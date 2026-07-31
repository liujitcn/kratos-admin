package kratosadmin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/log"
	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/health"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/localgrpc"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/script"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/startup"
	coreStatic "github.com/liujitcn/kratos-admin/backend/core/pkg/static"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	einoModel "github.com/liujitcn/kratos-admin/backend/internal/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/event"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/job"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	internalprojectdocs "github.com/liujitcn/kratos-admin/backend/internal/projectdocs"
	"github.com/liujitcn/kratos-admin/backend/internal/server"
	adminserver "github.com/liujitcn/kratos-admin/backend/internal/server/admin"
	appserver "github.com/liujitcn/kratos-admin/backend/internal/server/app"
	baseserver "github.com/liujitcn/kratos-admin/backend/internal/server/base"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware"
	adminservice "github.com/liujitcn/kratos-admin/backend/internal/service/admin"
	appservice "github.com/liujitcn/kratos-admin/backend/internal/service/app"
	baseservice "github.com/liujitcn/kratos-admin/backend/internal/service/base"
	"github.com/liujitcn/kratos-admin/backend/migration"
	"github.com/liujitcn/kratos-admin/backend/projectdoc"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Module 表示可注册服务并提供进程内 gRPC 客户端连接的 Backend 模块。
type Module interface {
	core.Module
	ClientConn() grpc.ClientConnInterface
}

// AdditionalModules 表示与 Backend 一起装配的扩展模块集合。
type AdditionalModules []core.Module

// Option 配置 Backend 模块及独立应用装配。
type Option func(*options)

type options struct {
	additionalModules          AdditionalModules
	configuredProjectDocuments projectdoc.ConfiguredDocuments
	databaseOptions            []databaseGorm.ClientOption
	migrations                 gormmigration.AdditionalMigrations
}

// WithAdditionalModules 追加由同一启动器挂载的扩展模块。
func WithAdditionalModules(modules ...core.Module) Option {
	return func(opts *options) {
		opts.additionalModules = append(opts.additionalModules, modules...)
	}
}

// WithProjectDocuments 追加由宿主项目直接提供的文档。
func WithProjectDocuments(documents ...projectdoc.Document) Option {
	return func(opts *options) {
		opts.configuredProjectDocuments = append(opts.configuredProjectDocuments, documents...)
	}
}

// WithDatabaseOptions 追加 Backend 数据库客户端选项。
func WithDatabaseOptions(values ...databaseGorm.ClientOption) Option {
	return func(opts *options) {
		opts.databaseOptions = append(opts.databaseOptions, values...)
	}
}

// WithMigrations 追加与 Backend 数据库一起执行的迁移贡献者。
func WithMigrations(contributors ...gormmigration.Contributor) Option {
	return func(opts *options) {
		opts.migrations = append(opts.migrations, contributors...)
	}
}

// Runtime 封装 Backend 服务注册、本地客户端和运行时贡献。
type Runtime struct {
	modules             server.Modules
	clientConn          *localgrpc.Conn
	httpMiddlewares     server.HTTPMiddlewares
	grpcMiddlewares     server.GRPCMiddlewares
	cronServer          *job.CronServer
	openAPIRegistry     *coreOpenAPI.Registry
	projectDocumentCase *adminbiz.ProjectDocumentCase
}

type standaloneRuntime struct {
	servers        []kratosTransport.Server
	startupManager *startup.Manager
}

var _ Module = (*Runtime)(nil)

