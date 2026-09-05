package admin

import (
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/oauth"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/passwordpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/sessionpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"
	coreBiz "github.com/liujitcn/kratos-core/biz"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 system.admin.v1 的服务实现。
type Services struct {
	Auth        *admin.AuthService
	BaseAPI     *admin.BaseApiService
	OauthClient *admin.OauthClientService

	BaseAPICase              *biz.BaseAPICase
	BaseUserRepository       *data.BaseUserRepository
	OauthClientRepository    *data.OauthClientRepository
	Authenticator            engine.Authenticator
	OauthCredentialProtector *oauthsecret.Protector
	LogMiddleware            middleware.Middleware
	BaseCase                 *coreBiz.BaseCase
	UserToken                *authData.UserToken

	BaseArea                *admin.BaseAreaService
	BaseConfig              *admin.BaseConfigService
	BaseDept                *admin.BaseDeptService
	BaseDict                *admin.BaseDictService
	BaseDictItem            *admin.BaseDictItemService
	BaseJob                 *admin.BaseJobService
	BaseJobLog              *admin.BaseJobLogService
	BaseLanguage            *admin.BaseLanguageService
	BaseLog                 *admin.BaseLogService
	BaseLoginLog            *admin.BaseLoginLogService
	BaseApiLog              *admin.BaseApiLogService
	BaseOperationLog        *admin.BaseOperationLogService
	BaseDataAccessLog       *admin.BaseDataAccessLogService
	BasePermissionLog       *admin.BasePermissionLogService
	BasePolicyEvaluationLog *admin.BasePolicyEvaluationLogService
	BaseDashboard           *admin.BaseDashboardService
	BaseFile                *admin.BaseFileService
	BaseMenu                *admin.BaseMenuService
	BaseMessage             *admin.BaseMessageService
	BaseMessageCategory     *admin.BaseMessageCategoryService
	BasePost                *admin.BasePostService
	BaseRole                *admin.BaseRoleService
	BaseTenant              *admin.BaseTenantService
	BaseThirdAccount        *admin.BaseThirdAccountService
	BaseI18n                *admin.BaseI18nService
	BaseUser                *admin.BaseUserService
	CodeGen                 *admin.CodeGenService
	CodeGenColumn           *admin.CodeGenColumnService
	CodeGenProto            *admin.CodeGenProtoService
	CodeGenTable            *admin.CodeGenTableService
	BaseMigration           *admin.BaseMigrationService
	OpsMonitoring           *admin.OpsMonitoringService
	Cache                   *admin.CacheService
	RuntimeLog              *admin.RuntimeLogService
	ProjectDocument         *admin.ProjectDocumentService
	BaseSession             *admin.BaseSessionService
	BaseLoginPolicy         *admin.BaseLoginPolicyService
	BaseTableArchive        *admin.BaseTableArchiveService
	BaseTableArchiveRecord  *admin.BaseTableArchiveRecordService
	BaseTableArchiveRestore *admin.BaseTableArchiveRestoreService
	BaseTableBackup         *admin.BaseTableBackupService
	BaseTableBackupRecord   *admin.BaseTableBackupRecordService
	BaseTableBackupRestore  *admin.BaseTableBackupRestoreService
	BaseTableSource         *admin.BaseTableSourceService
}

// RegisterGRPC 注册 system.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	adminv1.RegisterAuthServiceServer(srv, adminv1.RedactedAuthServiceServer(s.Auth))
	adminv1.RegisterBaseApiServiceServer(srv, adminv1.RedactedBaseApiServiceServer(s.BaseAPI))
	adminv1.RegisterOauthClientServiceServer(srv, adminv1.RedactedOauthClientServiceServer(s.OauthClient))

	adminv1.RegisterBaseAreaServiceServer(srv, adminv1.RedactedBaseAreaServiceServer(s.BaseArea))
	adminv1.RegisterBaseConfigServiceServer(srv, adminv1.RedactedBaseConfigServiceServer(s.BaseConfig))
	adminv1.RegisterBaseDeptServiceServer(srv, adminv1.RedactedBaseDeptServiceServer(s.BaseDept))
	adminv1.RegisterBaseDictServiceServer(srv, adminv1.RedactedBaseDictServiceServer(s.BaseDict))
	adminv1.RegisterBaseDictItemServiceServer(srv, adminv1.RedactedBaseDictItemServiceServer(s.BaseDictItem))
	adminv1.RegisterBaseJobServiceServer(srv, adminv1.RedactedBaseJobServiceServer(s.BaseJob))
	adminv1.RegisterBaseJobLogServiceServer(srv, adminv1.RedactedBaseJobLogServiceServer(s.BaseJobLog))
	adminv1.RegisterBaseLanguageServiceServer(srv, adminv1.RedactedBaseLanguageServiceServer(s.BaseLanguage))
	adminv1.RegisterBaseLogServiceServer(srv, adminv1.RedactedBaseLogServiceServer(s.BaseLog))
	adminv1.RegisterBaseLoginLogServiceServer(srv, adminv1.RedactedBaseLoginLogServiceServer(s.BaseLoginLog))
	adminv1.RegisterBaseApiLogServiceServer(srv, adminv1.RedactedBaseApiLogServiceServer(s.BaseApiLog))
	adminv1.RegisterBaseOperationLogServiceServer(srv, adminv1.RedactedBaseOperationLogServiceServer(s.BaseOperationLog))
	adminv1.RegisterBaseDataAccessLogServiceServer(srv, adminv1.RedactedBaseDataAccessLogServiceServer(s.BaseDataAccessLog))
	adminv1.RegisterBasePermissionLogServiceServer(srv, adminv1.RedactedBasePermissionLogServiceServer(s.BasePermissionLog))
	adminv1.RegisterBasePolicyEvaluationLogServiceServer(srv, adminv1.RedactedBasePolicyEvaluationLogServiceServer(s.BasePolicyEvaluationLog))
	adminv1.RegisterBaseDashboardServiceServer(srv, adminv1.RedactedBaseDashboardServiceServer(s.BaseDashboard))
	adminv1.RegisterBaseFileServiceServer(srv, adminv1.RedactedBaseFileServiceServer(s.BaseFile))
	adminv1.RegisterBaseMenuServiceServer(srv, adminv1.RedactedBaseMenuServiceServer(s.BaseMenu))
	adminv1.RegisterBaseMessageServiceServer(srv, adminv1.RedactedBaseMessageServiceServer(s.BaseMessage))
	adminv1.RegisterBaseMessageCategoryServiceServer(srv, adminv1.RedactedBaseMessageCategoryServiceServer(s.BaseMessageCategory))
	adminv1.RegisterBasePostServiceServer(srv, adminv1.RedactedBasePostServiceServer(s.BasePost))
	adminv1.RegisterBaseRoleServiceServer(srv, adminv1.RedactedBaseRoleServiceServer(s.BaseRole))
	adminv1.RegisterBaseTenantServiceServer(srv, adminv1.RedactedBaseTenantServiceServer(s.BaseTenant))
	adminv1.RegisterBaseThirdAccountServiceServer(srv, adminv1.RedactedBaseThirdAccountServiceServer(s.BaseThirdAccount))
	adminv1.RegisterBaseI18nServiceServer(srv, adminv1.RedactedBaseI18nServiceServer(s.BaseI18n))
	adminv1.RegisterBaseUserServiceServer(srv, adminv1.RedactedBaseUserServiceServer(s.BaseUser))
	adminv1.RegisterCodeGenServiceServer(srv, adminv1.RedactedCodeGenServiceServer(s.CodeGen))
	adminv1.RegisterCodeGenColumnServiceServer(srv, adminv1.RedactedCodeGenColumnServiceServer(s.CodeGenColumn))
	adminv1.RegisterCodeGenProtoServiceServer(srv, adminv1.RedactedCodeGenProtoServiceServer(s.CodeGenProto))
	adminv1.RegisterCodeGenTableServiceServer(srv, adminv1.RedactedCodeGenTableServiceServer(s.CodeGenTable))
	adminv1.RegisterBaseMigrationServiceServer(srv, adminv1.RedactedBaseMigrationServiceServer(s.BaseMigration))
	adminv1.RegisterOpsMonitoringServiceServer(srv, adminv1.RedactedOpsMonitoringServiceServer(s.OpsMonitoring))
	adminv1.RegisterCacheServiceServer(srv, adminv1.RedactedCacheServiceServer(s.Cache))
	adminv1.RegisterRuntimeLogServiceServer(srv, adminv1.RedactedRuntimeLogServiceServer(s.RuntimeLog))
	adminv1.RegisterProjectDocumentServiceServer(srv, adminv1.RedactedProjectDocumentServiceServer(s.ProjectDocument))
	adminv1.RegisterBaseSessionServiceServer(srv, adminv1.RedactedBaseSessionServiceServer(s.BaseSession))
	adminv1.RegisterBaseLoginPolicyServiceServer(srv, adminv1.RedactedBaseLoginPolicyServiceServer(s.BaseLoginPolicy))
	adminv1.RegisterBaseTableArchiveServiceServer(srv, adminv1.RedactedBaseTableArchiveServiceServer(s.BaseTableArchive))
	adminv1.RegisterBaseTableArchiveRecordServiceServer(srv, adminv1.RedactedBaseTableArchiveRecordServiceServer(s.BaseTableArchiveRecord))
	adminv1.RegisterBaseTableArchiveRestoreServiceServer(srv, adminv1.RedactedBaseTableArchiveRestoreServiceServer(s.BaseTableArchiveRestore))
	adminv1.RegisterBaseTableBackupServiceServer(srv, adminv1.RedactedBaseTableBackupServiceServer(s.BaseTableBackup))
	adminv1.RegisterBaseTableBackupRecordServiceServer(srv, adminv1.RedactedBaseTableBackupRecordServiceServer(s.BaseTableBackupRecord))
	adminv1.RegisterBaseTableBackupRestoreServiceServer(srv, adminv1.RedactedBaseTableBackupRestoreServiceServer(s.BaseTableBackupRestore))
	adminv1.RegisterBaseTableSourceServiceServer(srv, adminv1.RedactedBaseTableSourceServiceServer(s.BaseTableSource))
}

