package core

import (
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/task"
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

// HTTPMiddlewareContributor 表示可贡献 HTTP 服务端中间件的模块。
type HTTPMiddlewareContributor interface {
	HTTPMiddlewares() []kratosMiddleware.Middleware
}

// GRPCMiddlewareContributor 表示可贡献 gRPC 服务端中间件的模块。
type GRPCMiddlewareContributor interface {
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

// ScriptContributor 表示可贡献启动脚本或数据库迁移适配器的模块。
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
