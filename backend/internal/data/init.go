package data

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
)

// ProviderSet 汇总 Admin 数据访问依赖注入提供者和查询接口绑定。
var ProviderSet = wire.NewSet(
	data.ProviderSet,
	NewMessageDeliveryWriter,
	wire.Bind(new(data.QueryProvider), new(*data.Data)),
)
