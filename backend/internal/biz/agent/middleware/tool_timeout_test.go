package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

// TestToolTimeoutStopsExecution 验证外部结果未知的超时不会伪装为成功再交给模型重试。
func TestToolTimeoutStopsExecution(t *testing.T) {
	recorder := &callback.Recorder{}
	ctx := callback.WithRecorder(t.Context(), recorder)
	handler := NewToolMetricsHandler(nil, nil)
	wrapped, err := handler.WrapInvokableToolCall(ctx, func(context.Context, string, ...tool.Option) (string, error) { return "", context.DeadlineExceeded }, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped(ctx, "{}")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("工具超时被吞掉，模型仍会继续推断和重试")
	}
}

// TestContextBudgetIncludesToolResult 验证每次模型调用都检查大工具结果，超限前停止。
func TestContextBudgetIncludesToolResult(t *testing.T) {
	handler := NewToolFilterHandler(nil)
	handler.ContextWindow = 4096
	handler.MaxOutputTokens = 1024
	_, _, err := handler.BeforeModelRewriteState(t.Context(), &adk.TypedChatModelAgentState[*schema.AgenticMessage]{Messages: []*schema.AgenticMessage{message.UserText(strings.Repeat("长", 5000))}}, nil)
	if err == nil {
		t.Fatal("超大工具结果或正文绕过上下文预算")
	}
}

// TestContextBudgetUsesCurrentToolDefinitions 验证旧模型上下文不膨胀预算，已加载和服务端延迟定义仍计入预算。
func TestContextBudgetUsesCurrentToolDefinitions(t *testing.T) {
	search := &schema.ToolInfo{Name: "tool_search"}
	large := &schema.ToolInfo{Name: "private_tool", Desc: strings.Repeat("工具定义", 5000)}
	for _, test := range []struct {
		name     string
		active   []*schema.ToolInfo
		deferred []*schema.ToolInfo
		wantErr  bool
	}{
		{name: "未加载", active: []*schema.ToolInfo{search}},
		{name: "已加载", active: []*schema.ToolInfo{search, large}, wantErr: true},
		{name: "服务端延迟定义", active: []*schema.ToolInfo{search}, deferred: []*schema.ToolInfo{large}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewToolFilterHandler([]*schema.ToolInfo{search, large})
			handler.ContextWindow = 8192
			handler.MaxOutputTokens = 1024
			state := &adk.TypedChatModelAgentState[*schema.AgenticMessage]{
				Messages: []*schema.AgenticMessage{message.UserText("你好")}, ToolInfos: test.active, DeferredToolInfos: test.deferred,
			}
			mc := &adk.TypedModelContext[*schema.AgenticMessage]{Tools: []*schema.ToolInfo{search, large}}
			_, _, err := handler.BeforeModelRewriteState(t.Context(), state, mc)
			if (err != nil) != test.wantErr {
				t.Fatalf("预算未按实际发送的工具计算: %v", err)
			}
		})
	}
}
