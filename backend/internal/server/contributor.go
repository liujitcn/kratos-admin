package server

import (
	"github.com/liujitcn/kratos-admin/backend/core"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

// TaskContributor 表示可向调度运行时贡献具名任务的业务模块。
type TaskContributor = core.TaskContributor

// RegisterTasks 汇总模块任务并注册到调度运行时。
func RegisterTasks(registry *coreTask.Registry, contributors ...TaskContributor) error {
	tasks := make([]coreTask.Task, 0)
	for _, contributor := range contributors {
		tasks = append(tasks, contributor.Tasks()...)
	}
	return registry.Register(tasks...)
}

// OpenAPIDocuments 收集业务模块提供的具名 OpenAPI 文档。
func (modules Modules) OpenAPIDocuments() ([]coreOpenAPI.Document, error) {
	return core.Modules(modules).OpenAPIDocuments(), nil
}