// NewModule 创建可挂载到其他启动器的 Backend 模块。
func NewModule(ctx *bootstrap.Context, optionValues ...Option) (*Runtime, func(), error) {
	opts := newOptions(optionValues)
	return initModule(
		ctx,
		opts.additionalModules,
		opts.configuredProjectDocuments,
		opts.databaseOptions,
		opts.migrations,
	)
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

// newOptions 合并默认迁移和调用方提供的装配选项。
func newOptions(optionValues []Option) options {
	opts := options{
		databaseOptions: []databaseGorm.ClientOption{databaseGorm.WithMigrateModels(data.Models()...)},
		migrations:      migration.NewMigrations(),
	}
	for _, option := range optionValues {
		option(&opts)
	}
	return opts
}

// newAdditionalProjectDocuments 汇总宿主配置与外部模块贡献的项目文档。
func newAdditionalProjectDocuments(
	modules AdditionalModules,
	configuredDocuments projectdoc.ConfiguredDocuments,
) projectdoc.AdditionalDocuments {
	documents := append(projectdoc.AdditionalDocuments(nil), configuredDocuments...)
	for _, module := range modules {
		contributor, ok := module.(projectdoc.Contributor)
		if !ok {
			continue
		}
		documents = append(documents, contributor.ProjectDocuments()...)
	}
	return documents
}

// newProjectDocumentCatalog 合并内置目录与宿主、外部模块贡献的项目文档。
func newProjectDocumentCatalog(
	appInfo *bootstrapConfigv1.AppInfo,
	additionalDocuments projectdoc.AdditionalDocuments,
) (*projectdoc.Catalog, error) {
	embeddedCatalog, err := projectdoc.ParseCatalog(
		internalprojectdocs.CatalogData,
		appInfo.GetProject(),
		appInfo.GetName(),
	)
	if err != nil {
		return nil, fmt.Errorf("加载内置项目文档目录: %w", err)
	}
	documents := embeddedCatalog.Documents()
	documents = append(documents, additionalDocuments...)
	return projectdoc.NewCatalog(documents...)
}

// newModules 汇总 Backend 内置服务与调用方扩展模块。
func newModules(
	baseModule baseserver.Services,
	adminModule adminserver.Services,
	appModule appserver.Services,
	additionalModules AdditionalModules,
) server.Modules {
	modules := server.Modules{
		baseModule,
		adminModule,
		appModule,
	}
	return append(modules, additionalModules...)
}

// newRuntime 创建服务注册门面并初始化本地 gRPC 连接。
func newRuntime(
	modules server.Modules,
	httpMiddlewares server.HTTPMiddlewares,
	grpcMiddlewares server.GRPCMiddlewares,
	cronServer *job.CronServer,
	openAPIRegistry *coreOpenAPI.Registry,
	baseConfigCase *adminbiz.BaseConfigCase,
	projectDocumentCase *adminbiz.ProjectDocumentCase,
	_ server.MCPToolsReady,
	_ server.AgentToolsReady,
	_ server.OpenAPIReady,
) (*Runtime, error) {
	// 服务完成装配后再初始化配置缓存，失败时不向启动器暴露半初始化模块。
	err := baseConfigCase.RefreshBaseConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("初始化系统配置缓存: %w", err)
	}
	clientConn := localgrpc.NewConn()
	modules.RegisterGRPC(clientConn)
	return &Runtime{
		modules:             modules,
		clientConn:          clientConn,
		httpMiddlewares:     httpMiddlewares,
		grpcMiddlewares:     grpcMiddlewares,
		cronServer:          cronServer,
		openAPIRegistry:     openAPIRegistry,
		projectDocumentCase: projectDocumentCase,
	}, nil
}

// newStandaloneRuntime 收集扩展模块仅在独立部署形态下需要落地的运行时贡献。
func newStandaloneRuntime(
	runtime *Runtime,
	queue kitQueue.Queue,
	taskRegistry *coreTask.Registry,
	sseRegistry *coreSSE.Registry,
) (*standaloneRuntime, func(), error) {
	modules := core.Modules(runtime.modules)
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

// ClientConn 返回与生成 gRPC 客户端兼容的进程内连接。
func (r *Runtime) ClientConn() grpc.ClientConnInterface {
	return r.clientConn
}

// RegisterGRPC 将 Backend gRPC 服务注册到启动器提供的服务注册表。
func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	r.modules.RegisterGRPC(registrar)
}

// RegisterHTTP 将 Backend HTTP 服务注册到启动器提供的 HTTP Server。
func (r *Runtime) RegisterHTTP(httpServer *kratosHTTP.Server) {
	r.modules.RegisterHTTP(httpServer)
}

// RegisterMCP 将 Backend MCP 工具注册到启动器提供的 MCP Server。
func (r *Runtime) RegisterMCP(server *mcpserver.Server) {
	r.modules.RegisterMCP(server)
}

// HTTPMiddlewares 返回 Backend 认证、鉴权、校验和访问日志中间件。
func (r *Runtime) HTTPMiddlewares() []kratosMiddleware.Middleware {
	middlewares := append([]kratosMiddleware.Middleware(nil), r.httpMiddlewares...)
	return append(middlewares, core.Modules(r.modules).HTTPMiddlewares()...)
}

