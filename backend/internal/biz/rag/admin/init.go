package biz

import "github.com/google/wire"

// ProviderSet 汇总 rag 管理端业务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAiKnowledgeCase,
	NewAiKnowledgeDocCase,
	NewAiKnowledgeChunkCase,
)
