package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-core/errorsx"
)

// TestToolPermissionFailurePreservesReason 验证工具只返回权限原因和通用说明。
func TestToolPermissionFailurePreservesReason(t *testing.T) {
	handler := NewToolMetricsHandler(nil, nil)
	wrapped, err := handler.WrapInvokableToolCall(t.Context(), func(context.Context, string, ...tool.Option) (string, error) {
		return "", fmt.Errorf("工具包装: %w", errorsx.PermissionDenied("内部授权详情"))
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var output string
	output, err = wrapped(t.Context(), "{}")
	if err != nil {
		t.Fatal(err)
	}
	var failure map[string]any
	if err = json.Unmarshal([]byte(output), &failure); err != nil {
		t.Fatal(err)
	}
	if failure["reason"] != errorsx.ReasonPermissionDenied || len(failure) != 2 || strings.Contains(output, "内部授权详情") {
		t.Fatalf("权限错误输出未保留结构化原因或泄漏内部详情: %s", output)
	}
}

// TestToolTimeoutStopsExecution 验证外部结果未知的超时不会伪装为成功再交给模型重试。
func TestToolTimeoutStopsExecution(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		cause error
	}{
		{name: "timeout", cause: context.DeadlineExceeded},
		{name: "remote transport failure", cause: errorsx.Internal("远端调用失败").WithCause(ErrToolOutcomeUnknown)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx := callback.WithRecorder(t.Context(), &callback.Recorder{})
			handler := NewToolMetricsHandler(nil, nil)
			wrapped, err := handler.WrapInvokableToolCall(ctx, func(context.Context, string, ...tool.Option) (string, error) { return "", scenario.cause }, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = wrapped(ctx, "{}")
			if !errors.Is(err, ErrToolOutcomeUnknown) || !errors.Is(err, scenario.cause) {
				t.Fatalf("工具调用结果未知时仍允许模型继续推断和重试: %v", err)
			}
		})
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
