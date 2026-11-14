package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/liujitcn/kratos-kit/auth/data"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/plugin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-admin/backend/pkg/aiplugin"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// SyncBuiltinTools 在初始化种子存在后同步内置API快照，不复制base_api开关和提示词。
func (c *AiPluginCase) SyncBuiltinTools(ctx context.Context) error {
	var err error
	query := c.Query(ctx).AiPlugin
	var row *models.AiPlugin
	row, err = c.Find(ctx, repository.Where(query.Code.Eq("builtin")))
	if err != nil {
		return err
	}
	if row.ID != 1 {
		return errorsx.Conflict("内置插件编号不符合初始化契约")
	}
	var tenants []*models.BaseTenant
	tenants, err = c.baseTenantRepo.List(ctx)
	if err != nil {
		return err
	}
	for _, tenant := range tenants {
		_, err = c.findInstallation(ctx, row.ID, tenant.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = c.installationRepo.Create(ctx, &models.AiPluginInstallation{TenantID: tenant.ID, PluginID: row.ID, Enabled: 1, Terminals: "admin,app", Revision: 1})
		}
		if err != nil {
			return err
		}
	}

	names := make([]string, 0)
	for _, terminal := range []string{"admin", "app"} {
		for _, registered := range c.aiRuntime.RegisteredTools(terminal) {
			var info *schema.ToolInfo
			info, err = registered.Info(ctx)
			if err != nil {
				return err
			}
			if c.manifestOwnsTool(ctx, terminal, info.Name) {
				continue
			}
			raw := []byte(`{"type":"object"}`)
			if info.ParamsOneOf != nil {
				var jsonSchema *jsonschema.Schema
				jsonSchema, err = info.ToJSONSchema()
				if err != nil {
					return err
				}
				raw, err = json.Marshal(jsonSchema)
				if err != nil {
					return err
				}
			}
			apiQuery := c.mcpCase.baseAPIRepo.Query(ctx).BaseAPI
			var apis []*models.BaseAPI
			apis, err = c.mcpCase.baseAPIRepo.List(ctx, repository.Where(apiQuery.ToolName.Eq(info.Name)))
			if err != nil {
				return err
			}
			if len(apis) == 0 {
				continue
			}
			effect := "read"
			for _, api := range apis {
				if !agentToolReadOnly(api) {
					effect = "write"
				}
			}
			names = append(names, info.Name)
			err = c.saveToolSnapshot(ctx, row.ID, "local", "API", apis[0].Operation, info.Name, info.Desc, string(raw), effect)
			if err != nil {
				return err
			}
		}
	}
	toolQuery := c.toolRepo.Query(ctx).AiPluginTool
	opts := []repository.QueryOption{repository.Where(toolQuery.PluginID.Eq(row.ID)), repository.Where(toolQuery.DiscoveryScope.Eq("local")), repository.Select(toolQuery.State)}
	if len(names) > 0 {
		opts = append(opts, repository.Where(toolQuery.SourceName.NotIn(names...)))
	}
	return c.toolRepo.Update(ctx, &models.AiPluginTool{State: "removed"}, opts...)

}

// ResolveAutomaticAppToolPool 自动组装当前用户获准在应用端使用的只读插件工具。
func (c *AiPluginCase) ResolveAutomaticAppToolPool(ctx context.Context) (agent.ToolPool, error) {
	var err error
	var catalog *basev1.ListAiPluginResponse
	catalog, err = c.ListAiPlugin(ctx, &basev1.ListAiPluginRequest{Terminal: basev1.Terminal_TERMINAL_APP})
	if err != nil {
		return agent.ToolPool{}, err
	}
	var accounts *basev1.ListAiPluginConnectionResponse
	accounts, err = c.ListAiPluginConnection(ctx, &basev1.ListAiPluginConnectionRequest{})
	if err != nil {
		return agent.ToolPool{}, err
	}
	tools := make(map[string][]*basev1.AiPluginTool, len(catalog.Plugins))
	for _, definition := range catalog.Plugins {
		if definition.UnavailableReason != "" {
			continue
		}
		var response *basev1.ListAiPluginToolResponse
		response, err = c.ListAiPluginTool(ctx, &basev1.ListAiPluginToolRequest{PluginId: definition.Id, Terminal: basev1.Terminal_TERMINAL_APP})
		if err != nil {
			return agent.ToolPool{}, err
		}
		tools[definition.Id] = response.Tools
	}
	var selections []*basev1.AiPluginSelection
	selections, err = automaticAppPluginSelections(catalog.Plugins, tools, accounts.Connections, time.Now().Unix())
	if err != nil {
		return agent.ToolPool{}, err
	}
	return c.ResolveToolPool(ctx, "app", selections)
}