// RegisterHTTP 注册 system.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	policyMiddleware := passwordpolicy.NewMiddleware(s.BaseUserRepository, s.BaseCase.Cache)
	sessionMiddleware := sessionpolicy.NewMiddleware(s.BaseCase, s.UserToken)
	srv.Use("/*", oauth.NewIPMiddleware(s.OauthClientRepository), oauth.NewClientMiddleware(s.OauthClientRepository, s.BaseAPICase), s.LogMiddleware, sessionMiddleware, policyMiddleware)
	srv.Use("/system.admin.v1.*", middleware.Chain(s.LogMiddleware, sessionMiddleware, policyMiddleware))
	adminv1.RegisterAuthServiceHTTPServer(srv, adminv1.RedactedAuthServiceServer(s.Auth))
	adminv1.RegisterBaseApiServiceHTTPServer(srv, adminv1.RedactedBaseApiServiceServer(s.BaseAPI))
	adminv1.RegisterOauthClientServiceHTTPServer(srv, adminv1.RedactedOauthClientServiceServer(s.OauthClient))

	adminv1.RegisterBaseAreaServiceHTTPServer(srv, adminv1.RedactedBaseAreaServiceServer(s.BaseArea))
	adminv1.RegisterBaseConfigServiceHTTPServer(srv, adminv1.RedactedBaseConfigServiceServer(s.BaseConfig))
	adminv1.RegisterBaseDeptServiceHTTPServer(srv, adminv1.RedactedBaseDeptServiceServer(s.BaseDept))
	adminv1.RegisterBaseDictServiceHTTPServer(srv, adminv1.RedactedBaseDictServiceServer(s.BaseDict))
	adminv1.RegisterBaseDictItemServiceHTTPServer(srv, adminv1.RedactedBaseDictItemServiceServer(s.BaseDictItem))
	adminv1.RegisterBaseJobServiceHTTPServer(srv, adminv1.RedactedBaseJobServiceServer(s.BaseJob))
	adminv1.RegisterBaseJobLogServiceHTTPServer(srv, adminv1.RedactedBaseJobLogServiceServer(s.BaseJobLog))
	adminv1.RegisterBaseLanguageServiceHTTPServer(srv, adminv1.RedactedBaseLanguageServiceServer(s.BaseLanguage))
	adminv1.RegisterBaseLogServiceHTTPServer(srv, adminv1.RedactedBaseLogServiceServer(s.BaseLog))
	adminv1.RegisterBaseLoginLogServiceHTTPServer(srv, adminv1.RedactedBaseLoginLogServiceServer(s.BaseLoginLog))
	adminv1.RegisterBaseApiLogServiceHTTPServer(srv, adminv1.RedactedBaseApiLogServiceServer(s.BaseApiLog))
	adminv1.RegisterBaseOperationLogServiceHTTPServer(srv, adminv1.RedactedBaseOperationLogServiceServer(s.BaseOperationLog))
	adminv1.RegisterBaseDataAccessLogServiceHTTPServer(srv, adminv1.RedactedBaseDataAccessLogServiceServer(s.BaseDataAccessLog))
	adminv1.RegisterBasePermissionLogServiceHTTPServer(srv, adminv1.RedactedBasePermissionLogServiceServer(s.BasePermissionLog))
	adminv1.RegisterBasePolicyEvaluationLogServiceHTTPServer(srv, adminv1.RedactedBasePolicyEvaluationLogServiceServer(s.BasePolicyEvaluationLog))
	adminv1.RegisterBaseDashboardServiceHTTPServer(srv, adminv1.RedactedBaseDashboardServiceServer(s.BaseDashboard))
	adminv1.RegisterBaseFileServiceHTTPServer(srv, adminv1.RedactedBaseFileServiceServer(s.BaseFile))
	adminv1.RegisterBaseMenuServiceHTTPServer(srv, adminv1.RedactedBaseMenuServiceServer(s.BaseMenu))
	adminv1.RegisterBaseMessageServiceHTTPServer(srv, adminv1.RedactedBaseMessageServiceServer(s.BaseMessage))
	adminv1.RegisterBaseMessageCategoryServiceHTTPServer(srv, adminv1.RedactedBaseMessageCategoryServiceServer(s.BaseMessageCategory))
	adminv1.RegisterBasePostServiceHTTPServer(srv, adminv1.RedactedBasePostServiceServer(s.BasePost))
	adminv1.RegisterBaseRoleServiceHTTPServer(srv, adminv1.RedactedBaseRoleServiceServer(s.BaseRole))
	adminv1.RegisterBaseTenantServiceHTTPServer(srv, adminv1.RedactedBaseTenantServiceServer(s.BaseTenant))
	adminv1.RegisterBaseI18nServiceHTTPServer(srv, adminv1.RedactedBaseI18nServiceServer(s.BaseI18n))
	adminv1.RegisterBaseUserServiceHTTPServer(srv, adminv1.RedactedBaseUserServiceServer(s.BaseUser))
	adminv1.RegisterCodeGenServiceHTTPServer(srv, adminv1.RedactedCodeGenServiceServer(s.CodeGen))
	adminv1.RegisterCodeGenColumnServiceHTTPServer(srv, adminv1.RedactedCodeGenColumnServiceServer(s.CodeGenColumn))
	adminv1.RegisterCodeGenProtoServiceHTTPServer(srv, adminv1.RedactedCodeGenProtoServiceServer(s.CodeGenProto))
	adminv1.RegisterCodeGenTableServiceHTTPServer(srv, adminv1.RedactedCodeGenTableServiceServer(s.CodeGenTable))
	adminv1.RegisterBaseMigrationServiceHTTPServer(srv, adminv1.RedactedBaseMigrationServiceServer(s.BaseMigration))
	adminv1.RegisterOpsMonitoringServiceHTTPServer(srv, adminv1.RedactedOpsMonitoringServiceServer(s.OpsMonitoring))
	adminv1.RegisterCacheServiceHTTPServer(srv, adminv1.RedactedCacheServiceServer(s.Cache))
	adminv1.RegisterRuntimeLogServiceHTTPServer(srv, adminv1.RedactedRuntimeLogServiceServer(s.RuntimeLog))
	adminv1.RegisterProjectDocumentServiceHTTPServer(srv, adminv1.RedactedProjectDocumentServiceServer(s.ProjectDocument))
	adminv1.RegisterBaseSessionServiceHTTPServer(srv, adminv1.RedactedBaseSessionServiceServer(s.BaseSession))
	adminv1.RegisterBaseLoginPolicyServiceHTTPServer(srv, adminv1.RedactedBaseLoginPolicyServiceServer(s.BaseLoginPolicy))
	adminv1.RegisterBaseTableArchiveServiceHTTPServer(srv, adminv1.RedactedBaseTableArchiveServiceServer(s.BaseTableArchive))
	adminv1.RegisterBaseTableArchiveRecordServiceHTTPServer(srv, adminv1.RedactedBaseTableArchiveRecordServiceServer(s.BaseTableArchiveRecord))
	adminv1.RegisterBaseTableArchiveRestoreServiceHTTPServer(srv, adminv1.RedactedBaseTableArchiveRestoreServiceServer(s.BaseTableArchiveRestore))
	adminv1.RegisterBaseTableBackupServiceHTTPServer(srv, adminv1.RedactedBaseTableBackupServiceServer(s.BaseTableBackup))
	adminv1.RegisterBaseTableBackupRecordServiceHTTPServer(srv, adminv1.RedactedBaseTableBackupRecordServiceServer(s.BaseTableBackupRecord))
	adminv1.RegisterBaseTableBackupRestoreServiceHTTPServer(srv, adminv1.RedactedBaseTableBackupRestoreServiceServer(s.BaseTableBackupRestore))
	adminv1.RegisterBaseTableSourceServiceHTTPServer(srv, adminv1.RedactedBaseTableSourceServiceServer(s.BaseTableSource))
}

