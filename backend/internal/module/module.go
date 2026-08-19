package module

import (
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	baseServer "github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	adminServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/queue"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Module 聚合 Admin 注册到 Core 宿主的协议服务。
type Module struct {
	baseServices  *baseServer.Services
	adminServices *adminServer.Services
	appServices   *appServer.Services
}

var _ module.Module = (*Module)(nil)

// NewModules 创建包含 Admin 服务的模块集合。
func NewModules(
	baseServices *baseServer.Services,
	adminServices *adminServer.Services,
	appServices *appServer.Services,
) module.Modules {
	return module.Modules{
		&Module{
			baseServices:  baseServices,
			adminServices: adminServices,
			appServices:   appServices,
		},
	}
}

// NewQueueConsumers 提供 Admin 当前没有的队列消费者集合。
func NewQueueConsumers() queue.Consumers {
	return queue.Consumers{}
}

// RegisterGRPC 注册 Admin 的全部 gRPC 服务。
func (m *Module) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	m.baseServices.RegisterGRPC(registrar)
	m.adminServices.RegisterGRPC(registrar)
	m.appServices.RegisterGRPC(registrar)
}

// RegisterHTTP 注册 Admin 的全部 HTTP 服务。
func (m *Module) RegisterHTTP(server *kratosHTTP.Server) {
	m.baseServices.RegisterHTTP(server)
	m.adminServices.RegisterHTTP(server)
	m.appServices.RegisterHTTP(server)
}

// RegisterMCP 注册 Admin 的全部 MCP 工具。
func (m *Module) RegisterMCP(server *mcpserver.Server) {
	m.baseServices.RegisterMCP(server)
	m.adminServices.RegisterMCP(server)
	m.appServices.RegisterMCP(server)
}
