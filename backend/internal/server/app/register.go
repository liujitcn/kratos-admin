// Package app 注册 system.app.v1 传输层服务。
package app

import (
	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	einoTool "github.com/liujitcn/kratos-admin/backend/internal/agent/tool"
	host "github.com/liujitcn/kratos-admin/backend/internal/server"
	systemapp "github.com/liujitcn/kratos-admin/backend/internal/service/app"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 system.app.v1 的服务实现。
type Services struct {
	Auth     *systemapp.AuthService
	BaseArea *systemapp.BaseAreaService
	BaseDict *systemapp.BaseDictService
}

var _ host.Module = Services{}

// ProviderSet 汇总 system.app.v1 传输模块依赖注入提供者。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))

// RegisterGRPC 注册 system.app.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	systemappv1.RegisterAuthServiceServer(srv, s.Auth)
	systemappv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	systemappv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
}

// RegisterHTTP 注册 system.app.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	systemappv1.RegisterAuthServiceHTTPServer(srv, s.Auth)
	systemappv1.RegisterBaseAreaServiceHTTPServer(srv, s.BaseArea)
	systemappv1.RegisterBaseDictServiceHTTPServer(srv, s.BaseDict)
}

// RegisterMCP 注册 system.app.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	systemappv1.RegisterAuthServiceMCPTools(server.MCPServer(), s.Auth)
	systemappv1.RegisterBaseAreaServiceMCPTools(server.MCPServer(), s.BaseArea)
	systemappv1.RegisterBaseDictServiceMCPTools(server.MCPServer(), s.BaseDict)
}

// AppAgentTools 创建 system.app.v1 的应用端 AI 助手工具。
func (s Services) AppAgentTools() ([]einoTool.Invokable, error) {
	var tools []einoTool.Invokable
	tool, err := systemappv1.NewAuthServiceGetUserProfileAgentTool(s.Auth)
	if err != nil {
		return nil, err
	}
	tools = append(tools, tool)
	tool, err = systemappv1.NewAuthServiceUpdateUserProfileAgentTool(s.Auth)
	if err != nil {
		return nil, err
	}
	tools = append(tools, tool)
	var values []einoTool.Invokable
	values, err = systemappv1.NewBaseAreaServiceAgentTools(s.BaseArea)
	if err != nil {
		return nil, err
	}
	tools = append(tools, values...)
	values, err = systemappv1.NewBaseDictServiceAgentTools(s.BaseDict)
	if err != nil {
		return nil, err
	}
	return append(tools, values...), nil
}
