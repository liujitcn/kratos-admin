package config

import (
	"github.com/google/wire"
)

// ProviderSet 汇总系统配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	GetAppInfo,
	ParseAIModel,
	ParseOSS,
	ParseData,
	ParseDatabase,
	ParseRedis,
	ParseQueue,
	ParsePprof,
	ParseAuthnJWT,
	ParseOAuth,
)
