package codegen

import "github.com/google/wire"

// ProviderSet 汇总代码生成进度管理依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewManager,
)
