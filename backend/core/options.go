package core

import (
	stdhttp "net/http"

	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

type options struct {
	modules          Modules
	httpMiddlewares  []kratosMiddleware.Middleware
	grpcMiddlewares  []kratosMiddleware.Middleware
	servers          []kratosTransport.Server
	openAPIRegistry  *openapi.Registry
	openAPIOptions   openapi.HTTPOptions
	openAPIDocuments []openapi.Document
	taskRegistry     *task.Registry
	tasks            []task.Task
	taskObserver     task.Observer
	queue            kitQueue.Queue
	queueConsumers   []coreQueue.Consumer
	sseServer        *sseServer.Server
	sseRegistry      *coreSSE.Registry
	sseStreams       []coreSSE.Stream
	scripts          []script.Script
	startupHooks     []startup.Hook
	healthChecks     []health.Check
	staticMounts     []coreStatic.Mount
}

// Option 表示核心服务宿主的装配选项。
type Option func(*options)

// WithModules 挂载外部服务模块。
func WithModules(modules ...Module) Option {
	return func(opts *options) {
		opts.modules = append(opts.modules, modules...)
	}
}

// WithHTTPMiddlewares 追加 HTTP 服务端中间件。
func WithHTTPMiddlewares(middlewares ...kratosMiddleware.Middleware) Option {
	return func(opts *options) {
		opts.httpMiddlewares = append(opts.httpMiddlewares, middlewares...)
	}
}

// WithGRPCMiddlewares 追加 gRPC 服务端中间件。
func WithGRPCMiddlewares(middlewares ...kratosMiddleware.Middleware) Option {
	return func(opts *options) {
		opts.grpcMiddlewares = append(opts.grpcMiddlewares, middlewares...)
	}
}

// WithServers 追加定时任务、消费者等 Kratos 后台服务。
func WithServers(servers ...kratosTransport.Server) Option {
	return func(opts *options) {
		opts.servers = append(opts.servers, servers...)
	}
}

// WithOpenAPIRegistry 使用调用方创建的 OpenAPI 注册表。
func WithOpenAPIRegistry(registry *openapi.Registry) Option {
	return func(opts *options) {
		opts.openAPIRegistry = registry
	}
}

// WithOpenAPIDocuments 追加由组合根直接提供的 OpenAPI 文档。
func WithOpenAPIDocuments(documents ...openapi.Document) Option {
	return func(opts *options) {
		opts.openAPIDocuments = append(opts.openAPIDocuments, documents...)
	}
}

// WithOpenAPIPaths 配置原始文档和 Swagger UI 的路由前缀。
func WithOpenAPIPaths(documentPath, swaggerPath string) Option {
	return func(opts *options) {
		opts.openAPIOptions.DocumentPath = documentPath
		opts.openAPIOptions.SwaggerPath = swaggerPath
	}
}

// WithOpenAPIAuthorizer 设置 OpenAPI 文档访问校验函数。
func WithOpenAPIAuthorizer(authorizer func(*stdhttp.Request) bool) Option {
	return func(opts *options) {
		opts.openAPIOptions.Authorizer = authorizer
	}
}

// WithTaskRegistry 使用调用方创建的任务注册表，便于动态任务适配器共享。
func WithTaskRegistry(registry *task.Registry) Option {
	return func(opts *options) {
		opts.taskRegistry = registry
	}
}

// WithTasks 追加由组合根直接提供的任务定义。
func WithTasks(tasks ...task.Task) Option {
	return func(opts *options) {
		opts.tasks = append(opts.tasks, tasks...)
	}
}

// WithTaskObserver 设置任务执行结果观察器。
func WithTaskObserver(observer task.Observer) Option {
	return func(opts *options) {
		opts.taskObserver = observer
	}
}

// WithQueue 注入内存、Redis 或其他队列适配器。
func WithQueue(queue kitQueue.Queue) Option {
	return func(opts *options) {
		opts.queue = queue
	}
}

// WithQueueConsumers 追加由组合根直接提供的队列消费者。
func WithQueueConsumers(consumers ...coreQueue.Consumer) Option {
	return func(opts *options) {
		opts.queueConsumers = append(opts.queueConsumers, consumers...)
	}
}

// WithSSEServer 注入需要挂载或独立启动的 SSE 服务。
func WithSSEServer(server *sseServer.Server) Option {
	return func(opts *options) {
		opts.sseServer = server
	}
}

// WithSSERegistry 使用调用方创建的 SSE 流注册表。
func WithSSERegistry(registry *coreSSE.Registry) Option {
	return func(opts *options) {
		opts.sseRegistry = registry
	}
}

// WithSSEStreams 追加由组合根直接提供的 SSE 流定义。
func WithSSEStreams(streams ...coreSSE.Stream) Option {
	return func(opts *options) {
		opts.sseStreams = append(opts.sseStreams, streams...)
	}
}

// WithScripts 追加由组合根直接提供的启动脚本。
func WithScripts(scripts ...script.Script) Option {
	return func(opts *options) {
		opts.scripts = append(opts.scripts, scripts...)
	}
}

// WithStartupHooks 追加由组合根直接提供的服务启动和清理钩子。
func WithStartupHooks(hooks ...startup.Hook) Option {
	return func(opts *options) {
		opts.startupHooks = append(opts.startupHooks, hooks...)
	}
}

// WithHealthChecks 追加由组合根直接提供的就绪检查。
func WithHealthChecks(checks ...health.Check) Option {
	return func(opts *options) {
		opts.healthChecks = append(opts.healthChecks, checks...)
	}
}

// WithStaticMounts 追加由组合根直接提供的静态资源挂载。
func WithStaticMounts(mounts ...coreStatic.Mount) Option {
	return func(opts *options) {
		opts.staticMounts = append(opts.staticMounts, mounts...)
	}
}
