package plugin

import (
	"errors"
	"testing"
)

// TestIsAuthenticationRejected 只将远端认证拒绝识别为认证问题。
func TestIsAuthenticationRejected(t *testing.T) {
	for _, err := range []error{errors.New("Unauthorized"), errors.New("forbidden")} {
		if !IsAuthenticationRejected(err) {
			t.Fatalf("应识别认证拒绝: %v", err)
		}
	}
	if IsAuthenticationRejected(errors.New("connection refused")) {
		t.Fatal("网络连接错误不应被识别为认证拒绝")
	}
}
