package tool

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"testing"

	"github.com/cloudwego/eino/compose"
)

// approvalCheckpoint 仅在测试中保存框架原始快照，不接触业务数据库。
type approvalCheckpoint map[string][]byte

// Get 读取内存快照。
func (s approvalCheckpoint) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := s[key]
	return value, ok, nil
}

// Set 保存内存快照。
func (s approvalCheckpoint) Set(_ context.Context, key string, value []byte) error {
	s[key] = value
	return nil
}

// approvalSerializer 与ADK内部快照使用相同的Gob序列化，不借用Graph默认的其他注册表。
type approvalSerializer struct{}

// Marshal 编码审批中断快照。
func (approvalSerializer) Marshal(value any) ([]byte, error) {
	var buffer bytes.Buffer
	err := gob.NewEncoder(&buffer).Encode(value)
	return buffer.Bytes(), err
}

// Unmarshal 解码审批中断快照。
func (approvalSerializer) Unmarshal(value []byte, target any) error {
	return gob.NewDecoder(bytes.NewReader(value)).Decode(target)
}

// approvalTestTool 记录实际副作用入口是否被调用。
type approvalTestTool struct{ calls int }

// Info 返回固定工具契约。
func (*approvalTestTool) Info(context.Context) (*Info, error) {
	return &Info{Name: "write", Desc: "写入"}, nil
}

// InvokableRun 记录实际执行次数。
func (t *approvalTestTool) InvokableRun(context.Context, string, ...Option) (string, error) {
	t.calls++
	return "done", nil
}

// TestFrozenApprovalAndExecutionBoundary 验证冻结参数、拒绝、权限变化和停止不会被恢复路径绕过。
func TestFrozenApprovalAndExecutionBoundary(t *testing.T) {
	for _, scenario := range []string{"approved", "rejected", "tampered", "changed_arguments", "permission_revoked", "stopped"} {
		t.Run(scenario, func(t *testing.T) {
			actual := &approvalTestTool{}
			arguments := `{"tenant_id":7,"value":"original"}`
			allowed := true
			guarded := NewGuardedTool(actual, func(context.Context, string) (string, bool, error) {
				if !allowed {
					return "", false, errors.New("permission revoked")
				}
				return arguments, true, nil
			}, true)
			graph := compose.NewGraph[string, string]()
			err := graph.AddLambdaNode("write", compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
				return guarded.InvokableRun(ctx, input)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if err = graph.AddEdge(compose.START, "write"); err != nil {
				t.Fatal(err)
			}
			if err = graph.AddEdge("write", compose.END); err != nil {
				t.Fatal(err)
			}
			var runnable compose.Runnable[string, string]
			runnable, err = graph.Compile(t.Context(), compose.WithCheckPointStore(approvalCheckpoint{}), compose.WithSerializer(approvalSerializer{}))
			if err != nil {
				t.Fatal(err)
			}
			_, err = runnable.Invoke(t.Context(), arguments, compose.WithCheckPointID("approval"))
			interrupted, ok := compose.ExtractInterruptInfo(err)
			if !ok || len(interrupted.InterruptContexts) != 1 || actual.calls != 0 {
				t.Fatalf("未冻结审批或审批前已执行: %v", err)
			}
			decision := ApprovalDecision{ApprovalInfo: interrupted.InterruptContexts[0].Info.(ApprovalInfo), Approved: scenario != "rejected"}
			switch scenario {
			case "tampered":
				decision.Arguments = `{"tenant_id":8}`
			case "changed_arguments":
				arguments = `{"tenant_id":7,"value":"changed"}`
			case "permission_revoked":
				allowed = false
			}
			ctx := compose.ResumeWithData(t.Context(), interrupted.InterruptContexts[0].ID, decision)
			if scenario == "stopped" {
				ctx = ContextWithExecutionObserver(ctx, ExecutionObserver{Before: func(context.Context, string) error { return errors.New("stopped") }})
			}
			_, err = runnable.Invoke(ctx, "", compose.WithCheckPointID("approval"))
			if scenario == "approved" {
				if err != nil || actual.calls != 1 {
					t.Fatalf("原始审批未执行一次: calls=%d err=%v", actual.calls, err)
				}
			} else if actual.calls != 0 || (scenario != "rejected" && err == nil) {
				t.Fatalf("无效审批或停止后仍执行: calls=%d err=%v", actual.calls, err)
			}
		})
	}
}
