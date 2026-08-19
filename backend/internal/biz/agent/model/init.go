package model

import "github.com/google/wire"

// ProviderSet 汇总 AI 模型客户端依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewResponsesClient,
)
