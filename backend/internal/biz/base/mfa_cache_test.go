package biz

import (
	"strings"
	"testing"
)

// TestMfaTicketKeysHashOpaqueValues 验证 MFA 一次性票据不会以原文出现在缓存键中。
func TestMfaTicketKeysHashOpaqueValues(t *testing.T) {
	keys := []string{
		mfaLoginChallengeKey("login-ticket"),
		mfaSetupTicketKey("setup-ticket"),
		mfaDisableChallengeKey("disable-ticket"),
	}
	for _, key := range keys {
		for _, ticket := range []string{"login-ticket", "setup-ticket", "disable-ticket"} {
			if strings.Contains(key, ticket) {
				t.Fatalf("MFA cache key exposes ticket %q: %q", ticket, key)
			}
		}
	}
}
