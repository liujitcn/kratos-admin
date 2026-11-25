package ai

import (
	"context"
	"encoding/json"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// TestShortcutGraphStopsAndResumes 验证确认后的分支使用前一次工具结果，无需重新调用工具。
func TestShortcutGraphStopsAndResumes(t *testing.T) {
	graph := &adminv1.AiShortcutGraph{
		Nodes: []*adminv1.AiShortcutNode{
			{Id: "start", Type: "START", Step: "begin"},
			{Id: "tool", Type: "TOOL", Step: "fetch", ToolName: "read_count"},
			{Id: "confirm", Type: "CONFIRM", Step: "review", PayloadJson: `{"text":"继续"}`},
			{Id: "branch", Type: "BRANCH", Step: "check"},
			{Id: "matched", Type: "CONFIRM", Step: "matched", PayloadJson: `{"text":"匹配成功"}`},
			{Id: "end", Type: "END", Step: "finish"},
		},
		Edges: []*adminv1.AiShortcutEdge{
			{From: "start", To: "tool"}, {From: "tool", To: "confirm"}, {From: "confirm", To: "branch"},
			{From: "branch", To: "matched", Condition: "total == 1"}, {From: "branch", To: "end", Condition: "default"},
			{From: "matched", To: "end"},
		},
	}
	compiled, err := CompileShortcutGraph(graph)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	var tool agent.Tool
	tool, err = agent.InferTool("read_count", "读取数量", func(_ context.Context, _ struct{}) (map[string]int, error) {
		calls++
		return map[string]int{"total": 1}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{Runtime: agent.NewRuntimeWithTools(nil, tool)}
	const flow = "ai-shortcut/42/3"
	input := RuntimeInput{Terminal: "admin"}
	var reply *Response
	reply, err = r.RunShortcutGraph(t.Context(), compiled, input, &basev1.AiAction{Flow: flow, Type: "shortcut_start", Step: "start"}, "")
	if err != nil || reply.Step != "review" || calls != 1 {
		t.Fatalf("流程未停在确认节点: %v %v", reply, err)
	}
	var blocks []struct {
		Action struct {
			Flow string `json:"flow"`
			Step string `json:"step"`
			Type string `json:"type"`
		} `json:"action"`
	}
	if err = json.Unmarshal([]byte(reply.BlocksJSON), &blocks); err != nil || len(blocks) != 1 || blocks[0].Action.Flow != flow {
		t.Fatalf("缺少绑定修订号的继续动作: %v %v", blocks, err)
	}
	action := blocks[0].Action
	saved := ParseOutputContent(MarshalReplyContent(reply))
	if saved.FlowResult == nil {
		t.Fatal("消息没有保存确认前的工具结果")
	}
	reply, err = r.RunShortcutGraph(t.Context(), compiled, input, &basev1.AiAction{Flow: action.Flow, Type: action.Type, Step: action.Step}, *saved.FlowResult)
	if err != nil || reply.Step != "matched" || reply.Content != "匹配成功" || calls != 1 {
		t.Fatalf("恢复流程未按已保存的结果继续或重复调用了工具: %v %v", reply, err)
	}
	graph.Edges[2].To = "confirm"
	if _, err = CompileShortcutGraph(graph); err == nil {
		t.Fatal("自环流程未被拒绝")
	}
}
