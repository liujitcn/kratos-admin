package biz

import (
	"context"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	corebiz "github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

func TestLoginFailureKeyTenantAndClientIsolation(t *testing.T) {
	key := loginFailureKey("tenant-a", "alice", "192.0.2.1")
	if key != loginFailureKey("tenant-a", "alice", "192.0.2.1") {
		t.Fatal("login failure key is not deterministic")
	}
	if key == loginFailureKey("tenant-b", "alice", "192.0.2.1") {
		t.Fatal("login failure key is not tenant isolated")
	}
	if key == loginFailureKey("tenant-a", "alice", "192.0.2.2") {
		t.Fatal("login failure key is not client isolated")
	}
}

func TestLoginFailurePolicyLocksAndClears(t *testing.T) {
	cache, _, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	loginCase := &LoginCase{BaseCase: &corebiz.BaseCase{Cache: cache}}
	ctx := context.Background()
	maxAttempts := int(loginpolicy.DefaultMaxFailedAttempts)
	policySet := loginpolicy.PolicySet{Policies: []loginpolicy.Policy{{
		ScopeType:           loginpolicy.ScopeGlobal,
		Status:              loginpolicy.StatusEnable,
		MaxFailedAttempts:   int32(maxAttempts),
		LockDurationMinutes: loginpolicy.DefaultLockDurationMinutes,
	}}}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = loginCase.recordLoginFailure(ctx, "tenant-a", "alice", policySet, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice", policySet, 0, 0); err == nil {
		t.Fatal("expected login policy to lock the account")
	}
	if err = loginCase.clearLoginFailures(ctx, "tenant-a", "alice"); err != nil {
		t.Fatal(err)
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice", policySet, 0, 0); err != nil {
		t.Fatalf("expected cleared account to be available: %v", err)
	}
}

func TestLoginFailurePolicySkipsWithoutEnabledPolicy(t *testing.T) {
	cache, _, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	loginCase := &LoginCase{BaseCase: &corebiz.BaseCase{Cache: cache}}
	ctx := context.Background()
	policySet := loginpolicy.PolicySet{Policies: []loginpolicy.Policy{{
		ScopeType:           loginpolicy.ScopeGlobal,
		Status:              loginpolicy.StatusDisable,
		MaxFailedAttempts:   5,
		LockDurationMinutes: 15,
		PasswordMaxAgeDays:  90,
	}}}
	for attempt := 0; attempt < 10; attempt++ {
		if err = loginCase.recordLoginFailure(ctx, "tenant-a", "alice", policySet, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice", policySet, 0, 0); err != nil {
		t.Fatalf("disabled policy should not lock login: %v", err)
	}
}

// TestRequiresServerSession 验证应用端角色不受后台会话策略影响。
func TestRequiresServerSession(t *testing.T) {
	if requiresServerSession(_const.BASE_ROLE_CODE_USER) || requiresServerSession(_const.BASE_ROLE_CODE_AUTHUSER) {
		t.Fatal("应用端角色不应启用后台会话策略")
	}
	if !requiresServerSession("admin") {
		t.Fatal("管理角色应启用后台会话策略")
	}
}
