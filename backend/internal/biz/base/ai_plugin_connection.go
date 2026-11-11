package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/plugin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/sdk"
	"github.com/liujitcn/kratos-kit/secretcrypto"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListAiPluginConnection 只返回本人或本租户显式共享的账号，始终排除凭据。
func (c *AiPluginCase) ListAiPluginConnection(ctx context.Context, req *basev1.ListAiPluginConnectionRequest) (*basev1.ListAiPluginConnectionResponse, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.connectionRepo.Query(ctx).AiPluginConnection
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(identity.TenantId)), repository.Order(query.ID.Desc())}
	if req.PluginId != "" {
		var pluginID int64
		pluginID, err = strconv.ParseInt(req.PluginId, 10, 64)
		if err != nil || pluginID <= 0 {
			return nil, errorsx.InvalidArgument("插件编号不合法")
		}
		opts = append(opts, repository.Where(query.PluginID.Eq(pluginID)))
	}
	var rows []*models.AiPluginConnection
	rows, err = c.connectionRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := &basev1.ListAiPluginConnectionResponse{}
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.PluginID)
	}
	pluginQuery := c.Query(ctx).AiPlugin
	var definitions []*models.AiPlugin
	definitions, err = c.List(ctx, repository.Where(pluginQuery.ID.In(ids...)))
	if err != nil {
		return nil, err
	}
	plugins := make(map[int64]*models.AiPlugin, len(definitions))
	for _, definition := range definitions {
		plugins[definition.ID] = definition
	}
	for _, row := range rows {
		definition := plugins[row.PluginID]
		if definition == nil || !pluginCanUse(definition, identity) {
			continue
		}
		if row.OwnerUserID == identity.UserId || row.Sharing == "tenant" {
			result.Connections = append(result.Connections, pluginConnectionDTO(row, identity))
		}
	}
	return result, nil
}

// CreateAiPluginConnection 从当前身份创建真实连接；令牌只存入绑定账号的加密信封。
func (c *AiPluginCase) CreateAiPluginConnection(ctx context.Context, req *basev1.CreateAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var row *models.AiPlugin
	row, err = c.visiblePlugin(ctx, req.PluginId)
	if err != nil {
		return nil, err
	}
	if !pluginCanUse(row, identity) {
		return nil, errorsx.PermissionDenied("个人插件仅允许本人连接账号")
	}
	var installation *models.AiPluginInstallation
	installation, err = c.activeInstallation(ctx, row, identity.TenantId, "admin")
	if err != nil {
		return nil, err
	}
	if row.Source != "MCP" {
		return nil, errorsx.InvalidArgument("本地组件不需要账号连接")
	}
	var config plugin.Config
	config, err = plugin.ParseConfig(row.ConfigJSON)
	if err != nil {
		return nil, errorsx.InvalidArgument(err.Error())
	}
	if req.AuthType == "oauth" && config.OAuth == nil {
		return nil, errorsx.InvalidArgument("平台尚未配置此插件的OAuth客户端")
	}
	credential := dto.AiPluginCredential{}
	if req.AuthType == "token" {
		if req.Token.GetText() == "" {
			return nil, errorsx.InvalidArgument("请输入API令牌")
		}
		credential.Token = req.Token.GetText()
	}
	connection := &models.AiPluginConnection{InstallationID: installation.ID, PluginID: row.ID, TenantID: identity.TenantId, OwnerUserID: identity.UserId, Sharing: "private", Label: req.Label, AuthType: req.AuthType, State: "disconnected", Revision: 1, Scopes: "", Credential: ""}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		err = c.connectionRepo.Create(txCtx, connection)
		if err != nil {
			return err
		}
		connection.Credential, err = sealPluginCredential(txCtx, connection, credential)
		if err != nil {
			return err
		}
		return c.saveConnection(txCtx, connection)
	})
	if err != nil {
		return nil, err
	}
	if req.AuthType != "oauth" {
		err = c.refreshConnection(ctx, row, installation, connection)
		if err != nil {
			return nil, err
		}
	}
	return pluginConnectionDTO(connection, identity), nil
}

