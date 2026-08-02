package base

import (
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"

	"github.com/google/wire"
)

// ProviderSet 汇总基础服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	biz.NewAiSessionCase,
	biz.NewAiMessageCase,
	biz.NewAiToolCase,
	biz.NewBaseDeptCase,
	biz.NewBaseRoleCase,
	biz.NewBaseThirdAccountCase,
	biz.NewBaseUserCase,
	biz.NewConfigCase,
	biz.NewLanguageCase,
	biz.NewFileCase,
	biz.NewLoginCase,
	biz.NewMcpCase,
	biz.NewOauthCase,
	biz.NewSseCase,

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