// ResolveToolPool 对当前身份、安装、终端、账号和审核策略取交集；空选择产生空工具池。
func (c *AiPluginCase) ResolveToolPool(ctx context.Context, terminal string, selections []*basev1.AiPluginSelection) (agent.ToolPool, error) {
	var err error
	if terminal != "admin" && terminal != "app" {
		return agent.ToolPool{}, errorsx.PermissionDenied("执行终端无效")
	}
	if len(selections) > 64 {
		return agent.ToolPool{}, errorsx.InvalidArgument("本轮最多选用64个插件账号")
	}
	if selections == nil {
		selections = []*basev1.AiPluginSelection{{PluginId: "1"}}
	}
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return agent.ToolPool{}, err
	}
	frozen := make([]*basev1.AiPluginSelection, 0, len(selections))
	for _, selection := range selections {
		if selection == nil {
			return agent.ToolPool{}, errorsx.InvalidArgument("插件引用不能为空")
		}
		frozen = append(frozen, &basev1.AiPluginSelection{PluginId: selection.PluginId, ConnectionId: selection.ConnectionId})
	}
	var refs []dto.AiPluginResolvedTool
	refs, err = c.resolveToolRefs(ctx, terminal, frozen)
	if err != nil {
		return agent.ToolPool{}, err
	}
	pool := agent.ToolPool{Tools: make([]agent.Tool, 0, len(refs)), Revision: pluginPoolRevision(refs)}
	guard := &plugin.AccessGuard{}
	guard.Configs = func(callCtx context.Context, callTerminal string, names []string) (map[string]agent.ToolConfig, error) {
		var err error
		var caller *data.UserTokenPayload
		caller, err = c.GetAuthInfo(callCtx)
		if err != nil {
			return nil, err
		}
		if caller.TenantId != identity.TenantId || caller.UserId != identity.UserId {
			return nil, errorsx.PermissionDenied("工具池身份已变化")
		}
		var current []dto.AiPluginResolvedTool
		current, err = c.resolveToolRefs(callCtx, callTerminal, frozen)
		if err != nil {
			return nil, err
		}
		if callTerminal != terminal || pluginPoolRevision(current) != pool.Revision {
			return nil, errorsx.PermissionDenied("插件配置或授权已变化，请重新发起")
		}
		result := make(map[string]agent.ToolConfig)
		for _, ref := range current {
			if !slices.Contains(names, ref.Name) {
				continue
			}
			prompts := []string(nil)
			if ref.Policy != nil {
				prompts = parseToolPrompts(ref.Policy.Prompts)
			}
			result[ref.Name] = agent.ToolConfig{Enabled: true, Prompts: prompts}
		}
		var apiNames []string
		for _, ref := range current {
			if ref.Plugin.Source == "API" && slices.Contains(names, ref.Name) {
				apiNames = append(apiNames, ref.Name)
			}
		}
		if len(apiNames) > 0 {
			var apiConfigs map[string]agent.ToolConfig

			apiConfigs, err = c.mcpCase.ToolConfigs(callCtx, callTerminal, apiNames)
			if err != nil {
				return nil, err
			}
			for name, config := range apiConfigs {
				result[name] = config
			}
		}
		return result, nil
	}
	guard.Prepare = func(callCtx context.Context, callTerminal, name, arguments string) (string, bool, error) {
		var err error
		var caller *data.UserTokenPayload
		caller, err = c.GetAuthInfo(callCtx)
		if err != nil {
			return "", false, err
		}
		if caller.TenantId != identity.TenantId || caller.UserId != identity.UserId {
			return "", false, errorsx.PermissionDenied("工具池身份已变化")
		}
		var current []dto.AiPluginResolvedTool
		current, err = c.resolveToolRefs(callCtx, callTerminal, frozen)
		if err != nil {
			return "", false, err
		}
		if callTerminal != terminal || pluginPoolRevision(current) != pool.Revision {
			return "", false, errorsx.PermissionDenied("插件配置或授权已变化，请重新发起")
		}
		for _, ref := range current {
			if ref.Name == name {
				return c.prepareResolvedTool(callCtx, callTerminal, ref, arguments)
			}
		}
		return "", false, errorsx.PermissionDenied("工具未在本轮授权范围内")
	}
	pool.Checker = guard
	for _, ref := range refs {
		var base agent.Tool
		if ref.Plugin.Source == "MCP" {
			base = &plugin.RemoteTool{Name: ref.Name, Description: ref.Snapshot.Description, InputSchema: ref.Snapshot.InputSchema, Invoke: func(callCtx context.Context, arguments string) (*mcp.CallToolResult, error) {
				var err error
				_, _, err = guard.PrepareToolCall(callCtx, terminal, ref.Name, arguments)
				if err != nil {
					return nil, err
				}
				return c.invokeRemoteTool(callCtx, terminal, ref, arguments)
			}}
		} else {
			base = ref.Binding.Tool
		}
		pool.Tools = append(pool.Tools, base)
	}
	return pool, nil
}

