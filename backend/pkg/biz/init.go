package biz

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/oauth"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
)

// ProviderSet 汇总业务层依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewBaseAPICase,
	NewBaseTenantCase,
	NewCasbinRuleCase,
	NewBaseCase,
	oss.NewOSS,
	gorm.NewGormClient,
	queue.NewQueue,
	cache.NewCache,
	pprof.NewPprof,
	oauth.NewManager,
)
