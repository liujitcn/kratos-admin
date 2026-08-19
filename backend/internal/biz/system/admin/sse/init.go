package sse

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
)

// ProviderSet 提供 Admin SSE 流定义和生命周期管理器。
var ProviderSet = wire.NewSet(
	logstream.DefaultHub,
	NewCodegen,
	NewOpsMonitoring,
	NewRuntimeConsole,
	NewStreams,
)
