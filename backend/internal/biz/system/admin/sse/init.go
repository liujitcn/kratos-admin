package sse

import (
	"github.com/google/wire"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
)

// ProviderSet 提供 Admin SSE 流定义和生命周期管理器。
var ProviderSet = wire.NewSet(
	logstream.DefaultHub,
	biz.ProviderSet,
	codegen.ProviderSet,
	NewCodegen,
	NewNotification,
	NewOpsMonitoring,
	NewRuntimeConsole,
	NewStreams,
)