// resolveToolRefs 重建当前可执行引用；配置缺失或账号越权绝不扩大到其他连接。
func (c *AiPluginCase) resolveToolRefs(ctx context.Context, terminal string, selections []*basev1.AiPluginSelection) ([]dto.AiPluginResolvedTool, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.AiPluginResolvedTool, 0)
	names := make(map[string]bool)
	for _, selection := range selections {
		var row *models.AiPlugin
		row, err = c.visiblePlugin(ctx, selection.PluginId)
		if err != nil {
			return nil, err
		}
		if !pluginCanUse(row, identity) {
			return nil, errorsx.PermissionDenied("个人插件仅允许本人使用")
		}
		var installation *models.AiPluginInstallation
		installation, err = c.activeInstallation(ctx, row, identity.TenantId, terminal)
		if err != nil {
			return nil, err
		}
		var approvedRevision string
		approvedRevision, err = c.approvedMCPRevision(ctx, row, identity.TenantId, terminal)
		if err != nil {
			return nil, err
		}
		scope := "local"
		var connection *models.AiPluginConnection
		if row.Source == "MCP" {
			connection, err = c.ownedConnection(ctx, selection.ConnectionId, false)
			if err != nil {
				return nil, err
			}
			if connection.PluginID != row.ID || connection.InstallationID != installation.ID || connection.State != "connected" {
				return nil, errorsx.PermissionDenied("所选账号与插件不匹配或连接已失效")
			}
			scope = selection.ConnectionId
		} else if selection.ConnectionId != "" {
			return nil, errorsx.InvalidArgument("本地插件不能指定远程账号")
		}
		query := c.toolRepo.Query(ctx).AiPluginTool
		var snapshots []*models.AiPluginTool
		snapshots, err = c.toolRepo.List(ctx, repository.Where(query.PluginID.Eq(row.ID)), repository.Where(query.DiscoveryScope.Eq(scope)), repository.Where(query.State.Eq("discovered")), repository.Order(query.ID.Asc()))
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots {
			ref := dto.AiPluginResolvedTool{Plugin: row, Installation: installation, Snapshot: snapshot, Connection: connection, Name: snapshot.SourceName}
			if row.Source == "MCP" {
				sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", row.ID, connection.ID, snapshot.SourceName)))
				ref.Name = "mcp_" + hex.EncodeToString(sum[:24])
			}
			if row.Source != "API" {
				ref.Policy, err = c.findToolPolicy(ctx, installation.ID, snapshot.SourceName)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				if err != nil {
					return nil, err
				}
				if ref.Policy.Allowed != 1 || ref.Policy.ContractHash != snapshot.ContractHash {
					continue
				}
				if terminal == "app" && ref.Policy.Effect != "read" {
					continue
				}
			}
			if row.Source != "MCP" {
				ref.Binding = c.localBinding(ctx, row.Code, terminal, snapshot.SourceName)
				if ref.Binding == nil {
					continue
				}
				if row.Source == "LOCAL" && terminal == "app" && ref.Binding.Effect != "read" {
					continue
				}
			}
			apiRevision := ""
			if row.Source == "API" {
				var configs map[string]agent.ToolConfig

				configs, err = c.mcpCase.ToolConfigs(ctx, terminal, []string{snapshot.SourceName})
				if err != nil {
					return nil, err
				}
				if !configs[snapshot.SourceName].Enabled {
					continue
				}
				apiQuery := c.mcpCase.baseAPIRepo.Query(ctx).BaseAPI
				var apis []*models.BaseAPI
				apis, err = c.mcpCase.baseAPIRepo.List(ctx, repository.Where(apiQuery.ToolName.Eq(snapshot.SourceName)), repository.Order(apiQuery.ID.Asc()))
				if err != nil {
					return nil, err
				}
				for _, api := range apis {
					apiRevision += fmt.Sprintf(":api%d:%d:%d:%s", api.ID, api.AgentStatus, api.McpStatus, api.ToolPrompts)
				}

			}
			if names[ref.Name] {
				return nil, errorsx.Conflict("本轮工具名称重复，请只选择一个确定账号")
			}
			names[ref.Name] = true
			ref.Revision = fmt.Sprintf("%d:%d:%d:%d:%d:%s", row.ID, row.Revision, installation.ID, installation.Revision, snapshot.Revision, snapshot.ContractHash)
			ref.Revision += ":" + approvedRevision
			ref.Revision += apiRevision
			if row.Code != "builtin" && row.Source != "MCP" {
				c.manifestMu.RLock()
				manifest := c.manifests[row.Code]
				c.manifestMu.RUnlock()
				ref.Revision += fmt.Sprintf(":manifest%d:%s:%s", manifest.Revision, ref.Binding.Effect, ref.Binding.Operation)
			}

			if ref.Policy != nil {
				ref.Revision += fmt.Sprintf(":p%d:%d", ref.Policy.ID, ref.Policy.Revision)
			}
			if connection != nil {
				ref.Revision += fmt.Sprintf(":c%d:%d", connection.ID, connection.Revision)
			}
			result = append(result, ref)
		}
	}
	return result, nil
}

