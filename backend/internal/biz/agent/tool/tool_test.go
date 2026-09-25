package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestExecuteCallLocalizesDisabledTool verifies disabled-tool output uses the configured message localizer.
func TestExecuteCallLocalizesDisabledTool(t *testing.T) {
	localize := func(_ context.Context, key string, args map[string]any, _ string) string {
		if key != "base.ai.tool.disabled" {
			t.Fatalf("message key = %q", key)
		}
		return "localized:" + args["Name"].(string)
	}
	result := ExecuteCall(context.Background(), map[string]Invokable{}, nil, Call{Name: "disabled_tool"}, WithMessageLocalizer(localize))
	if result.Status != "error" {
		t.Fatalf("status = %q, want error", result.Status)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["error"] != "localized:disabled_tool" || result.Content != result.Output {
		t.Fatalf("unexpected disabled-tool result: output=%q content=%q", result.Output, result.Content)
	}
}

// TestExecuteCallLocalizesToolExecutionFailure 验证工具执行错误按请求语言返回且不透出底层异常内容。
func TestExecuteCallLocalizesToolExecutionFailure(t *testing.T) {
	localize := func(_ context.Context, key string, args map[string]any, _ string) string {
		if key != "system.ai.chat.error.tool_failed" {
			t.Fatalf("message key = %q", key)
		}
		return "localized:" + args["Name"].(string)
	}
	infos := []*Info{{Name: "failing_tool"}}
	result := ExecuteCall(context.Background(), map[string]Invokable{"failing_tool": failingInvokable{}}, infos, Call{Name: "failing_tool"}, WithMessageLocalizer(localize))
	if result.Status != "error" {
		t.Fatalf("status = %q, want error", result.Status)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["error"] != "localized:failing_tool" || result.Content != result.Output {
		t.Fatalf("unexpected tool failure: output=%q content=%q", result.Output, result.Content)
	}
	if strings.Contains(result.Output, "secret-token") {
		t.Fatal("tool output must not include the raw provider error")
	}
}

type failingInvokable struct{}

// Info 返回失败工具的稳定名称。
func (failingInvokable) Info(context.Context) (*Info, error) {
	return &Info{Name: "failing_tool"}, nil
}

// InvokableRun 返回带有敏感片段的模拟底层错误。
func (failingInvokable) InvokableRun(context.Context, string, ...Option) (string, error) {
	return "", errors.New("provider rejected request: secret-token")
}
