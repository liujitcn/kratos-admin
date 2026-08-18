package app

import (
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	einoTool "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	systemapp "github.com/liujitcn/kratos-admin/backend/internal/service/system/app/v1"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 system.app.v1 的服务实现。
type Services struct {
	Auth     *systemapp.AuthService
	BaseArea *systemapp.BaseAreaService
	BaseDict *systemapp.BaseDictService
	BaseMenu *systemapp.BaseMenuService
}

// RegisterGRPC 注册 system.app.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	systemappv1.RegisterAuthServiceServer(srv, s.Auth)
	systemappv1.RegisterBaseAreaServiceServer(srv, s.BaseArea)
	systemappv1.RegisterBaseDictServiceServer(srv, s.BaseDict)
	systemappv1.RegisterBaseMenuServiceServer(srv, s.BaseMenu)
}

// RegisterHTTP 注册 system.app.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *kratosHTTP.Server) {
	systemappv1.RegisterAuthServiceHTTPServer(srv, s.Auth)
	systemappv1.RegisterBaseAreaServiceHTTPServer(srv, s.BaseArea)
	systemappv1.RegisterBaseDictServiceHTTPServer(srv, s.BaseDict)
	systemappv1.RegisterBaseMenuServiceHTTPServer(srv, s.BaseMenu)
}

// RegisterMCP 注册 system.app.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcpserver.Server) {
	systemappv1.RegisterAuthServiceMCPTools(server.MCPServer(), s.Auth)
	systemappv1.RegisterBaseAreaServiceMCPTools(server.MCPServer(), s.BaseArea)
	systemappv1.RegisterBaseDictServiceMCPTools(server.MCPServer(), s.BaseDict)
	systemappv1.RegisterBaseMenuServiceMCPTools(server.MCPServer(), s.BaseMenu)
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
	tools = append(tools, values...)
	values, err = systemappv1.NewBaseMenuServiceAgentTools(s.BaseMenu)
	if err != nil {
		return nil, err
	}
	return append(tools, values...), nil
}
