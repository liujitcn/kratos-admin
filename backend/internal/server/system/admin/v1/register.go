package admin

import (
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/oauth"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/passwordpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
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
	AuditLogMiddleware       middleware.Middleware

	BaseArea                *admin.BaseAreaService
	BaseConfig              *admin.BaseConfigService
	BaseDept                *admin.BaseDeptService
	BaseDict                *admin.BaseDictService
	BaseJob                 *admin.BaseJobService
	BaseLanguage            *admin.BaseLanguageService
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
	RuntimeLog              *admin.RuntimeLogService
	ProjectDocument         *admin.ProjectDocumentService
	BaseSession             *admin.BaseSessionService
	BaseLoginPolicy         *admin.BaseLoginPolicyService
}

// RegisterGRPC 注册 system.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	adminv1.RegisterAuthServiceServer(srv, s.Auth)
	adminv1.RegisterBaseApiServiceServer(srv, s.BaseAPI)
	adminv1.RegisterOauthClientServiceServer(srv, s.OauthClient)

	adminv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	adminv1.RegisterBaseConfigServiceServer(srv, s.BaseConfig)
	adminv1.RegisterBaseDeptServiceServer(srv, s.BaseDept)
	adminv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
	adminv1.RegisterBaseJobServiceServer(srv, s.BaseJob)
	adminv1.RegisterBaseLanguageServiceServer(srv, s.BaseLanguage)
	adminv1.RegisterBaseLoginLogServiceServer(srv, s.BaseLoginLog)
	adminv1.RegisterBaseApiLogServiceServer(srv, s.BaseApiLog)
	adminv1.RegisterBaseOperationLogServiceServer(srv, s.BaseOperationLog)
	adminv1.RegisterBaseDataAccessLogServiceServer(srv, s.BaseDataAccessLog)
	adminv1.RegisterBasePermissionLogServiceServer(srv, s.BasePermissionLog)
	adminv1.RegisterBasePolicyEvaluationLogServiceServer(srv, s.BasePolicyEvaluationLog)
	adminv1.RegisterBaseDashboardServiceServer(srv, s.BaseDashboard)
	adminv1.RegisterBaseFileServiceServer(srv, s.BaseFile)
	adminv1.RegisterBaseMenuServiceServer(srv, s.BaseMenu)
	adminv1.RegisterBaseMessageServiceServer(srv, s.BaseMessage)
	adminv1.RegisterBaseMessageCategoryServiceServer(srv, s.BaseMessageCategory)
	adminv1.RegisterBasePostServiceServer(srv, s.BasePost)
	adminv1.RegisterBaseRoleServiceServer(srv, s.BaseRole)
	adminv1.RegisterBaseTenantServiceServer(srv, s.BaseTenant)
	adminv1.RegisterBaseThirdAccountServiceServer(srv, s.BaseThirdAccount)
	adminv1.RegisterBaseI18nServiceServer(srv, s.BaseI18n)
	adminv1.RegisterBaseUserServiceServer(srv, s.BaseUser)
	adminv1.RegisterCodeGenServiceServer(srv, s.CodeGen)
	adminv1.RegisterCodeGenColumnServiceServer(srv, s.CodeGenColumn)
	adminv1.RegisterCodeGenProtoServiceServer(srv, s.CodeGenProto)
	adminv1.RegisterCodeGenTableServiceServer(srv, s.CodeGenTable)
	adminv1.RegisterBaseMigrationServiceServer(srv, s.BaseMigration)
	adminv1.RegisterOpsMonitoringServiceServer(srv, s.OpsMonitoring)
	adminv1.RegisterRuntimeLogServiceServer(srv, s.RuntimeLog)
	adminv1.RegisterProjectDocumentServiceServer(srv, s.ProjectDocument)
	adminv1.RegisterBaseSessionServiceServer(srv, s.BaseSession)
	adminv1.RegisterBaseLoginPolicyServiceServer(srv, s.BaseLoginPolicy)
}

