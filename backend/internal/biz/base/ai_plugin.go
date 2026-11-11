package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/plugin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/aiplugin"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AiPluginCase 管理插件、租户策略、账号连接及请求工具池。
type AiPluginCase struct {
	*biz.BaseCase
	*data.AiPluginRepository
	tx               data.Transaction
	installationRepo *data.AiPluginInstallationRepository
	toolRepo         *data.AiPluginToolRepository
	policyRepo       *data.AiPluginToolPolicyRepository
	connectionRepo   *data.AiPluginConnectionRepository
	baseTenantRepo   *data.BaseTenantRepository
	aiRuntime        *ai.Runtime
	mcpCase          *McpCase
	manifestMu       sync.RWMutex
	manifests        map[string]aiplugin.Manifest
}

// NewAiPluginCase 创建插件业务并注入生成仓储与原有 API 权限边界。
func NewAiPluginCase(baseCase *biz.BaseCase, tx data.Transaction, pluginRepo *data.AiPluginRepository, installationRepo *data.AiPluginInstallationRepository, toolRepo *data.AiPluginToolRepository, policyRepo *data.AiPluginToolPolicyRepository, connectionRepo *data.AiPluginConnectionRepository, baseTenantRepo *data.BaseTenantRepository, aiRuntime *ai.Runtime, mcpCase *McpCase) *AiPluginCase {
	return &AiPluginCase{BaseCase: baseCase, AiPluginRepository: pluginRepo, tx: tx, installationRepo: installationRepo, toolRepo: toolRepo, policyRepo: policyRepo, connectionRepo: connectionRepo, baseTenantRepo: baseTenantRepo, aiRuntime: aiRuntime, mcpCase: mcpCase, manifests: make(map[string]aiplugin.Manifest)}
}

