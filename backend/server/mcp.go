package server

import (
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// MCPToolsReady 表示 MCP 工具已完成注册。
type MCPToolsReady struct{}

// NewMCPHandler 创建未注册业务工具的进程内 MCP 服务。
func NewMCPHandler(ctx *bootstrap.Context) (*mcpserver.Server, error) {
	return rpc.CreateMcpHandler(ctx.GetConfig())
}

// NewMCPToolsReady 将已启用业务模块注册到 MCP 服务。
func NewMCPToolsReady(mcpSrv *mcpserver.Server, modules Modules) MCPToolsReady {
	modules.RegisterMCP(mcpSrv)
	return MCPToolsReady{}
}
