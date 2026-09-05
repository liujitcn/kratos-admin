package kit

import "github.com/google/wire"

// ProviderSet 汇总 kratos-kit 脱敏运行时适配器。
var ProviderSet = wire.NewSet(
	NewStorageValueStore,
	NewRedactRuntime,
)
