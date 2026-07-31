package server

import (
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"

	"github.com/liujitcn/kratos-admin/backend/core"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

// Module 表示可挂载到服务端宿主的业务模块。
type Module = core.Module

// TaskContributor 表示可向调度运行时贡献具名任务的业务模块。
type TaskContributor = core.TaskContributor

// Modules 表示当前进程启用的业务模块集合。
type Modules []Module

// RegisterTasks 汇总模块任务并注册到调度运行时。
func RegisterTasks(registry *coreTask.Registry, contributors ...TaskContributor) error {
	tasks := make([]coreTask.Task, 0)
	for _, contributor := range contributors {
		tasks = append(tasks, contributor.Tasks()...)
	}
	return registry.Register(tasks...)
}

// RegisterGRPC 将全部业务模块注册到 GRPC 服务。
func (modules Modules) RegisterGRPC(srv grpc.ServiceRegistrar) {
	for _, module := range modules {
		module.RegisterGRPC(srv)
	}
}

// RegisterHTTP 将全部业务模块注册到 HTTP 服务。
func (modules Modules) RegisterHTTP(srv *kratosHTTP.Server) {
	for _, module := range modules {
		module.RegisterHTTP(srv)
	}
}

// OpenAPIDocuments 收集业务模块提供的具名 OpenAPI 文档。
func (modules Modules) OpenAPIDocuments() ([]coreOpenAPI.Document, error) {
	return core.Modules(modules).OpenAPIDocuments(), nil
}

// RegisterMCP 将全部业务模块注册到 MCP 服务。
func (modules Modules) RegisterMCP(srv *mcpserver.Server) {
	for _, module := range modules {
		module.RegisterMCP(srv)
	}
}
