package i18n

import (
	"errors"
	"testing"
)

// TestEncodeMessage 编码稳定消息键及可由前端解析的参数。
func TestEncodeMessage(t *testing.T) {
	message := EncodeMessage("system.backup.result.backup_completed", map[string]string{
		"Count": "3",
		"Name":  "main db",
	})
	if message != "__I18N__:system.backup.result.backup_completed?Count=3&Name=main+db" {
		t.Fatalf("EncodeMessage() = %q", message)
	}
}

// TestWrapMessageError 保留稳定消息标记并支持沿错误链检查 cause。
func TestWrapMessageError(t *testing.T) {
	cause := errors.New("database unavailable")
	err := WrapMessageError("system.backup.error.backup_task_failed", cause)
	if !errors.Is(err, cause) {
		t.Fatal("WrapMessageError() lost the cause")
	}
	if want := "__I18N__:system.backup.error.backup_task_failed\ndatabase unavailable"; err.Error() != want {
		t.Fatalf("WrapMessageError() = %q, want %q", err, want)
	}
}
