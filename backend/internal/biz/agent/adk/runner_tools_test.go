package adk

import (
	"context"
	"fmt"
	"sync"
	"testing"

	agenttool "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"

	componentsModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

// singleToolModel 每轮只返回一个明确调用，隔离供应商与真实执行链路。
type singleToolModel struct {
	dynamic   bool
	calls     int
	duplicate bool
	distinct  bool
	conflict  bool
	replay    bool
	emptyArgs string
}

// Generate 先发现再调用一次工具，最后返回结果。
func (m *singleToolModel) Generate(context.Context, []*schema.AgenticMessage, ...componentsModel.Option) (*schema.AgenticMessage, error) {
	m.calls++
	name, id, arguments := "read_counter", "read-1", `{}`
	if m.dynamic && m.calls == 1 {
		name, id, arguments = "tool_search", "search-1", `{"query":"select:read_counter"}`
	} else if !m.replay && m.calls > 1 && (!m.dynamic || m.calls > 2) {
		return message.AIText("完成"), nil
	}
	blocks := []*schema.ContentBlock{schema.NewContentBlock(&schema.FunctionToolCall{CallID: id, Name: name, Arguments: arguments})}
	if m.duplicate {
		secondID, secondArgs := id, arguments
		if m.distinct {
			secondID = id + "-second"
		}
		if m.conflict {
			secondArgs = `{"different":true}`
		}
		if m.emptyArgs == "first" {
			blocks[0].FunctionToolCall.Arguments = ""
		} else if m.emptyArgs == "last" {
			secondArgs = ""
		}
		blocks = append(blocks, schema.NewContentBlock(&schema.FunctionToolCall{CallID: secondID, Name: name, Arguments: secondArgs}))
	}
	return &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: blocks}, nil
}

// Stream 使用完整消息模拟供应商，仍经过真实流式Runner。
func (m *singleToolModel) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...componentsModel.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	value, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{value}), nil
}

// TestRunnerCompletesToolCallPlaceholders 验证同编号同名的空参数占位块不丢失完整参数且只执行一次。
func TestRunnerCompletesToolCallPlaceholders(t *testing.T) {
	for _, position := range []string{"first", "last"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("empty=%s/stream=%t", position, stream), func(t *testing.T) {
				executions := 0
				info := &schema.ToolInfo{Name: "read_counter", Desc: "读取计数"}
				value := utils.NewTool(info, func(context.Context, map[string]any) (string, error) { executions++; return `{"count":42}`, nil })
				recorder := &callback.Recorder{}
				runner := NewRunner(Config{Model: &singleToolModel{duplicate: true, emptyArgs: position}, Name: "test"})
				result, err := runner.Run(t.Context(), Request{Messages: []*message.AgenticMessage{message.UserText("读取")}, Tools: []tool.BaseTool{value}, ToolInfos: []*schema.ToolInfo{info}, Recorder: recorder, Stream: stream, ModelStreaming: stream})
				if err != nil {
					t.Fatal(err)
				}
				if executions != 1 || len(recorder.ToolCalls()) != 1 || message.AITextOnly(result.Message) != "完成" {
					t.Fatalf("占位块合并后执行=%d，记录=%+v，结果=%+v", executions, recorder.ToolCalls(), result.Message)
				}
			})
		}
	}
}

// TestRunnerExecutesAndRecordsEachToolOnce 防止动态工具中间件重包装造成执行或记录重复。
func TestRunnerExecutesAndRecordsEachToolOnce(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		for _, dynamic := range []bool{false, true} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("duplicate=%t/dynamic=%t/stream=%t", duplicate, dynamic, stream), func(t *testing.T) {
					executions := 0
					info := &schema.ToolInfo{Name: "read_counter", Desc: "读取计数"}
					value := utils.NewTool(info, func(context.Context, map[string]any) (string, error) { executions++; return `{"count":42}`, nil })
					recorder := &callback.Recorder{}
					model := &singleToolModel{dynamic: dynamic, duplicate: duplicate}
					runner := NewRunner(Config{Model: model, Name: "test", Description: "测试"})
					result, err := runner.Run(t.Context(), Request{Messages: []*message.AgenticMessage{message.UserText("读取")}, Tools: []tool.BaseTool{value}, ToolInfos: []*schema.ToolInfo{info}, DynamicTools: dynamic, Recorder: recorder, Stream: stream, ModelStreaming: stream})
					if err != nil {
						t.Fatal(err)
					}
					if executions != 1 {
						t.Fatalf("实际执行次数=%d，期望1", executions)
					}
					want := 1
					if dynamic {
						want = 2
					}
					if calls := recorder.ToolCalls(); len(calls) != want {
						t.Fatalf("工具记录=%+v，期望%d条", calls, want)
					}
					if message.AITextOnly(result.Message) != "完成" {
						t.Fatalf("最终回复=%+v", result.Message)
					}
				})
			}
		}
	}
}

