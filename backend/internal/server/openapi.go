package server

import (
	"fmt"

	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	appBiz "github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/server/assets"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

const (
	defaultOpenAPIKey  = "admin"
	defaultOpenAPIName = "Kratos Admin"
	defaultOpenAPIPath = "/api/docs/openapi"
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
func NewOpenAPIReady(ctx *bootstrap.Context, baseCase *appBiz.BaseCase, registry *coreOpenAPI.Registry, modules Modules) (OpenAPIReady, error) {
	documents, err := modules.OpenAPIDocuments()
	if err != nil {
		return OpenAPIReady{}, err
	}
	err = registry.Register(documents...)
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("注册 OpenAPI 文档: %w", err)
	}
	err = baseCase.InitializeOpenAPI(ctx.Context(), registry.Documents())
	if err != nil {
		return OpenAPIReady{}, fmt.Errorf("初始化 OpenAPI 接口数据: %w", err)
	}
	return OpenAPIReady{}, nil
}
