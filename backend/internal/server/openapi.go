package server

import (
	"fmt"

	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/cmd/server/assets"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

const (
	defaultOpenAPIKey  = "admin"
	defaultOpenAPIName = "系统管理"
)

// OpenAPIReady 表示全部模块文档和接口权限数据已经完成初始化。
type OpenAPIReady struct{}

// NewOpenAPIRegistry 创建包含系统内置文档的 OpenAPI 注册表。
func NewOpenAPIRegistry() (*coreOpenAPI.Registry, error) {
	return coreOpenAPI.NewRegistry(coreOpenAPI.Document{
		Key:  defaultOpenAPIKey,
		Name: defaultOpenAPIName,
		Data: assets.OpenAPIData,
	})
}

// NewOpenAPIReady 注册扩展模块文档并同步接口权限数据。
func NewOpenAPIReady(
	ctx *bootstrap.Context,
	baseAPICase *adminBiz.BaseAPICase,
	baseTenantCase *adminBiz.BaseTenantCase,
	casbinRuleCase *adminBiz.CasbinRuleCase,
	registry *coreOpenAPI.Registry,
	modules Modules,
) (OpenAPIReady, error) {
	documents, err := modules.OpenAPIDocuments()
	if err != nil {
		return OpenAPIReady{}, err
	}
	err = registry.Register(documents...)
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("注册 OpenAPI 文档: %w", err)
	}
	err = baseAPICase.InitializeOpenAPI(ctx.Context(), registry.Documents())
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("初始化 OpenAPI 接口数据: %w", err)
	}
	err = baseTenantCase.SyncTenantRoleMenus(ctx.Context())
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("同步租户管理员菜单: %w", err)
	}
	err = casbinRuleCase.RebuildAllCasbinRules(ctx.Context())
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("重建 Casbin 权限规则: %w", err)
	}
	return OpenAPIReady{}, nil
}
