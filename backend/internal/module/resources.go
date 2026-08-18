package module

import (
	"testing/fstest"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	adminData "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/docs"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-admin/backend/internal/openapi"
	"github.com/liujitcn/kratos-admin/backend/migration"
	"github.com/liujitcn/kratos-core/module"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// NewModuleResources 创建 Admin 在业务对象构建前提供给 Core 的静态资源。
func NewModuleResources() module.Resources {
	return module.Resources{
		ProjectKey:  _const.Project,
		ProjectName: _const.Name,
		Models:      map[string][]interface{}{databaseGorm.DefaultClientName: adminData.Models()},
		OpenAPI:     openapi.Assets(),
		Docs:        fstest.MapFS{"docs.json": &fstest.MapFile{Data: docs.DocsData}},
		Migrations: []module.Migration{
			{Name: migration.ModuleName, FS: migration.Assets(), Path: "."},
		},
		I18n: i18n.Assets(),
	}
}