// RegisterMCP 注册 system.admin.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcp.Server) {
	mcpSrv := server.MCPServer()
	adminv1.RegisterAuthServiceMCPTools(mcpSrv, s.Auth)
	adminv1.RegisterBaseApiServiceMCPTools(mcpSrv, s.BaseAPI)

	adminv1.RegisterBaseAreaServiceMCPTools(mcpSrv, s.BaseArea)
	adminv1.RegisterBaseConfigServiceMCPTools(mcpSrv, s.BaseConfig)
	adminv1.RegisterBaseDeptServiceMCPTools(mcpSrv, s.BaseDept)
	adminv1.RegisterBaseDictServiceMCPTools(mcpSrv, s.BaseDict)
	adminv1.RegisterBaseDictItemServiceMCPTools(mcpSrv, s.BaseDictItem)
	adminv1.RegisterBaseJobServiceMCPTools(mcpSrv, s.BaseJob)
	adminv1.RegisterBaseJobLogServiceMCPTools(mcpSrv, s.BaseJobLog)
	adminv1.RegisterBaseLanguageServiceMCPTools(mcpSrv, s.BaseLanguage)
	adminv1.RegisterBaseMenuServiceMCPTools(mcpSrv, s.BaseMenu)
	adminv1.RegisterBasePostServiceMCPTools(mcpSrv, s.BasePost)
	adminv1.RegisterBaseRoleServiceMCPTools(mcpSrv, s.BaseRole)
	adminv1.RegisterBaseI18nServiceMCPTools(mcpSrv, s.BaseI18n)
	// 角色切换仅开放 gRPC，MCP 显式注册其余用户管理方法。
	adminv1.RegisterBaseUserServiceOptionBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceListBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServicePageBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceGetBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceCreateBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceUpdateBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceDeleteBaseUserMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceSetBaseUserStatusMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterBaseUserServiceResetBaseUserPasswordMCPTool(mcpSrv, s.BaseUser)
	adminv1.RegisterCodeGenServiceMCPTools(mcpSrv, s.CodeGen)
	adminv1.RegisterCodeGenColumnServiceMCPTools(mcpSrv, s.CodeGenColumn)
	adminv1.RegisterCodeGenProtoServiceMCPTools(mcpSrv, s.CodeGenProto)
	adminv1.RegisterCodeGenTableServiceMCPTools(mcpSrv, s.CodeGenTable)
	adminv1.RegisterBaseMigrationServiceMCPTools(mcpSrv, s.BaseMigration)
	adminv1.RegisterOpsMonitoringServiceMCPTools(mcpSrv, s.OpsMonitoring)
	adminv1.RegisterCacheServiceMCPTools(mcpSrv, s.Cache)
	adminv1.RegisterProjectDocumentServiceMCPTools(mcpSrv, s.ProjectDocument)
	adminv1.RegisterBaseSessionServiceMCPTools(mcpSrv, s.BaseSession)
	adminv1.RegisterBaseLoginPolicyServiceMCPTools(mcpSrv, s.BaseLoginPolicy)
	adminv1.RegisterBaseDashboardServiceMCPTools(mcpSrv, s.BaseDashboard)
	adminv1.RegisterBaseFileServiceMCPTools(mcpSrv, s.BaseFile)
	adminv1.RegisterBaseTableSourceServiceMCPTools(mcpSrv, s.BaseTableSource)
}

