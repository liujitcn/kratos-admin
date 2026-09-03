package sessionstate

import (
	"errors"
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// TestEvaluate 验证空闲超时与绝对生命周期边界。
func TestEvaluate(t *testing.T) {
	startedAt := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	state := State{CreatedAt: startedAt, LastActiveAt: startedAt.Add(20 * time.Minute), TokenIssuedAt: startedAt}
	policy := Policy{IdleTimeout: 30 * time.Minute, MaxLifetime: 12 * time.Hour}
	if err := Evaluate(state, startedAt.Add(49*time.Minute), policy); err != nil {
		t.Fatalf("有效会话不应过期: %v", err)
	}
	if err := Evaluate(state, startedAt.Add(50*time.Minute), policy); !errors.Is(err, ErrIdleExpired) {
		t.Fatalf("预期空闲超时，实际错误: %v", err)
	}
	state.LastActiveAt = startedAt.Add(11 * time.Hour)
	if err := Evaluate(state, startedAt.Add(12*time.Hour), policy); !errors.Is(err, ErrMaxLifetimeExpired) {
		t.Fatalf("预期绝对生命周期超时，实际错误: %v", err)
	}
}

// TestPolicyFromSessionConfig 验证引导配置中的会话时长会转换为运行策略。
func TestPolicyFromSessionConfig(t *testing.T) {
	policy := policyFromSessionConfig(&configv1.Authentication_Session{
		IdleTimeout: durationpb.New(45 * time.Minute),
		MaxLifetime: durationpb.New(24 * time.Hour),
	})
	if policy.IdleTimeout != 45*time.Minute || policy.MaxLifetime != 24*time.Hour {
		t.Fatalf("unexpected session policy: %+v", policy)
	}
}
