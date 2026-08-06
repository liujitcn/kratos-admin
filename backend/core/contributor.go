package core

import (
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

// HTTPMiddlewareContributor 表示可贡献 HTTP 服务端中间件的模块。
type HTTPMiddlewareContributor interface {
	// HTTPMiddlewares 返回模块贡献的 HTTP 服务端中间件。
	HTTPMiddlewares() []kratosMiddleware.Middleware
}

// GRPCMiddlewareContributor 表示可贡献 gRPC 服务端中间件的模块。
type GRPCMiddlewareContributor interface {
	// GRPCMiddlewares 返回模块贡献的 gRPC 服务端中间件。
	GRPCMiddlewares() []kratosMiddleware.Middleware
}

// ServerContributor 表示可贡献定时任务、消费者等后台服务的模块。
type ServerContributor interface {
	// Servers 返回需要纳入 Kratos 生命周期的后台服务。
	Servers() []kratosTransport.Server
}

// OpenAPIContributor 表示可贡献具名 OpenAPI 文档的模块。
type OpenAPIContributor interface {
	// OpenAPIDocuments 返回模块拥有的独立 OpenAPI 文档。
	OpenAPIDocuments() []openapi.Document
}

// ProjectDocumentContributor 表示可贡献项目文档的模块。
type ProjectDocumentContributor interface {
	// ProjectDocuments 返回模块提供的项目文档。
	ProjectDocuments() []docs.Document
}

// TaskContributor 表示可贡献具名任务的模块。
type TaskContributor interface {
	// Tasks 返回模块提供的任务定义。
	Tasks() []task.Task
}

// QueueConsumerContributor 表示可贡献队列消费者的模块。
type QueueConsumerContributor interface {
	// QueueConsumers 返回模块提供的队列消费者。
	QueueConsumers() []coreQueue.Consumer
}

// SSEContributor 表示可贡献 SSE 流的模块。
type SSEContributor interface {
	// SSEStreams 返回模块提供的 SSE 流定义。
	SSEStreams() []coreSSE.Stream
}

// SSEPublisherAware 表示需要使用宿主 SSE 发布器的模块。
type SSEPublisherAware interface {
	// SetSSEPublisher 注入与宿主订阅端点共享的 SSE 发布器。
	SetSSEPublisher(*coreSSE.Publisher)
}

// SSERegistryAware 表示需要解析宿主 SSE 流定义的模块。
type SSERegistryAware interface {
	// SetSSERegistry 注入与宿主共享的 SSE 流注册表。
	SetSSERegistry(*coreSSE.Registry)
}

// ScriptContributor 表示可贡献启动脚本的模块。
type ScriptContributor interface {
	// Scripts 返回模块提供的启动脚本。
	Scripts() []script.Script
}

// StartupContributor 表示可贡献服务启动和清理钩子的模块。
type StartupContributor interface {
	// StartupHooks 返回模块提供的服务启动和清理钩子。
	StartupHooks() []startup.Hook
}

// HealthContributor 表示可贡献外部依赖就绪检查的模块。
type HealthContributor interface {
	// HealthChecks 返回模块提供的就绪检查。
	HealthChecks() []health.Check
}

// StaticContributor 表示可贡献静态资源或单页应用挂载的模块。
type StaticContributor interface {
	// StaticMounts 返回模块提供的静态资源挂载。
	StaticMounts() []coreStatic.Mount
}

// HTTPMiddlewares 汇总全部外部模块贡献的 HTTP 中间件。
func (modules Modules) HTTPMiddlewares() []kratosMiddleware.Middleware {
	middlewares := make([]kratosMiddleware.Middleware, 0)
	for _, module := range modules {
		contributor, ok := module.(HTTPMiddlewareContributor)
		if !ok {
			continue
		}
		middlewares = append(middlewares, contributor.HTTPMiddlewares()...)
	}
	return middlewares
}

// GRPCMiddlewares 汇总全部外部模块贡献的 gRPC 中间件。
func (modules Modules) GRPCMiddlewares() []kratosMiddleware.Middleware {
	middlewares := make([]kratosMiddleware.Middleware, 0)
	for _, module := range modules {
		contributor, ok := module.(GRPCMiddlewareContributor)
		if !ok {
			continue
		}
		middlewares = append(middlewares, contributor.GRPCMiddlewares()...)
	}
	return middlewares
}

// Servers 汇总全部外部模块贡献的后台服务。
func (modules Modules) Servers() []kratosTransport.Server {
	servers := make([]kratosTransport.Server, 0)
	for _, module := range modules {
		contributor, ok := module.(ServerContributor)
		if !ok {
			continue
		}
		servers = append(servers, contributor.Servers()...)
	}
	return servers
}

// OpenAPIDocuments 汇总全部外部模块贡献的 OpenAPI 文档。
func (modules Modules) OpenAPIDocuments() []openapi.Document {
	documents := make([]openapi.Document, 0)
	for _, module := range modules {
		contributor, ok := module.(OpenAPIContributor)
		if !ok {
			continue
		}
		documents = append(documents, contributor.OpenAPIDocuments()...)
	}
	return documents
}

// ProjectDocuments 汇总全部外部模块贡献的项目文档。
func (modules Modules) ProjectDocuments() []docs.Document {
	documents := make([]docs.Document, 0)
	for _, module := range modules {
		contributor, ok := module.(ProjectDocumentContributor)
		if !ok {
			continue
		}
		documents = append(documents, contributor.ProjectDocuments()...)
	}
	return documents
}

// Tasks 汇总全部外部模块贡献的任务定义。
func (modules Modules) Tasks() []task.Task {
	tasks := make([]task.Task, 0)
	for _, module := range modules {
		contributor, ok := module.(TaskContributor)
		if !ok {
			continue
		}
		tasks = append(tasks, contributor.Tasks()...)
	}
	return tasks
}

// QueueConsumers 汇总全部外部模块贡献的队列消费者。
func (modules Modules) QueueConsumers() []coreQueue.Consumer {
	consumers := make([]coreQueue.Consumer, 0)
	for _, module := range modules {
		contributor, ok := module.(QueueConsumerContributor)
		if !ok {
			continue
		}
		consumers = append(consumers, contributor.QueueConsumers()...)
	}
	return consumers
}

// SSEStreams 汇总全部外部模块贡献的 SSE 流。
func (modules Modules) SSEStreams() []coreSSE.Stream {
	streams := make([]coreSSE.Stream, 0)
	for _, module := range modules {
		contributor, ok := module.(SSEContributor)
		if !ok {
			continue
		}
		streams = append(streams, contributor.SSEStreams()...)
	}
	return streams
}

// SetSSEPublisher 向需要发布 SSE 消息的模块注入宿主发布器。
func (modules Modules) SetSSEPublisher(publisher *coreSSE.Publisher) {
	for _, module := range modules {
		aware, ok := module.(SSEPublisherAware)
		if !ok {
			continue
		}
		aware.SetSSEPublisher(publisher)
	}
}

// SetSSERegistry 向需要解析流定义的模块透传宿主注册表。
func (modules Modules) SetSSERegistry(registry *coreSSE.Registry) {
	for _, module := range modules {
		aware, ok := module.(SSERegistryAware)
		if !ok {
			continue
		}
		aware.SetSSERegistry(registry)
	}
}

// Scripts 汇总全部外部模块贡献的启动脚本。
func (modules Modules) Scripts() []script.Script {
	scripts := make([]script.Script, 0)
	for _, module := range modules {
		contributor, ok := module.(ScriptContributor)
		if !ok {
			continue
		}
		scripts = append(scripts, contributor.Scripts()...)
	}
	return scripts
}

// StartupHooks 汇总全部外部模块贡献的服务启动和清理钩子。
func (modules Modules) StartupHooks() []startup.Hook {
	hooks := make([]startup.Hook, 0)
	for _, module := range modules {
		contributor, ok := module.(StartupContributor)
		if !ok {
			continue
		}
		hooks = append(hooks, contributor.StartupHooks()...)
	}
	return hooks
}

// HealthChecks 汇总全部外部模块贡献的就绪检查。
func (modules Modules) HealthChecks() []health.Check {
	checks := make([]health.Check, 0)
	for _, module := range modules {
		contributor, ok := module.(HealthContributor)
		if !ok {
			continue
		}
		checks = append(checks, contributor.HealthChecks()...)
	}
	return checks
}

// StaticMounts 汇总全部外部模块贡献的静态资源挂载。
func (modules Modules) StaticMounts() []coreStatic.Mount {
	mounts := make([]coreStatic.Mount, 0)
	for _, module := range modules {
		contributor, ok := module.(StaticContributor)
		if !ok {
			continue
		}
		mounts = append(mounts, contributor.StaticMounts()...)
	}
	return mounts
}
