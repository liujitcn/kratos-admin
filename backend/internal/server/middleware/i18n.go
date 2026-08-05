package middleware

import (
	"fmt"

	coreI18n "github.com/liujitcn/kratos-admin/backend/core/pkg/i18n"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n/locales"
)

var defaultCatalog = mustLoadCatalog()

// mustLoadCatalog 加载 Admin 内嵌错误目录，目录损坏属于启动期不可恢复错误。
func mustLoadCatalog() *coreI18n.Catalog {
	catalog, err := coreI18n.NewCatalog(locales.Files, coreLocale.Supported(), coreLocale.Default)
	if err != nil {
		panic(fmt.Errorf("加载 Admin 国际化目录: %w", err))
	}
	return catalog
}