// RegisterHTTP 注册 system.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	policyMiddleware := passwordpolicy.NewMiddleware(s.BaseUserRepository)
	srv.Use("/*", oauth.NewIPMiddleware(s.OauthClientRepository), oauth.NewClientMiddleware(s.OauthClientRepository, s.BaseAPICase), s.AuditLogMiddleware, policyMiddleware)
	srv.Use("/system.admin.v1.*", middleware.Chain(s.AuditLogMiddleware, policyMiddleware))
	adminv1.RegisterAuthServiceHTTPServer(srv, s.Auth)
	adminv1.RegisterBaseApiServiceHTTPServer(srv, s.BaseAPI)
	adminv1.RegisterOauthClientServiceHTTPServer(srv, s.OauthClient)

	adminv1.RegisterBaseAreaServiceHTTPServer(srv, s.BaseArea)
	adminv1.RegisterBaseConfigServiceHTTPServer(srv, s.BaseConfig)
	adminv1.RegisterBaseDeptServiceHTTPServer(srv, s.BaseDept)
	adminv1.RegisterBaseDictServiceHTTPServer(srv, s.BaseDict)
	adminv1.RegisterBaseJobServiceHTTPServer(srv, s.BaseJob)
	adminv1.RegisterBaseLanguageServiceHTTPServer(srv, s.BaseLanguage)
	adminv1.RegisterBaseLoginLogServiceHTTPServer(srv, s.BaseLoginLog)
	adminv1.RegisterBaseApiLogServiceHTTPServer(srv, s.BaseApiLog)
	adminv1.RegisterBaseOperationLogServiceHTTPServer(srv, s.BaseOperationLog)
	adminv1.RegisterBaseDataAccessLogServiceHTTPServer(srv, s.BaseDataAccessLog)
	adminv1.RegisterBasePermissionLogServiceHTTPServer(srv, s.BasePermissionLog)
	adminv1.RegisterBasePolicyEvaluationLogServiceHTTPServer(srv, s.BasePolicyEvaluationLog)
	adminv1.RegisterBaseDashboardServiceHTTPServer(srv, s.BaseDashboard)
	adminv1.RegisterBaseFileServiceHTTPServer(srv, s.BaseFile)
	adminv1.RegisterBaseMenuServiceHTTPServer(srv, s.BaseMenu)
	adminv1.RegisterBaseMessageServiceHTTPServer(srv, s.BaseMessage)
	adminv1.RegisterBaseMessageCategoryServiceHTTPServer(srv, s.BaseMessageCategory)
	adminv1.RegisterBasePostServiceHTTPServer(srv, s.BasePost)
	adminv1.RegisterBaseRoleServiceHTTPServer(srv, s.BaseRole)
	adminv1.RegisterBaseTenantServiceHTTPServer(srv, s.BaseTenant)
	adminv1.RegisterBaseI18nServiceHTTPServer(srv, s.BaseI18n)
	adminv1.RegisterBaseUserServiceHTTPServer(srv, s.BaseUser)
	adminv1.RegisterCodeGenServiceHTTPServer(srv, s.CodeGen)
	adminv1.RegisterCodeGenColumnServiceHTTPServer(srv, s.CodeGenColumn)
	adminv1.RegisterCodeGenProtoServiceHTTPServer(srv, s.CodeGenProto)
	adminv1.RegisterCodeGenTableServiceHTTPServer(srv, s.CodeGenTable)
	adminv1.RegisterBaseMigrationServiceHTTPServer(srv, s.BaseMigration)
	adminv1.RegisterOpsMonitoringServiceHTTPServer(srv, s.OpsMonitoring)
	adminv1.RegisterRuntimeLogServiceHTTPServer(srv, s.RuntimeLog)
	adminv1.RegisterProjectDocumentServiceHTTPServer(srv, s.ProjectDocument)
	adminv1.RegisterBaseSessionServiceHTTPServer(srv, s.BaseSession)
	adminv1.RegisterBaseLoginPolicyServiceHTTPServer(srv, s.BaseLoginPolicy)
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
	adminv1.RegisterBaseJobServiceMCPTools(mcpSrv, s.BaseJob)
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
	adminv1.RegisterProjectDocumentServiceMCPTools(mcpSrv, s.ProjectDocument)
	adminv1.RegisterBaseSessionServiceMCPTools(mcpSrv, s.BaseSession)
	adminv1.RegisterBaseLoginPolicyServiceMCPTools(mcpSrv, s.BaseLoginPolicy)
	adminv1.RegisterBaseDashboardServiceMCPTools(mcpSrv, s.BaseDashboard)
	adminv1.RegisterBaseFileServiceMCPTools(mcpSrv, s.BaseFile)
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
		func() ([]tool.Invokable, error) { return adminv1.NewBaseJobServiceAgentTools(s.BaseJob) },
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
			return adminv1.NewOpsMonitoringServiceAgentTools(s.OpsMonitoring)
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
