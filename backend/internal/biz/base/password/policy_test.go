package password

import (
	"testing"
	"time"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

func TestValidateComplexity(t *testing.T) {
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
			err := ValidateComplexity(test.value)
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
	t.Setenv("PASSWORD_MAX_AGE_DAYS", "90")
	store, _, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	hash, err := crypto.Encrypt("Admin123!")
	if err != nil {
		t.Fatal(err)
	}
	if err = RecordHistory(store, 7, hash); err != nil {
		t.Fatal(err)
	}
	if err = CheckHistory(store, 7, "Admin123!"); err == nil {
		t.Fatal("expected recent password to be rejected")
	}
	if err = MarkChanged(store, 7); err != nil {
		t.Fatal(err)
	}
	if IsExpired(store, 7, time.Now().Add(91*24*time.Hour)) != (MaxAgeDays() > 0) {
		t.Fatal("unexpected password expiry state")
	}
}

func TestPersistentHistoryAndExpiry(t *testing.T) {
	t.Setenv("PASSWORD_HISTORY_COUNT", "2")
	first, err := crypto.Encrypt("Admin123!")
	if err != nil {
		t.Fatal(err)
	}
	history, err := AppendHistoryJSON("[]", first)
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckHistoryJSON(history, "Admin123!"); err == nil {
		t.Fatal("expected persistent history to reject reused password")
	}
	changedAt := time.Now().Add(-91 * 24 * time.Hour)
	if !IsExpiredAt(changedAt, time.Now()) {
		t.Fatal("expected persistent password state to expire")
	}
}
