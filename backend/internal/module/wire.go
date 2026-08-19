//go:build wireinject
// +build wireinject

package module

import (
	"github.com/google/wire"
	bizProvider "github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	adminSSE "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/sse"
	configProvider "github.com/liujitcn/kratos-admin/backend/internal/config"
	adminData "github.com/liujitcn/kratos-admin/backend/internal/data"
	serverProvider "github.com/liujitcn/kratos-admin/backend/internal/server"
	serviceProvider "github.com/liujitcn/kratos-admin/backend/internal/service"
	taskProvider "github.com/liujitcn/kratos-admin/backend/internal/task"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	coreJob "github.com/liujitcn/kratos-core/job"
	coreModule "github.com/liujitcn/kratos-core/module"
	coreQueue "github.com/liujitcn/kratos-core/queue"
	coreDocs "github.com/liujitcn/kratos-core/resource/docs"
	coreI18n "github.com/liujitcn/kratos-core/resource/i18n"
	coreOpenAPI "github.com/liujitcn/kratos-core/resource/openapi"
	coreSSE "github.com/liujitcn/kratos-core/sse"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// BuildModules 通过 Admin 内部依赖装配协议服务。
func BuildModules(
	config *bootstrapConfigv1.Bootstrap,
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jobRuntime *coreJob.Job,
	sseRuntime *coreSSE.SSE,
	docsRuntime *coreDocs.Docs,
	catalog *coreI18n.I18n,
	openAPIRuntime *coreOpenAPI.OpenAPI,
) (coreModule.Modules, func(), error) {
	panic(wire.Build(
		ParseAdminAgentTools,
		ParseAppAgentTools,
		configProvider.ProviderSet,
		logstream.DefaultHub,
		NewModules,
		bizProvider.ProviderSet,
		adminData.ProviderSet,
		serviceProvider.ProviderSet,
		serverProvider.ProviderSet,
	))
}

// BuildTasks 通过最小依赖集合装配 Admin 定时任务。
func BuildTasks(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (coreJob.Tasks, func(), error) {
	panic(wire.Build(
		adminData.ProviderSet,
		taskProvider.ProviderSet,
	))
}

// BuildStreams 通过最小依赖集合装配 Admin SSE 流。
func BuildStreams(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (coreSSE.Streams, func(), error) {
	panic(wire.Build(
		adminData.ProviderSet,
		adminSSE.ProviderSet,
	))
}

// BuildQueueConsumers 装配 Admin 队列消费者集合。
func BuildQueueConsumers() coreQueue.Consumers {
	panic(wire.Build(NewQueueConsumers))
}