// GRPCMiddlewares 返回 Backend 认证、鉴权、校验和访问日志中间件。
func (r *Runtime) GRPCMiddlewares() []kratosMiddleware.Middleware {
	middlewares := append([]kratosMiddleware.Middleware(nil), r.grpcMiddlewares...)
	return append(middlewares, core.Modules(r.modules).GRPCMiddlewares()...)
}

// Servers 返回 Backend 定时任务和扩展模块后台服务。
func (r *Runtime) Servers() []kratosTransport.Server {
	servers := []kratosTransport.Server{r.cronServer}
	return append(servers, core.Modules(r.modules).Servers()...)
}

// OpenAPIDocuments 返回 Backend 及扩展模块的具名 OpenAPI 文档。
func (r *Runtime) OpenAPIDocuments() []coreOpenAPI.Document {
	return r.openAPIRegistry.Documents()
}

// ProjectDocuments 返回 Backend、宿主配置及扩展模块贡献的项目文档。
func (r *Runtime) ProjectDocuments() []projectdoc.Document {
	return r.projectDocumentCase.ProjectDocuments()
}

// Tasks 返回扩展模块贡献的静态任务。
func (r *Runtime) Tasks() []coreTask.Task {
	return core.Modules(r.modules).Tasks()
}

// QueueConsumers 返回扩展模块贡献的队列消费者。
func (r *Runtime) QueueConsumers() []coreQueue.Consumer {
	return core.Modules(r.modules).QueueConsumers()
}

// SSEStreams 返回扩展模块贡献的 SSE 流。
func (r *Runtime) SSEStreams() []coreSSE.Stream {
	return core.Modules(r.modules).SSEStreams()
}

// Scripts 返回扩展模块贡献的启动脚本。
func (r *Runtime) Scripts() []script.Script {
	return core.Modules(r.modules).Scripts()
}

// StartupHooks 返回扩展模块贡献的启动和清理钩子。
func (r *Runtime) StartupHooks() []startup.Hook {
	return core.Modules(r.modules).StartupHooks()
}

// HealthChecks 返回扩展模块贡献的健康检查。
func (r *Runtime) HealthChecks() []health.Check {
	return core.Modules(r.modules).HealthChecks()
}

// StaticMounts 返回扩展模块贡献的静态资源挂载。
func (r *Runtime) StaticMounts() []coreStatic.Mount {
	return core.Modules(r.modules).StaticMounts()
}

// repositoryProviderSet 只装配仓储，数据库客户端统一由配置层完成多数据源和迁移初始化。
var repositoryProviderSet = data.RepositoryProviderSet

var moduleProviderSet = wire.NewSet(
	event.NewUserEvents,
	job.ProviderSet,
	coreSSE.NewRegistry,
	coreSSE.NewPublisher,
	wire.NewSet(
		gormmigration.NewRegistry,
		gormmigration.NewRunner,
		gormmigration.NewReady,
	),
	biz.ProviderSet,
	einoModel.NewResponsesClient,
	ai.NewRuntime,
	systemConfig.ProviderSet,
	repositoryProviderSet,
	middleware.ProviderSet,
	newAdditionalProjectDocuments,
	newProjectDocumentCatalog,
	adminservice.ProviderSet,
	appservice.ProviderSet,
	baseservice.ProviderSet,
	baseserver.ProviderSet,
	adminserver.ProviderSet,
	appserver.ProviderSet,
	baseserver.NewSSEHandler,
	newModules,
	wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
	server.ModuleProviderSet,
	newRuntime,
)

var appProviderSet = wire.NewSet(
	moduleProviderSet,
	newStandaloneRuntime,
	server.NewGRPCServer,
	server.NewHTTPServer,
	newKratosApp,
)

var (
	_ core.HTTPMiddlewareContributor = (*Runtime)(nil)
	_ core.GRPCMiddlewareContributor = (*Runtime)(nil)
	_ core.ServerContributor         = (*Runtime)(nil)
	_ core.OpenAPIContributor        = (*Runtime)(nil)
	_ projectdoc.Contributor         = (*Runtime)(nil)
	_ core.TaskContributor           = (*Runtime)(nil)
	_ core.QueueConsumerContributor  = (*Runtime)(nil)
	_ core.SSEContributor            = (*Runtime)(nil)
	_ core.ScriptContributor         = (*Runtime)(nil)
	_ core.StartupContributor        = (*Runtime)(nil)
	_ core.HealthContributor         = (*Runtime)(nil)
	_ core.StaticContributor         = (*Runtime)(nil)
)
