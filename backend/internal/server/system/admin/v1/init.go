package admin

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/log"
)

// ProviderSet 汇总 system.admin.v1 服务注册依赖注入提供者。
var ProviderSet = wire.NewSet(
	logmiddleware.NewMiddleware,
	wire.Struct(new(Services), "*"),
)
