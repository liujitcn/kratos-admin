package agent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	componentsModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
)

// TestContextEstimateAndSummarySource 验证中英文预算及历史、模型和工具变化不会错误复用摘要。
func TestContextEstimateAndSummarySource(t *testing.T) {
	if message.EstimateTokens([]*message.AgenticMessage{message.UserText(strings.Repeat("汉", 1000))}, nil) < 1000 {
		t.Fatal("中文不能按四字符一个Token低估")
	}
	history := []Message{{Role: RoleUser, Content: "原问题"}, {Role: RoleAI, Content: "原回答"}}
	summary := &ContextSummary{Count: 2, Fingerprint: historyFingerprint(history), Scope: "model-a/tools-a", Content: "摘要"}
	if !summaryMatches(summary, history, "model-a/tools-a") {
		t.Fatal("相同来源应复用摘要")
	}
	if summaryMatches(summary, history, "model-b/tools-a") || summaryMatches(summary, history, "model-a/tools-b") {
		t.Fatal("模型或工具修订变化后不能复用摘要")
	}
	history[0].Content = "修改后的问题"
	if summaryMatches(summary, history, summary.Scope) || summaryMatches(summary, history[:1], summary.Scope) {
		t.Fatal("编辑或删除历史后不能复用摘要")
	}
}

// summaryTestModel 只返回可计费的确定性摘要，不连接外部模型。
type summaryTestModel struct {
	calls int
	fail  bool
}

// Generate 生成短摘要并携带供应商用量，便于验证缓存与计费。
func (m *summaryTestModel) Generate(_ context.Context, _ []*schema.AgenticMessage, _ ...componentsModel.Option) (*schema.AgenticMessage, error) {
	m.calls++
	if m.fail {
		return nil, errors.New("summary unavailable")
	}
	response := message.AIText("已确认原始目标，后续按原约束执行")
	response.ResponseMeta = &schema.AgenticResponseMeta{TokenUsage: &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}
	return response, nil
}

// Stream 防止上下文准备意外走流式输出，摘要不应作为聊天正文发送。
func (m *summaryTestModel) Stream(context.Context, []*schema.AgenticMessage, ...componentsModel.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	return nil, errors.New("unexpected stream")
}

// TestRollingContextPreservesRecentAndUsage 验证早期摘要、近期原文、缓存、模型切换和失败计费。
func TestRollingContextPreservesRecentAndUsage(t *testing.T) {
	fake := &summaryTestModel{}
	client := &model.AssistantClient{AgenticModel: fake, ContextWindow: 32768, MaxOutputTokens: 1024}
	runtime := &Runtime{}
	input := RuntimeInput{Content: "当前问题", ModelName: "model-a"}
	for i := range 30 {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAI
		}
		input.History = append(input.History, Message{Role: role, Content: fmt.Sprintf("原文%d", i)})
	}
	original := append([]Message(nil), input.History...)
	var cached ContextSummary
	input.SaveContextSummary = func(_ context.Context, value ContextSummary) error { cached = value; return nil }
	prepared, usage, err := runtime.prepareContext(t.Context(), input, client, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.History) != 4 || prepared.History[0].Content != "原文26" || prepared.Content != "当前问题" || !reflect.DeepEqual(input.History, original) {
		t.Fatal("压缩改写了原文、近期对话或当前问题")
	}
	if usage.Total != 15 || cached.Count != 26 || prepared.ContextSummary == nil {
		t.Fatalf("摘要未持久化或消耗丢失: %+v %+v", usage, cached)
	}
	input.ContextSummary = &cached
	_, usage, err = runtime.prepareContext(t.Context(), input, client, nil)
	if err != nil || fake.calls != 1 || usage.Total != 0 {
		t.Fatalf("相同来源未复用摘要: %v %+v", err, usage)
	}
	input.ModelName = "model-b"
	_, _, err = runtime.prepareContext(t.Context(), input, client, nil)
	if err != nil || fake.calls != 2 {
		t.Fatal("模型切换错误复用旧摘要")
	}
	fake.fail = true
	input.ContextSummary = nil
	_, _, err = runtime.prepareContext(t.Context(), input, client, nil)
	if err == nil {
		t.Fatal("摘要失败后静默丢弃历史继续执行")
	}
	fake.fail = false
	input.SaveContextSummary = func(context.Context, ContextSummary) error { return errors.New("storage unavailable") }
	_, usage, err = runtime.prepareContext(t.Context(), input, client, nil)
	if err == nil || usage.Total != 15 {
		t.Fatal("缓存落库失败丢失已发生的模型费用")
	}
	input.Content = strings.Repeat("超", 40000)
	calls := fake.calls
	_, _, err = runtime.prepareContext(t.Context(), input, client, nil)
	if err == nil || fake.calls != calls {
		t.Fatal("当前超大问题仍发出模型请求")
	}
}