// RefreshAiPluginConnection 检测并完整刷新本人连接，失败保留最后成功快照但停止执行。
func (c *AiPluginCase) RefreshAiPluginConnection(ctx context.Context, req *basev1.RefreshAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	var err error
	var connection *models.AiPluginConnection
	connection, err = c.ownedConnection(ctx, req.Id, true)
	if err != nil {
		return nil, err
	}
	if connection.Revision != req.Revision {
		return nil, errorsx.Conflict("连接配置已变化")
	}
	var row *models.AiPlugin
	row, err = c.visiblePlugin(ctx, strconv.FormatInt(connection.PluginID, 10))
	if err != nil {
		return nil, err
	}
	var installation *models.AiPluginInstallation
	installation, err = c.activeInstallation(ctx, row, connection.TenantID, "admin")
	if err != nil {
		return nil, err
	}
	err = c.refreshConnection(ctx, row, installation, connection)
	if err != nil {
		return nil, err
	}
	return pluginConnectionDTO(connection, nil), nil
}

// DisconnectAiPluginConnection 先持久化撤销和清空密文，再尽力撤销供应商令牌且保留公开结果。
func (c *AiPluginCase) DisconnectAiPluginConnection(ctx context.Context, req *basev1.DisconnectAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	var err error
	var connection *models.AiPluginConnection
	var credential dto.AiPluginCredential
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		connection, err = c.ownedConnection(txCtx, req.Id, true, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if connection.Revision != req.Revision {
			return errorsx.Conflict("连接配置已变化")
		}
		connection.LastError = ""
		if connection.AuthType == "oauth" && connection.Credential != "" {
			credential, err = openPluginCredential(txCtx, connection)
			if err != nil {
				// 解密失败仍须先关闭本地授权，不能阻止用户断开损坏的连接。
				connection.LastError = "本地已断开；凭据无法读取，请到供应商处撤销授权"
			}
		} else if connection.AuthType == "token" {
			connection.LastError = "本地已断开；请到供应商处撤销API令牌"
		}
		connection.State = "disconnected"
		connection.Credential = ""
		connection.Scopes = ""
		connection.ExpiresAt = 0
		connection.Revision++
		return c.saveConnection(txCtx, connection)
	})
	if err != nil {
		return nil, err
	}
	if credential.OAuth != nil {
		query := c.Query(ctx).AiPlugin
		var row *models.AiPlugin
		row, err = c.Find(ctx, repository.Where(query.ID.Eq(connection.PluginID)))
		if err == nil {
			var config plugin.Config
			config, err = plugin.ParseConfig(row.ConfigJSON)
			if err == nil {
				err = plugin.RevokeOAuth(ctx, credential.OAuth, config)
			}
		}
		if err != nil {
			err = c.failConnection(ctx, connection, "disconnected", "本地已断开；供应商未确认令牌撤销，请到供应商处核对授权")
			if err != nil {
				return nil, err
			}
		}
	}
	return pluginConnectionDTO(connection, nil), nil
}

