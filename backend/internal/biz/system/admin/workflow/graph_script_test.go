package workflow

import "testing"

// TestScriptGraphRoundTrip 校验脚本配置和画布坐标在发布序列化后保留。
func TestScriptGraphRoundTrip(t *testing.T) {
	graph, err := Parse(`{"version":2,"hooks":{"before_submit":9},"nodes":[{"id":"s","type":"START","x":1.5,"y":2},{"id":"run","type":"SCRIPT","script_id":3},{"id":"task","type":"TASK","approver_script_id":4},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"run"},{"from":"run","to":"task"},{"from":"task","to":"e"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	err = Validate(graph)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := graph.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var restored *Graph
	restored, err = Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Nodes["s"].X != 1.5 || restored.Hooks.BeforeSubmit != 9 || len(restored.ScriptIDs()) != 3 {
		t.Fatal("配置未保留")
	}
	restored.Nodes["task"].Approvers = []Approver{{Type: ApproverTypeUser, ID: 1}}
	if Validate(restored) == nil {
		t.Fatal("混合审批配置必须失败")
	}
}

// TestGraphRejectsCycle 校验即使所有节点双向可达也不允许循环。
func TestGraphRejectsCycle(t *testing.T) {
	graph, err := Parse(`{"version":2,"nodes":[{"id":"s","type":"START"},{"id":"g","type":"EXCLUSIVE_GATEWAY"},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"g"},{"from":"g","to":"g","condition":"x > 1"},{"from":"g","to":"e","condition":"default"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if Validate(graph) == nil {
		t.Fatal("循环必须失败")
	}
}

// TestScriptReferencesValidation 校验脚本引用和脚本节点的边界约束。
func TestScriptReferencesValidation(t *testing.T) {
	cases := []string{
		`{"version":2,"hooks":{"approved":-1},"nodes":[{"id":"s","type":"START"},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"e"}]}`,
		`{"version":2,"nodes":[{"id":"s","type":"START"},{"id":"x","type":"SCRIPT","script_id":0},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"x"},{"from":"x","to":"e"}]}`,
		`{"version":2,"nodes":[{"id":"s","type":"START","script_id":1},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"e"}]}`,
	}
	for _, raw := range cases {
		graph, err := Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if Validate(graph) == nil {
			t.Fatalf("无效脚本配置通过校验: %s", raw)
		}
	}
}
