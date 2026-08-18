package module

import (
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	baseServer "github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	adminServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
	"github.com/liujitcn/kratos-core/module"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
	"github.com/liujitcn/kratos-kit/transport/sse"
	sseTransport "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/grpc"
)

// Module 聚合 Admin 注册到 Core 宿主的服务和业务资源。
type Module struct {
	baseServices  *baseServer.Services
	adminServices *adminServer.Services
	appServices   *appServer.Services
	task          []*cronTransport.Task
	sseStreams    []sseTransport.SSEStream
	resources     module.Resources
}

var _ module.Module = (*Module)(nil)

// NewModules 创建包含 Admin 服务和资源的模块集合。
func NewModules(
	baseServices *baseServer.Services,
	adminServices *adminServer.Services,
	appServices *appServer.Services,
	task []*cronTransport.Task,
	sseStreams []sseTransport.SSEStream,
	resources module.Resources,
) []module.Module {
	return []module.Module{
		&Module{
			baseServices:  baseServices,
			adminServices: adminServices,
			appServices:   appServices,
			task:          task,
			sseStreams:    sseStreams,
			resources:     resources,
		},
	}
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

// RegisterQueue 注册 Admin 的队列消费者；当前 Admin 没有独立队列消费者。
func (*Module) RegisterQueue(*queueTransport.Server) {}

// RegisterCron 注册 Admin 的数据库任务执行器。
func (m *Module) RegisterCron(server *cronTransport.Server) error {
	return server.RegisterTask(m.task...)
}

// RegisterSSE 注册 Admin 的业务 SSE 流定义。
func (m *Module) RegisterSSE(server *sse.Server) error {
	return server.RegisterStream(m.sseStreams...)
}

// Resources 返回 Admin 提供给 Core 的静态资源。
func (m *Module) Resources() module.Resources {
	if m == nil {
		return module.Resources{}
	}
	return m.resources
}
