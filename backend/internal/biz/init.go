package biz

import (
	"github.com/google/wire"
	agentModel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	baseBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	baseAI "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	adminCodegen "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	appBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app"
)

// ProviderSet 汇总 Admin 各业务目录的依赖注入提供者。
var ProviderSet = wire.NewSet(
	agentModel.ProviderSet,
	baseAI.ProviderSet,
	baseBiz.ProviderSet,
	adminCodegen.ProviderSet,
	adminBiz.ProviderSet,
	appBiz.ProviderSet,
)