// RegisterManifest 启动时登记唯一工具归属，并同步本地装配的持久化目录与契约快照。
func (c *AiPluginCase) RegisterManifest(manifest aiplugin.Manifest) error {
	var err error
	if manifest.Code == "" || manifest.Code == "builtin" || manifest.Name == "" || (manifest.Source != "API" && manifest.Source != "LOCAL") || manifest.Revision <= 0 {
		return fmt.Errorf("本地插件清单不合法")
	}
	c.manifestMu.Lock()
	defer c.manifestMu.Unlock()
	if _, exists := c.manifests[manifest.Code]; exists {
		return fmt.Errorf("插件编码重复: %s", manifest.Code)
	}
	names := make(map[string]bool)
	ctx := context.Background()
	for _, binding := range manifest.Tools {
		if binding.Tool == nil || (binding.Terminal != "admin" && binding.Terminal != "app") || (binding.Effect != "read" && binding.Effect != "write" && binding.Effect != "unknown") || (manifest.Source == "LOCAL" && binding.Validate == nil) {
			return fmt.Errorf("插件工具绑定不合法")
		}
		var info *schema.ToolInfo
		info, err = binding.Tool.Info(ctx)
		if err != nil {
			return err
		}
		for _, registered := range c.aiRuntime.RegisteredTools(binding.Terminal) {
			var registeredInfo *schema.ToolInfo
			registeredInfo, err = registered.Info(ctx)
			if err != nil {
				return err
			}
			if registeredInfo.Name == info.Name {
				return fmt.Errorf("工具执行名与已装配工具重复: %s", info.Name)
			}
		}
		key := binding.Terminal + ":" + info.Name
		if info.Name == "tool_search" || names[key] {
			return fmt.Errorf("插件占用保留名或重复工具: %s", info.Name)
		}
		names[key] = true
		for _, registered := range c.manifests {
			for _, other := range registered.Tools {
				if other.Terminal != binding.Terminal {
					continue
				}
				var otherInfo *schema.ToolInfo
				otherInfo, err = other.Tool.Info(ctx)
				if err != nil {
					return err
				}
				if otherInfo.Name == info.Name {
					return fmt.Errorf("工具归属冲突: %s", info.Name)
				}
			}
		}
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		query := c.Query(txCtx).AiPlugin
		var row *models.AiPlugin
		row, err = c.Find(txCtx, repository.Where(query.Code.Eq(manifest.Code)))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = &models.AiPlugin{Code: manifest.Code, Name: manifest.Name, Description: manifest.Description, Source: manifest.Source, Visibility: "system", Enabled: 1, ConfigJSON: "{}", Revision: manifest.Revision}
			err = c.Create(txCtx, row)
		}
		if err != nil {
			return err
		}
		if row.Source != manifest.Source {
			return errorsx.Conflict("插件编码已被其他来源占用")
		}
		for _, binding := range manifest.Tools {
			var info *schema.ToolInfo
			info, err = binding.Tool.Info(txCtx)
			if err != nil {
				return err
			}
			var inputSchema *schema.ParamsOneOf
			inputSchema = info.ParamsOneOf
			raw := []byte(`{"type":"object"}`)
			if inputSchema != nil {
				var jsonSchema *jsonschema.Schema
				jsonSchema, err = inputSchema.ToJSONSchema()
				if err != nil {
					return err
				}
				raw, err = json.Marshal(jsonSchema)
				if err != nil {
					return err
				}
			}
			err = c.saveToolSnapshot(txCtx, row.ID, "local", manifest.Source, binding.Operation, info.Name, info.Desc, string(raw), binding.Effect)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	manifest.Tools = append([]aiplugin.ToolBinding(nil), manifest.Tools...)
	c.manifests[manifest.Code] = manifest
	return nil
}

// ListAiPlugin 按当前身份查询目录，并按会话工具能力标记可用状态。
func (c *AiPluginCase) ListAiPlugin(ctx context.Context, req *basev1.ListAiPluginRequest) (*basev1.ListAiPluginResponse, error) {
	var err error
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiPlugin
	opts := []repository.QueryOption{repository.Order(query.ID.Asc())}
	var rows []*models.AiPlugin
	rows, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := &basev1.ListAiPluginResponse{}
	result.CanManageTools = pluginPlatform(identity) || identity.RoleCode == "tenant"
	result.CanManageCatalog = result.CanManageTools
	for _, row := range rows {
		if !pluginVisible(row, identity) {
			continue
		}
		if req.Keyword != "" && !strings.Contains(row.Name, req.Keyword) && !strings.Contains(row.Code, req.Keyword) {
			continue
		}
		var item *basev1.AiPlugin
		item, err = c.pluginDTO(ctx, row, identity.TenantId, ai.NormalizeTerminalString(ai.NormalizeTerminal(req.Terminal)))
		if err != nil {
			return nil, err
		}
		if row.Source == "MCP" && row.Visibility == "system" && row.Enabled == 1 && item.Installation != nil && item.Installation.Enabled && pluginTerminalAllowed(strings.Join(item.Installation.Terminals, ","), "admin") {
			result.CanCreatePrivate = true
		}
		result.Plugins = append(result.Plugins, item)
	}
	return result, nil
}

// GetAiPlugin 查询已校验可见范围的插件详情。
func (c *AiPluginCase) GetAiPlugin(ctx context.Context, req *basev1.GetAiPluginRequest) (*basev1.AiPlugin, error) {
	var err error
	var row *models.AiPlugin
	row, err = c.visiblePlugin(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	return c.pluginDTO(ctx, row, identity.TenantId, "admin")
}

// CreateAiPlugin 创建平台定义或复用平台批准端点的个人定义。
func (c *AiPluginCase) CreateAiPlugin(ctx context.Context, req *basev1.CreateAiPluginRequest) (*basev1.AiPlugin, error) {
	var err error
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.Form
	config := form.ConfigJson
	if !pluginPlatform(identity) {
		if form.Source != "MCP" || (form.Visibility != "private" && !(identity.RoleCode == "tenant" && form.Visibility == "tenant")) || form.ConfigJson != "" {
			return nil, errorsx.PermissionDenied("只能创建平台批准端点的个人连接定义")
		}
		query := c.Query(ctx).AiPlugin
		var approved []*models.AiPlugin
		approved, err = c.List(ctx, repository.Where(query.Source.Eq("MCP")), repository.Where(query.Visibility.Eq("system")), repository.Where(query.Endpoint.Eq(form.Endpoint)), repository.Where(query.Enabled.Eq(1)))
		if err != nil {
			return nil, err
		}
		if len(approved) == 0 {
			return nil, errorsx.PermissionDenied("端点尚未经平台批准")
		}
		_, err = c.activeInstallation(ctx, approved[0], identity.TenantId, "admin")
		if err != nil {
			return nil, errorsx.PermissionDenied("本租户未允许该自定义接入端点")
		}
		config = approved[0].ConfigJSON
	}
	if form.Source == "MCP" {
		var cfg plugin.Config
		cfg, err = plugin.ParseConfig(config)
		if err != nil {
			return nil, errorsx.InvalidArgument(err.Error())
		}
		_, err = cfg.ValidateURL(form.Endpoint)
		if err != nil {
			return nil, errorsx.InvalidArgument(err.Error())
		}
	}
	row := &models.AiPlugin{Code: form.Code, Name: form.Name, Description: form.Description, Source: form.Source, Visibility: form.Visibility, TenantID: identity.TenantId, OwnerUserID: identity.UserId, Enabled: pluginBool(form.Enabled), Endpoint: form.Endpoint, ConfigJSON: config, Revision: 1}
	if form.Visibility == "system" {
		row.TenantID = 0
		row.OwnerUserID = 0
	}
	err = c.Create(ctx, row)
	if err != nil {
		return nil, errorsx.Conflict("插件编码已存在或创建失败").WithCause(err)
	}
	return c.pluginDTO(ctx, row, identity.TenantId, "admin")
}

// UpdateAiPlugin 使用锁和修订号更新定义，个人不能更换平台批准的端点策略。
func (c *AiPluginCase) UpdateAiPlugin(ctx context.Context, req *basev1.UpdateAiPluginRequest) (*basev1.AiPlugin, error) {
	var err error
	var result *basev1.AiPlugin
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var row *models.AiPlugin
		row, err = c.visiblePlugin(txCtx, req.Id, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		var identity *authdata.UserTokenPayload
		identity, err = c.GetAuthInfo(txCtx)
		if err != nil {
			return err
		}
		if !pluginCanManageDefinition(row, identity) {
			return errorsx.PermissionDenied("无权修改插件目录")
		}
		if row.Revision != req.Revision {
			return errorsx.Conflict("插件配置已变化，请刷新后重试")
		}
		form := req.Form
		if form.Source != row.Source || form.Code != row.Code || form.Visibility != row.Visibility {
			return errorsx.InvalidArgument("插件来源、范围和稳定编码不可修改")
		}
		if !pluginPlatform(identity) && (form.Endpoint != row.Endpoint || form.Visibility != row.Visibility || form.ConfigJson != "") {
			return errorsx.PermissionDenied("个人不能修改接入安全范围")
		}
		row.Name = form.Name
		row.Description = form.Description
		row.Enabled = pluginBool(form.Enabled)
		if pluginPlatform(identity) {
			row.Visibility = form.Visibility
			row.Endpoint = form.Endpoint
			row.ConfigJSON = form.ConfigJson
			if row.Source == "MCP" {
				var cfg plugin.Config
				cfg, err = plugin.ParseConfig(row.ConfigJSON)
				if err != nil {
					return errorsx.InvalidArgument(err.Error())
				}
				_, err = cfg.ValidateURL(row.Endpoint)
				if err != nil {
					return errorsx.InvalidArgument(err.Error())
				}
			}
		}
		row.Revision++
		query := c.Query(txCtx).AiPlugin
		err = c.Update(txCtx, row, repository.Where(query.ID.Eq(row.ID)), repository.Select(query.Name, query.Description, query.Enabled, query.Visibility, query.Endpoint, query.ConfigJSON, query.Revision))
		if err != nil {
			return err
		}
		result, err = c.pluginDTO(txCtx, row, identity.TenantId, "admin")
		return err
	})
	return result, err
}

// DeleteAiPlugin 拒绝删除仍有启用安装或未断开账号的定义。
func (c *AiPluginCase) DeleteAiPlugin(ctx context.Context, req *basev1.DeleteAiPluginRequest) (*emptypb.Empty, error) {
	var err error
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var row *models.AiPlugin
		row, err = c.visiblePlugin(txCtx, req.Id, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		var identity *authdata.UserTokenPayload
		identity, err = c.GetAuthInfo(txCtx)
		if err != nil {
			return err
		}
		if !pluginCanManageDefinition(row, identity) {
			return errorsx.PermissionDenied("无权删除插件")
		}
		if row.Revision != req.Revision {
			return errorsx.Conflict("插件配置已变化")
		}
		installationQuery := c.installationRepo.Query(txCtx).AiPluginInstallation
		var installations []*models.AiPluginInstallation
		installations, err = c.installationRepo.List(txCtx, repository.Where(installationQuery.PluginID.Eq(row.ID)), repository.Where(installationQuery.Enabled.Eq(1)))
		if err != nil {
			return err
		}
		if len(installations) > 0 {
			return errorsx.Conflict("请先停用所有租户安装")
		}
		query := c.connectionRepo.Query(txCtx).AiPluginConnection
		var connections []*models.AiPluginConnection
		connections, err = c.connectionRepo.List(txCtx, repository.Where(query.PluginID.Eq(row.ID)), repository.Where(query.State.Neq("disconnected")))
		if err != nil {
			return err
		}
		if len(connections) > 0 {
			return errorsx.Conflict("请先断开全部连接")
		}
		return c.DeleteByID(txCtx, row.ID)
	})
	return &emptypb.Empty{}, err
}