// StartAiPluginOAuth 绑定当前账号、策略版本与一次性PKCE，返回真实供应商授权页。
func (c *AiPluginCase) StartAiPluginOAuth(ctx context.Context, req *basev1.StartAiPluginOAuthRequest) (*basev1.StartAiPluginOAuthResponse, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var connection *models.AiPluginConnection
	connection, err = c.ownedConnection(ctx, req.Id, true)
	if err != nil {
		return nil, err
	}
	if connection.AuthType != "oauth" || connection.Revision != req.Revision {
		return nil, errorsx.Conflict("连接类型或配置已变化")
	}
	var row *models.AiPlugin
	row, err = c.visiblePlugin(ctx, strconv.FormatInt(connection.PluginID, 10))
	if err != nil {
		return nil, err
	}
	var installation *models.AiPluginInstallation
	installation, err = c.activeInstallation(ctx, row, identity.TenantId, "admin")
	if err != nil {
		return nil, err
	}
	var config plugin.Config
	config, err = plugin.ParseConfig(row.ConfigJSON)
	if err != nil || config.OAuth == nil {
		return nil, errorsx.InvalidArgument("平台尚未配置OAuth客户端")
	}
	var approvedRevision string
	approvedRevision, err = c.approvedMCPRevision(ctx, row, identity.TenantId, "admin")
	if err != nil {
		return nil, err
	}
	if c.Cache == nil {
		return nil, errorsx.Internal("授权状态缓存不可用")
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var current *models.AiPluginConnection
		current, err = c.ownedConnection(txCtx, req.Id, true, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if current.Revision != req.Revision {
			return errorsx.Conflict("连接配置已变化")
		}
		current.State = "authorizing"
		current.LastError = ""
		current.Revision++
		err = c.saveConnection(txCtx, current)
		if err != nil {
			return err
		}
		connection = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	state := uuid.NewString()
	verifier := oauth2.GenerateVerifier()
	binding := dto.AiPluginOAuthState{Identity: *identity, ConnectionID: connection.ID, ConnectionRevision: connection.Revision, PluginRevision: row.Revision, InstallationRevision: installation.Revision, ApprovedRevision: approvedRevision, Verifier: verifier}
	var raw []byte
	raw, err = json.Marshal(binding)
	if err != nil {
		return nil, err
	}
	err = c.Cache.Set("ai:plugin:oauth:"+state, string(raw), 10*time.Minute)
	if err != nil {
		return nil, err
	}
	oauthConfig := pluginOAuthConfig(config.OAuth)
	return &basev1.StartAiPluginOAuthResponse{AuthorizationUrl: oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("resource", row.Endpoint))}, nil
}

// CompleteAiPluginOAuth 不要求第三方携带登录令牌，仅信任原子消费的服务端授权绑定。
func (c *AiPluginCase) CompleteAiPluginOAuth(ctx context.Context, req *basev1.CompleteAiPluginOAuthRequest) (*basev1.AiPluginConnection, error) {
	var err error
	if c.Cache == nil {
		return nil, errorsx.Internal("授权状态缓存不可用")
	}
	var raw string
	raw, err = c.Cache.GetDel("ai:plugin:oauth:" + req.State)
	if err != nil || raw == "" {
		return nil, errorsx.PermissionDenied("授权状态已过期或已使用")
	}
	var binding dto.AiPluginOAuthState
	err = json.Unmarshal([]byte(raw), &binding)
	if err != nil {
		return nil, errorsx.PermissionDenied("授权状态无效")
	}
	query := c.connectionRepo.Query(ctx).AiPluginConnection
	var connection *models.AiPluginConnection
	connection, err = c.connectionRepo.Find(ctx, repository.Where(query.ID.Eq(binding.ConnectionID)), repository.Where(query.TenantID.Eq(binding.Identity.TenantId)), repository.Where(query.OwnerUserID.Eq(binding.Identity.UserId)))
	if err != nil {
		return nil, errorsx.PermissionDenied("授权连接已失效")
	}
	pluginQuery := c.Query(ctx).AiPlugin
	var row *models.AiPlugin
	row, err = c.Find(ctx, repository.Where(pluginQuery.ID.Eq(connection.PluginID)))
	if err != nil {
		return nil, err
	}
	var installation *models.AiPluginInstallation
	installation, err = c.activeInstallation(ctx, row, connection.TenantID, "admin")
	if err != nil {
		return nil, err
	}
	var approvedRevision string
	approvedRevision, err = c.approvedMCPRevision(ctx, row, connection.TenantID, "admin")
	if err != nil {
		return nil, err
	}
	if approvedRevision != binding.ApprovedRevision || connection.Revision != binding.ConnectionRevision || row.Revision != binding.PluginRevision || installation.Revision != binding.InstallationRevision {
		return nil, errorsx.Conflict("授权期间配置已变化，请重新授权")
	}
	var config plugin.Config
	config, err = plugin.ParseConfig(row.ConfigJSON)
	if err != nil || config.OAuth == nil {
		return nil, errorsx.InvalidArgument("OAuth配置不可用")
	}
	if req.Error != "" || req.Code == "" {
		err = c.failConnection(ctx, connection, "reauthorization_required", "用户拒绝或供应商未返回授权码")
		if err != nil {
			return nil, err
		}
		return pluginConnectionDTO(connection, nil), nil
	}
	if req.Iss != config.OAuth.Issuer {
		err = c.failConnection(ctx, connection, "reauthorization_required", "OAuth发行方与配置不匹配，请重新授权")
		if err != nil {
			return nil, err
		}
		return pluginConnectionDTO(connection, nil), nil
	}
	oauthConfig := pluginOAuthConfig(config.OAuth)
	var token *oauth2.Token
	token, err = oauthConfig.Exchange(context.WithValue(ctx, oauth2.HTTPClient, config.Client("")), req.Code, oauth2.VerifierOption(binding.Verifier), oauth2.SetAuthURLParam("resource", row.Endpoint))
	if err != nil {
		err = c.failConnection(ctx, connection, "reauthorization_required", "OAuth令牌交换失败，请重新授权")
		if err != nil {
			return nil, err
		}
		return pluginConnectionDTO(connection, nil), nil
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		var currentPlugin *models.AiPlugin
		currentPlugin, err = c.Find(txCtx, repository.Where(pluginQuery.ID.Eq(row.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		var currentInstallation *models.AiPluginInstallation
		currentInstallation, err = c.activeInstallation(txCtx, currentPlugin, connection.TenantID, "admin")
		if err != nil {
			return err
		}
		var currentApprovedRevision string
		currentApprovedRevision, err = c.approvedMCPRevision(txCtx, currentPlugin, connection.TenantID, "admin")
		if err != nil {
			return err
		}
		if currentApprovedRevision != binding.ApprovedRevision || currentPlugin.Revision != binding.PluginRevision || currentInstallation.Revision != binding.InstallationRevision {
			return errorsx.Conflict("授权期间插件或安装策略已变化")
		}

		var locked *models.AiPluginConnection
		locked, err = c.connectionRepo.Find(txCtx, repository.Where(query.ID.Eq(connection.ID)), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if locked.Revision != binding.ConnectionRevision {
			return errorsx.Conflict("连接已变化，请重新授权")
		}
		connection = locked
		connection.Credential, err = sealPluginCredential(txCtx, connection, dto.AiPluginCredential{OAuth: token})
		if err != nil {
			return err
		}
		if scope, ok := token.Extra("scope").(string); ok {
			connection.Scopes = scope
		} else {
			connection.Scopes = ""
		}
		connection.ExpiresAt = 0
		if !token.Expiry.IsZero() {
			connection.ExpiresAt = token.Expiry.Unix()
		}
		connection.State = "checking"
		connection.Revision++
		return c.saveConnection(txCtx, connection)
	})
	if err != nil {
		return nil, err
	}
	err = c.refreshConnection(ctx, row, installation, connection)
	if err != nil {
		return nil, err
	}
	return pluginConnectionDTO(connection, nil), nil
}

// ReauthorizeAiPluginConnection 在原账号内替换加密令牌并使所有旧许可失效。
func (c *AiPluginCase) ReauthorizeAiPluginConnection(ctx context.Context, req *basev1.ReauthorizeAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	var err error
	if req.Token.GetText() == "" {
		return nil, errorsx.InvalidArgument("请输入API令牌")
	}
	var connection *models.AiPluginConnection
	var row *models.AiPlugin
	var installation *models.AiPluginInstallation
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		connection, err = c.ownedConnection(txCtx, req.Id, true, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if connection.Revision != req.Revision || connection.AuthType != "token" {
			return errorsx.Conflict("连接类型或配置已变化")
		}
		row, err = c.visiblePlugin(txCtx, strconv.FormatInt(connection.PluginID, 10))
		if err != nil {
			return err
		}
		installation, err = c.activeInstallation(txCtx, row, connection.TenantID, "admin")
		if err != nil {
			return err
		}
		connection.Credential, err = sealPluginCredential(txCtx, connection, dto.AiPluginCredential{Token: req.Token.GetText()})
		if err != nil {
			return err
		}
		connection.State = "checking"
		connection.LastError = ""
		connection.Revision++
		return c.saveConnection(txCtx, connection)
	})
	if err != nil {
		return nil, err
	}
	err = c.refreshConnection(ctx, row, installation, connection)
	if err != nil {
		return nil, err
	}
	return pluginConnectionDTO(connection, nil), nil
}

// SetAiPluginConnectionSharing 仅管理员能显式共享本人账号，变更即撤销旧许可。
func (c *AiPluginCase) SetAiPluginConnectionSharing(ctx context.Context, req *basev1.SetAiPluginConnectionSharingRequest) (*basev1.AiPluginConnection, error) {
	var err error
	_, err = c.pluginAdministrator(ctx)
	if err != nil {
		return nil, err
	}
	var connection *models.AiPluginConnection
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		connection, err = c.ownedConnection(txCtx, req.Id, true, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if connection.Revision != req.Revision {
			return errorsx.Conflict("连接配置已变化")
		}
		if req.Sharing != "private" && req.Sharing != "tenant" {
			return errorsx.InvalidArgument("共享范围不合法")
		}
		if req.Sharing == "tenant" {
			var definition *models.AiPlugin
			definition, err = c.visiblePlugin(txCtx, strconv.FormatInt(connection.PluginID, 10))
			if err != nil {
				return err
			}
			if definition.Visibility == "private" {
				return errorsx.PermissionDenied("个人插件账号不能共享给租户")
			}
		}
		connection.Sharing = req.Sharing
		connection.Revision++
		query := c.connectionRepo.Query(txCtx).AiPluginConnection
		return c.connectionRepo.Update(txCtx, connection, repository.Where(query.ID.Eq(connection.ID)), repository.Select(query.Sharing, query.Revision))
	})
	if err != nil {
		return nil, err
	}
	return pluginConnectionDTO(connection, nil), nil
}

// ownedConnection 同时校验租户与账号归属，写操作不允许共享访客管理。
func (c *AiPluginCase) ownedConnection(ctx context.Context, rawID string, ownerOnly bool, extra ...repository.QueryOption) (*models.AiPluginConnection, error) {
	var err error
	var identity *data.UserTokenPayload
	identity, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	id, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return nil, errorsx.InvalidArgument("连接编号不合法")
	}
	query := c.connectionRepo.Query(ctx).AiPluginConnection
	opts := append([]repository.QueryOption{repository.Where(query.ID.Eq(id)), repository.Where(query.TenantID.Eq(identity.TenantId))}, extra...)
	var row *models.AiPluginConnection
	row, err = c.connectionRepo.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.ResourceNotFound("连接不存在")
	}
	if row.OwnerUserID != identity.UserId {
		if ownerOnly || row.Sharing != "tenant" {
			return nil, errorsx.PermissionDenied("无权访问此账号连接")
		}
		var definition *models.AiPlugin
		definition, err = c.visiblePlugin(ctx, strconv.FormatInt(row.PluginID, 10))
		if err != nil {
			return nil, err
		}
		if !pluginCanUse(definition, identity) {
			return nil, errorsx.PermissionDenied("个人插件账号仅允许本人使用")
		}
	}
	return row, nil
}

