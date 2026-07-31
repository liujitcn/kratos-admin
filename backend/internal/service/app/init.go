package app

import (
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/app"

	"github.com/google/wire"
)

// ProviderSet 汇总系统应用端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	biz.NewAuthCase,
	biz.NewBaseAreaCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseMenuCase,
	biz.NewBaseRoleCase,
	biz.NewBaseUserCase,
	NewAuthService,
	NewBaseAreaService,
	NewBaseDictService,
	NewBaseMenuService,
)
