package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
)

// TestNormalizeJobLogErrorLocalizesCoreSchedulerFailures 验证调度器绕过任务实现时的失败文案使用稳定国际化键。
func TestNormalizeJobLogErrorLocalizesCoreSchedulerFailures(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		wantKey string
	}{
		{name: "missing target", input: "[]", message: "调用目标不存在", wantKey: "system.base.job.log.error.target_missing"},
		{name: "running elsewhere", input: "[]", message: "任务正在其他实例执行", wantKey: "system.base.job.log.error.running_elsewhere"},
		{name: "panic", input: "[]", message: "任务执行异常: index out of range", wantKey: "system.base.job.log.error.execution_panic"},
		{name: "invalid arguments", input: "[", message: "unexpected end of JSON input", wantKey: "system.base.job.log.error.arguments_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, want := normalizeJobLogError(test.input, test.message), i18n.EncodeMessage(test.wantKey, nil); got != want {
				t.Fatalf("normalized error = %q, want %q", got, want)
			}
		})
	}
}

// TestNormalizeJobLogErrorPreservesEncodedAndUnknownMessages 验证已有翻译标记和有效执行参数的诊断信息不被改写。
func TestNormalizeJobLogErrorPreservesEncodedAndUnknownMessages(t *testing.T) {
	encoded := i18n.EncodeMessage("system.base.job.error.message_dispatch_failed", nil)
	if got := normalizeJobLogError("[]", encoded); got != encoded {
		t.Fatalf("encoded error = %q, want %q", got, encoded)
	}
	const diagnostic = "database connection refused"
	if got := normalizeJobLogError(`[{"key":"source","value":"main"}]`, diagnostic); got != diagnostic {
		t.Fatalf("diagnostic error = %q, want %q", got, diagnostic)
	}
}
