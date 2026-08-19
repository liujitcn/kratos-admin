// Package client 提供 Backend 生成的 gRPC 客户端集合。
package client

import (
	"context"

	"github.com/go-kratos/kratos/v3/registry"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	coreclient "github.com/liujitcn/kratos-core/client"
	"github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// Connection 是 Core 客户端提供的统一 gRPC 连接。
type Connection = coreclient.Connection

// Option 是 Core 客户端连接的可选配置。
type Option = coreclient.Option

// LocalServiceRegistrar 描述向进程内 gRPC 客户端注册服务的函数。
type LocalServiceRegistrar = coreclient.LocalServiceRegistrar

// Client 汇总 Backend 的全部生成 gRPC 客户端。
type Client struct {
	// Connection 是所有业务客户端共用的 gRPC 连接。
	Connection *Connection
	// Base 提供 base.v1 服务客户端。
	Base BaseClient
	// SystemAdmin 提供 system.admin.v1 服务客户端。
	SystemAdmin SystemAdminClient
	// SystemApp 提供 system.app.v1 服务客户端。
	SystemApp SystemAppClient
}

// BaseClient 汇总 base.v1 服务客户端。
type BaseClient struct {
	// AiMessage 是 AI 消息服务客户端。
	AiMessage basev1.AiMessageServiceClient
	// AiSession 是 AI 会话服务客户端。
	AiSession basev1.AiSessionServiceClient
	// AiTool 是 AI 工具服务客户端。
	AiTool basev1.AiToolServiceClient
	// Config 是配置服务客户端。
	Config basev1.ConfigServiceClient
	// File 是文件服务客户端。
	File basev1.FileServiceClient
	// Language 是语言服务客户端。
	Language basev1.LanguageServiceClient
	// Login 是登录服务客户端。
	Login basev1.LoginServiceClient
	// Mcp 是 MCP 服务客户端。
	Mcp basev1.McpServiceClient
	// Oauth 是 OAuth 服务客户端。
	Oauth basev1.OauthServiceClient
	// Sse 是 SSE 服务客户端。
	Sse basev1.SseServiceClient
}

// SystemAdminClient 汇总 system.admin.v1 服务客户端。
type SystemAdminClient struct {
	// Auth 是管理端认证服务客户端。
	Auth adminv1.AuthServiceClient
	// BaseAPI 是管理端 API 服务客户端。
	BaseAPI adminv1.BaseApiServiceClient
	// BaseArea 是管理端行政区划服务客户端。
	BaseArea adminv1.BaseAreaServiceClient
	// BaseConfig 是管理端配置服务客户端。
	BaseConfig adminv1.BaseConfigServiceClient
	// BaseDept 是管理端部门服务客户端。
	BaseDept adminv1.BaseDeptServiceClient
	// BaseDict 是管理端字典服务客户端。
	BaseDict adminv1.BaseDictServiceClient
	// BaseJob 是管理端岗位服务客户端。
	BaseJob adminv1.BaseJobServiceClient
	// BaseLanguage 是管理端语言服务客户端。
	BaseLanguage adminv1.BaseLanguageServiceClient
	// BaseLog 是管理端日志服务客户端。
	BaseLog adminv1.BaseLogServiceClient
	// BaseMenu 是管理端菜单服务客户端。
	BaseMenu adminv1.BaseMenuServiceClient
	// BaseMigration 是管理端迁移服务客户端。
	BaseMigration adminv1.BaseMigrationServiceClient
	// BasePost 是管理端岗位服务客户端。
	BasePost adminv1.BasePostServiceClient
	// BaseRole 是管理端角色服务客户端。
	BaseRole adminv1.BaseRoleServiceClient
	// BaseTenant 是管理端租户服务客户端。
	BaseTenant adminv1.BaseTenantServiceClient
	// BaseThirdAccount 是管理端三方账号服务客户端。
	BaseThirdAccount adminv1.BaseThirdAccountServiceClient
	// BaseI18n 是管理端国际化服务客户端。
	BaseI18n adminv1.BaseI18nServiceClient
	// BaseUser 是管理端用户服务客户端。
	BaseUser adminv1.BaseUserServiceClient
	// CodeGen 是代码生成服务客户端。
	CodeGen adminv1.CodeGenServiceClient
	// CodeGenColumn 是代码生成列服务客户端。
	CodeGenColumn adminv1.CodeGenColumnServiceClient
	// CodeGenProto 是代码生成 Proto 服务客户端。
	CodeGenProto adminv1.CodeGenProtoServiceClient
	// CodeGenTable 是代码生成表服务客户端。
	CodeGenTable adminv1.CodeGenTableServiceClient
	// OpsMonitoring 是运维监控服务客户端。
	OpsMonitoring adminv1.OpsMonitoringServiceClient
	// ProjectDocument 是项目文档服务客户端。
	ProjectDocument adminv1.ProjectDocumentServiceClient
}

// SystemAppClient 汇总 system.app.v1 服务客户端。
type SystemAppClient struct {
	// Auth 是应用端认证服务客户端。
	Auth appv1.AuthServiceClient
	// BaseArea 是应用端行政区划服务客户端。
	BaseArea appv1.BaseAreaServiceClient
	// BaseDict 是应用端字典服务客户端。
	BaseDict appv1.BaseDictServiceClient
	// BaseMenu 是应用端菜单服务客户端。
	BaseMenu appv1.BaseMenuServiceClient
}

// NewClient 根据客户端配置创建 Backend 的全部 gRPC 客户端。
func NewClient(ctx context.Context, clientConfig *configv1.Client, options ...Option) (*Client, func(), error) {
	connection, cleanup, err := coreclient.NewConnection(ctx, clientConfig, options...)
	if err != nil {
		return nil, nil, err
	}
	return &Client{
		Connection: connection,
		Base: BaseClient{
			AiMessage: basev1.NewAiMessageServiceClient(connection),
			AiSession: basev1.NewAiSessionServiceClient(connection),
			AiTool:    basev1.NewAiToolServiceClient(connection),
			Config:    basev1.NewConfigServiceClient(connection),
			File:      basev1.NewFileServiceClient(connection),
			Language:  basev1.NewLanguageServiceClient(connection),
			Login:     basev1.NewLoginServiceClient(connection),
			Mcp:       basev1.NewMcpServiceClient(connection),
			Oauth:     basev1.NewOauthServiceClient(connection),
			Sse:       basev1.NewSseServiceClient(connection),
		},
		SystemAdmin: SystemAdminClient{
			Auth:             adminv1.NewAuthServiceClient(connection),
			BaseAPI:          adminv1.NewBaseApiServiceClient(connection),
			BaseArea:         adminv1.NewBaseAreaServiceClient(connection),
			BaseConfig:       adminv1.NewBaseConfigServiceClient(connection),
			BaseDept:         adminv1.NewBaseDeptServiceClient(connection),
			BaseDict:         adminv1.NewBaseDictServiceClient(connection),
			BaseJob:          adminv1.NewBaseJobServiceClient(connection),
			BaseLanguage:     adminv1.NewBaseLanguageServiceClient(connection),
			BaseLog:          adminv1.NewBaseLogServiceClient(connection),
			BaseMenu:         adminv1.NewBaseMenuServiceClient(connection),
			BaseMigration:    adminv1.NewBaseMigrationServiceClient(connection),
			BasePost:         adminv1.NewBasePostServiceClient(connection),
			BaseRole:         adminv1.NewBaseRoleServiceClient(connection),
			BaseTenant:       adminv1.NewBaseTenantServiceClient(connection),
			BaseThirdAccount: adminv1.NewBaseThirdAccountServiceClient(connection),
			BaseI18n:         adminv1.NewBaseI18nServiceClient(connection),
			BaseUser:         adminv1.NewBaseUserServiceClient(connection),
			CodeGen:          adminv1.NewCodeGenServiceClient(connection),
			CodeGenColumn:    adminv1.NewCodeGenColumnServiceClient(connection),
			CodeGenProto:     adminv1.NewCodeGenProtoServiceClient(connection),
			CodeGenTable:     adminv1.NewCodeGenTableServiceClient(connection),
			OpsMonitoring:    adminv1.NewOpsMonitoringServiceClient(connection),
			ProjectDocument:  adminv1.NewProjectDocumentServiceClient(connection),
		},
		SystemApp: SystemAppClient{
			Auth:     appv1.NewAuthServiceClient(connection),
			BaseArea: appv1.NewBaseAreaServiceClient(connection),
			BaseDict: appv1.NewBaseDictServiceClient(connection),
			BaseMenu: appv1.NewBaseMenuServiceClient(connection),
		},
	}, cleanup, nil
}

// NewConnection 根据客户端配置创建底层 gRPC 连接。
func NewConnection(ctx context.Context, clientConfig *configv1.Client, options ...Option) (*Connection, func(), error) {
	return coreclient.NewConnection(ctx, clientConfig, options...)
}

// WithDiscovery 为使用 discovery:/// 地址的连接注入服务发现器。
func WithDiscovery(discovery registry.Discovery) Option {
	return coreclient.WithDiscovery(discovery)
}

// WithLocalServices 配置进程内 gRPC 客户端需要注册的服务。
func WithLocalServices(registrars ...LocalServiceRegistrar) Option {
	return coreclient.WithLocalServices(registrars...)
}
