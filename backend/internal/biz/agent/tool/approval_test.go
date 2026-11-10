package tool

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

// TestGuardedToolRequiresApproval 验证未确认和直接调用均不能触发写入副作用。
func TestGuardedToolRequiresApproval(t *testing.T) {
	calls := 0
	value, err := utils.InferTool("write", "写入", func(context.Context, struct{}) (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	prepare := func(_ context.Context, arguments string) (string, bool, error) { return arguments, true, nil }
	guarded := NewGuardedTool(value, prepare, true)
	_, err = guarded.InvokableRun(t.Context(), "{}")
	if err == nil || calls != 0 {
		t.Fatalf("未确认工具产生副作用: calls=%d err=%v", calls, err)
	}
	_, interrupted := compose.IsInterruptRerunError(err)
	if !interrupted {
		t.Fatalf("应当返回 Eino 中断: %v", err)
	}
	guarded = NewGuardedTool(value, prepare, false)
	_, err = guarded.InvokableRun(t.Context(), "{}")
	if err == nil || calls != 0 {
		t.Fatalf("直接调用绕过审批: calls=%d err=%v", calls, err)
	}
}