// SetAiPluginInstallation 设置本租户安装，不能启用平台停用的插件。
func (c *AiPluginCase) SetAiPluginInstallation(ctx context.Context, req *basev1.SetAiPluginInstallationRequest) (*basev1.AiPluginInstallation, error) {
	var err error
	var identity *authdata.UserTokenPayload
	identity, err = c.pluginAdministrator(ctx)
	if err != nil {
		return nil, err
	}
	var result *basev1.AiPluginInstallation
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var row *models.AiPlugin
		row, err = c.visiblePlugin(txCtx, req.PluginId, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if row.Enabled != 1 && req.Enabled {
			return errorsx.PermissionDenied("插件目录已停用")
		}
		var installation *models.AiPluginInstallation
		installation, err = c.findInstallation(txCtx, row.ID, identity.TenantId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if installation == nil {
			if req.Revision != 0 {
				return errorsx.Conflict("安装策略已变化")
			}
			installation = &models.AiPluginInstallation{PluginID: row.ID, TenantID: identity.TenantId, Revision: 1, Enabled: pluginBool(req.Enabled), Terminals: strings.Join(req.Terminals, ",")}
			err = c.installationRepo.Create(txCtx, installation)
		} else {
			if installation.Revision != req.Revision {
				return errorsx.Conflict("安装策略已变化")
			}
			installation.Enabled = pluginBool(req.Enabled)
			installation.Terminals = strings.Join(req.Terminals, ",")
			installation.Revision++
			query := c.installationRepo.Query(txCtx).AiPluginInstallation
			err = c.installationRepo.Update(txCtx, installation, repository.Where(query.ID.Eq(installation.ID)), repository.Select(query.Enabled, query.Terminals, query.Revision))
		}
		if err != nil {
			return err
		}
		result = pluginInstallationDTO(installation)
		return nil
	})
	return result, err
}

