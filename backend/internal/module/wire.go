//go:build wireinject
// +build wireinject

package module

import (
	"github.com/google/wire"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	adminCodegen "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	adminSSE "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/sse"
	adminData "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	adminTask "github.com/liujitcn/kratos-admin/backend/internal/task"
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

// BuildModules 通过 Admin 内部 ProviderSet 装配协议服务。
func BuildModules(
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
) (coreModule.Modules, func(), error) {
	panic(wire.Build(ProviderSet))
}

// BuildTasks 通过最小依赖集合装配 Admin 定时任务。
func BuildTasks(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (coreJob.Tasks, func(), error) {
	panic(wire.Build(
		adminData.ProviderSet,
		wire.Bind(new(adminData.QueryProvider), new(*adminData.Data)),
		adminBiz.ProviderSet,
		adminTask.ProviderSet,
	))
}

// BuildStreams 通过最小依赖集合装配 Admin SSE 流。
func BuildStreams(
	databases map[string]*databaseGorm.Client,
	baseCase *coreBiz.BaseCase,
) (coreSSE.Streams, func(), error) {
	panic(wire.Build(
		adminData.ProviderSet,
		wire.Bind(new(adminData.QueryProvider), new(*adminData.Data)),
		adminBiz.ProviderSet,
		adminCodegen.ProviderSet,
		adminSSE.ProviderSet,
	))
}

// BuildQueueConsumers 装配 Admin 队列消费者集合。
func BuildQueueConsumers() coreQueue.Consumers {
	panic(wire.Build(NewQueueConsumers))
}
