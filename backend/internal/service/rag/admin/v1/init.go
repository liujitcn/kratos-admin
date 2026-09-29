package admin

import "github.com/google/wire"

// ProviderSet 汇总 rag 管理端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAiKnowledgeService,
	NewAiKnowledgeDocService,
	NewAiKnowledgeChunkService,
)
