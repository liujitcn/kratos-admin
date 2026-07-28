package kratosadmin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
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
	systemadminbiz "github.com/liujitcn/kratos-admin/backend/service/system/admin/biz"
	systemapp "github.com/liujitcn/kratos-admin/backend/service/system/app"
	"github.com/liujitcn/kratos-kit/bootstrap"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// Module 表示可挂载到基础应用宿主的业务模块。
type Module = server.Module

// Modules 表示当前进程启用的业务模块集合。
type Modules = server.Modules

// AdditionalModules 表示由业务项目注入的扩展模块集合。
type AdditionalModules []Module

// TaskContributor 表示可向调度运行时贡献具名任务的业务模块。
type TaskContributor = server.TaskContributor

// RegisterTasks 将业务模块贡献的任务注册到基础调度运行时。
func RegisterTasks(registry *job.Registry, contributors ...TaskContributor) error {
	return server.RegisterTasks(registry, contributors...)
}

// ProviderSet 汇总基础应用及其 HTTP、gRPC、MCP 服务依赖。
var ProviderSet = wire.NewSet(
	event.NewUserEvents,
	job.ProviderSet,
	transportSSE.NewRegistry,
	transportSSE.NewPublisher,
	wire.NewSet(
		gormmigration.NewRegistry,
		gormmigration.NewRunner,
		gormmigration.NewReady,
	),
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
	NewModules,
	wire.Value([]databaseGorm.ClientOption(nil)),
	wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
	server.ProviderSet,
)

// NewModules 汇总当前进程启用的基础、系统与业务模块。
func NewModules(
	baseModule baseserver.Services,
	systemAdminModule systemadminserver.Services,
	systemAppModule systemappserver.Services,
	additionalModules AdditionalModules,
) (Modules, error) {
	modules := Modules{
		baseModule,
		systemAdminModule,
		systemAppModule,
	}
	return append(modules, additionalModules...), nil
}

// NewApp 创建并挂载基础应用宿主提供的服务。
func NewApp(
	ctx *bootstrap.Context,
	baseConfigCase *systemadminbiz.BaseConfigCase,
	cron *job.CronServer,
	grpcServer *grpc.Server,
	httpServer *http.Server,
) (*kratos.App, error) {
	// 所有服务完成装配后再初始化缓存，初始化失败时不启动定时任务及传输服务。
	err := baseConfigCase.RefreshBaseConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("初始化系统配置缓存: %w", err)
	}

	servers := make([]kratosTransport.Server, 0, 3)
	if cron != nil {
		servers = append(servers, cron)
	}
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	return bootstrap.NewApp(ctx, servers...), nil
}
