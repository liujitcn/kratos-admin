package loginpolicy

import (
	"testing"
	"time"
)

func TestEvaluateForRules(t *testing.T) {
	now := time.Date(2026, 8, 29, 23, 30, 0, 0, time.UTC)
	policy := PolicySet{Policies: []Policy{{
		ScopeType: ScopeGlobal,
		Status:    StatusEnable,
		Rules: []Rule{
			{RestrictionType: RestrictionWhitelist, RestrictionMethod: MethodIP, RestrictionValue: "10.0.0.0/8", Status: StatusEnable},
			{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodTime, RestrictionValue: "22:00-06:00", Status: StatusEnable},
			{RestrictionType: RestrictionWhitelist, RestrictionMethod: MethodDevice, RestrictionValue: "Chrome", Status: StatusEnable},
		},
	}}}
	blocked, reason := policy.EvaluateFor(1, 2, "192.0.2.1", "", "", "Chrome", now)
	if !blocked || reason != "登录 IP 不在白名单" {
		t.Fatalf("expected non-whitelisted IP to be blocked, got blocked=%v reason=%q", blocked, reason)
	}
	blocked, reason = policy.EvaluateFor(1, 2, "10.0.0.1", "", "", "Chrome", now)
	if !blocked || reason != "当前时间不允许登录" {
		t.Fatalf("expected time window block, got blocked=%v reason=%q", blocked, reason)
	}
	blocked, reason = (PolicySet{Policies: []Policy{{
		ScopeType: ScopeGlobal,
		Status:    StatusEnable,
		Rules:     []Rule{{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodDevice, RestrictionValue: "Chrome", Status: StatusEnable}},
	}}}).EvaluateFor(1, 2, "10.0.0.1", "", "", "Chrome", time.Time{})
	if !blocked || reason != "登录设备命中黑名单" {
		t.Fatalf("expected device blacklist block, got blocked=%v reason=%q", blocked, reason)
	}
}

func TestEvaluateForScopes(t *testing.T) {
	policy := PolicySet{Policies: []Policy{
		{ScopeType: ScopeGlobal, Status: StatusEnable, Rules: []Rule{{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodIP, RestrictionValue: "192.0.2.0/24", Status: StatusEnable}}},
		{ScopeType: ScopeTenant, TenantID: 2, Status: StatusEnable, Rules: []Rule{{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodIP, RestrictionValue: "198.51.100.0/24", Status: StatusEnable}}},
		{ScopeType: ScopeUser, TenantID: 2, UserID: 3, Status: StatusEnable, Rules: []Rule{{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodDevice, RestrictionValue: "mobile", Status: StatusEnable}}},
	}}
	if blocked, _ := policy.EvaluateFor(2, 4, "198.51.100.5", "", "", "Chrome", time.Now()); !blocked {
		t.Fatal("expected tenant policy to block")
	}
	if blocked, _ := policy.EvaluateFor(2, 3, "203.0.113.5", "", "", "Mobile Safari", time.Now()); !blocked {
		t.Fatal("expected user policy to block")
	}
	if blocked, _ := policy.EvaluateFor(4, 4, "203.0.113.5", "", "", "Chrome", time.Now()); blocked {
		t.Fatal("unexpected block for non-matching scope")
	}
}

func TestLoadPasswordFallback(t *testing.T) {
	config := Load().PasswordConfigFor(1, 2)
	if config.MinLength != DefaultPasswordMinLength || config.HistoryCount != DefaultPasswordHistoryCount || config.MinComplexityClasses != DefaultPasswordMinComplexityClasses || config.MaxAgeDays != DefaultPasswordMaxAgeDays {
		t.Fatalf("unexpected password fallback: %+v", config)
	}
}

func TestValidateRules(t *testing.T) {
	if err := (PolicySet{Policies: []Policy{{
		ScopeType:           ScopeGlobal,
		Status:              StatusEnable,
		MaxFailedAttempts:   1,
		LockDurationMinutes: 1,
		Rules:               []Rule{{RestrictionType: RestrictionBlacklist, RestrictionMethod: MethodIP, RestrictionValue: "not-an-ip", Status: StatusEnable}},
	}}}).Validate(); err == nil {
		t.Fatal("expected invalid rule IP")
	}
	if err := (PolicySet{Policies: []Policy{
		{ScopeType: ScopeGlobal, Status: StatusEnable, MaxFailedAttempts: 1, LockDurationMinutes: 1},
		{ScopeType: ScopeGlobal, Status: StatusDisable, MaxFailedAttempts: 1, LockDurationMinutes: 1},
	}}).Validate(); err == nil {
		t.Fatal("expected duplicate policy scope")
	}
}

