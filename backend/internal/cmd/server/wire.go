//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/bootstrap"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"

	einoModel "github.com/liujitcn/kratos-admin/backend/pkg/agent/eino/model"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	systemConfig "github.com/liujitcn/kratos-admin/backend/pkg/config"
	"github.com/liujitcn/kratos-admin/backend/pkg/event"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/job"
	"github.com/liujitcn/kratos-admin/backend/pkg/middleware"
	transportSSE "github.com/liujitcn/kratos-admin/backend/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/server"
	baseserver "github.com/liujitcn/kratos-admin/backend/server/base"
	systemadminserver "github.com/liujitcn/kratos-admin/backend/server/system/admin"
	systemappserver "github.com/liujitcn/kratos-admin/backend/server/system/app"
	"github.com/liujitcn/kratos-admin/backend/service/base"
	"github.com/liujitcn/kratos-admin/backend/service/base/agent/ai"
	systemadmin "github.com/liujitcn/kratos-admin/backend/service/system/admin"
	systemapp "github.com/liujitcn/kratos-admin/backend/service/system/app"
)

// initApp 初始化 Kratos 应用实例。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		event.NewUserEvents,
		job.ProviderSet,
		transportSSE.NewRegistry,
		transportSSE.NewPublisher,
		biz.ProviderSet,
		einoModel.NewResponsesClient,
		ai.NewRuntime,
		systemConfig.ProviderSet,
		data.ProviderSet,
		middleware.ProviderSet,
		systemadmin.ProviderSet,
		systemapp.ProviderSet,
		base.ProviderSet,
		baseserver.ProviderSet,
		systemadminserver.ProviderSet,
		systemappserver.ProviderSet,
		baseserver.NewSSEHandler,
		wire.Value([]databaseGorm.ClientOption(nil)),
		newModules,
		wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
		server.ProviderSet,
		newApp,
	))
}