// prepareResolvedTool 对每次执行应用来源特定校验；未知效果必须经过现有人工确认。
func (c *AiPluginCase) prepareResolvedTool(ctx context.Context, terminal string, ref dto.AiPluginResolvedTool, arguments string) (string, bool, error) {
	var err error
	if ref.Plugin.Source == "API" {
		return c.mcpCase.PrepareToolCall(ctx, terminal, ref.Snapshot.SourceName, arguments)
	}
	if err = plugin.ValidateArguments(ref.Snapshot.InputSchema, arguments); err != nil {
		return "", false, errorsx.InvalidArgument("工具参数不符合已审查契约").WithCause(err)
	}
	requiresApproval := ref.Policy.Effect != "read" || (ref.Binding != nil && ref.Binding.Effect != "read")
	if terminal == "app" && requiresApproval {
		return "", false, errorsx.PermissionDenied("请在电脑端确认写入操作")
	}
	if ref.Plugin.Source == "LOCAL" {
		var prepared string
		prepared, err = ref.Binding.Validate(ctx, arguments)
		return prepared, requiresApproval, err
	}
	return arguments, requiresApproval, nil
}

// invokeRemoteTool 执行前完整发现并核对冻结契约，写入只发送一次且不自动重放。
func (c *AiPluginCase) invokeRemoteTool(ctx context.Context, terminal string, ref dto.AiPluginResolvedTool, arguments string) (*mcp.CallToolResult, error) {
	var err error
	var config plugin.Config
	config, err = plugin.ParseConfig(ref.Plugin.ConfigJSON)
	if err != nil {
		return nil, err
	}
	var token string
	token, err = c.connectionToken(ctx, ref.Plugin, ref.Connection, config)
	if err != nil {
		return nil, err
	}
	var tools []*mcp.Tool
	tools, err = plugin.Discover(ctx, ref.Plugin.Endpoint, token, config)
	if err != nil {
		if errors.Is(err, plugin.ErrReauthorizationRequired) {
			err = c.failConnection(ctx, ref.Connection, "reauthorization_required", "远端授权已失效，请重新授权")
			if err != nil {
				return nil, err
			}
			return nil, errorsx.PermissionDenied("连接授权已失效，请重新授权")
		}
		return nil, errorsx.InvalidArgument("工具发现失败，请检测连接后重新发起")
	}
	valid := false
	for _, remote := range tools {
		if remote.Name != ref.Snapshot.SourceName {
			continue
		}
		var raw []byte
		raw, err = json.Marshal(remote.InputSchema)
		if err != nil {
			return nil, err
		}
		var hash string
		hash, err = plugin.ContractHash(remote.Name, remote.Description, string(raw))
		if err != nil {
			return nil, err
		}
		valid = hash == ref.Snapshot.ContractHash
	}
	if !valid {
		return nil, errorsx.PermissionDenied("远端工具已移除或契约变化，请刷新并重新审查")
	}
	var fresh []dto.AiPluginResolvedTool
	fresh, err = c.resolveToolRefs(ctx, terminal, []*basev1.AiPluginSelection{{PluginId: strconv.FormatInt(ref.Plugin.ID, 10), ConnectionId: strconv.FormatInt(ref.Connection.ID, 10)}})
	if err != nil {
		return nil, err
	}
	current := false
	for _, item := range fresh {
		if item.Name == ref.Name && item.Revision == ref.Revision {
			current = true
		}
	}
	if !current {
		return nil, errorsx.PermissionDenied("执行前授权配置已变化，请重新发起")
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "kratos-admin", Version: "1"}, nil)
	var session *mcp.ClientSession
	session, err = client.Connect(ctx, plugin.Transport(ref.Plugin.Endpoint, token, config), nil)
	if err != nil {
		if errors.Is(err, plugin.ErrReauthorizationRequired) {
			err = c.failConnection(ctx, ref.Connection, "reauthorization_required", "远端授权已失效，请重新授权")
			if err != nil {
				return nil, err
			}
			return nil, errorsx.PermissionDenied("连接授权已失效，请重新授权")
		}
		return nil, errorsx.InvalidArgument("连接失败，请检测账号后重新发起")
	}
	// 关闭仅释放传输会话；远端清理失败不能把已发送操作当作可重试失败。
	defer session.Close()
	var values map[string]any
	err = json.Unmarshal([]byte(arguments), &values)
	if err != nil {
		return nil, err
	}
	var result *mcp.CallToolResult
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: ref.Snapshot.SourceName, Arguments: values})
	if err != nil {
		if errors.Is(err, plugin.ErrReauthorizationRequired) {
			err = c.failConnection(ctx, ref.Connection, "reauthorization_required", "远端授权已失效，请重新授权")
			if err != nil {
				return nil, err
			}
			return nil, errorsx.PermissionDenied("连接授权已失效，请重新授权")
		}
		return nil, errorsx.Internal("远端操作结果未知，请核对外部系统；本系统不会自动重试")
	}
	return result, nil
}

