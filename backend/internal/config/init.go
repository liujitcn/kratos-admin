package config

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
)

// ProviderSet 汇总 Admin 配置解析依赖注入提供者。
var ProviderSet = wire.NewSet(
	ParseAIModel,
	ParseMfaConfig,
	ParseOAuthManager,
	oauthsecret.NewProtector,
)