// visiblePlugin 将插件可见性与资源查询绑定，避免知道ID即可跨租户访问。
func (c *AiPluginCase) visiblePlugin(ctx context.Context, rawID string, extra ...repository.QueryOption) (*models.AiPlugin, error) {
	var err error
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	id, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return nil, errorsx.InvalidArgument("插件编号不合法")
	}
	query := c.Query(ctx).AiPlugin
	opts := append([]repository.QueryOption{repository.Where(query.ID.Eq(id))}, extra...)
	var row *models.AiPlugin
	row, err = c.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("插件不存在")
	}
	if err != nil {
		return nil, err
	}
	if !pluginVisible(row, identity) {
		return nil, errorsx.PermissionDenied("无权访问插件")
	}
	return row, nil
}

// pluginAdministrator 校验当前登录用户的平台或租户管理身份。
func (c *AiPluginCase) pluginAdministrator(ctx context.Context) (*authdata.UserTokenPayload, error) {
	var err error
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if !pluginPlatform(identity) && identity.RoleCode != "tenant" {
		return nil, errorsx.PermissionDenied("需要租户管理权限")
	}
	return identity, nil
}

// findInstallation 按租户与插件查找唯一安装策略。
func (c *AiPluginCase) findInstallation(ctx context.Context, pluginID, tenantID int64) (*models.AiPluginInstallation, error) {
	query := c.installationRepo.Query(ctx).AiPluginInstallation
	opts := []repository.QueryOption{repository.Where(query.PluginID.Eq(pluginID)), repository.Where(query.TenantID.Eq(tenantID))}
	return c.installationRepo.Find(ctx, opts...)
}

