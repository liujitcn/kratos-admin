package server

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"

	"github.com/liujitcn/kratos-admin/backend/pkg/job"
)

// Module 表示可挂载到服务端宿主的业务模块。
type Module interface {
	RegisterGRPC(*grpc.Server)
	RegisterHTTP(*kratosHTTP.Server)
	RegisterMCP(*mcpserver.Server)
}

// TaskContributor 表示可向调度运行时贡献具名任务的业务模块。
type TaskContributor interface {
	Tasks() []job.Task
}

// Modules 表示当前进程启用的业务模块集合。
type Modules []Module

// RegisterTasks 汇总模块任务并注册到调度运行时。
func RegisterTasks(registry *job.Registry, contributors ...TaskContributor) error {
	tasks := make([]job.Task, 0)
	for _, contributor := range contributors {
		tasks = append(tasks, contributor.Tasks()...)
	}
	return registry.Register(tasks...)
}

// RegisterGRPC 将全部业务模块注册到 GRPC 服务。
func (modules Modules) RegisterGRPC(srv *grpc.Server) {
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

// RegisterMCP 将全部业务模块注册到 MCP 服务。
func (modules Modules) RegisterMCP(srv *mcpserver.Server) {
	for _, module := range modules {
		module.RegisterMCP(srv)
	}
}
