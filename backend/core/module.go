package core

import (
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Module 表示可挂载到核心服务宿主的外部服务模块。
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*kratosHTTP.Server)
	RegisterMCP(*mcpserver.Server)
}

// ModuleAdapter 为不需要全部传输协议的外部模块提供空注册实现。
type ModuleAdapter struct{}

// RegisterGRPC 保持 gRPC 注册为空。
func (ModuleAdapter) RegisterGRPC(grpc.ServiceRegistrar) {}

// RegisterHTTP 保持 HTTP 注册为空。
func (ModuleAdapter) RegisterHTTP(*kratosHTTP.Server) {}

// RegisterMCP 保持 MCP 注册为空。
func (ModuleAdapter) RegisterMCP(*mcpserver.Server) {}

// Modules 表示当前进程启用的外部服务模块集合。
type Modules []Module

// RegisterGRPC 将全部外部模块注册到 gRPC 服务。
func (modules Modules) RegisterGRPC(server grpc.ServiceRegistrar) {
	for _, module := range modules {
		module.RegisterGRPC(server)
	}
}

// RegisterHTTP 将全部外部模块注册到 HTTP 服务。
func (modules Modules) RegisterHTTP(server *kratosHTTP.Server) {
	for _, module := range modules {
		module.RegisterHTTP(server)
	}
}

// RegisterMCP 将全部外部模块注册到 MCP 服务。
func (modules Modules) RegisterMCP(server *mcpserver.Server) {
	for _, module := range modules {
		module.RegisterMCP(server)
	}
}
