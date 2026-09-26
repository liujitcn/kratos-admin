package kit

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/server/middleware/ratelimit"
	"github.com/liujitcn/kratos-kit/redact"
)

// ProviderSet 汇总 kratos-kit 脱敏运行时适配器。
var ProviderSet = wire.NewSet(
	NewRateLimitPolicyResolver,
	NewRedactPolicyResolver,
	wire.Bind(new(ratelimit.PolicyResolver), new(*RateLimitPolicyResolver)),
	wire.Bind(new(redact.PolicyResolver), new(*RedactPolicyResolver)),
)
