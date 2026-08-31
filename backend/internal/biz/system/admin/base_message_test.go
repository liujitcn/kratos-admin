package biz

import (
	"strings"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

func TestMessageDispatchRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int32
		want    int64
	}{
		{attempt: 0, want: int64(messageDispatchRetryBase)},
		{attempt: 1, want: int64(messageDispatchRetryBase)},
		{attempt: 2, want: int64(2 * time.Minute)},
		{attempt: 4, want: int64(8 * time.Minute)},
		{attempt: 5, want: int64(messageDispatchRetryMaxDelay)},
	}
	for _, test := range tests {
		if got := int64(messageDispatchRetryDelay(test.attempt)); got != test.want {
			t.Errorf("messageDispatchRetryDelay(%d) = %d, want %d", test.attempt, got, test.want)
		}
	}
}

func TestMessageRetentionDaysDefaultsZero(t *testing.T) {
	if got := messageRetentionDays(&models.BaseMessageCategory{}); got != defaultMessageRetentionDays {
		t.Fatalf("messageRetentionDays(0) = %d, want %d", got, defaultMessageRetentionDays)
	}
	if got := messageRetentionDays(&models.BaseMessageCategory{RetentionDays: 365}); got != 365 {
		t.Fatalf("messageRetentionDays(365) = %d, want 365", got)
	}
}

func TestValidateMessageActionParamsBounds(t *testing.T) {
	if err := validateMessageActionParams(`{"target":"ok"}`); err != nil {
		t.Fatalf("valid action params rejected: %v", err)
	}
	if err := validateMessageActionParams(`{"target":"` + strings.Repeat("a", maxMessageActionStringRunes+1) + `"}`); err == nil {
		t.Fatal("oversized action value should be rejected")
	}
}
