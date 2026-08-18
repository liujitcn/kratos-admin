package module

import (
	"errors"

	"github.com/google/wire"
	einoModel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	baseBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	baseAI "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	adminCodegen "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	adminSSE "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/sse"
	appBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app"
	adminData "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	baseServer "github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	adminServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
	baseService "github.com/liujitcn/kratos-admin/backend/internal/service/base/v1"
	adminService "github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"
	appService "github.com/liujitcn/kratos-admin/backend/internal/service/system/app/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/task"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/oauth"
)

var ProviderSet = wire.NewSet(
	NewModuleResources,
	ParseAIModel,
	ParseAdminAgentTools,
	ParseAppAgentTools,
	ParseOAuthManager,
	NewModules,
	adminSSE.ProviderSet,
	einoModel.NewResponsesClient,
	adminData.ProviderSet,
	wire.Bind(new(adminData.QueryProvider), new(*adminData.Data)),
	baseAI.NewRuntime,
	adminCodegen.ProviderSet,
	baseBiz.ProviderSet,
	adminBiz.ProviderSet,
	appBiz.ProviderSet,
	baseService.ProviderSet,
	adminService.ProviderSet,
	appService.ProviderSet,
	task.ProviderSet,
	wire.Struct(new(baseServer.Services), "*"),
	wire.Struct(new(adminServer.Services), "*"),
	wire.Struct(new(appServer.Services), "*"),
)

// ParseAIModel 提取本地 AI 模型配置。
func ParseAIModel(cfg *configv1.Bootstrap) (*configv1.AI_Model, error) {
	if cfg == nil || cfg.GetAi() == nil {
		return nil, errors.New("ai相关配置为空")
	}
	return cfg.GetAi().GetModel(), nil
}

// ParseAdminAgentTools 创建管理端服务导出的 Eino 工具。
func ParseAdminAgentTools(services adminServer.Services) (baseAI.AdminTools, error) {
	return services.AgentTools()
}

// ParseAppAgentTools 创建应用端服务导出的 Eino 工具。
func ParseAppAgentTools(services appServer.Services) (baseAI.AppTools, error) {
	return services.AppAgentTools()
}

// ParseOAuthManager 根据 Admin 启动配置创建 OAuth 管理器。
func ParseOAuthManager(cfg *configv1.Bootstrap) (*oauth.Manager, error) {
	return oauth.NewManager(cfg.GetOauth())
}
