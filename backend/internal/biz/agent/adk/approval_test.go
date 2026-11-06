package adk

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

// approvalModel 用固定输出验证实际 Eino 中断与恢复链路。
type approvalModel struct {
	replies []*schema.AgenticMessage
	index   int
}

// Generate 按顺序返回工具调用和最终文本。
func (m *approvalModel) Generate(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.AgenticMessage, error) {
	if m.index >= len(m.replies) {
		return nil, fmt.Errorf("unexpected model retry")
	}
	reply := m.replies[m.index]
	m.index++
	return reply, nil
}

// Stream 将固定输出转成单消息流。
func (m *approvalModel) Stream(ctx context.Context, input []*schema.AgenticMessage, options ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	reply, err := m.Generate(ctx, input, options...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{reply}), nil
}

// approvalStore 保存真实 Eino 序列化后的 checkpoint。
type approvalStore struct {
	values sync.Map
	fail   bool
}

// Get 读取快照。
func (s *approvalStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := s.values.Load(key)
	if !ok {
		return nil, false, nil
	}
	return value.([]byte), true, nil
}

// Set 模拟快照写入或持久化失败。
func (s *approvalStore) Set(_ context.Context, key string, value []byte) error {
	if s.fail {
		return errors.New("checkpoint unavailable")
	}
	s.values.Store(key, value)
	return nil
}

// TestRunnerApproval 验证多调用完整暂停、批准、拒绝、参数篡改和权限撤销。
func TestRunnerApproval(t *testing.T) {
	for _, scenario := range []string{"approve", "reject", "tamper", "revoke", "checkpoint_error", "second_approval", "large_media", "oversized_input"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			var executions atomic.Int32
			value, err := utils.InferTool("write", "业务写入", func(context.Context, struct{}) (string, error) {
				executions.Add(1)
				return "ok", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			revoked := false
			guarded := tool.NewGuardedTool(value, func(_ context.Context, args string) (string, bool, error) {
				if revoked {
					return "", false, errors.New("permission revoked")
				}
				return args, true, nil
			}, true)
			var info *schema.ToolInfo
			info, err = value.Info(ctx)
			if err != nil {
				t.Fatal(err)
			}
			calls := &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.FunctionToolCall{CallID: "a", Name: "write", Arguments: "{}"}),
				schema.NewContentBlock(&schema.FunctionToolCall{CallID: "b", Name: "write", Arguments: "{}"}),
			}}
			fake := &approvalModel{replies: []*schema.AgenticMessage{calls, message.AIText("done")}}
			if scenario == "second_approval" {
				fake.replies = []*schema.AgenticMessage{calls, calls, message.AIText("done")}
			}
			store := &approvalStore{fail: scenario == "checkpoint_error"}
			config := Config{Name: "test", Model: fake, CheckPointStore: store}
			request := Request{CheckPointID: "run", Messages: []*schema.AgenticMessage{message.UserText("write")}, Tools: []tool.Base{guarded}, ToolInfos: []*schema.ToolInfo{info}}
			if scenario == "large_media" || scenario == "oversized_input" {
				size := 20 * 1024 * 1024
				if scenario == "oversized_input" {
					size += 2 * 1024 * 1024
				}
				media := make([]byte, size)
				if _, err = rand.Read(media); err != nil {
					t.Fatal(err)
				}
				request.Messages = []*schema.AgenticMessage{message.UserTextWithMedia("write", nil, nil, []message.FileData{{Bytes: media, MIMEType: "application/pdf", Name: "report.pdf"}})}
			}
			var pending *Result
			pending, err = NewRunner(config).Run(ctx, request)
			if scenario == "oversized_input" {
				if err == nil || fake.index != 0 || executions.Load() != 0 {
					t.Fatalf("超出持久化输入限额应在模型执行前拒绝: model_calls=%d executions=%d err=%v", fake.index, executions.Load(), err)
				}
				return
			}
			if scenario == "checkpoint_error" {
				if err == nil || executions.Load() != 0 {
					t.Fatalf("快照失败不应允许确认或执行: executions=%d err=%v", executions.Load(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(pending.Pending) != 2 || executions.Load() != 0 {
				t.Fatalf("未完整中断: pending=%d executions=%d", len(pending.Pending), executions.Load())
			}
			if scenario == "large_media" {
				var data []byte
				data, _, err = store.Get(ctx, "run")
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("20 MiB 附件的 Eino checkpoint: %d bytes", len(data))
			}
			request.Decisions = make(map[string]any)
			for _, call := range pending.Pending {
				decision := tool.ApprovalDecision{ApprovalInfo: call.ApprovalInfo, Approved: scenario != "reject"}
				if scenario == "tamper" {
					decision.Arguments = `{"changed":true}`
				}
				request.Decisions[call.InterruptID] = decision
			}
			revoked = scenario == "revoke"
			var resumed *Result
			resumed, err = NewRunner(config).Run(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			want := int32(0)
			if scenario == "approve" || scenario == "second_approval" || scenario == "large_media" {
				want = 2
			}
			if executions.Load() != want {
				t.Fatalf("错误执行次数: got=%d want=%d", executions.Load(), want)
			}
			if scenario == "second_approval" && len(resumed.Pending) != 2 {
				t.Fatalf("后续新调用未再次等待确认: %+v", resumed)
			}
		})
	}
}
