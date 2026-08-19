package server

import (
	"github.com/google/wire"
	baseServer "github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	adminServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
)

// ProviderSet 汇总 Admin 各协议服务目录的依赖注入提供者。
var ProviderSet = wire.NewSet(
	baseServer.ProviderSet,
	adminServer.ProviderSet,
	appServer.ProviderSet,
)