// pluginDTO 组合租户安装和指定工具能力的可用状态，管理权限仍按身份判断。
func (c *AiPluginCase) pluginDTO(ctx context.Context, row *models.AiPlugin, tenantID int64, terminal string) (*basev1.AiPlugin, error) {
	var err error
	result := mapper.NewCopierMapper[basev1.AiPlugin, models.AiPlugin]().ToDTO(row)
	result.Id = strconv.FormatInt(row.ID, 10)
	result.Enabled = row.Enabled == 1
	var identity *authdata.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	result.CanManage = pluginCanManageDefinition(row, identity)
	if pluginPlatform(identity) && result.CanManage {
		result.ConfigJson = row.ConfigJSON
	}
	c.manifestMu.RLock()
	_, result.Assembled = c.manifests[row.Code]
	c.manifestMu.RUnlock()
	if row.Source == "MCP" || row.Code == "builtin" {
		result.Assembled = true
	}
	var installation *models.AiPluginInstallation
	installation, err = c.findInstallation(ctx, row.ID, tenantID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if installation != nil {
		result.Installation = pluginInstallationDTO(installation)
	}
	if !pluginCanUse(row, identity) {
		result.UnavailableReason = "个人插件仅本人可使用；管理员可维护安装和工具审查"
	} else if !result.Assembled {
		result.UnavailableReason = "组件尚未装配"
	} else if row.Enabled != 1 {
		result.UnavailableReason = "插件已停用"
	} else if installation == nil || installation.Enabled != 1 {
		result.UnavailableReason = "租户尚未启用"
	} else if !pluginTerminalAllowed(installation.Terminals, terminal) {
		result.UnavailableReason = "当前终端不允许使用此插件"
	} else {
		_, err = c.approvedMCPRevision(ctx, row, tenantID, terminal)
		if err != nil {
			if kratosErrors.FromError(err).Reason != errorsx.ReasonPermissionDenied {
				return nil, err
			}
			result.UnavailableReason = "平台或租户接入策略已撤销或变化，请重建自定义定义"
		}
	}
	return result, nil
}

// pluginVisible 允许本租户管理员审查个人目录元数据，账号和执行权限仍独立校验。
func pluginVisible(row *models.AiPlugin, identity *authdata.UserTokenPayload) bool {
	return row.Visibility == "system" || (row.TenantID == identity.TenantId && (row.Visibility == "tenant" || (row.Visibility == "private" && (row.OwnerUserID == identity.UserId || identity.RoleCode == "tenant" || pluginPlatform(identity)))))
}

// pluginCanUse 将个人定义使用权始终限制在本人，不因管理员可审查目录而扩大。
func pluginCanUse(row *models.AiPlugin, identity *authdata.UserTokenPayload) bool {
	return pluginVisible(row, identity) && (row.Visibility != "private" || row.OwnerUserID == identity.UserId)
}

// pluginCanManageDefinition 个人定义只有本人可修改，管理员仅管理其安装及工具策略。
func pluginCanManageDefinition(row *models.AiPlugin, identity *authdata.UserTokenPayload) bool {
	if !pluginVisible(row, identity) {
		return false
	}
	if row.Visibility == "private" {
		return row.TenantID == identity.TenantId && row.OwnerUserID == identity.UserId
	}
	return pluginPlatform(identity) || (row.Visibility == "tenant" && row.TenantID == identity.TenantId && identity.RoleCode == "tenant")
}

// pluginPlatform 默认租户的 super/admin 才能修改平台安全底线。
func pluginPlatform(identity *authdata.UserTokenPayload) bool {
	return identity.TenantId == 1 && (identity.RoleCode == "super" || identity.RoleCode == "admin")
}

// pluginBool 将协议布尔值映射为数据库开关。
func pluginBool(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// pluginInstallationDTO 转换安装的逗号终端策略。
func pluginInstallationDTO(row *models.AiPluginInstallation) *basev1.AiPluginInstallation {
	return &basev1.AiPluginInstallation{Id: strconv.FormatInt(row.ID, 10), PluginId: strconv.FormatInt(row.PluginID, 10), Enabled: row.Enabled == 1, Terminals: strings.FieldsFunc(row.Terminals, func(value rune) bool { return value == ',' }), Revision: row.Revision}
}
