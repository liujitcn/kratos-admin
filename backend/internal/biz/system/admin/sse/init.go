package sse

import (
	"github.com/google/wire"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	adminCodegen "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
)

// ProviderSet 提供 Admin SSE 流定义和生命周期管理器。
var ProviderSet = wire.NewSet(
	logstream.DefaultHub,
	adminBiz.ProviderSet,
	adminCodegen.ProviderSet,
	NewCodegen,
	NewOpsMonitoring,
	NewRuntimeConsole,
	NewStreams,
)
