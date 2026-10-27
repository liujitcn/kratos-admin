package workflow

import (
	"encoding/json"
	"testing"
)

// buildGraphJSON 构造节点图配置 JSON 文本。
func buildGraphJSON(t *testing.T, version int, nodes []*Node, edges []*Edge) string {
	t.Helper()
	raw, err := json.Marshal(graphJSON{Version: version, Nodes: nodes, Edges: edges})
	if err != nil {
		t.Fatalf("marshal graph failed: %v", err)
	}
	return string(raw)
}

// linearGraph 构造 start→task→end 的最小合法图。
func linearGraph() ([]*Node, []*Edge) {
	nodes := []*Node{
		{ID: "start", Type: NodeTypeStart},
		{ID: "task1", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypeUser, ID: 2}}, Strategy: StrategyAll},
		{ID: "end", Type: NodeTypeEnd},
	}
	edges := []*Edge{
		{From: "start", To: "task1"},
		{From: "task1", To: "end"},
	}
	return nodes, edges
}

func TestParseAndValidateLinear(t *testing.T) {
	nodes, edges := linearGraph()
	graph, err := Parse(buildGraphJSON(t, graphVersion, nodes, edges))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if err := Validate(graph); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if graph.StartNode().ID != "start" {
		t.Fatalf("start node mismatch: %s", graph.StartNode().ID)
	}
	if graph.Nodes["task1"].InDegree != 1 || len(graph.Nodes["task1"].OutEdges) != 1 {
		t.Fatalf("task1 degree mismatch")
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []struct {
		name       string
		nodeConfig string
	}{
		{"非法JSON", `{`},
		{"版本不支持", buildGraphJSON(t, 1, mustNodesLinear(t), mustEdgesLinear(t))},
		{"节点元素为空", `{"version":2,"nodes":[null],"edges":[]}`},
		{"边元素为空", `{"version":2,"nodes":[{"id":"a","type":"START"},{"id":"b","type":"END"}],"edges":[null]}`},
		{"节点ID重复", `{"version":2,"nodes":[{"id":"a","type":"START"},{"id":"a","type":"END"}],"edges":[]}`},
		{"边引用不存在节点", `{"version":2,"nodes":[{"id":"a","type":"START"},{"id":"b","type":"END"}],"edges":[{"from":"a","to":"c"}]}`},
	}
	for _, c := range cases {
		if _, err := Parse(c.nodeConfig); err == nil {
			t.Fatalf("%s: 期望解析失败", c.name)
		}
	}
}

// mustNodesLinear 返回线性图节点，供需要改写的用例使用。
func mustNodesLinear(t *testing.T) []*Node {
	t.Helper()
	nodes, _ := linearGraph()
	return nodes
}

// mustEdgesLinear 返回线性图边，供需要改写的用例使用。
func mustEdgesLinear(t *testing.T) []*Edge {
	t.Helper()
	_, edges := linearGraph()
	return edges
}

func TestValidateInvalid(t *testing.T) {
	cases := []struct {
		name  string
		build func() ([]*Node, []*Edge)
	}{
		{"开始节点多条出边", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "task2", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypeUser, ID: 3}}})
			edges = append(edges, &Edge{From: "start", To: "task2"}, &Edge{From: "task2", To: "end"})
			return nodes, edges
		}},
		{"缺少结束节点", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			return nodes[:2], edges[:1]
		}},
		{"审批节点缺审批人", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes[1].Approvers = nil
			return nodes, edges
		}},
		{"审批人类型不支持", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes[1].Approvers = []Approver{{Type: "LEADER", ID: 2}}
			return nodes, edges
		}},
		{"审批人重复", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes[1].Approvers = []Approver{{Type: ApproverTypeUser, ID: 2}, {Type: ApproverTypeUser, ID: 2}}
			return nodes, edges
		}},
		{"审批策略无效", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes[1].Strategy = "VOTE"
			return nodes, edges
		}},
		{"排他网关缺默认边", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "gw", Type: NodeTypeExclusiveGateway})
			edges[1] = &Edge{From: "task1", To: "gw"}
			edges = append(edges, &Edge{From: "gw", To: "end", Condition: "amount > 100"}, &Edge{From: "gw", To: "end", Condition: "amount > 200"})
			return nodes, edges
		}},
		{"排他网关出边缺条件", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "gw", Type: NodeTypeExclusiveGateway})
			edges[1] = &Edge{From: "task1", To: "gw"}
			edges = append(edges, &Edge{From: "gw", To: "end"}, &Edge{From: "gw", To: "end", Condition: ConditionDefault})
			return nodes, edges
		}},
		{"并行分支不足", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "fork", Type: NodeTypeParallelFork})
			edges[1] = &Edge{From: "task1", To: "fork"}
			edges = append(edges, &Edge{From: "fork", To: "end"})
			return nodes, edges
		}},
		{"并行汇聚入度不足", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "join", Type: NodeTypeParallelJoin})
			edges[1] = &Edge{From: "task1", To: "join"}
			edges = append(edges, &Edge{From: "join", To: "end"})
			return nodes, edges
		}},
		{"节点类型不支持", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "sub", Type: "SUBPROCESS"})
			edges[1] = &Edge{From: "task1", To: "sub"}
			edges = append(edges, &Edge{From: "sub", To: "end"})
			return nodes, edges
		}},
		{"节点不可达", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "task2", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypeUser, ID: 3}}})
			edges = append(edges, &Edge{From: "task2", To: "end"})
			return nodes, edges
		}},
		{"节点不可达结束", func() ([]*Node, []*Edge) {
			nodes, edges := linearGraph()
			nodes = append(nodes, &Node{ID: "task2", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypeUser, ID: 3}}})
			edges = append(edges, &Edge{From: "start", To: "task2"})
			return nodes, edges
		}},
	}
	for _, c := range cases {
		nodes, edges := c.build()
		graph, err := Parse(buildGraphJSON(t, graphVersion, nodes, edges))
		if err != nil {
			t.Fatalf("%s: 解析失败: %v", c.name, err)
		}
		if err := Validate(graph); err == nil {
			t.Fatalf("%s: 期望校验失败", c.name)
		}
	}
}

func TestValidateParallelHappyPath(t *testing.T) {
	nodes := []*Node{
		{ID: "start", Type: NodeTypeStart},
		{ID: "fork", Type: NodeTypeParallelFork},
		{ID: "taskA", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypeRole, ID: 5}}},
		{ID: "taskB", Type: NodeTypeTask, Approvers: []Approver{{Type: ApproverTypePost, ID: 6}}, Strategy: StrategyAny},
		{ID: "join", Type: NodeTypeParallelJoin},
		{ID: "end", Type: NodeTypeEnd},
	}
	edges := []*Edge{
		{From: "start", To: "fork"},
		{From: "fork", To: "taskA"},
		{From: "fork", To: "taskB"},
		{From: "taskA", To: "join"},
		{From: "taskB", To: "join"},
		{From: "join", To: "end"},
	}
	graph, err := Parse(buildGraphJSON(t, graphVersion, nodes, edges))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if err := Validate(graph); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if graph.Nodes["join"].InDegree != 2 {
		t.Fatalf("join in-degree mismatch: %d", graph.Nodes["join"].InDegree)
	}
	if !graph.Nodes["taskB"].IsAnyStrategy() {
		t.Fatalf("taskB 应为或签")
	}
}
