package admin

import (
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	systemadmin "github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 system.admin.v1 的服务实现。
type Services struct {
	Auth    *systemadmin.AuthService
	BaseAPI *systemadmin.BaseApiService

	BaseArea         *systemadmin.BaseAreaService
	BaseConfig       *systemadmin.BaseConfigService
	BaseDept         *systemadmin.BaseDeptService
	BaseDict         *systemadmin.BaseDictService
	BaseJob          *systemadmin.BaseJobService
	BaseLanguage     *systemadmin.BaseLanguageService
	BaseLog          *systemadmin.BaseLogService
	BaseMenu         *systemadmin.BaseMenuService
	BasePost         *systemadmin.BasePostService
	BaseRole         *systemadmin.BaseRoleService
	BaseTenant       *systemadmin.BaseTenantService
	BaseThirdAccount *systemadmin.BaseThirdAccountService
	BaseI18n         *systemadmin.BaseI18nService
	BaseUser         *systemadmin.BaseUserService
	CodeGen          *systemadmin.CodeGenService
	CodeGenColumn    *systemadmin.CodeGenColumnService
	CodeGenProto     *systemadmin.CodeGenProtoService
	CodeGenTable     *systemadmin.CodeGenTableService
	BaseMigration    *systemadmin.BaseMigrationService
	OpsMonitoring    *systemadmin.OpsMonitoringService
	RuntimeLog       *systemadmin.RuntimeLogService
	ProjectDocument  *systemadmin.ProjectDocumentService
}

// RegisterGRPC 注册 system.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	adminv1.RegisterAuthServiceServer(srv, s.Auth)
	adminv1.RegisterBaseApiServiceServer(srv, s.BaseAPI)

	adminv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	adminv1.RegisterBaseConfigServiceServer(srv, s.BaseConfig)
	adminv1.RegisterBaseDeptServiceServer(srv, s.BaseDept)
	adminv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
	adminv1.RegisterBaseJobServiceServer(srv, s.BaseJob)
	adminv1.RegisterBaseLanguageServiceServer(srv, s.BaseLanguage)
	adminv1.RegisterBaseLogServiceServer(srv, s.BaseLog)
	adminv1.RegisterBaseMenuServiceServer(srv, s.BaseMenu)
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
}

// RegisterHTTP 注册 system.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	adminv1.RegisterAuthServiceHTTPServer(srv, s.Auth)
	adminv1.RegisterBaseApiServiceHTTPServer(srv, s.BaseAPI)

	adminv1.RegisterBaseAreaServiceHTTPServer(srv, s.BaseArea)
	adminv1.RegisterBaseConfigServiceHTTPServer(srv, s.BaseConfig)
	adminv1.RegisterBaseDeptServiceHTTPServer(srv, s.BaseDept)
	adminv1.RegisterBaseDictServiceHTTPServer(srv, s.BaseDict)
	adminv1.RegisterBaseJobServiceHTTPServer(srv, s.BaseJob)
	adminv1.RegisterBaseLanguageServiceHTTPServer(srv, s.BaseLanguage)
	adminv1.RegisterBaseLogServiceHTTPServer(srv, s.BaseLog)
	adminv1.RegisterBaseMenuServiceHTTPServer(srv, s.BaseMenu)
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
	adminv1.RegisterBaseLogServiceMCPTools(mcpSrv, s.BaseLog)
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
		func() ([]tool.Invokable, error) { return adminv1.NewBaseLogServiceAgentTools(s.BaseLog) },
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
