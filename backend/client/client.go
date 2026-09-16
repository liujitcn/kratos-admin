package client

import (
	"context"

	"github.com/go-kratos/kratos/v3/registry"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	appv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	"github.com/liujitcn/kratos-core/client"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// Connection 是 Core 客户端提供的统一 gRPC 连接。
type Connection = client.Connection

// Option 是 Core 客户端连接的可选配置。
type Option = client.Option

// LocalServiceRegistrar 描述向进程内 gRPC 客户端注册服务的函数。
type LocalServiceRegistrar = client.LocalServiceRegistrar

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
	// Mfa 是登录阶段多因素认证服务客户端。
	Mfa basev1.MfaServiceClient
	// Mcp 是 MCP 服务客户端。
	Mcp basev1.McpServiceClient
	// Notification 是通知服务客户端。
	Notification basev1.NotificationServiceClient
	// Oauth 是 OAuth 服务客户端。
	Oauth basev1.OauthServiceClient
	// OauthClient 是开放授权客户端令牌服务客户端。
	OauthClient basev1.OauthClientServiceClient
	// Sse 是 SSE 服务客户端。
	Sse basev1.SseServiceClient
}

// SystemAdminClient 汇总 system.admin.v1 服务客户端。
type SystemAdminClient struct {
	// BaseApiLog 是管理端API 日志服务客户端。
	BaseApiLog adminv1.BaseApiLogServiceClient
	// BaseDashboard 是管理端工作台服务客户端。
	BaseDashboard adminv1.BaseDashboardServiceClient
	// BaseDataAccessLog 是管理端数据访问日志服务客户端。
	BaseDataAccessLog adminv1.BaseDataAccessLogServiceClient
	// BaseDictItem 是管理端字典项服务客户端。
	BaseDictItem adminv1.BaseDictItemServiceClient
	// BaseFile 是管理端文件资产服务客户端。
	BaseFile adminv1.BaseFileServiceClient
	// BaseI18nCustom 是管理端自定义翻译服务客户端。
	BaseI18nCustom adminv1.BaseI18nCustomServiceClient
	// BaseJobLog 是管理端定时任务日志服务客户端。
	BaseJobLog adminv1.BaseJobLogServiceClient
	// BaseLog 是管理端日志追踪服务客户端。
	BaseLog adminv1.BaseLogServiceClient
	// BaseLoginLog 是管理端登录日志服务客户端。
	BaseLoginLog adminv1.BaseLoginLogServiceClient
	// BaseMessage 是管理端消息服务客户端。
	BaseMessage adminv1.BaseMessageServiceClient
	// BaseMessageCategory 是管理端消息分类服务客户端。
	BaseMessageCategory adminv1.BaseMessageCategoryServiceClient
	// BaseOperationLog 是管理端操作日志服务客户端。
	BaseOperationLog adminv1.BaseOperationLogServiceClient
	// BasePermissionLog 是管理端权限日志服务客户端。
	BasePermissionLog adminv1.BasePermissionLogServiceClient
	// BasePolicyEvaluationLog 是管理端策略评估日志服务客户端。
	BasePolicyEvaluationLog adminv1.BasePolicyEvaluationLogServiceClient
	// BaseRedactOutputPolicy 是管理端输出脱敏策略服务客户端。
	BaseRedactOutputPolicy adminv1.BaseRedactOutputPolicyServiceClient
	// BaseRedactRule 是管理端脱敏规则服务客户端。
	BaseRedactRule adminv1.BaseRedactRuleServiceClient
	// BaseRedactStoragePolicy 是管理端存储脱敏策略服务客户端。
	BaseRedactStoragePolicy adminv1.BaseRedactStoragePolicyServiceClient
	// BaseTableArchive 是管理端数据归档配置服务客户端。
	BaseTableArchive adminv1.BaseTableArchiveServiceClient
	// BaseTableArchiveRecord 是管理端数据归档记录服务客户端。
	BaseTableArchiveRecord adminv1.BaseTableArchiveRecordServiceClient
	// BaseTableArchiveRestore 是管理端数据归档恢复服务客户端。
	BaseTableArchiveRestore adminv1.BaseTableArchiveRestoreServiceClient
	// BaseTableBackup 是管理端数据备份配置服务客户端。
	BaseTableBackup adminv1.BaseTableBackupServiceClient
	// BaseTableBackupRecord 是管理端数据备份记录服务客户端。
	BaseTableBackupRecord adminv1.BaseTableBackupRecordServiceClient
	// BaseTableBackupRestore 是管理端数据备份恢复服务客户端。
	BaseTableBackupRestore adminv1.BaseTableBackupRestoreServiceClient
	// BaseTableSource 是管理端数据源服务客户端。
	BaseTableSource adminv1.BaseTableSourceServiceClient
	// RuntimeLog 是管理端运行日志服务客户端。
	RuntimeLog adminv1.RuntimeLogServiceClient
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
	// BaseJob 是管理端定时任务服务客户端。
	BaseJob adminv1.BaseJobServiceClient
	// BaseLanguage 是管理端语言服务客户端。
	BaseLanguage adminv1.BaseLanguageServiceClient
	// BaseMenu 是管理端菜单服务客户端。
	BaseMenu adminv1.BaseMenuServiceClient
	// BaseMigration 是管理端迁移服务客户端。
	BaseMigration adminv1.BaseMigrationServiceClient
	// BasePost 是管理端岗位服务客户端。
	BasePost adminv1.BasePostServiceClient
	// BaseTenantProject 是租户项目客户端。
	BaseTenantProject adminv1.BaseTenantProjectServiceClient
	// BaseTenantProjectGrant 是四维项目授权客户端。
	BaseTenantProjectGrant adminv1.BaseTenantProjectGrantServiceClient
	// BaseSession 是管理端会话服务客户端。
	BaseSession adminv1.BaseSessionServiceClient
	// BaseLoginPolicy 是管理端登录策略服务客户端。
	BaseLoginPolicy adminv1.BaseLoginPolicyServiceClient
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
	// Cache 是运行时缓存查询服务客户端。
	Cache adminv1.CacheServiceClient
	// OauthClient 是开放授权客户端管理服务客户端。
	OauthClient adminv1.OauthClientServiceClient
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
	connection, cleanup, err := client.NewConnection(ctx, clientConfig, options...)
	if err != nil {
		return nil, nil, err
	}
	return &Client{
		Connection: connection,
		Base: BaseClient{
			AiMessage:    basev1.NewAiMessageServiceClient(connection),
			AiSession:    basev1.NewAiSessionServiceClient(connection),
			AiTool:       basev1.NewAiToolServiceClient(connection),
			Config:       basev1.NewConfigServiceClient(connection),
			File:         basev1.NewFileServiceClient(connection),
			Language:     basev1.NewLanguageServiceClient(connection),
			Login:        basev1.NewLoginServiceClient(connection),
			Mfa:          basev1.NewMfaServiceClient(connection),
			Mcp:          basev1.NewMcpServiceClient(connection),
			Notification: basev1.NewNotificationServiceClient(connection),
			Oauth:        basev1.NewOauthServiceClient(connection),
			OauthClient:  basev1.NewOauthClientServiceClient(connection),
			Sse:          basev1.NewSseServiceClient(connection),
		},
		SystemAdmin: SystemAdminClient{
			BaseApiLog:              adminv1.NewBaseApiLogServiceClient(connection),
			BaseDashboard:           adminv1.NewBaseDashboardServiceClient(connection),
			BaseDataAccessLog:       adminv1.NewBaseDataAccessLogServiceClient(connection),
			BaseDictItem:            adminv1.NewBaseDictItemServiceClient(connection),
			BaseFile:                adminv1.NewBaseFileServiceClient(connection),
			BaseI18nCustom:          adminv1.NewBaseI18nCustomServiceClient(connection),
			BaseJobLog:              adminv1.NewBaseJobLogServiceClient(connection),
			BaseLog:                 adminv1.NewBaseLogServiceClient(connection),
			BaseLoginLog:            adminv1.NewBaseLoginLogServiceClient(connection),
			BaseMessage:             adminv1.NewBaseMessageServiceClient(connection),
			BaseMessageCategory:     adminv1.NewBaseMessageCategoryServiceClient(connection),
			BaseOperationLog:        adminv1.NewBaseOperationLogServiceClient(connection),
			BasePermissionLog:       adminv1.NewBasePermissionLogServiceClient(connection),
			BasePolicyEvaluationLog: adminv1.NewBasePolicyEvaluationLogServiceClient(connection),
			BaseRedactOutputPolicy:  adminv1.NewBaseRedactOutputPolicyServiceClient(connection),
			BaseRedactRule:          adminv1.NewBaseRedactRuleServiceClient(connection),
			BaseRedactStoragePolicy: adminv1.NewBaseRedactStoragePolicyServiceClient(connection),
			BaseTableArchive:        adminv1.NewBaseTableArchiveServiceClient(connection),
			BaseTableArchiveRecord:  adminv1.NewBaseTableArchiveRecordServiceClient(connection),
			BaseTableArchiveRestore: adminv1.NewBaseTableArchiveRestoreServiceClient(connection),
			BaseTableBackup:         adminv1.NewBaseTableBackupServiceClient(connection),
			BaseTableBackupRecord:   adminv1.NewBaseTableBackupRecordServiceClient(connection),
			BaseTableBackupRestore:  adminv1.NewBaseTableBackupRestoreServiceClient(connection),
			BaseTableSource:         adminv1.NewBaseTableSourceServiceClient(connection),
			RuntimeLog:              adminv1.NewRuntimeLogServiceClient(connection),
			Auth:                    adminv1.NewAuthServiceClient(connection),
			BaseAPI:                 adminv1.NewBaseApiServiceClient(connection),
			BaseArea:                adminv1.NewBaseAreaServiceClient(connection),
			BaseConfig:              adminv1.NewBaseConfigServiceClient(connection),
			BaseDept:                adminv1.NewBaseDeptServiceClient(connection),
			BaseDict:                adminv1.NewBaseDictServiceClient(connection),
			BaseJob:                 adminv1.NewBaseJobServiceClient(connection),
			BaseLanguage:            adminv1.NewBaseLanguageServiceClient(connection),
			BaseMenu:                adminv1.NewBaseMenuServiceClient(connection),
			BaseMigration:           adminv1.NewBaseMigrationServiceClient(connection),
			BasePost:                adminv1.NewBasePostServiceClient(connection),
			BaseTenantProject:       adminv1.NewBaseTenantProjectServiceClient(connection),
			BaseTenantProjectGrant:  adminv1.NewBaseTenantProjectGrantServiceClient(connection),
			BaseSession:             adminv1.NewBaseSessionServiceClient(connection),
			BaseLoginPolicy:         adminv1.NewBaseLoginPolicyServiceClient(connection),
			BaseRole:                adminv1.NewBaseRoleServiceClient(connection),
			BaseTenant:              adminv1.NewBaseTenantServiceClient(connection),
			BaseThirdAccount:        adminv1.NewBaseThirdAccountServiceClient(connection),
			BaseI18n:                adminv1.NewBaseI18nServiceClient(connection),
			BaseUser:                adminv1.NewBaseUserServiceClient(connection),
			CodeGen:                 adminv1.NewCodeGenServiceClient(connection),
			CodeGenColumn:           adminv1.NewCodeGenColumnServiceClient(connection),
			CodeGenProto:            adminv1.NewCodeGenProtoServiceClient(connection),
			CodeGenTable:            adminv1.NewCodeGenTableServiceClient(connection),
			OpsMonitoring:           adminv1.NewOpsMonitoringServiceClient(connection),
			Cache:                   adminv1.NewCacheServiceClient(connection),
			OauthClient:             adminv1.NewOauthClientServiceClient(connection),
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
	return client.NewConnection(ctx, clientConfig, options...)
}

// WithDiscovery 为使用 discovery:/// 地址的连接注入服务发现器。
func WithDiscovery(discovery registry.Discovery) Option {
	return client.WithDiscovery(discovery)
}

// WithLocalServices 配置进程内 gRPC 客户端需要注册的服务。
func WithLocalServices(registrars ...LocalServiceRegistrar) Option {
	return client.WithLocalServices(registrars...)
}
