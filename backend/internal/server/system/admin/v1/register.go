package admin

import (
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	einoTool "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	systemadmin "github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
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
	ProjectDocument  *systemadmin.ProjectDocumentService
}

// RegisterGRPC 注册 system.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	systemadminv1.RegisterAuthServiceServer(srv, s.Auth)
	systemadminv1.RegisterBaseApiServiceServer(srv, s.BaseAPI)

	systemadminv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	systemadminv1.RegisterBaseConfigServiceServer(srv, s.BaseConfig)
	systemadminv1.RegisterBaseDeptServiceServer(srv, s.BaseDept)
	systemadminv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
	systemadminv1.RegisterBaseJobServiceServer(srv, s.BaseJob)
	systemadminv1.RegisterBaseLanguageServiceServer(srv, s.BaseLanguage)
	systemadminv1.RegisterBaseLogServiceServer(srv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceServer(srv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceServer(srv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceServer(srv, s.BaseRole)
	systemadminv1.RegisterBaseTenantServiceServer(srv, s.BaseTenant)
	systemadminv1.RegisterBaseThirdAccountServiceServer(srv, s.BaseThirdAccount)
	systemadminv1.RegisterBaseI18nServiceServer(srv, s.BaseI18n)
	systemadminv1.RegisterBaseUserServiceServer(srv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceServer(srv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceServer(srv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceServer(srv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceServer(srv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceServer(srv, s.BaseMigration)
	systemadminv1.RegisterOpsMonitoringServiceServer(srv, s.OpsMonitoring)
	systemadminv1.RegisterProjectDocumentServiceServer(srv, s.ProjectDocument)
}

// RegisterHTTP 注册 system.admin.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	systemadminv1.RegisterAuthServiceHTTPServer(srv, s.Auth)
	systemadminv1.RegisterBaseApiServiceHTTPServer(srv, s.BaseAPI)

	systemadminv1.RegisterBaseAreaServiceHTTPServer(srv, s.BaseArea)
	systemadminv1.RegisterBaseConfigServiceHTTPServer(srv, s.BaseConfig)
	systemadminv1.RegisterBaseDeptServiceHTTPServer(srv, s.BaseDept)
	systemadminv1.RegisterBaseDictServiceHTTPServer(srv, s.BaseDict)
	systemadminv1.RegisterBaseJobServiceHTTPServer(srv, s.BaseJob)
	systemadminv1.RegisterBaseLanguageServiceHTTPServer(srv, s.BaseLanguage)
	systemadminv1.RegisterBaseLogServiceHTTPServer(srv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceHTTPServer(srv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceHTTPServer(srv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceHTTPServer(srv, s.BaseRole)
	systemadminv1.RegisterBaseTenantServiceHTTPServer(srv, s.BaseTenant)
	systemadminv1.RegisterBaseI18nServiceHTTPServer(srv, s.BaseI18n)
	systemadminv1.RegisterBaseUserServiceHTTPServer(srv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceHTTPServer(srv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceHTTPServer(srv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceHTTPServer(srv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceHTTPServer(srv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceHTTPServer(srv, s.BaseMigration)
	systemadminv1.RegisterOpsMonitoringServiceHTTPServer(srv, s.OpsMonitoring)
	systemadminv1.RegisterProjectDocumentServiceHTTPServer(srv, s.ProjectDocument)
}

// RegisterMCP 注册 system.admin.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	mcpSrv := server.MCPServer()
	systemadminv1.RegisterAuthServiceMCPTools(mcpSrv, s.Auth)
	systemadminv1.RegisterBaseApiServiceMCPTools(mcpSrv, s.BaseAPI)

	systemadminv1.RegisterBaseAreaServiceMCPTools(mcpSrv, s.BaseArea)
	systemadminv1.RegisterBaseConfigServiceMCPTools(mcpSrv, s.BaseConfig)
	systemadminv1.RegisterBaseDeptServiceMCPTools(mcpSrv, s.BaseDept)
	systemadminv1.RegisterBaseDictServiceMCPTools(mcpSrv, s.BaseDict)
	systemadminv1.RegisterBaseJobServiceMCPTools(mcpSrv, s.BaseJob)
	systemadminv1.RegisterBaseLanguageServiceMCPTools(mcpSrv, s.BaseLanguage)
	systemadminv1.RegisterBaseLogServiceMCPTools(mcpSrv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceMCPTools(mcpSrv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceMCPTools(mcpSrv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceMCPTools(mcpSrv, s.BaseRole)
	systemadminv1.RegisterBaseI18nServiceMCPTools(mcpSrv, s.BaseI18n)
	// 角色切换仅开放 gRPC，MCP 显式注册其余用户管理方法。
	systemadminv1.RegisterBaseUserServiceOptionBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceListBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServicePageBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceGetBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceCreateBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceUpdateBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceDeleteBaseUserMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceSetBaseUserStatusMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterBaseUserServiceResetBaseUserPasswordMCPTool(mcpSrv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceMCPTools(mcpSrv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceMCPTools(mcpSrv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceMCPTools(mcpSrv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceMCPTools(mcpSrv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceMCPTools(mcpSrv, s.BaseMigration)
	systemadminv1.RegisterOpsMonitoringServiceMCPTools(mcpSrv, s.OpsMonitoring)
	systemadminv1.RegisterProjectDocumentServiceMCPTools(mcpSrv, s.ProjectDocument)
}

// AgentTools 创建 system.admin.v1 的管理端 AI 助手工具。
func (s Services) AgentTools() ([]einoTool.Invokable, error) {
	builders := []func() ([]einoTool.Invokable, error){
		func() ([]einoTool.Invokable, error) { return systemadminv1.NewAuthServiceAgentTools(s.Auth) },
		func() ([]einoTool.Invokable, error) { return systemadminv1.NewBaseApiServiceAgentTools(s.BaseAPI) },
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseAreaServiceAgentTools(s.BaseArea)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseConfigServiceAgentTools(s.BaseConfig)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseDeptServiceAgentTools(s.BaseDept)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseDictServiceAgentTools(s.BaseDict)
		},
		func() ([]einoTool.Invokable, error) { return systemadminv1.NewBaseJobServiceAgentTools(s.BaseJob) },
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseLanguageServiceAgentTools(s.BaseLanguage)
		},
		func() ([]einoTool.Invokable, error) { return systemadminv1.NewBaseLogServiceAgentTools(s.BaseLog) },
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseMenuServiceAgentTools(s.BaseMenu)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBasePostServiceAgentTools(s.BasePost)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseRoleServiceAgentTools(s.BaseRole)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseTenantServiceAgentTools(s.BaseTenant)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseThirdAccountServiceAgentTools(s.BaseThirdAccount)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseI18nServiceAgentTools(s.BaseI18n)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseUserServiceAgentTools(s.BaseUser)
		},
		func() ([]einoTool.Invokable, error) { return systemadminv1.NewCodeGenServiceAgentTools(s.CodeGen) },
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewCodeGenColumnServiceAgentTools(s.CodeGenColumn)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewCodeGenProtoServiceAgentTools(s.CodeGenProto)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewCodeGenTableServiceAgentTools(s.CodeGenTable)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewBaseMigrationServiceAgentTools(s.BaseMigration)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewOpsMonitoringServiceAgentTools(s.OpsMonitoring)
		},
		func() ([]einoTool.Invokable, error) {
			return systemadminv1.NewProjectDocumentServiceAgentTools(s.ProjectDocument)
		},
	}
	tools := make([]einoTool.Invokable, 0)
	var err error
	for _, build := range builders {
		var values []einoTool.Invokable
		values, err = build()
		if err != nil {
			return nil, err
		}
		tools = append(tools, values...)
	}
	return tools, nil
}
