package agent

import (
	"context"
	"testing"
)

// TestInvokeToolCarriesAssistantClient 验证业务工具可以沿用本轮对话已解析的模型客户端。
func TestInvokeToolCarriesAssistantClient(t *testing.T) {
	client := &AssistantClient{}
	var actual *AssistantClient
	capture, err := InferTool[struct{}, string]("capture_model", "读取本轮模型", func(ctx context.Context, _ struct{}) (string, error) {
		actual = AssistantClientFromContext(ctx)
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(RuntimeConfig{AdminTools: []Tool{capture}})
	ctx := ContextWithAssistantClient(context.Background(), client)
	if _, err = runtime.InvokeTool(ctx, "admin", "capture_model", "{}"); err != nil {
		t.Fatal(err)
	}
	if actual != client {
		t.Fatal("业务工具丢失了本轮模型客户端")
	}
}

// TestInvokeToolCarriesTerminal 验证直接工具调用能读取运行时传入的真实终端。
func TestInvokeToolCarriesTerminal(t *testing.T) {
	for _, terminal := range []string{"app", "admin"} {
		t.Run(terminal, func(t *testing.T) {
			var actual string
			capture, err := InferTool[struct{}, string]("capture_terminal", "读取调用终端", func(ctx context.Context, _ struct{}) (string, error) {
				actual = TerminalFromContext(ctx)
				return actual, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			runtime := NewRuntime(RuntimeConfig{AdminTools: []Tool{capture}, AppTools: []Tool{capture}})
			if _, err = runtime.InvokeTool(context.Background(), terminal, "capture_terminal", "{}"); err != nil {
				t.Fatal(err)
			}
			if actual != terminal {
				t.Fatalf("工具调用终端 = %q，期望 %q", actual, terminal)
			}
		})
	}
}

// TestResumeCarriesRuntimeContext 验证审批恢复会向下游传递原始终端与模型客户端。
func TestResumeCarriesRuntimeContext(t *testing.T) {
	client := NewAssistantClient(nil)
	client.AgenticModel = &pdfInputModel{}
	store := &resumeContextStore{}
	runtime := NewRuntime(RuntimeConfig{Client: client, CheckPointStore: store})
	_, err := runtime.Resume(context.Background(), "context-run", RunState{Terminal: "admin"}, []ApprovalCall{{InterruptID: "pending"}}, true)
	if err == nil {
		t.Fatal("读取测试快照应返回取消错误")
	}
	if store.ctx == nil || TerminalFromContext(store.ctx) != "admin" || AssistantClientFromContext(store.ctx) != client {
		t.Fatal("审批恢复丢失了原始终端或模型客户端")
	}
}

// resumeContextStore 记录恢复入口传给快照存储的上下文。
type resumeContextStore struct {
	CheckPointStore
	ctx context.Context
}

// Get 记录恢复上下文并终止测试快照读取。
func (s *resumeContextStore) Get(ctx context.Context, _ string) ([]byte, bool, error) {
	s.ctx = ctx
	return nil, false, context.Canceled
}
