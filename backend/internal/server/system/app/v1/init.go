package app

import "github.com/google/wire"

// ProviderSet 汇总 system.app.v1 服务注册依赖注入提供者。
var ProviderSet = wire.NewSet(
	wire.Struct(new(Services), "*"),
)
