package app

import (
	"github.com/go-kratos/kratos/v3/transport/http"
	appv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	einoTool "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	"github.com/liujitcn/kratos-admin/backend/internal/service/system/app/v1"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Services 汇总 system.app.v1 的服务实现。
type Services struct {
	Auth     *app.AuthService
	BaseArea *app.BaseAreaService
	BaseDict *app.BaseDictService
	BaseMenu *app.BaseMenuService
}

// RegisterGRPC 注册 system.app.v1 的 gRPC 服务。
func (s Services) RegisterGRPC(srv grpc.ServiceRegistrar) {
	appv1.RegisterAuthServiceServer(srv, appv1.RedactedAuthServiceServer(s.Auth))
	appv1.RegisterBaseAreaServiceServer(srv, appv1.RedactedBaseAreaServiceServer(s.BaseArea))
	appv1.RegisterBaseDictServiceServer(srv, appv1.RedactedBaseDictServiceServer(s.BaseDict))
	appv1.RegisterBaseMenuServiceServer(srv, appv1.RedactedBaseMenuServiceServer(s.BaseMenu))
}

// RegisterHTTP 注册 system.app.v1 的 HTTP 服务。
func (s Services) RegisterHTTP(srv *http.Server) {
	appv1.RegisterAuthServiceHTTPServer(srv, appv1.RedactedAuthServiceServer(s.Auth))
	appv1.RegisterBaseAreaServiceHTTPServer(srv, appv1.RedactedBaseAreaServiceServer(s.BaseArea))
	appv1.RegisterBaseDictServiceHTTPServer(srv, appv1.RedactedBaseDictServiceServer(s.BaseDict))
	appv1.RegisterBaseMenuServiceHTTPServer(srv, appv1.RedactedBaseMenuServiceServer(s.BaseMenu))
}

// RegisterMCP 注册 system.app.v1 的 MCP 工具。
func (s Services) RegisterMCP(server *mcp.Server) {
	appv1.RegisterAuthServiceMCPTools(server.MCPServer(), s.Auth)
	appv1.RegisterBaseAreaServiceMCPTools(server.MCPServer(), s.BaseArea)
	appv1.RegisterBaseDictServiceMCPTools(server.MCPServer(), s.BaseDict)
	appv1.RegisterBaseMenuServiceMCPTools(server.MCPServer(), s.BaseMenu)
}

// AppAgentTools 创建 system.app.v1 的应用端 AI 助手工具。
func (s Services) AppAgentTools() ([]einoTool.Invokable, error) {
	var tools []einoTool.Invokable
	tool, err := appv1.NewAuthServiceGetUserProfileAgentTool(s.Auth)
	if err != nil {
		return nil, err
	}
	tools = append(tools, tool)
	tool, err = appv1.NewAuthServiceUpdateUserProfileAgentTool(s.Auth)
	if err != nil {
		return nil, err
	}
	tools = append(tools, tool)
	var values []einoTool.Invokable
	values, err = appv1.NewBaseAreaServiceAgentTools(s.BaseArea)
	if err != nil {
		return nil, err
	}
	tools = append(tools, values...)
	values, err = appv1.NewBaseDictServiceAgentTools(s.BaseDict)
	if err != nil {
		return nil, err
	}
	tools = append(tools, values...)
	values, err = appv1.NewBaseMenuServiceAgentTools(s.BaseMenu)
	if err != nil {
		return nil, err
	}
	return append(tools, values...), nil
}
