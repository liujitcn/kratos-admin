package biz

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"

	"github.com/google/wire"
)

// ProviderSet 汇总 base 业务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAiSessionCase,
	NewAiMessageCase,
	NewAiToolCase,
	NewBaseDeptCase,
	NewBaseRoleCase,
	NewBaseThirdAccountCase,
	NewBaseUserCase,
	NewConfigCase,
	NewLanguageCase,
	NewFileCase,
	NewLoginCase,
	NewMcpCase,
	wire.Bind(new(ai.ToolAccessChecker), new(*McpCase)),
	NewOauthCase,
	NewSseCase,
)
