package ai

import "github.com/google/wire"

// ProviderSet 汇总 Admin AI 运行时依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewRuntime,
)