// TestApprovalScopeInvalidation 验证配置修订和有效工具权限变化后不能沿用旧审批。
func TestApprovalScopeInvalidation(t *testing.T) {
	runtime := &Runtime{}
	state := RunState{Terminal: "admin", ToolRevision: "v1", ToolNames: []string{"write"}}
	calls := []ApprovalCall{{ApprovalInfo: ApprovalInfo{Name: "write", Arguments: "{}"}, InterruptID: "call-1"}}
	for _, revision := range []string{"v1", "v2"} {
		ctx := ContextWithToolPool(t.Context(), ToolPool{Revision: revision})
		if err := runtime.ValidateApproval(ctx, state, calls); err == nil {
			t.Fatal("权限移除或工具修订后仍能恢复审批")
		}
	}
}

// TestSummaryHonorsExecutionStop 验证摘要请求也遵守自动化停止闸门，不能绕过正式模型请求的边界。
func TestSummaryHonorsExecutionStop(t *testing.T) {
	fake := &summaryTestModel{}
	client := &model.AssistantClient{AgenticModel: fake, ContextWindow: 32768, MaxOutputTokens: 1024}
	runtime := &Runtime{}
	input := RuntimeInput{Content: "当前问题", History: make([]Message, 30)}
	for i := range input.History {
		input.History[i] = Message{Role: RoleUser, Content: "历史原文"}
	}
	stopped := errors.New("automation stopped")
	ctx := ContextWithToolExecutionObserver(t.Context(), ToolExecutionObserver{BeforeModel: func(context.Context) error { return stopped }})
	_, _, err := runtime.prepareContext(ctx, input, client, nil)
	if !errors.Is(err, stopped) || fake.calls != 0 {
		t.Fatalf("停止后仍发出摘要模型请求: calls=%d err=%v", fake.calls, err)
	}
}

// contextBudgetModel 记录实际模型工具定义，并按需触发一次真实动态发现。
type contextBudgetModel struct {
	tools      [][]string
	selectTool string
}

// Generate 先检索指定工具或直接回复，不访问外部模型。
func (m *contextBudgetModel) Generate(_ context.Context, _ []*schema.AgenticMessage, opts ...componentsModel.Option) (*schema.AgenticMessage, error) {
	names := make([]string, 0)
	for _, info := range componentsModel.GetCommonOptions(&componentsModel.Options{}, opts...).Tools {
		names = append(names, info.Name)
	}
	m.tools = append(m.tools, names)
	if len(m.tools) == 1 && m.selectTool != "" {
		return &schema.AgenticMessage{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{schema.NewContentBlock(&schema.FunctionToolCall{
				CallID: "search-1", Name: "tool_search", Arguments: fmt.Sprintf(`{"query":"select:%s"}`, m.selectTool),
			})},
		}, nil
	}
	return message.AIText("你好"), nil
}

// Stream 防止测试在供应商禁用流式时意外绕过非流式预算检查。
func (m *contextBudgetModel) Stream(context.Context, []*schema.AgenticMessage, ...componentsModel.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	return nil, errors.New("unexpected stream")
}