// activeInstallation 获取目录、租户和受信执行终端的共同允许范围。
func (c *AiPluginCase) activeInstallation(ctx context.Context, row *models.AiPlugin, tenantID int64, terminal string) (*models.AiPluginInstallation, error) {
	var err error
	if row.Enabled != 1 {
		return nil, errorsx.PermissionDenied("插件目录已停用")
	}
	var installation *models.AiPluginInstallation
	installation, err = c.findInstallation(ctx, row.ID, tenantID)
	if err != nil || installation.Enabled != 1 {
		return nil, errorsx.PermissionDenied("租户未启用此插件")
	}
	if !strings.Contains(","+installation.Terminals+",", ","+terminal+",") {
		return nil, errorsx.PermissionDenied("当前终端不允许使用此插件")
	}
	_, err = c.approvedMCPRevision(ctx, row, tenantID, terminal)
	if err != nil {
		return nil, err
	}
	return installation, nil
}

// approvedMCPRevision 子定义必须持续处于平台批准的端点及租户安装范围，复制配置不能绕过撤销。
func (c *AiPluginCase) approvedMCPRevision(ctx context.Context, row *models.AiPlugin, tenantID int64, terminal string) (string, error) {
	if row.Source != "MCP" || row.Visibility == "system" {
		return "", nil
	}
	query := c.Query(ctx).AiPlugin
	opts := []repository.QueryOption{repository.Where(query.Source.Eq("MCP")), repository.Where(query.Visibility.Eq("system")), repository.Where(query.Endpoint.Eq(row.Endpoint)), repository.Where(query.Enabled.Eq(1)), repository.Order(query.ID.Asc())}
	approved, err := c.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	for _, definition := range approved {
		if definition.ConfigJSON != row.ConfigJSON {
			continue
		}
		var installation *models.AiPluginInstallation
		installation, err = c.findInstallation(ctx, definition.ID, tenantID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		if installation.Enabled == 1 && pluginTerminalAllowed(installation.Terminals, terminal) {
			return fmt.Sprintf("approved:%d:%d:%d:%d", definition.ID, definition.Revision, installation.ID, installation.Revision), nil
		}
	}
	return "", errorsx.PermissionDenied("平台或租户已撤销接入端点，或接入安全配置已变化，请重建自定义定义")
}

// saveConnection 显式选择可归零的状态字段，避免ORM忽略禁用和清空操作。
func (c *AiPluginCase) saveConnection(ctx context.Context, row *models.AiPluginConnection) error {
	query := c.connectionRepo.Query(ctx).AiPluginConnection
	return c.connectionRepo.Update(ctx, row, repository.Where(query.ID.Eq(row.ID)), repository.Select(query.Credential, query.Scopes, query.ExpiresAt, query.State, query.LastError, query.Revision))
}

// sealPluginCredential 复用SecretCrypto信封与宿主派生密钥，将密文绑定具体租户账号。
func sealPluginCredential(ctx context.Context, row *models.AiPluginConnection, value dto.AiPluginCredential) (string, error) {
	var err error
	key := sdk.Runtime.GetKey()
	if key == nil {
		return "", errorsx.Internal("连接凭据加密密钥未初始化")
	}
	var derived []byte
	derived, err = key.Derive(ctx, "kratos-admin:ai-plugin:credential")
	if err != nil {
		return "", err
	}
	var raw []byte
	raw, err = json.Marshal(value)
	if err != nil {
		return "", err
	}
	return secretcrypto.EncryptFieldEnvelope(derived, fmt.Sprintf("ai_plugin_connection:%d:%d:%d", row.TenantID, row.OwnerUserID, row.ID), string(raw))
}

// openPluginCredential 仅在服务端连接和调用边界解密，不向DTO、模型或检查点返回。
func openPluginCredential(ctx context.Context, row *models.AiPluginConnection) (dto.AiPluginCredential, error) {
	var err error
	var value dto.AiPluginCredential
	key := sdk.Runtime.GetKey()
	if key == nil {
		return value, errorsx.Internal("连接凭据加密密钥未初始化")
	}
	var derived []byte
	derived, err = key.Derive(ctx, "kratos-admin:ai-plugin:credential")
	if err != nil {
		return value, err
	}
	var raw string
	raw, err = secretcrypto.DecryptFieldEnvelope(derived, fmt.Sprintf("ai_plugin_connection:%d:%d:%d", row.TenantID, row.OwnerUserID, row.ID), row.Credential)
	if err != nil {
		return value, err
	}
	err = json.Unmarshal([]byte(raw), &value)
	return value, err
}

// pluginConnectionDTO 公开账号状态与归属，排除所有令牌和加密信封。
func pluginConnectionDTO(row *models.AiPluginConnection, identity *data.UserTokenPayload) *basev1.AiPluginConnection {
	result := &basev1.AiPluginConnection{Id: strconv.FormatInt(row.ID, 10), PluginId: strconv.FormatInt(row.PluginID, 10), Label: row.Label, AuthType: row.AuthType, State: row.State, Sharing: row.Sharing, Scopes: row.Scopes, ExpiresAt: row.ExpiresAt, Revision: row.Revision, LastError: row.LastError, OwnerUserId: row.OwnerUserID}
	result.CanManage = (identity == nil || identity.UserId == row.OwnerUserID)
	return result
}

// pluginOAuthConfig 使用平台预注册的公开PKCE客户端，不使用登录OAuth配置。
func pluginOAuthConfig(config *plugin.OAuthConfig) *oauth2.Config {
	return &oauth2.Config{ClientID: config.ClientID, RedirectURL: config.RedirectURL, Scopes: config.Scopes, Endpoint: oauth2.Endpoint{AuthURL: config.AuthorizationURL, TokenURL: config.TokenURL, AuthStyle: oauth2.AuthStyleInParams}}
}
