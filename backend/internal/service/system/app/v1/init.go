package app

import "github.com/google/wire"

// ProviderSet 汇总系统应用端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAuthService,
	NewBaseAreaService,
	NewBaseDictService,
	NewBaseMenuService,
)
