package sse

import "github.com/google/wire"

// ProviderSet 提供 Admin SSE 流定义和生命周期管理器。
var ProviderSet = wire.NewSet(
	NewCodegen,
	NewOpsMonitoring,
	NewStreams,
)
