package middleware

import (
	"context"
	"testing"
)

// TestToolFailureMessageUsesRequestLocale 验证 ADK 工具错误通过当前请求的本地化器生成。
func TestToolFailureMessageUsesRequestLocale(t *testing.T) {
	localize := func(_ context.Context, key string, args map[string]any, _ string) string {
		if key != "system.ai.chat.error.tool_failed" {
			t.Fatalf("message key = %q", key)
		}
		if args["Name"] != "lookup" {
			t.Fatalf("tool name argument = %v", args["Name"])
		}
		return "localized tool failure"
	}
	if got := toolFailureMessage(context.Background(), "lookup", localize); got != "localized tool failure" {
		t.Fatalf("tool failure message = %q", got)
	}
}
