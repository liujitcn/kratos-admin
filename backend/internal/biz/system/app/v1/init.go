package biz

import "github.com/google/wire"

// ProviderSet 汇总 system.app.v1 业务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAuthCase,
	NewBaseAreaCase,
	NewBaseDeptCase,
	NewBaseDictCase,
	NewBaseDictItemCase,
	NewBaseMenuCase,
	NewBaseRoleCase,
	NewBaseUserCase,
)
