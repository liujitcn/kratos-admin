package loginpolicy

import (
	"testing"
	"time"
)

func TestEvaluate(t *testing.T) {
	now := time.Date(2026, 8, 29, 23, 30, 0, 0, time.UTC)
	policy := Policy{
		Enabled:         true,
		IPWhitelist:     []string{"10.0.0.0/8"},
		TimeWindows:     []string{"22:00-06:00"},
		DeviceWhitelist: []string{"Chrome"},
	}
	blocked, reason := policy.Evaluate("192.0.2.1", "Chrome", now)
	if !blocked || reason == "" {
		t.Fatal("expected non-whitelisted IP to be blocked")
	}
	policy.IPWhitelist = nil
	blocked, reason = policy.Evaluate("10.0.0.1", "Chrome", now)
	if !blocked || reason != "当前时间不允许登录" {
		t.Fatalf("expected time window block, got blocked=%v reason=%q", blocked, reason)
	}
	blocked, reason = (Policy{Enabled: true, DeviceBlacklist: []string{"Chrome"}}).Evaluate("10.0.0.1", "Chrome", time.Time{})
	if !blocked || reason != "登录设备命中黑名单" {
		t.Fatal("expected device blacklist block")
	}
}

func TestEvaluateForTargetedRules(t *testing.T) {
	policy := Policy{
		Rules: []Rule{
			{TargetType: "TENANT", TargetValue: "tenant-a", Enabled: true, IPBlacklist: []string{"192.0.2.0/24"}},
			{TargetType: "USER", TargetValue: "alice", Enabled: true, DeviceBlacklist: []string{"mobile"}},
		},
	}
	if blocked, _ := policy.EvaluateFor("tenant-a", "bob", "192.0.2.5", "Chrome", time.Now()); !blocked {
		t.Fatal("expected tenant-targeted rule to block")
	}
	if blocked, _ := policy.EvaluateFor("tenant-b", "alice", "198.51.100.10", "Mobile Safari", time.Now()); !blocked {
		t.Fatal("expected user-targeted rule to block")
	}
	if blocked, _ := policy.EvaluateFor("tenant-b", "bob", "198.51.100.10", "Chrome", time.Now()); blocked {
		t.Fatal("unexpected block for non-matching target")
	}
}

func TestValidateTargetedRules(t *testing.T) {
	if err := (Policy{Rules: []Rule{{TargetType: "GROUP", TargetValue: "ops", Enabled: true}}}).Validate(); err == nil {
		t.Fatal("expected invalid targeted rule type")
	}
	if err := (Policy{Rules: []Rule{{TargetType: "USER", TargetValue: "alice", Enabled: true, IPWhitelist: []string{"not-an-ip"}}}}).Validate(); err == nil {
		t.Fatal("expected invalid targeted rule IP")
	}
	if err := (Policy{Rules: []Rule{
		{TargetType: "USER", TargetValue: "alice", Enabled: true},
		{TargetType: "USER", TargetValue: "alice", Enabled: false},
	}}).Validate(); err == nil {
		t.Fatal("expected duplicate targeted rule to be rejected")
	}
}
