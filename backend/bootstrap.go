package backend

import (
	"github.com/google/wire"
	adminModule "github.com/liujitcn/kratos-admin/backend/internal/module"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	coreJob "github.com/liujitcn/kratos-core/job"
	coreModule "github.com/liujitcn/kratos-core/module"
	coreQueue "github.com/liujitcn/kratos-core/queue"
	coreDocs "github.com/liujitcn/kratos-core/resource/docs"
	coreI18n "github.com/liujitcn/kratos-core/resource/i18n"
	coreOpenAPI "github.com/liujitcn/kratos-core/resource/openapi"
	coreSSE "github.com/liujitcn/kratos-core/sse"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// AdminResources 表示 Admin 提供的静态资源集合。
type AdminResources coreModule.Resources

// AdminModules 表示 Admin 提供的协议模块集合。
type AdminModules coreModule.Modules

// AdminTasks 表示 Admin 提供的定时任务集合。
type AdminTasks coreJob.Tasks

// AdminStreams 表示 Admin 提供的 SSE 业务流集合。
type AdminStreams coreSSE.Streams

// AdminConsumers 表示 Admin 提供的队列消费者集合。
type AdminConsumers coreQueue.Consumers

// ProviderSet 提供可被外部 Core 宿主复用的 Backend 业务、模块和运行时能力。
//
// 外部项目将本集合与其他业务模块的具名贡献合并后，再交给 kratos-core.ProviderSet
// 统一创建 HTTP、gRPC、MCP、SSE、队列和定时任务运行时。
var ProviderSet = wire.NewSet(
	NewModuleResources,
	NewModules,
	NewTasks,
	NewStreams,
	NewQueueConsumers,
)

// NewModuleResources 返回 Backend 提供给 Core 的模型、迁移、文档、OpenAPI 和语言资源。
func NewModuleResources() AdminResources {
	return AdminResources(adminModule.NewModuleResources())
}

// NewModules 创建 Backend 注册到 Core 的协议模块集合。
//
// 参数均来自 kratos-core.ProviderSet：数据库客户端由模块资源驱动创建，
// BaseCase、Job、SSE、文档和 OpenAPI 运行时由 Core 统一提供。Admin 业务依赖
// 在 Backend 内部完成装配，避免外部项目的生成代码引用 backend/internal 包。
func NewModules(
	config *configv1.Bootstrap,
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jobRuntime *coreJob.Job,
	sseRuntime *coreSSE.SSE,
	docsRuntime *coreDocs.Docs,
	catalog *coreI18n.I18n,
	openAPIRuntime *coreOpenAPI.OpenAPI,
) (AdminModules, func(), error) {
	modules, cleanup, err := adminModule.BuildModules(config, databases, baseCase, authorizer, userToken, jobRuntime, sseRuntime, docsRuntime, catalog, openAPIRuntime)
	return AdminModules(modules), cleanup, err
}

// NewTasks 创建 Backend 提供给 Core 调度器的定时任务集合。
func NewTasks(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (AdminTasks, func(), error) {
	tasks, cleanup, err := adminModule.BuildTasks(databases, baseCase)
	return AdminTasks(tasks), cleanup, err
}

// NewStreams 创建 Backend 提供给 Core SSE 服务的业务流集合。
func NewStreams(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (AdminStreams, func(), error) {
	streams, cleanup, err := adminModule.BuildStreams(databases, baseCase)
	return AdminStreams(streams), cleanup, err
}

// NewQueueConsumers 创建 Backend 提供给 Core 队列服务的消费者集合。
func NewQueueConsumers() AdminConsumers {
	return AdminConsumers(adminModule.BuildQueueConsumers())
}
