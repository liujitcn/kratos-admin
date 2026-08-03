package base

import "github.com/google/wire"

// ProviderSet 汇总基础服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAiSessionService,
	NewAiToolService,
	NewAiMessageService,
	NewConfigService,
	NewLanguageService,
	NewFileService,
	NewLoginService,
	NewMcpService,
	NewOauthService,
	NewSseService,
)
