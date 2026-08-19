package module

import (
	"fmt"
	"io/fs"

	"github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/docs"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-admin/backend/internal/openapi"
	"github.com/liujitcn/kratos-admin/backend/migration"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

type resource struct {
	projectKey  string
	projectName string
	models      module.Models
	docs        fs.FS
	i18n        fs.FS
	openAPI     fs.FS
	migrations  module.Migrations
}

// ProjectKey 返回 Admin 项目的稳定标识。
func (r *resource) ProjectKey() string {
	return r.projectKey
}

// ProjectName 返回 Admin 项目的展示名称。
func (r *resource) ProjectName() string {
	return r.projectName
}

// Models 返回 Admin 数据库自动迁移所需的模型。
func (r *resource) Models() module.Models {
	return r.models
}

// Docs 返回 Admin 项目文档文件系统。
func (r *resource) Docs() fs.FS {
	return r.docs
}

// I18n 返回 Admin 项目语言文件系统。
func (r *resource) I18n() fs.FS {
	return r.i18n
}

// OpenAPI 返回 Admin OpenAPI 文件系统。
func (r *resource) OpenAPI() fs.FS {
	return r.openAPI
}

// Migrations 返回 Admin 数据库迁移资源。
func (r *resource) Migrations() module.Migrations {
	return r.migrations
}

var _ module.Resource = (*resource)(nil)

// NewModuleResources 创建 Admin 在业务对象构建前提供给 Core 的静态资源。
func NewModuleResources() module.Resources {
	docsFS, err := fs.Sub(docs.DocsFS, "assets")
	if err != nil {
		panic(fmt.Errorf("读取内嵌项目文档资源: %w", err))
	}
	return module.Resources{
		&resource{
			projectKey:  _const.Project,
			projectName: _const.Name,
			models:      module.Models{gorm.DefaultClientName: data.Models()},
			openAPI:     openapi.Assets(),
			docs:        docsFS,
			migrations: module.Migrations{
				{Name: migration.ModuleName, FS: migration.Assets(), Path: "."},
			},
			i18n: i18n.Assets(),
		},
	}
}
