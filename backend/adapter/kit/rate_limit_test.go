package kit

import (
	"context"
	"testing"
	"time"

	"github.com/liujitcn/kratos-core/server/middleware/ratelimit"
)

// TestRateLimitPolicyResolverResolveReturnsFirstPolicy 验证同一接口按策略顺序只返回首条启用策略。
func TestRateLimitPolicyResolverResolveReturnsFirstPolicy(t *testing.T) {
	resolver := &RateLimitPolicyResolver{
		policies: map[string][]ratelimit.Policy{
			"/system.admin.v1.TestService/Call": {
				{Dimension: ratelimit.DimensionGlobal, TokensPerSecond: 1, Burst: 1},
				{Dimension: ratelimit.DimensionUser, TokensPerSecond: 2, Burst: 2},
			},
		},
		loadedAt: time.Now(),
	}

	policies, err := resolver.Resolve(context.Background(), "/system.admin.v1.TestService/Call")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(policies) != 1 || policies[0].Dimension != ratelimit.DimensionGlobal {
		t.Fatalf("Resolve() policies = %#v, want only the first policy", policies)
	}
}

// TestRateLimitPolicyResolverResolveReturnsNoPolicy 验证未匹配接口不会返回限流策略。
func TestRateLimitPolicyResolverResolveReturnsNoPolicy(t *testing.T) {
	resolver := &RateLimitPolicyResolver{
		policies: map[string][]ratelimit.Policy{},
		loadedAt: time.Now(),
	}

	policies, err := resolver.Resolve(context.Background(), "/system.admin.v1.TestService/Call")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("Resolve() policies = %#v, want no policy", policies)
	}
}