// AgentTools 创建 system.admin.v1 的管理端 AI 助手工具。
func (s Services) AgentTools() ([]tool.Invokable, error) {
	builders := []func() ([]tool.Invokable, error){
		func() ([]tool.Invokable, error) { return adminv1.NewAuthServiceAgentTools(s.Auth) },
		func() ([]tool.Invokable, error) { return adminv1.NewBaseApiServiceAgentTools(s.BaseAPI) },
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseAreaServiceAgentTools(s.BaseArea)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseConfigServiceAgentTools(s.BaseConfig)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseDeptServiceAgentTools(s.BaseDept)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseDictServiceAgentTools(s.BaseDict)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseDictItemServiceAgentTools(s.BaseDictItem)
		},
		func() ([]tool.Invokable, error) { return adminv1.NewBaseJobServiceAgentTools(s.BaseJob) },
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseJobLogServiceAgentTools(s.BaseJobLog)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseLanguageServiceAgentTools(s.BaseLanguage)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseMenuServiceAgentTools(s.BaseMenu)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBasePostServiceAgentTools(s.BasePost)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseRoleServiceAgentTools(s.BaseRole)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseTenantServiceAgentTools(s.BaseTenant)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseThirdAccountServiceAgentTools(s.BaseThirdAccount)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseI18nServiceAgentTools(s.BaseI18n)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseUserServiceAgentTools(s.BaseUser)
		},
		func() ([]tool.Invokable, error) { return adminv1.NewCodeGenServiceAgentTools(s.CodeGen) },
		func() ([]tool.Invokable, error) {
			return adminv1.NewCodeGenColumnServiceAgentTools(s.CodeGenColumn)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewCodeGenProtoServiceAgentTools(s.CodeGenProto)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewCodeGenTableServiceAgentTools(s.CodeGenTable)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseMigrationServiceAgentTools(s.BaseMigration)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseSessionServiceAgentTools(s.BaseSession)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseLoginPolicyServiceAgentTools(s.BaseLoginPolicy)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseDashboardServiceAgentTools(s.BaseDashboard)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseFileServiceAgentTools(s.BaseFile)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewBaseTableSourceServiceAgentTools(s.BaseTableSource)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewOpsMonitoringServiceAgentTools(s.OpsMonitoring)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewCacheServiceAgentTools(s.Cache)
		},
		func() ([]tool.Invokable, error) {
			return adminv1.NewProjectDocumentServiceAgentTools(s.ProjectDocument)
		},
	}
	tools := make([]tool.Invokable, 0)
	var err error
	for _, build := range builders {
		var values []tool.Invokable
		values, err = build()
		if err != nil {
			return nil, err
		}
		tools = append(tools, values...)
	}
	return tools, nil
}
