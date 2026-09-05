//go:build wireinject
// +build wireinject

package module

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/adapter/kit"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/sse"
	configProvider "github.com/liujitcn/kratos-admin/backend/internal/config"
	"github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/server"
	logmiddleware "github.com/liujitcn/kratos-admin/backend/internal/server/middleware/log"
	"github.com/liujitcn/kratos-admin/backend/internal/service"
	"github.com/liujitcn/kratos-admin/backend/internal/task"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/job"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-core/resource/docs"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"github.com/liujitcn/kratos-core/resource/openapi"
	coreSSE "github.com/liujitcn/kratos-core/sse"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BuildModules 通过 Admin 内部依赖装配协议服务。
func BuildModules(
	config *configv1.Bootstrap,
	databases map[string]*gorm.Client,
	baseCase *coreBiz.BaseCase,
	authorizer engine.Engine,
	authenticator authnEngine.Authenticator,
	userToken *authData.UserToken,
	jobRuntime *job.Job,
	sseRuntime *coreSSE.SSE,
	docsRuntime *docs.Docs,
	catalog *i18n.I18n,
	openAPIRuntime *openapi.OpenAPI,
) (module.Modules, func(), error) {
	panic(wire.Build(
		ParseAdminAgentTools,
		ParseAppAgentTools,
		configProvider.ProviderSet,
		logstream.DefaultHub,
		kit.ProviderSet,
		NewModules,
		biz.ProviderSet,
		biz.MessageProviderSet,
		data.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
	))
}

// BuildTasks 通过最小依赖集合装配 Admin 定时任务。
func BuildTasks(
	databases map[string]*gorm.Client,
	baseCase *coreBiz.BaseCase,
	sseRuntime *coreSSE.SSE,
) (job.Tasks, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		biz.MessageProviderSet,
		task.ProviderSet,
	))
}

// BuildStreams 通过最小依赖集合装配 Admin SSE 流。
func BuildStreams(
	databases map[string]*gorm.Client,
	baseCase *coreBiz.BaseCase,
	catalog *i18n.I18n,
) (coreSSE.Streams, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		sse.ProviderSet,
	))
}

// BuildQueueConsumers 装配 Admin 队列消费者集合。
func BuildQueueConsumers(
	databases map[string]*gorm.Client,
	baseCase *coreBiz.BaseCase,
	sseRuntime *coreSSE.SSE,
) (queue.Consumers, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		biz.MessageProviderSet,
		logmiddleware.NewConsumer,
		NewQueueConsumers,
	))
}
