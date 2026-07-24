package job

import "github.com/google/wire"

// ProviderSet 注册定时任务模块依赖。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewCronServer,
)
