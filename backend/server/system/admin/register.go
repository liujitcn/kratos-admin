// Package admin 注册 system.admin.v1 传输层服务。
package admin

import (
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	host "github.com/liujitcn/kratos-admin/backend/server"
	systemadmin "github.com/liujitcn/kratos-admin/backend/service/system/admin"

	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// Services 汇总 system.admin.v1 的服务实现。
type Services struct {
	Auth    *systemadmin.AuthService
	BaseAPI *systemadmin.BaseApiService

	BaseArea      *systemadmin.BaseAreaService
	BaseConfig    *systemadmin.BaseConfigService
	BaseDept      *systemadmin.BaseDeptService
	BaseDict      *systemadmin.BaseDictService
	BaseJob       *systemadmin.BaseJobService
	BaseLog       *systemadmin.BaseLogService
	BaseMenu      *systemadmin.BaseMenuService
	BasePost      *systemadmin.BasePostService
	BaseRole      *systemadmin.BaseRoleService
	BaseTenant    *systemadmin.BaseTenantService
	BaseUser      *systemadmin.BaseUserService
	CodeGen       *systemadmin.CodeGenService
	CodeGenColumn *systemadmin.CodeGenColumnService
	CodeGenProto  *systemadmin.CodeGenProtoService
	CodeGenTable  *systemadmin.CodeGenTableService
	BaseMigration *systemadmin.BaseMigrationService
}

var _ host.Module = Services{}

// ProviderSet 汇总 system.admin.v1 传输模块依赖注入提供者。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))

// RegisterGRPC 注册 system.admin.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv *grpc.Server) {
	systemadminv1.RegisterAuthServiceServer(srv, s.Auth)
	systemadminv1.RegisterBaseApiServiceServer(srv, s.BaseAPI)

	systemadminv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	systemadminv1.RegisterBaseConfigServiceServer(srv, s.BaseConfig)
	systemadminv1.RegisterBaseDeptServiceServer(srv, s.BaseDept)
	systemadminv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
	systemadminv1.RegisterBaseJobServiceServer(srv, s.BaseJob)
	systemadminv1.RegisterBaseLogServiceServer(srv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceServer(srv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceServer(srv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceServer(srv, s.BaseRole)
	systemadminv1.RegisterBaseTenantServiceServer(srv, s.BaseTenant)
	systemadminv1.RegisterBaseUserServiceServer(srv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceServer(srv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceServer(srv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceServer(srv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceServer(srv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceServer(srv, s.BaseMigration)
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
	systemadminv1.RegisterBaseLogServiceHTTPServer(srv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceHTTPServer(srv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceHTTPServer(srv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceHTTPServer(srv, s.BaseRole)
	systemadminv1.RegisterBaseTenantServiceHTTPServer(srv, s.BaseTenant)
	systemadminv1.RegisterBaseUserServiceHTTPServer(srv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceHTTPServer(srv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceHTTPServer(srv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceHTTPServer(srv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceHTTPServer(srv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceHTTPServer(srv, s.BaseMigration)
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
	systemadminv1.RegisterBaseLogServiceMCPTools(mcpSrv, s.BaseLog)
	systemadminv1.RegisterBaseMenuServiceMCPTools(mcpSrv, s.BaseMenu)
	systemadminv1.RegisterBasePostServiceMCPTools(mcpSrv, s.BasePost)
	systemadminv1.RegisterBaseRoleServiceMCPTools(mcpSrv, s.BaseRole)
	systemadminv1.RegisterBaseUserServiceMCPTools(mcpSrv, s.BaseUser)
	systemadminv1.RegisterCodeGenServiceMCPTools(mcpSrv, s.CodeGen)
	systemadminv1.RegisterCodeGenColumnServiceMCPTools(mcpSrv, s.CodeGenColumn)
	systemadminv1.RegisterCodeGenProtoServiceMCPTools(mcpSrv, s.CodeGenProto)
	systemadminv1.RegisterCodeGenTableServiceMCPTools(mcpSrv, s.CodeGenTable)
	systemadminv1.RegisterBaseMigrationServiceMCPTools(mcpSrv, s.BaseMigration)
}
