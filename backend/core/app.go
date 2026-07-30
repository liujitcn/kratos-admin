package core

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/log"
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

const defaultMCPPath = "/mcp"

// NewApp 创建只包含传输层、拦截器和外部模块挂载能力的 Kratos 应用。
func NewApp(ctx *bootstrap.Context, optionValues ...Option) (*kratos.App, func(), error) {
	opts := options{}
	for _, option := range optionValues {
		option(&opts)
	}

	var err error
	var openAPIRegistry *openapi.Registry
	openAPIRegistry, err = prepareOpenAPI(opts)
	if err != nil {
		return nil, nil, err
	}
	var taskRegistry *task.Registry
	taskRegistry, err = prepareTasks(opts)
	if err != nil {
		return nil, nil, err
	}
	err = prepareSSE(opts)
	if err != nil {
		return nil, nil, err
	}
	var healthRegistry *health.Registry
	healthRegistry, err = prepareHealth(opts)
	if err != nil {
		return nil, nil, err
	}
	var startupManager *startup.Manager
	startupManager, err = prepareStartup(opts)
	if err != nil {
		return nil, nil, err
	}

	var mcpServer *mcpserver.Server
	var configuredSSEServer *sseServer.Server
	var grpcServer *grpc.Server
	var httpServer *kratosHTTP.Server
	serversOwnedByApp := false
	defer func() {
		if serversOwnedByApp {
			return
		}
		var stopErr error
		if httpServer != nil {
			stopErr = httpServer.Stop(context.Background())
			if stopErr != nil {
				log.Error("清理 HTTP 服务初始化资源失败", "error", stopErr)
			}
		}
		if grpcServer != nil {
			stopErr = grpcServer.Stop(context.Background())
			if stopErr != nil {
				log.Error("清理 gRPC 服务初始化资源失败", "error", stopErr)
			}
		}
		if configuredSSEServer != nil && opts.sseServer == nil {
			stopErr = configuredSSEServer.Stop(context.Background())
			if stopErr != nil {
				log.Error("清理 SSE 服务初始化资源失败", "error", stopErr)
			}
		}
		if mcpServer != nil {
			stopErr = mcpServer.Stop(context.Background())
			if stopErr != nil {
				log.Error("清理 MCP 服务初始化资源失败", "error", stopErr)
			}
		}
	}()

	mcpServer, err = newMCPServer(ctx.GetConfig(), opts.modules)
	if err != nil {
		return nil, nil, err
	}
	configuredSSEServer, err = newSSEServer(ctx.GetConfig(), opts.sseServer)
	if err != nil {
		return nil, nil, err
	}

	grpcServer, err = newGRPCServer(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	httpServer, err = newHTTPServer(ctx, opts, mcpServer, configuredSSEServer, openAPIRegistry, healthRegistry)
	if err != nil {
		return nil, nil, err
	}

	servers := append([]kratosTransport.Server(nil), opts.servers...)
	servers = append(servers, opts.modules.Servers()...)
	queueConsumers := append([]coreQueue.Consumer(nil), opts.queueConsumers...)
	queueConsumers = append(queueConsumers, opts.modules.QueueConsumers()...)
	if opts.queue != nil {
		var queueRuntime *coreQueue.Runtime
		queueRuntime, err = coreQueue.NewRuntime(opts.queue)
		if err != nil {
			return nil, nil, err
		}
		err = queueRuntime.Register(queueConsumers...)
		if err != nil {
			return nil, nil, fmt.Errorf("注册队列消费者: %w", err)
		}
		servers = append(servers, queueRuntime)
	} else if len(queueConsumers) > 0 {
		return nil, nil, fmt.Errorf("模块提供了队列消费者，但未通过 WithQueue 注入队列适配器")
	}
	if taskRegistry.Scheduled() {
		var scheduler *task.Scheduler
		scheduler, err = task.NewScheduler(taskRegistry, opts.taskObserver)
		if err != nil {
			return nil, nil, err
		}
		servers = append(servers, scheduler)
	}
	if configuredSSEServer != nil && ctx.GetConfig().GetServer().GetSse().GetTransport() == bootstrapConfigv1.Server_Sse_HTTP {
		servers = append(servers, configuredSSEServer)
	}
	if mcpServer != nil && !mcpInProcess(ctx.GetConfig()) {
		servers = append(servers, mcpServer)
	}
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	err = startupManager.Start(ctx.Context())
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		stopErr := startupManager.Stop(context.Background())
		if stopErr != nil {
			log.Error("清理核心服务启动资源失败", "error", stopErr)
		}
	}
	serversOwnedByApp = true
	return bootstrap.NewApp(ctx, servers...), cleanup, nil
}

