package password

import (
	"testing"
	"time"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

func TestValidateComplexity(t *testing.T) {
	config := loginpolicy.PasswordConfig{MinLength: 8, MinComplexityClasses: 3}
	tests := []struct {
		name    string
		value   string
		invalid bool
	}{
		{name: "strong", value: "Admin123!"},
		{name: "short", value: "Aa1!", invalid: true},
		{name: "two classes", value: "admin123", invalid: true},
		{name: "digits only", value: "12345678", invalid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateComplexity(test.value, config)
			if test.invalid && err == nil {
				t.Fatal("expected password validation error")
			}
			if !test.invalid && err != nil {
				t.Fatalf("unexpected password validation error: %v", err)
			}
		})
	}
}

func TestHistoryAndExpiry(t *testing.T) {
	store, _, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	hash, err := crypto.Encrypt("Admin123!")
	if err != nil {
		t.Fatal(err)
	}
	if err = RecordHistory(store, 7, hash, 3); err != nil {
		t.Fatal(err)
	}
	if err = CheckHistory(store, 7, "Admin123!", 3); err == nil {
		t.Fatal("expected recent password to be rejected")
	}
	if err = MarkChanged(store, 7); err != nil {
		t.Fatal(err)
	}
	if IsExpired(store, 7, time.Now().Add(91*24*time.Hour), 90) != true {
		t.Fatal("unexpected password expiry state")
	}
}

func TestPersistentHistoryAndExpiry(t *testing.T) {
	first, err := crypto.Encrypt("Admin123!")
	if err != nil {
		t.Fatal(err)
	}
	history, err := AppendHistoryJSON("[]", first, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckHistoryJSON(history, "Admin123!", 2); err == nil {
		t.Fatal("expected persistent history to reject reused password")
	}
	changedAt := time.Now().Add(-91 * 24 * time.Hour)
	if !IsExpiredAt(changedAt, time.Now(), 90) {
		t.Fatal("expected persistent password state to expire")
	}
}

func TestCheckHistoryUsesConfiguredCount(t *testing.T) {
	first, err := crypto.Encrypt("FirstAdmin123!")
	if err != nil {
		t.Fatal(err)
	}
	second, err := crypto.Encrypt("SecondAdmin123!")
	if err != nil {
		t.Fatal(err)
	}
	history, err := AppendHistoryJSON("[]", first, 2)
	if err != nil {
		t.Fatal(err)
	}
	history, err = AppendHistoryJSON(history, second, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckHistoryJSON(history, "SecondAdmin123!", 1); err == nil {
		t.Fatal("expected newest password to be rejected")
	}
	if err = CheckHistoryJSON(history, "FirstAdmin123!", 1); err != nil {
		t.Fatalf("older password should be allowed after reducing history count: %v", err)
	}
}

func TestAppendHistoryDisabledKeepsValidJSON(t *testing.T) {
	history, err := AppendHistoryJSON("", "hash", 0)
	if err != nil {
		t.Fatal(err)
	}
	if history != "[]" {
		t.Fatalf("expected empty history JSON, got %q", history)
	}
}

func TestExpiryWithPolicyAge(t *testing.T) {
	now := time.Now()
	changedAt := now.Add(-31 * 24 * time.Hour)
	if !IsExpiredAtWithMaxAge(changedAt, now, 30) {
		t.Fatal("expected password to expire at configured age")
	}
	if IsExpiredAtWithMaxAge(changedAt, now, 0) {
		t.Fatal("zero password age should disable expiry")
	}
}
