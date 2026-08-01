package kratosadmin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/log"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/core"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	"github.com/liujitcn/kratos-admin/backend/internal/server"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

type standaloneRuntime struct {
	servers        []kratosTransport.Server
	startupManager *startup.Manager
}

// NewApp 创建包含 Backend 模块和独立 HTTP、gRPC 服务的 Kratos 应用。
func NewApp(ctx *bootstrap.Context, optionValues ...Option) (*kratos.App, func(), error) {
	opts := newOptions(optionValues)
	return initApp(
		ctx,
		opts.additionalModules,
		opts.configuredProjectDocuments,
		opts.databaseOptions,
		opts.migrations,
	)
}

// newStandaloneRuntime 收集扩展模块仅在独立部署形态下需要落地的运行时贡献。
func newStandaloneRuntime(
	ctx *bootstrap.Context,
	runtime *Runtime,
	queue kitQueue.Queue,
	taskRegistry *coreTask.Registry,
	sseRegistry *coreSSE.Registry,
	ssePublisher *coreSSE.Publisher,
	sseServer *sseServer.Server,
) (*standaloneRuntime, func(), error) {
	modules := core.Modules(runtime.modules)
	modules.SetSSEPublisher(ssePublisher)
	modules.SetSSERegistry(sseRegistry)
	var err error
	err = taskRegistry.Register(modules.Tasks()...)
	if err != nil {
		return nil, nil, fmt.Errorf("注册扩展模块任务: %w", err)
	}
	err = sseRegistry.Register(modules.SSEStreams()...)
	if err != nil {
		return nil, nil, fmt.Errorf("注册扩展模块 SSE 流: %w", err)
	}

	servers := append([]kratosTransport.Server(nil), runtime.Servers()...)
	if ctx.GetConfig().GetServer().GetSse() != nil && ctx.GetConfig().GetServer().GetSse().GetTransport() != bootstrapConfigv1.Server_Sse_IN_PROCESS && sseServer != nil {
		servers = append(servers, sseServer)
	}
	queueConsumers := modules.QueueConsumers()
	if len(queueConsumers) > 0 {
		var queueRuntime *coreQueue.Runtime
		queueRuntime, err = coreQueue.NewRuntime(queue)
		if err != nil {
			return nil, nil, err
		}
		err = queueRuntime.Register(queueConsumers...)
		if err != nil {
			return nil, nil, fmt.Errorf("注册扩展模块队列消费者: %w", err)
		}
		servers = append(servers, queueRuntime)
	}
	if taskRegistry.Scheduled() {
		var scheduler *coreTask.Scheduler
		scheduler, err = coreTask.NewScheduler(taskRegistry, nil)
		if err != nil {
			return nil, nil, err
		}
		servers = append(servers, scheduler)
	}

	// 扩展脚本在统一启动管理器中执行，失败时由管理器按注册顺序回滚。
	scripts := modules.Scripts()
	scriptRegistry := script.NewRegistry()
	err = scriptRegistry.Register(scripts...)
	if err != nil {
		return nil, nil, fmt.Errorf("注册扩展模块启动脚本: %w", err)
	}
	startupManager := startup.NewManager()
	if len(scripts) > 0 {
		err = startupManager.Register(startup.Hook{
			Name:  "backend.scripts",
			Start: scriptRegistry.Run,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	err = startupManager.Register(modules.StartupHooks()...)
	if err != nil {
		return nil, nil, fmt.Errorf("注册扩展模块启动钩子: %w", err)
	}

	standalone := &standaloneRuntime{
		servers:        servers,
		startupManager: startupManager,
	}
	cleanup := func() {
		stopErr := startupManager.Stop(context.Background())
		if stopErr != nil {
			log.Error("清理扩展模块启动资源失败", "error", stopErr)
		}
	}
	return standalone, cleanup, nil
}

// newKratosApp 将 Backend 后台服务和独立传输服务加入 Kratos 生命周期。
func newKratosApp(
	ctx *bootstrap.Context,
	runtime *standaloneRuntime,
	grpcServer *kratosGRPC.Server,
	httpServer *kratosHTTP.Server,
) (*kratos.App, error) {
	err := runtime.startupManager.Start(ctx.Context())
	if err != nil {
		return nil, err
	}
	servers := append([]kratosTransport.Server(nil), runtime.servers...)
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	return bootstrap.NewApp(ctx, servers...), nil
}

var appProviderSet = wire.NewSet(
	appModuleProviderSet,
	newStandaloneRuntime,
	server.NewGRPCServer,
	server.NewHTTPServer,
	newKratosApp,
)
