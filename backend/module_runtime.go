package kratosadmin

import (
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/projectdoc"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// ClientConn 返回与生成 gRPC 客户端兼容的进程内连接。
func (r *Runtime) ClientConn() grpc.ClientConnInterface {
	return r.clientConn
}

// RegisterGRPC 将 Backend gRPC 服务注册到启动器提供的服务注册表。
func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	r.modules.RegisterGRPC(registrar)
}

// RegisterHTTP 将 Backend HTTP 服务注册到启动器提供的 HTTP Server。
func (r *Runtime) RegisterHTTP(httpServer *kratosHTTP.Server) {
	r.modules.RegisterHTTP(httpServer)
}

// RegisterMCP 将 Backend MCP 工具注册到启动器提供的 MCP Server。
func (r *Runtime) RegisterMCP(server *mcpserver.Server) {
	r.modules.RegisterMCP(server)
}

// HTTPMiddlewares 返回 Backend 认证、鉴权、校验和访问日志中间件。
func (r *Runtime) HTTPMiddlewares() []kratosMiddleware.Middleware {
	middlewares := append([]kratosMiddleware.Middleware(nil), r.httpMiddlewares...)
	return append(middlewares, core.Modules(r.modules).HTTPMiddlewares()...)
}

// GRPCMiddlewares 返回 Backend 认证、鉴权、校验和访问日志中间件。
func (r *Runtime) GRPCMiddlewares() []kratosMiddleware.Middleware {
	middlewares := append([]kratosMiddleware.Middleware(nil), r.grpcMiddlewares...)
	return append(middlewares, core.Modules(r.modules).GRPCMiddlewares()...)
}

// Servers 返回 Backend 定时任务和扩展模块后台服务。
func (r *Runtime) Servers() []kratosTransport.Server {
	servers := []kratosTransport.Server{r.cronServer}
	return append(servers, core.Modules(r.modules).Servers()...)
}

// OpenAPIDocuments 返回 Backend 及扩展模块的具名 OpenAPI 文档。
func (r *Runtime) OpenAPIDocuments() []coreOpenAPI.Document {
	return r.openAPIRegistry.Documents()
}

// ProjectDocuments 返回 Backend、宿主配置及扩展模块贡献的项目文档。
func (r *Runtime) ProjectDocuments() []projectdoc.Document {
	return r.projectDocumentCase.ProjectDocuments()
}

// Tasks 返回 Backend 内置和扩展模块贡献的任务执行器。
func (r *Runtime) Tasks() []coreTask.Task {
	tasks := core.Modules(r.modules).Tasks()
	return append(tasks, r.translationTask.Task())
}

// QueueConsumers 返回扩展模块贡献的队列消费者。
func (r *Runtime) QueueConsumers() []coreQueue.Consumer {
	return core.Modules(r.modules).QueueConsumers()
}

// SSEStreams 返回扩展模块贡献的 SSE 流。
func (r *Runtime) SSEStreams() []coreSSE.Stream {
	return core.Modules(r.modules).SSEStreams()
}

// SetSSEPublisher 向需要发布 SSE 消息的扩展模块透传宿主发布器。
func (r *Runtime) SetSSEPublisher(publisher *coreSSE.Publisher) {
	core.Modules(r.modules).SetSSEPublisher(publisher)
}

// SetSSERegistry 向需要解析 SSE 流定义的扩展模块透传宿主注册表。
func (r *Runtime) SetSSERegistry(registry *coreSSE.Registry) {
	core.Modules(r.modules).SetSSERegistry(registry)
}

// Scripts 返回扩展模块贡献的启动脚本。
func (r *Runtime) Scripts() []script.Script {
	return core.Modules(r.modules).Scripts()
}

// StartupHooks 返回扩展模块贡献的启动和清理钩子。
func (r *Runtime) StartupHooks() []startup.Hook {
	return core.Modules(r.modules).StartupHooks()
}

// HealthChecks 返回扩展模块贡献的健康检查。
func (r *Runtime) HealthChecks() []health.Check {
	return core.Modules(r.modules).HealthChecks()
}

// StaticMounts 返回扩展模块贡献的静态资源挂载。
func (r *Runtime) StaticMounts() []coreStatic.Mount {
	return core.Modules(r.modules).StaticMounts()
}

var (
	_ core.HTTPMiddlewareContributor  = (*Runtime)(nil)
	_ core.GRPCMiddlewareContributor  = (*Runtime)(nil)
	_ core.ServerContributor          = (*Runtime)(nil)
	_ core.OpenAPIContributor         = (*Runtime)(nil)
	_ core.ProjectDocumentContributor = (*Runtime)(nil)
	_ core.TaskContributor            = (*Runtime)(nil)
	_ core.QueueConsumerContributor   = (*Runtime)(nil)
	_ core.SSEContributor             = (*Runtime)(nil)
	_ core.SSEPublisherAware          = (*Runtime)(nil)
	_ core.SSERegistryAware           = (*Runtime)(nil)
	_ core.ScriptContributor          = (*Runtime)(nil)
	_ core.StartupContributor         = (*Runtime)(nil)
	_ core.HealthContributor          = (*Runtime)(nil)
	_ core.StaticContributor          = (*Runtime)(nil)
)
