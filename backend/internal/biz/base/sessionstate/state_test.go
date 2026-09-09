package sessionstate

import (
	"errors"
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/cache/memory"
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

// TestIndependentSessionLifetime 验证新设备登录和活动均不会延长旧设备的空闲与绝对生命周期。
func TestIndependentSessionLifetime(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	startedAt := time.Now()
	_, err = Start(store, "first", "192.0.2.1", "first device", startedAt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Start(store, "second", "192.0.2.2", "second device", startedAt.Add(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Touch(store, "second", startedAt.Add(29*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Validate(store, "first", startedAt.Add(30*time.Minute))
	if !errors.Is(err, ErrIdleExpired) {
		t.Fatalf("第二台设备活动后第一台仍应空闲超时: %v", err)
	}
	var first State
	first, err = Read(store, "first")
	if err != nil {
		t.Fatal(err)
	}
	first.LastActiveAt = startedAt.Add(12 * time.Hour)
	if err = Evaluate(first, startedAt.Add(12*time.Hour), PolicyFromConfig()); !errors.Is(err, ErrMaxLifetimeExpired) {
		t.Fatalf("新设备登录不能重置旧设备绝对生命周期: %v", err)
	}
	if err = Clear(store, "first"); err != nil {
		t.Fatal(err)
	}
	_, err = Validate(store, "second", startedAt.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("撤销第一台设备不应影响第二台: %v", err)
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
