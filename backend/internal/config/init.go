package config

import "github.com/google/wire"

// ProviderSet 汇总 Admin 配置解析依赖注入提供者。
var ProviderSet = wire.NewSet(
	ParseAIModel,
	ParseOAuthManager,
)
