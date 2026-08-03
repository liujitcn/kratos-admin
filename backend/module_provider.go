package kratosadmin

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/localgrpc"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/projectdoc"
	coreSSE "github.com/liujitcn/kratos-admin/backend/core/pkg/sse"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	einoModel "github.com/liujitcn/kratos-admin/backend/internal/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/v1/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/event"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/job"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"
	admincodegen "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/codegen"
	appbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app/v1"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/docs"
	"github.com/liujitcn/kratos-admin/backend/internal/server"
	baseserver "github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware"
	adminserver "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appserver "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
	service "github.com/liujitcn/kratos-admin/backend/internal/service"
	backendtask "github.com/liujitcn/kratos-admin/backend/internal/task"
	backendmodule "github.com/liujitcn/kratos-admin/backend/module"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// newAdditionalProjectDocuments 汇总宿主配置与外部模块贡献的项目文档。
func newAdditionalProjectDocuments(
	modules AdditionalModules,
	configuredDocuments projectdoc.ConfiguredDocuments,
) projectdoc.AdditionalDocuments {
	documents := append(projectdoc.AdditionalDocuments(nil), configuredDocuments...)
	documents = append(documents, core.Modules(modules).ProjectDocuments()...)
	return documents
}

// newProjectDocumentCatalog 合并 Admin 内置目录与宿主、外部模块贡献的项目文档。
func newProjectDocumentCatalog(
	additionalDocuments projectdoc.AdditionalDocuments,
) (*projectdoc.Catalog, error) {
	embeddedCatalog, err := projectdoc.ParseCatalog(
		docs.DocsData,
		_const.PROJECT_KEY,
		_const.PROJECT_NAME,
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
	taskRegistry *coreTask.Registry,
	translationTask *backendtask.BaseTranslationTask,
	openAPIRegistry *coreOpenAPI.Registry,
	baseConfigCase *adminbiz.BaseConfigCase,
	projectDocumentCase *adminbiz.ProjectDocumentCase,
	aiRuntime *ai.Runtime,
	userEvents *event.UserEvents,
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
	err = registerModuleExtensions(modules, aiRuntime, userEvents)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		contributor, ok := module.(backendmodule.RuntimeReadyContributor)
		if !ok {
			continue
		}
		err = contributor.RuntimeReady(clientConn)
		if err != nil {
			return nil, fmt.Errorf("通知扩展模块运行时就绪: %w", err)
		}
	}
	runtime := &Runtime{
		modules:             modules,
		clientConn:          clientConn,
		httpMiddlewares:     httpMiddlewares,
		grpcMiddlewares:     grpcMiddlewares,
		cronServer:          cronServer,
		translationTask:     translationTask,
		openAPIRegistry:     openAPIRegistry,
		projectDocumentCase: projectDocumentCase,
	}
	err = taskRegistry.Register(runtime.Tasks()...)
	if err != nil {
		return nil, fmt.Errorf("注册 Backend 任务: %w", err)
	}
	return runtime, nil
}

// registerModuleExtensions 注册扩展模块贡献的 AI 固定流程和用户事件订阅者。
func registerModuleExtensions(modules server.Modules, aiRuntime *ai.Runtime, userEvents *event.UserEvents) error {
	var err error
	for _, module := range modules {
		aiContributor, ok := module.(backendmodule.AIFixedFlowContributor)
		if ok {
			for _, provider := range aiContributor.AIFixedFlowProviders() {
				err = aiRuntime.RegisterFixedFlow(provider)
				if err != nil {
					return fmt.Errorf("注册扩展模块 AI 固定流程: %w", err)
				}
			}
		}
		var userContributor backendmodule.UserSubscriberContributor
		userContributor, ok = module.(backendmodule.UserSubscriberContributor)
		if !ok {
			continue
		}
		for _, subscriber := range userContributor.UserSubscribers() {
			userEvents.Subscribe(subscriber)
		}
	}
	return nil
}

// repositoryProviderSet 只装配仓储，数据库客户端统一由配置层完成多数据源和迁移初始化。
var repositoryProviderSet = data.RepositoryProviderSet

var migrationProviderSet = wire.NewSet(
	gormmigration.NewRegistry,
	gormmigration.NewRunner,
	gormmigration.NewReady,
)

var moduleCommonProviderSet = wire.NewSet(
	event.NewUserEvents,
	job.ProviderSet,
	coreSSE.NewRegistry,
	coreSSE.NewPublisher,
	migrationProviderSet,
	biz.ProviderSet,
	biz.InfrastructureProviderSet,
	basebiz.ProviderSet,
	adminbiz.ProviderSet,
	appbiz.ProviderSet,
	admincodegen.ProviderSet,
	backendtask.ProviderSet,
	einoModel.NewResponsesClient,
	ai.NewRuntime,
	systemConfig.ProviderSet,
	repositoryProviderSet,
	middleware.ProviderSet,
	newAdditionalProjectDocuments,
	newProjectDocumentCatalog,
	service.ProviderSet,
	baseserver.ProviderSet,
	adminserver.ProviderSet,
	appserver.ProviderSet,
	newModules,
	wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
	server.ModuleProviderSet,
	newRuntime,
)

var moduleProviderSet = wire.NewSet(
	moduleCommonProviderSet,
	baseserver.NewSSEHandler,
)

var appModuleProviderSet = wire.NewSet(
	moduleCommonProviderSet,
	baseserver.NewSSEServer,
)