// Run 创建引导上下文并启动核心服务宿主。
func Run(parent context.Context, appInfo *bootstrapConfigv1.AppInfo, optionValues ...Option) error {
	ctx := bootstrap.NewContext(parent, appInfo)
	return bootstrap.RunApp(ctx, func(ctx *bootstrap.Context) (*kratos.App, func(), error) {
		return NewApp(ctx, optionValues...)
	})
}

// prepareOpenAPI 收集并校验组合根和模块贡献的 OpenAPI 文档。
func prepareOpenAPI(opts options) (*openapi.Registry, error) {
	registry := opts.openAPIRegistry
	var err error
	if registry == nil {
		registry, err = openapi.NewRegistry()
		if err != nil {
			return nil, err
		}
	}
	documents := append([]openapi.Document(nil), opts.openAPIDocuments...)
	documents = append(documents, opts.modules.OpenAPIDocuments()...)
	err = registry.Register(documents...)
	if err != nil {
		return nil, fmt.Errorf("注册 OpenAPI 文档: %w", err)
	}
	return registry, nil
}

// prepareTasks 收集并校验组合根和模块贡献的任务。
func prepareTasks(opts options) (*task.Registry, error) {
	registry := opts.taskRegistry
	if registry == nil {
		registry = task.NewRegistry()
	}
	tasks := append([]task.Task(nil), opts.tasks...)
	tasks = append(tasks, opts.modules.Tasks()...)
	var err error
	err = registry.Register(tasks...)
	if err != nil {
		return nil, fmt.Errorf("注册定时任务: %w", err)
	}
	return registry, nil
}

// prepareSSE 收集并校验组合根和模块贡献的 SSE 流。
func prepareSSE(opts options) error {
	registry := opts.sseRegistry
	if registry == nil {
		registry = coreSSE.NewRegistry()
	}
	streams := append([]coreSSE.Stream(nil), opts.sseStreams...)
	streams = append(streams, opts.modules.SSEStreams()...)
	var err error
	err = registry.Register(streams...)
	if err != nil {
		return fmt.Errorf("注册 SSE 流: %w", err)
	}
	return nil
}

// prepareHealth 收集并校验组合根和模块贡献的健康检查。
func prepareHealth(opts options) (*health.Registry, error) {
	registry := health.NewRegistry()
	checks := append([]health.Check(nil), opts.healthChecks...)
	checks = append(checks, opts.modules.HealthChecks()...)
	var err error
	err = registry.Register(checks...)
	if err != nil {
		return nil, fmt.Errorf("注册健康检查: %w", err)
	}
	return registry, nil
}

// prepareStartup 收集启动脚本及服务启动和清理钩子。
func prepareStartup(opts options) (*startup.Manager, error) {
	var err error
	scriptRegistry := script.NewRegistry()
	scripts := append([]script.Script(nil), opts.scripts...)
	scripts = append(scripts, opts.modules.Scripts()...)
	err = scriptRegistry.Register(scripts...)
	if err != nil {
		return nil, fmt.Errorf("注册启动脚本: %w", err)
	}

	manager := startup.NewManager()
	if len(scripts) > 0 {
		err = manager.Register(startup.Hook{
			Name:  "core.scripts",
			Start: scriptRegistry.Run,
		})
		if err != nil {
			return nil, err
		}
	}
	hooks := append([]startup.Hook(nil), opts.startupHooks...)
	hooks = append(hooks, opts.modules.StartupHooks()...)
	err = manager.Register(hooks...)
	if err != nil {
		return nil, fmt.Errorf("注册启动钩子: %w", err)
	}
	return manager, nil
}

// newMCPServer 创建 MCP 服务并注册外部模块工具。
func newMCPServer(cfg *bootstrapConfigv1.Bootstrap, modules Modules) (*mcpserver.Server, error) {
	if cfg.GetServer().GetMcp() == nil {
		return nil, nil
	}
	var err error
	var server *mcpserver.Server
	if mcpInProcess(cfg) {
		server, err = rpc.CreateMcpHandler(cfg)
	} else {
		server, err = rpc.CreateMcpServer(cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("创建 MCP 服务: %w", err)
	}
	modules.RegisterMCP(server)
	return server, nil
}

// newSSEServer 根据配置创建或复用 SSE 服务。
func newSSEServer(cfg *bootstrapConfigv1.Bootstrap, configured *sseServer.Server) (*sseServer.Server, error) {
	if configured != nil || cfg.GetServer().GetSse() == nil {
		return configured, nil
	}
	var err error
	var server *sseServer.Server
	if cfg.GetServer().GetSse().GetTransport() == bootstrapConfigv1.Server_Sse_HTTP {
		server, err = rpc.CreateSseServer(cfg)
		if err != nil {
			return nil, fmt.Errorf("创建 SSE 服务: %w", err)
		}
		return server, nil
	}
	server, err = rpc.CreateSseHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 SSE 处理器: %w", err)
	}
	return server, nil
}

