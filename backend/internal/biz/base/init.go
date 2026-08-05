package biz

import "github.com/google/wire"

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
	NewOauthCase,
	NewSseCase,
)
