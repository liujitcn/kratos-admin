package service

import (
	"github.com/liujitcn/kratos-admin/backend/internal/service/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/service/system/app/v1"

	"github.com/google/wire"
)

// ProviderSet 汇总各 Proto 版本目录下的 Service 依赖注入提供者。
var ProviderSet = wire.NewSet(
	base.ProviderSet,
	admin.ProviderSet,
	app.ProviderSet,
)
