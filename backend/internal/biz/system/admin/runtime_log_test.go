package biz

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSanitizeRuntimeLogFile 验证历史日志下载不会返回常见敏感值原文。
func TestSanitizeRuntimeLogFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "info.log")
	content := "phone=13812345678 email=alice@example.com access_token=secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := sanitizeRuntimeLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if got == content || got != "phone=138****5678 email=al***@example.com access_token=\"[REDACTED]\"\n" {
		t.Fatalf("unexpected sanitized log: %q", got)
	}
}
