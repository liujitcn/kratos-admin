package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-kit/auth/data"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/plugin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListAiPluginTool 按会话工具能力查询可见账号快照，沿用原 API 权限和审核策略。
func (c *AiPluginCase) ListAiPluginTool(ctx context.Context, req *basev1.ListAiPluginToolRequest) (*basev1.ListAiPluginToolResponse, error) {
	var err error
	var row *models.AiPlugin
	row, err = c.visiblePlugin(ctx, req.PluginId)
	if err != nil {
		return nil, err
	}
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var installation *models.AiPluginInstallation
	installation, err = c.findInstallation(ctx, row.ID, identity.TenantId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	query := c.toolRepo.Query(ctx).AiPluginTool
	opts := []repository.QueryOption{repository.Where(query.PluginID.Eq(row.ID)), repository.Order(query.ID.Asc())}
	var rows []*models.AiPluginTool
	rows, err = c.toolRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	terminal := ai.NormalizeTerminalString(ai.NormalizeTerminal(req.Terminal))
	approvalUnavailable := ""
	_, err = c.approvedMCPRevision(ctx, row, identity.TenantId, terminal)
	if err != nil {
		if kratosErrors.FromError(err).Reason != errorsx.ReasonPermissionDenied {
			return nil, err
		}
		approvalUnavailable = "平台或租户接入策略已撤销或变化，请重建自定义定义"
	}
	result := &basev1.ListAiPluginToolResponse{}
	for _, snapshot := range rows {
		if snapshot.DiscoveryScope != "local" {
			_, err = c.ownedConnection(ctx, snapshot.DiscoveryScope, false)
			if err != nil {
				if !pluginPlatform(identity) && identity.RoleCode != "tenant" {
					continue
				}
				var connectionID int64
				connectionID, err = strconv.ParseInt(snapshot.DiscoveryScope, 10, 64)
				if err != nil {
					continue
				}
				connectionQuery := c.connectionRepo.Query(ctx).AiPluginConnection
				_, err = c.connectionRepo.Find(ctx, repository.Where(connectionQuery.ID.Eq(connectionID)), repository.Where(connectionQuery.TenantID.Eq(identity.TenantId)))
				if err != nil {
					continue
				}
			}
		}
		item := &basev1.AiPluginTool{Id: strconv.FormatInt(snapshot.ID, 10), PluginId: req.PluginId, SourceName: snapshot.SourceName, Description: snapshot.Description, InputSchema: snapshot.InputSchema, ContractHash: snapshot.ContractHash, State: snapshot.State, Effect: snapshot.Effect}
		if snapshot.DiscoveryScope != "local" {
			item.ConnectionId = snapshot.DiscoveryScope
		}
		if row.Source == "API" {
			apiQuery := c.mcpCase.baseAPIRepo.Query(ctx).BaseAPI
			var apis []*models.BaseAPI
			apis, err = c.mcpCase.baseAPIRepo.List(ctx, repository.Where(apiQuery.ToolName.Eq(snapshot.SourceName)))
			if err != nil {
				return nil, err
			}
			item.Allowed = len(apis) > 0
			for _, api := range apis {
				item.Allowed = item.Allowed && api.AgentStatus == 1 && !agentToolBlocked(api)
				item.Prompts = append(item.Prompts, parseToolPrompts(api.ToolPrompts)...)
			}
			item.Revision = snapshot.Revision
		} else if installation != nil {
			var policy *models.AiPluginToolPolicy
			policy, err = c.findToolPolicy(ctx, installation.ID, snapshot.SourceName)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			if policy != nil {
				item.Revision = policy.Revision
				item.Effect = policy.Effect
				item.Prompts = parseToolPrompts(policy.Prompts)
				item.Allowed = policy.Allowed == 1 && policy.ContractHash == snapshot.ContractHash
			}
		}
		item.EffectiveEnabled = item.Allowed && row.Enabled == 1 && installation != nil && installation.Enabled == 1 && snapshot.State == "discovered" && pluginCanUse(row, identity) && approvalUnavailable == ""
		if installation != nil && !pluginTerminalAllowed(installation.Terminals, terminal) {
			item.EffectiveEnabled = false
		}
		if terminal == "app" && item.Effect != "read" {
			item.EffectiveEnabled = false
		}
		if item.EffectiveEnabled && row.Source == "API" {
			var configs map[string]agent.ToolConfig

			configs, err = c.mcpCase.ToolConfigs(ctx, terminal, []string{snapshot.SourceName})
			if err != nil {
				return nil, err
			}
			item.EffectiveEnabled = configs[snapshot.SourceName].Enabled
		}
		if item.EffectiveEnabled && snapshot.DiscoveryScope != "local" {
			var connection *models.AiPluginConnection
			connection, err = c.ownedConnection(ctx, snapshot.DiscoveryScope, false)
			item.EffectiveEnabled = err == nil && connection.State == "connected"
		}
		if item.EffectiveEnabled && row.Source != "MCP" {
			binding := c.localBinding(ctx, row.Code, terminal, snapshot.SourceName)
			if binding == nil || row.Source == "LOCAL" && terminal == "app" && binding.Effect != "read" {
				item.EffectiveEnabled = false
			}
		}
		if !item.EffectiveEnabled {
			item.UnavailableReason = "插件、终端、连接或工具契约尚未获准"
			if approvalUnavailable != "" {
				item.UnavailableReason = approvalUnavailable
			}
		}
		result.Tools = append(result.Tools, item)
	}
	return result, nil
}

// SetAiPluginToolPolicy 以当前契约哈希和修订号审查，API仍修改原平台开关与提示词。
func (c *AiPluginCase) SetAiPluginToolPolicy(ctx context.Context, req *basev1.SetAiPluginToolPolicyRequest) (*basev1.AiPluginTool, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.pluginAdministrator(ctx)
	if err != nil {
		return nil, err
	}
	var result *basev1.AiPluginTool
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var row *models.AiPlugin
		row, err = c.visiblePlugin(txCtx, req.PluginId, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		var installation *models.AiPluginInstallation
		installation, err = c.findInstallation(txCtx, row.ID, identity.TenantId)
		if err != nil {
			return errorsx.PermissionDenied("租户尚未安装插件")
		}
		var toolID int64
		toolID, err = strconv.ParseInt(req.ToolId, 10, 64)
		if err != nil || toolID <= 0 {
			return errorsx.InvalidArgument("工具编号不合法")
		}
		query := c.toolRepo.Query(txCtx).AiPluginTool
		var snapshot *models.AiPluginTool
		snapshot, err = c.toolRepo.Find(txCtx, repository.Where(query.ID.Eq(toolID)), repository.Where(query.PluginID.Eq(row.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return errorsx.ResourceNotFound("工具快照不存在")
		}
		if snapshot.DiscoveryScope != "local" {
			var connectionID int64
			connectionID, err = strconv.ParseInt(snapshot.DiscoveryScope, 10, 64)
			if err != nil {
				return errorsx.InvalidArgument("发现范围不合法")
			}
			connectionQuery := c.connectionRepo.Query(txCtx).AiPluginConnection
			_, err = c.connectionRepo.Find(txCtx, repository.Where(connectionQuery.ID.Eq(connectionID)), repository.Where(connectionQuery.TenantID.Eq(identity.TenantId)))
			if err != nil {
				return errorsx.PermissionDenied("工具快照不属于当前租户")
			}
		}
		if snapshot.ContractHash != req.ContractHash || snapshot.State != "discovered" {
			return errorsx.Conflict("工具契约已变化，请重新审查")
		}
		var prompts []byte
		prompts, err = json.Marshal(req.Prompts)
		if err != nil {
			return err
		}
		revision := int64(1)
		if row.Source == "API" {
			if !pluginPlatform(identity) {
				return errorsx.PermissionDenied("API工具开关由平台管理员维护")
			}
			if snapshot.Revision != req.Revision {
				return errorsx.Conflict("工具配置已变化")
			}
			apiQuery := c.mcpCase.baseAPIRepo.Query(txCtx).BaseAPI
			api := &models.BaseAPI{AgentStatus: pluginBool(req.Allowed), ToolPrompts: string(prompts)}
			err = c.mcpCase.baseAPIRepo.Update(txCtx, api, repository.Where(apiQuery.ToolName.Eq(snapshot.SourceName)), repository.Select(apiQuery.AgentStatus, apiQuery.ToolPrompts))
			if err != nil {
				return err
			}
			snapshot.Revision++
			revision = snapshot.Revision
			err = c.toolRepo.Update(txCtx, snapshot, repository.Where(query.ID.Eq(snapshot.ID)), repository.Select(query.Revision))
		} else {
			var policy *models.AiPluginToolPolicy
			policy, err = c.findToolPolicy(txCtx, installation.ID, snapshot.SourceName)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if policy == nil {
				if req.Revision != 0 {
					return errorsx.Conflict("工具策略已变化")
				}
				policy = &models.AiPluginToolPolicy{InstallationID: installation.ID, ToolKey: snapshot.SourceName, Allowed: pluginBool(req.Allowed), ContractHash: snapshot.ContractHash, Effect: req.Effect, Prompts: string(prompts), Revision: 1}
				err = c.policyRepo.Create(txCtx, policy)
			} else {
				if policy.Revision != req.Revision {
					return errorsx.Conflict("工具策略已变化")
				}
				policy.Allowed = pluginBool(req.Allowed)
				policy.ContractHash = snapshot.ContractHash
				policy.Effect = req.Effect
				policy.Prompts = string(prompts)
				policy.Revision++
				policyQuery := c.policyRepo.Query(txCtx).AiPluginToolPolicy
				err = c.policyRepo.Update(txCtx, policy, repository.Where(policyQuery.ID.Eq(policy.ID)), repository.Select(policyQuery.Allowed, policyQuery.ContractHash, policyQuery.Effect, policyQuery.Prompts, policyQuery.Revision))
			}
			revision = policy.Revision
		}
		if err != nil {
			return err
		}
		installation.Revision++
		installationQuery := c.installationRepo.Query(txCtx).AiPluginInstallation
		err = c.installationRepo.Update(txCtx, installation, repository.Where(installationQuery.ID.Eq(installation.ID)), repository.Select(installationQuery.Revision))
		if err != nil {
			return err
		}
		result = &basev1.AiPluginTool{Id: req.ToolId, PluginId: req.PluginId, SourceName: snapshot.SourceName, Description: snapshot.Description, InputSchema: snapshot.InputSchema, ContractHash: snapshot.ContractHash, State: snapshot.State, Allowed: req.Allowed, Effect: req.Effect, Revision: revision, Prompts: req.Prompts}
		return nil
	})
	return result, err
}

// refreshConnection 刷新只在完整发现成功后提交快照；失败使连接不可执行而不删除旧工具。
func (c *AiPluginCase) refreshConnection(ctx context.Context, row *models.AiPlugin, installation *models.AiPluginInstallation, connection *models.AiPluginConnection) error {
	var err error
	var config plugin.Config
	config, err = plugin.ParseConfig(row.ConfigJSON)
	if err != nil {
		return err
	}
	var token string
	token, err = c.connectionToken(ctx, row, connection, config)
	if err != nil {
		err = c.failConnection(ctx, connection, "reauthorization_required", "授权已失效，请重新连接")
		if err != nil {
			return err
		}
		return errorsx.PermissionDenied("授权已失效，请重新连接")
	}
	var tools []*mcp.Tool
	tools, err = plugin.Discover(ctx, row.Endpoint, token, config)
	if err != nil {
		state := "error"
		message := "握手或完整工具发现失败，请检查端点和授权"
		if errors.Is(err, plugin.ErrReauthorizationRequired) {
			state = "reauthorization_required"
		} else if plugin.IsAuthenticationRejected(err) {
			switch connection.AuthType {
			case "none":
				message = "远端要求Bearer Token，请选择token认证并录入API令牌"
			case "token":
				message = "远端拒绝当前令牌，请重新录入API令牌"
			case "oauth":
				message = "远端拒绝当前授权，请重新授权"
			}
		}
		err = c.failConnection(ctx, connection, state, message)
		if err != nil {
			return err
		}
		return errorsx.InvalidArgument(message)
	}
	var refreshed *models.AiPluginConnection
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		query := c.connectionRepo.Query(txCtx).AiPluginConnection
		var locked *models.AiPluginConnection
		locked, err = c.connectionRepo.Find(txCtx, repository.Where(query.ID.Eq(connection.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if locked.Revision != connection.Revision {
			return errorsx.Conflict("发现期间连接已变化")
		}
		scope := strconv.FormatInt(connection.ID, 10)
		names := make([]string, 0, len(tools))
		for _, remote := range tools {
			var raw []byte
			raw, err = json.Marshal(remote.InputSchema)
			if err != nil {
				return err
			}
			_, err = plugin.CompileSchema(string(raw))
			if err != nil {
				return errorsx.InvalidArgument("远端工具Schema不支持，完整快照未更新").WithCause(err)
			}
			err = c.saveToolSnapshot(txCtx, row.ID, scope, "MCP", "", remote.Name, remote.Description, string(raw), "unknown")
			if err != nil {
				return err
			}
			names = append(names, remote.Name)
		}
		toolQuery := c.toolRepo.Query(txCtx).AiPluginTool
		opts := []repository.QueryOption{repository.Where(toolQuery.PluginID.Eq(row.ID)), repository.Where(toolQuery.DiscoveryScope.Eq(scope))}
		if len(names) > 0 {
			opts = append(opts, repository.Where(toolQuery.SourceName.NotIn(names...)))
		}
		err = c.toolRepo.Update(txCtx, &models.AiPluginTool{State: "removed"}, append(opts, repository.Select(toolQuery.State))...)
		if err != nil {
			return err
		}
		locked.State = "connected"
		locked.LastError = ""
		locked.Revision++
		err = c.saveConnection(txCtx, locked)
		if err != nil {
			return err
		}
		refreshed = locked
		return nil
	})
	if err != nil {
		snapshotErr := err
		err = c.failConnection(ctx, connection, "error", "完整工具发现或Schema校验失败，旧快照已停止使用")
		return errors.Join(snapshotErr, err)
	}
	*connection = *refreshed
	return nil
}

// connectionToken 在连接行锁内刷新OAuth，跨实例不会并发覆盖刷新令牌；不重放工具调用。
func (c *AiPluginCase) connectionToken(ctx context.Context, row *models.AiPlugin, connection *models.AiPluginConnection, config plugin.Config) (string, error) {
	var err error
	if connection.AuthType == "none" {
		return "", nil
	}
	var credential dto.AiPluginCredential
	credential, err = openPluginCredential(ctx, connection)
	if err != nil {
		return "", err
	}
	if connection.AuthType == "token" {
		if credential.Token == "" {
			return "", errorsx.PermissionDenied("连接令牌已撤销")
		}
		return credential.Token, nil
	}
	if credential.OAuth == nil || config.OAuth == nil {
		return "", errorsx.PermissionDenied("需要重新授权")
	}
	if credential.OAuth.Valid() {
		return credential.OAuth.AccessToken, nil
	}
	token := ""
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		query := c.connectionRepo.Query(txCtx).AiPluginConnection
		var locked *models.AiPluginConnection
		locked, err = c.connectionRepo.Find(txCtx, repository.Where(query.ID.Eq(connection.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if locked.Revision != connection.Revision || locked.State == "disconnected" {
			return errorsx.PermissionDenied("连接已被撤销或修改")
		}
		var current dto.AiPluginCredential
		current, err = openPluginCredential(txCtx, locked)
		if err != nil {
			return err
		}
		if current.OAuth == nil {
			return errorsx.PermissionDenied("需要重新授权")
		}
		oauthConfig := pluginOAuthConfig(config.OAuth)
		var refreshed *oauth2.Token
		refreshed, err = oauthConfig.TokenSource(context.WithValue(txCtx, oauth2.HTTPClient, config.Client("")), current.OAuth).Token()
		if err != nil {
			return errorsx.PermissionDenied("OAuth续期失败，请重新授权")
		}
		current.OAuth = refreshed
		locked.Credential, err = sealPluginCredential(txCtx, locked, current)
		if err != nil {
			return err
		}
		locked.ExpiresAt = 0
		if !refreshed.Expiry.IsZero() {
			locked.ExpiresAt = refreshed.Expiry.Unix()
		}
		if scopes, ok := refreshed.Extra("scope").(string); ok && scopes != locked.Scopes {
			locked.Scopes = scopes
			locked.Revision++
		}
		err = c.saveConnection(txCtx, locked)
		if err != nil {
			return err
		}
		*connection = *locked
		token = refreshed.AccessToken
		return nil
	})
	return token, err
}

// saveToolSnapshot 以连接独立契约保存工具，哈希变化不会继承旧审批。
func (c *AiPluginCase) saveToolSnapshot(ctx context.Context, pluginID int64, scope, source, operation, name, description, inputSchema, effect string) error {
	var err error
	var hash string
	hash, err = plugin.ContractHash(name, description, inputSchema)
	if err != nil {
		return err
	}
	query := c.toolRepo.Query(ctx).AiPluginTool
	opts := []repository.QueryOption{repository.Where(query.PluginID.Eq(pluginID)), repository.Where(query.DiscoveryScope.Eq(scope)), repository.Where(query.SourceName.Eq(name))}
	var row *models.AiPluginTool
	row, err = c.toolRepo.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.toolRepo.Create(ctx, &models.AiPluginTool{PluginID: pluginID, DiscoveryScope: scope, SourceName: name, Source: source, Operation: operation, Description: description, InputSchema: inputSchema, ContractHash: hash, Effect: effect, State: "discovered", Revision: 1})
	}
	if err != nil {
		return err
	}
	if row.ContractHash == hash && row.State == "discovered" {
		return nil
	}
	row.ContractHash = hash
	row.Description = description
	row.InputSchema = inputSchema
	row.Effect = effect
	row.State = "discovered"
	row.Revision++
	return c.toolRepo.Update(ctx, row, append(opts, repository.Select(query.ContractHash, query.Description, query.InputSchema, query.Effect, query.State, query.Revision))...)
}

// findToolPolicy 查询租户安装中已审核的稳定工具键。
func (c *AiPluginCase) findToolPolicy(ctx context.Context, installationID int64, key string) (*models.AiPluginToolPolicy, error) {
	query := c.policyRepo.Query(ctx).AiPluginToolPolicy
	opts := []repository.QueryOption{repository.Where(query.InstallationID.Eq(installationID)), repository.Where(query.ToolKey.Eq(key))}
	return c.policyRepo.Find(ctx, opts...)
}

// failConnection 只在原修订仍有效时登记失败，绝不让迟到的网络响应复活已撤销账号。
func (c *AiPluginCase) failConnection(ctx context.Context, connection *models.AiPluginConnection, state, message string) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		query := c.connectionRepo.Query(txCtx).AiPluginConnection
		var locked *models.AiPluginConnection
		locked, err = c.connectionRepo.Find(txCtx, repository.Where(query.ID.Eq(connection.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if locked.Revision != connection.Revision {
			return errorsx.Conflict("连接已变化，请刷新状态")
		}
		locked.State = state
		locked.LastError = message
		locked.Revision++
		err = c.saveConnection(txCtx, locked)
		if err != nil {
			return err
		}
		*connection = *locked
		return nil
	})
}
