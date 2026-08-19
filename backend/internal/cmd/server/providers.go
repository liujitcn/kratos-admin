package main

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend"
	"github.com/liujitcn/kratos-core/job"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-core/sse"
)

// hostProviderSet 将 Admin 的具名贡献收口为 Core 最终集合。
var hostProviderSet = wire.NewSet(
	provideResources,
	provideModules,
	provideTasks,
	provideStreams,
	provideConsumers,
)

// provideResources 将 Admin 静态资源转换为 Core 静态资源集合。
func provideResources(adminResources backend.AdminResources) module.Resources {
	return module.Resources(adminResources)
}

// provideModules 将 Admin 协议模块转换为 Core 协议模块集合。
func provideModules(adminModules backend.AdminModules) module.Modules {
	return module.Modules(adminModules)
}

// provideTasks 将 Admin 定时任务转换为 Core 定时任务集合。
func provideTasks(adminTasks backend.AdminTasks) job.Tasks {
	return job.Tasks(adminTasks)
}

// provideStreams 将 Admin SSE 业务流转换为 Core SSE 业务流集合。
func provideStreams(adminStreams backend.AdminStreams) sse.Streams {
	return sse.Streams(adminStreams)
}

// provideConsumers 将 Admin 队列消费者转换为 Core 队列消费者集合。
func provideConsumers(adminConsumers backend.AdminConsumers) queue.Consumers {
	return queue.Consumers(adminConsumers)
}