// localBinding 只从宿主显式登记或既有Proto执行器绑定工具，配置不能注入代码。
func (c *AiPluginCase) localBinding(ctx context.Context, code, terminal, name string) *aiplugin.ToolBinding {
	var err error
	if code == "builtin" {
		if c.manifestOwnsTool(ctx, terminal, name) {
			return nil
		}
		for _, registered := range c.aiRuntime.RegisteredTools(terminal) {
			var info *schema.ToolInfo
			info, err = registered.Info(ctx)
			if err == nil && info.Name == name {
				return &aiplugin.ToolBinding{Terminal: terminal, Tool: registered}
			}
		}
		return nil
	}
	c.manifestMu.RLock()
	defer c.manifestMu.RUnlock()
	manifest, exists := c.manifests[code]
	if !exists {
		return nil
	}
	for _, binding := range manifest.Tools {
		if binding.Terminal != terminal {
			continue
		}
		var info *schema.ToolInfo
		info, err = binding.Tool.Info(ctx)
		if err == nil && info.Name == name {
			return &binding
		}
	}
	return nil
}

// manifestOwnsTool 排除已归属可选组件的工具，避免通过内置集合绕过插件停用。
func (c *AiPluginCase) manifestOwnsTool(ctx context.Context, terminal, name string) bool {
	var err error
	c.manifestMu.RLock()
	defer c.manifestMu.RUnlock()
	for _, manifest := range c.manifests {
		for _, binding := range manifest.Tools {
			if binding.Terminal != terminal {
				continue
			}
			var info *schema.ToolInfo
			info, err = binding.Tool.Info(ctx)
			if err == nil && info.Name == name {
				return true
			}
		}
	}
	return false
}

