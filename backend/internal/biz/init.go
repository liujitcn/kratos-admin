package biz

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	adminBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	appBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app"
)

// ProviderSet 汇总 Admin 各业务目录的依赖注入提供者。
var ProviderSet = wire.NewSet(
	model.ProviderSet,
	ai.ProviderSet,
	biz.ProviderSet,
	codegen.ProviderSet,
	adminBiz.ProviderSet,
	appBiz.ProviderSet,
)

// MessageProviderSet 汇总站内信专用业务依赖。
var MessageProviderSet = adminBiz.MessageProviderSet