func TestFailureConfig(t *testing.T) {
	policy := PolicySet{Policies: []Policy{
		{ScopeType: ScopeGlobal, Status: StatusEnable, MaxFailedAttempts: 5, LockDurationMinutes: 15},
		{ScopeType: ScopeTenant, TenantID: 2, Status: StatusEnable, MaxFailedAttempts: 3, LockDurationMinutes: 10},
		{ScopeType: ScopeUser, TenantID: 2, UserID: 3, Status: StatusEnable, MaxFailedAttempts: 2, LockDurationMinutes: 6},
	}}
	maxAttempts, window := policy.FailureConfig(2, 3)
	if maxAttempts != 2 || window != 6*time.Minute {
		t.Fatalf("unexpected user failure config: %d/%s", maxAttempts, window)
	}
	maxAttempts, window = policy.FailureConfig(2, 4)
	if maxAttempts != 3 || window != 10*time.Minute {
		t.Fatalf("unexpected tenant failure config: %d/%s", maxAttempts, window)
	}
	maxAttempts, window = (PolicySet{Policies: []Policy{{
		ScopeType:           ScopeGlobal,
		Status:              StatusDisable,
		MaxFailedAttempts:   5,
		LockDurationMinutes: 15,
	}}}).FailureConfig(2, 4)
	if maxAttempts != 0 || window != 0 {
		t.Fatalf("disabled policy should not provide failure config: %d/%s", maxAttempts, window)
	}
}

func TestPasswordMaxAgeDaysFor(t *testing.T) {
	policy := PolicySet{Policies: []Policy{
		{ScopeType: ScopeGlobal, Status: StatusEnable, PasswordMaxAgeDays: 90},
		{ScopeType: ScopeTenant, TenantID: 2, Status: StatusEnable, PasswordMaxAgeDays: 30},
		{ScopeType: ScopeUser, TenantID: 2, UserID: 3, Status: StatusEnable, PasswordMaxAgeDays: 0},
	}}
	if got := policy.PasswordMaxAgeDaysFor(2, 3); got != 0 {
		t.Fatalf("unexpected user password age: %d", got)
	}
	if got := policy.PasswordMaxAgeDaysFor(2, 4); got != 30 {
		t.Fatalf("unexpected tenant password age: %d", got)
	}
	if got := policy.PasswordMaxAgeDaysFor(4, 4); got != 90 {
		t.Fatalf("unexpected global password age: %d", got)
	}
	if got := (PolicySet{Policies: []Policy{{
		ScopeType:          ScopeGlobal,
		Status:             StatusDisable,
		PasswordMaxAgeDays: 90,
	}}}).PasswordMaxAgeDaysFor(4, 4); got != 0 {
		t.Fatalf("disabled policy should not provide password age: %d", got)
	}
}

func TestPasswordConfigFor(t *testing.T) {
	policy := PolicySet{Policies: []Policy{
		{ScopeType: ScopeGlobal, Status: StatusEnable, PasswordMinLength: 10, PasswordHistoryCount: 4, PasswordMinComplexityClasses: 4, PasswordMaxAgeDays: 90, InitialPasswordHash: "global"},
		{ScopeType: ScopeTenant, TenantID: 2, Status: StatusEnable, PasswordMinLength: 12, PasswordHistoryCount: 2, PasswordMinComplexityClasses: 3, PasswordMaxAgeDays: 30, InitialPasswordHash: "tenant"},
		{ScopeType: ScopeUser, TenantID: 2, UserID: 3, Status: StatusEnable, PasswordMinLength: 14, PasswordHistoryCount: 0, PasswordMinComplexityClasses: 2, PasswordMaxAgeDays: 0, InitialPasswordHash: "user"},
	}}
	config := policy.PasswordConfigFor(2, 3)
	if config.MinLength != 14 || config.HistoryCount != 0 || config.MinComplexityClasses != 2 || config.MaxAgeDays != 0 || config.InitialPasswordHash != "user" {
		t.Fatalf("unexpected user password config: %+v", config)
	}
	config = policy.PasswordConfigFor(2, 4)
	if config.MinLength != 12 || config.HistoryCount != 2 || config.MinComplexityClasses != 3 || config.MaxAgeDays != 30 || config.InitialPasswordHash != "tenant" {
		t.Fatalf("unexpected tenant password config: %+v", config)
	}
	config = policy.PasswordConfigFor(4, 4)
	if config.MinLength != 10 || config.HistoryCount != 4 || config.MinComplexityClasses != 4 || config.MaxAgeDays != 90 || config.InitialPasswordHash != "global" {
		t.Fatalf("unexpected global password config: %+v", config)
	}
}