// TestDynamicToolBudgetUsesVisibleDefinitions 验证大授权目录不阻断普通问答，发现后的大定义仍受预算限制。
func TestDynamicToolBudgetUsesVisibleDefinitions(t *testing.T) {
	for _, test := range []struct {
		name       string
		stream     bool
		selectTool string
		wantErr    bool
	}{
		{name: "普通问答"},
		{name: "流式入口", stream: true},
		{name: "加载小工具", selectTool: "private_29"},
		{name: "加载超大工具", selectTool: "private_00", wantErr: true},
		{name: "流式入口加载超大工具", stream: true, selectTool: "private_00", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &contextBudgetModel{selectTool: test.selectTool}
			client := &model.AssistantClient{AgenticModel: fake, ContextWindow: 8192, MaxOutputTokens: 1024, DisableStreaming: true}
			var executions int
			tools := make([]Tool, 0, 30)
			for i := range 30 {
				info := &schema.ToolInfo{Name: fmt.Sprintf("private_%02d", i), Desc: strings.Repeat("工具定义", 5000)}
				if i == 29 {
					info.Desc = "小工具定义"
				}
				tools = append(tools, utils.NewTool(info, func(context.Context, map[string]any) (string, error) {
					executions++
					return "{}", nil
				}))
			}
			runtime := NewRuntimeWithTools(client, tools...)
			input := RuntimeInput{Terminal: "admin", Content: "你好"}
			var response *Response
			var err error
			if test.stream {
				response, err = runtime.RunStream(t.Context(), input, func(string) {})
			} else {
				response, err = runtime.Run(t.Context(), input)
			}
			if !test.wantErr {
				if err != nil || response == nil || response.Content != "你好" {
					t.Fatalf("未加载的工具定义阻断普通问答: response=%+v err=%v", response, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "上下文预算") {
				t.Fatalf("加载超大工具后未在下一次模型请求前拒绝: %v", err)
			}
			wantTools := [][]string{{"tool_search"}}
			if test.selectTool != "" && !test.wantErr {
				wantTools = append(wantTools, []string{"tool_search", test.selectTool})
			}
			if !reflect.DeepEqual(fake.tools, wantTools) || executions != 0 {
				t.Fatalf("实际模型工具或执行次数错误: tools=%v executions=%d", fake.tools, executions)
			}
		})
	}
}

// TestSummaryScopeIncludesDeferredDefinitions 验证未加载工具的有效定义变化也使旧摘要失效。
func TestSummaryScopeIncludesDeferredDefinitions(t *testing.T) {
	fake := &summaryTestModel{}
	client := &model.AssistantClient{AgenticModel: fake, ContextWindow: 32768, MaxOutputTokens: 1024}
	info := &schema.ToolInfo{Name: "private_tool", Desc: "原定义"}
	runtime := NewRuntimeWithTools(client, utils.NewTool(info, func(context.Context, map[string]any) (string, error) { return "{}", nil }))
	input := RuntimeInput{Content: "当前问题", History: make([]Message, 30)}
	for i := range input.History {
		input.History[i] = Message{Role: RoleUser, Content: "历史原文"}
	}
	input.SaveContextSummary = func(_ context.Context, value ContextSummary) error { input.ContextSummary = &value; return nil }
	infos := []*schema.ToolInfo{info}
	_, _, err := runtime.prepareContext(t.Context(), input, client, infos)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runtime.prepareContext(t.Context(), input, client, infos)
	if err != nil || fake.calls != 1 {
		t.Fatalf("相同授权定义未复用摘要: calls=%d err=%v", fake.calls, err)
	}
	info.Desc = "修订后的有效定义"
	_, _, err = runtime.prepareContext(t.Context(), input, client, infos)
	if err != nil || fake.calls != 2 {
		t.Fatalf("未加载工具定义变化后错误复用摘要: calls=%d err=%v", fake.calls, err)
	}
}

// TestContextBudgetReservesDiscoveryOverhead 验证发现提示与搜索定义也能触发较早历史摘要，而非等正式请求时才拒绝。
func TestContextBudgetReservesDiscoveryOverhead(t *testing.T) {
	fake := &summaryTestModel{}
	client := &model.AssistantClient{AgenticModel: fake, ContextWindow: 8192, MaxOutputTokens: 1024}
	info := &schema.ToolInfo{Name: "private_tool", Desc: "只读工具"}
	runtime := NewRuntimeWithTools(client, utils.NewTool(info, func(context.Context, map[string]any) (string, error) { return "{}", nil }))
	input := RuntimeInput{Content: "当前问题", History: make([]Message, 12)}
	for i := range input.History {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAI
		}
		input.History[i] = Message{Role: role, Content: strings.Repeat("history ", 130)}
	}
	prepared, _, err := runtime.prepareContext(t.Context(), input, client, []*schema.ToolInfo{info})
	if err != nil || fake.calls != 1 || prepared.ContextSummary == nil || len(prepared.History) != 4 {
		t.Fatalf("发现开销未触发历史摘要: calls=%d history=%d err=%v", fake.calls, len(prepared.History), err)
	}
}
