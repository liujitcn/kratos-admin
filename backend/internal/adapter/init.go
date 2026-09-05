package adapter

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/adapter/core"
	"github.com/liujitcn/kratos-admin/backend/internal/adapter/kit"
)

// ProviderSet 汇总 Backend 的全部外部适配器。
var ProviderSet = wire.NewSet(
	core.ProviderSet,
	kit.ProviderSet,
)