// pluginPoolRevision 确定性绑定当前工具和所有业务配置版本，不保存凭据。
func pluginPoolRevision(refs []dto.AiPluginResolvedTool) string {
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, ref.Name+":"+ref.Revision)
	}
	slices.Sort(values)
	sum := sha256.Sum256([]byte(strings.Join(values, "\n")))
	return hex.EncodeToString(sum[:])
}

// pluginTerminalAllowed 严格匹配持久化终端集合。
func pluginTerminalAllowed(terminals, terminal string) bool {
	return (terminal == "admin" || terminal == "app") && slices.Contains(strings.Split(terminals, ","), terminal)
}

// automaticAppPluginSelections 从公开目录和本人可见账号中筛选已审核的只读引用，保持具体账号归属。
func automaticAppPluginSelections(definitions []*basev1.AiPlugin, tools map[string][]*basev1.AiPluginTool, accounts []*basev1.AiPluginConnection, now int64) ([]*basev1.AiPluginSelection, error) {
	connections := make(map[string]*basev1.AiPluginConnection, len(accounts))
	for _, account := range accounts {
		if account.State == "connected" && (account.ExpiresAt == 0 || account.ExpiresAt > now) {
			connections[account.Id] = account
		}
	}
	selections := make([]*basev1.AiPluginSelection, 0)
	seen := make(map[string]bool)
	for _, definition := range definitions {
		if definition.UnavailableReason != "" || !definition.Enabled || !definition.Assembled || !definition.GetInstallation().GetEnabled() || !slices.Contains(definition.GetInstallation().GetTerminals(), "app") {
			continue
		}
		for _, tool := range tools[definition.Id] {
			if tool.PluginId != definition.Id || !tool.EffectiveEnabled || tool.Effect != "read" || tool.State != "discovered" {
				continue
			}
			if definition.Source == "MCP" {
				account := connections[tool.ConnectionId]
				if account == nil || account.PluginId != definition.Id {
					continue
				}
			} else if tool.ConnectionId != "" {
				continue
			}
			key := definition.Id + ":" + tool.ConnectionId
			if seen[key] {
				continue
			}
			seen[key] = true
			selections = append(selections, &basev1.AiPluginSelection{PluginId: definition.Id, ConnectionId: tool.ConnectionId})
		}
	}
	if len(selections) > 64 {
		return nil, errorsx.InvalidArgument("可自动使用的插件账号超过64个，请在管理端调整可用范围")
	}
	return selections, nil
}