// newGRPCServer 创建 gRPC 服务并挂载外部模块。
func newGRPCServer(ctx *bootstrap.Context, opts options) (*grpc.Server, error) {
	cfg := ctx.GetConfig()
	if cfg.GetServer().GetGrpc() == nil {
		return nil, nil
	}
	middlewares := defaultMiddlewares(ctx, cfg.GetServer().GetGrpc().GetMiddleware().GetEnableLogging())
	middlewares = append(middlewares, opts.grpcMiddlewares...)
	middlewares = append(middlewares, opts.modules.GRPCMiddlewares()...)
	var err error
	var server *grpc.Server
	server, err = rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 服务: %w", err)
	}
	opts.modules.RegisterGRPC(server)
	return server, nil
}

// newHTTPServer 创建 HTTP 服务并挂载健康检查、MCP 和外部模块。
func newHTTPServer(
	ctx *bootstrap.Context,
	opts options,
	mcpServer *mcpserver.Server,
	configuredSSEServer *sseServer.Server,
	openAPIRegistry *openapi.Registry,
	healthRegistry *health.Registry,
) (*kratosHTTP.Server, error) {
	cfg := ctx.GetConfig()
	if cfg.GetServer().GetHttp() == nil {
		return nil, nil
	}
	middlewares := defaultMiddlewares(ctx, cfg.GetServer().GetHttp().GetMiddleware().GetEnableLogging())
	middlewares = append(middlewares, opts.httpMiddlewares...)
	middlewares = append(middlewares, opts.modules.HTTPMiddlewares()...)
	var err error
	var server *kratosHTTP.Server
	server, err = rpc.CreateHttpServer(cfg, middlewares...)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 服务: %w", err)
	}
	serverReturned := false
	defer func() {
		if serverReturned {
			return
		}
		stopErr := server.Stop(context.Background())
		if stopErr != nil {
			log.Error("清理 HTTP 服务初始化资源失败", "error", stopErr)
		}
	}()
	health.RegisterHTTP(server, healthRegistry)
	opts.modules.RegisterHTTP(server)
	staticMounts := append([]coreStatic.Mount(nil), opts.staticMounts...)
	staticMounts = append(staticMounts, opts.modules.StaticMounts()...)
	if err = coreStatic.RegisterHTTP(server, staticMounts...); err != nil {
		return nil, fmt.Errorf("注册静态资源: %w", err)
	}
	if cfg.GetServer().GetHttp().GetEnableSwagger() && len(openAPIRegistry.Documents()) > 0 {
		openapi.RegisterHTTP(server, openAPIRegistry, opts.openAPIOptions)
	}
	if configuredSSEServer != nil && cfg.GetServer().GetSse().GetTransport() != bootstrapConfigv1.Server_Sse_HTTP {
		path := cfg.GetServer().GetSse().GetPath()
		if path == "" {
			path = "/events"
		}
		server.Handle(path, configuredSSEServer)
	}
	if mcpServer != nil && mcpInProcess(cfg) {
		var handler stdhttp.Handler
		handler, err = mcpServer.HTTPHandler()
		if err != nil {
			return nil, fmt.Errorf("创建 MCP HTTP 处理器: %w", err)
		}
		path := cfg.GetServer().GetMcp().GetPath()
		if path == "" {
			path = defaultMCPPath
		}
		server.Handle(path, handler)
	}
	serverReturned = true
	return server, nil
}

// mcpInProcess 判断 MCP 服务是否挂载到当前 HTTP 服务。
func mcpInProcess(cfg *bootstrapConfigv1.Bootstrap) bool {
	transport := cfg.GetServer().GetMcp().GetTransport()
	return transport == bootstrapConfigv1.Server_Mcp_UNSPECIFIED || transport == bootstrapConfigv1.Server_Mcp_IN_PROCESS
}

// defaultMiddlewares 根据传输配置创建框架级访问日志中间件。
func defaultMiddlewares(ctx *bootstrap.Context, enableLogging bool) []kratosMiddleware.Middleware {
	middlewares := make([]kratosMiddleware.Middleware, 0, 1)
	if enableLogging && ctx.GetLogger() != nil {
		middlewares = append(middlewares, logging.Server(ctx.GetLogger()))
	}
	return middlewares
}
