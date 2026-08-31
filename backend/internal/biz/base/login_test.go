package biz

import (
	"context"
	"testing"

	corebiz "github.com/liujitcn/kratos-core/biz"
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
	for attempt := 0; attempt < loginFailureMaxAttempts; attempt++ {
		if err = loginCase.recordLoginFailure(ctx, "tenant-a", "alice"); err != nil {
			t.Fatal(err)
		}
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice"); err == nil {
		t.Fatal("expected login policy to lock the account")
	}
	if err = loginCase.clearLoginFailures(ctx, "tenant-a", "alice"); err != nil {
		t.Fatal(err)
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice"); err != nil {
		t.Fatalf("expected cleared account to be available: %v", err)
	}
}
