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

// TestParseRateLimitParamsValidatesAlgorithmSpecificShapes 验证各算法只接受对应参数结构。
func TestParseRateLimitParamsValidatesAlgorithmSpecificShapes(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		params    string
		valid     bool
	}{
		{name: "令牌桶", algorithm: "TOKEN_BUCKET", params: `{"tokens_per_second":10,"burst":20}`, valid: true},
		{name: "固定窗口", algorithm: "FIXED_WINDOW", params: `{"limit":100,"window_seconds":60}`, valid: true},
		{name: "滑动窗口计数", algorithm: "SLIDING_WINDOW_COUNTER", params: `{"limit":100,"window_seconds":60}`, valid: true},
		{name: "滑动窗口日志", algorithm: "SLIDING_WINDOW_LOG", params: `{"limit":100,"window_seconds":60}`, valid: true},
		{name: "漏桶", algorithm: "LEAKY_BUCKET", params: `{"leak_rate_per_second":10,"capacity":20}`, valid: true},
		{name: "算法参数不匹配", algorithm: "FIXED_WINDOW", params: `{"tokens_per_second":10,"burst":20}`, valid: false},
		{name: "未知参数", algorithm: "LEAKY_BUCKET", params: `{"leak_rate_per_second":10,"capacity":20,"burst":5}`, valid: false},
		{name: "滑动窗口日志上限", algorithm: "SLIDING_WINDOW_LOG", params: `{"limit":10001,"window_seconds":60}`, valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params, err := ParseRateLimitParams(test.algorithm, test.params)
			if test.valid && err != nil {
				t.Fatalf("ParseRateLimitParams() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatalf("ParseRateLimitParams() params = %#v, want validation error", params)
			}
		})
	}
}
