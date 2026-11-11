package base

import "github.com/google/wire"

// ProviderSet 汇总基础服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAccountService,
	NewAiSessionService,
	NewAiModelService,
	NewAiPluginService,
	NewAiAutomationService,
	NewAiFileService,
	NewAiToolService,
	NewAiMessageService,
	NewAiSearchService,
	NewAiKnowledgeService,
	NewConfigService,
	NewLanguageService,
	NewFileService,
	NewLoginService,
	NewSecretCryptoService,
	NewMfaService,
	NewMcpService,
	NewNotificationService,
	NewOauthService,
	NewOauthClientService,
	NewSseService,
)