// TestRunnerPreservesDistinctCallsAndRejectsConflictingIDs 验证按调用编号处理重复，不按工具名或参数误删合法调用。
func TestRunnerPreservesDistinctCallsAndRejectsConflictingIDs(t *testing.T) {
	for _, test := range []struct {
		name                       string
		distinct, conflict, replay bool
		want                       int
		wantErr                    bool
	}{
		{name: "不同编号允许同参数调用", distinct: true, want: 2},
		{name: "同编号冲突参数停止", conflict: true, wantErr: true},
		{name: "后续轮次重用编号停止", replay: true, want: 1, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			executions := 0
			info := &schema.ToolInfo{Name: "read_counter"}
			value := utils.NewTool(info, func(context.Context, map[string]any) (string, error) { executions++; return "{}", nil })
			runner := NewRunner(Config{Model: &singleToolModel{duplicate: !test.replay, distinct: test.distinct, conflict: test.conflict, replay: test.replay}, Name: "test"})
			_, err := runner.Run(t.Context(), Request{Messages: []*message.AgenticMessage{message.UserText("读取")}, Tools: []tool.BaseTool{value}, ToolInfos: []*schema.ToolInfo{info}})
			if (err != nil) != test.wantErr || executions != test.want {
				t.Fatalf("执行=%d 期望=%d err=%v", executions, test.want, err)
			}
		})
	}
}

// runnerCheckpoint 为完整Runner审批测试保存隔离的内存快照。
type runnerCheckpoint struct{ values sync.Map }

// Get 返回框架快照，不接触业务数据库。
func (s *runnerCheckpoint) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := s.values.Load(key)
	if !ok {
		return nil, false, nil
	}
	return value.([]byte), true, nil
}

// Set 保存框架快照。
func (s *runnerCheckpoint) Set(_ context.Context, key string, value []byte) error {
	s.values.Store(key, value)
	return nil
}

// TestRunnerDuplicateApprovalExecutesOnce 验证重复工具块只形成一个审批，确认只执行一次，拒绝不执行。
func TestRunnerDuplicateApprovalExecutesOnce(t *testing.T) {
	for _, approved := range []bool{false, true} {
		t.Run(fmt.Sprintf("approved=%t", approved), func(t *testing.T) {
			executions := 0
			info := &schema.ToolInfo{Name: "read_counter", Desc: "测试写入"}
			value := utils.NewTool(info, func(context.Context, map[string]any) (string, error) { executions++; return "{}", nil })
			guarded := agenttool.NewGuardedTool(value, func(_ context.Context, args string) (string, bool, error) { return args, true, nil }, true)
			runner := NewRunner(Config{Model: &singleToolModel{duplicate: true}, CheckPointStore: &runnerCheckpoint{}, Name: "test"})
			request := Request{CheckPointID: "approval", Messages: []*message.AgenticMessage{message.UserText("写入")}, Tools: []tool.BaseTool{guarded}, ToolInfos: []*schema.ToolInfo{info}}
			result, err := runner.Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Pending) != 1 || executions != 0 {
				t.Fatalf("审批=%+v 执行次数=%d", result.Pending, executions)
			}
			call := result.Pending[0]
			request.Decisions = map[string]any{call.InterruptID: agenttool.ApprovalDecision{ApprovalInfo: call.ApprovalInfo, Approved: approved}}
			result, err = runner.Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if approved {
				want = 1
			}
			if executions != want || len(result.Pending) != 0 {
				t.Fatalf("恢复后执行=%d 期望%d 审批=%+v", executions, want, result.Pending)
			}
		})
	}
}
