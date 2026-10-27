package biz

import (
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
)

// TestDraftNodeConfig 校验客户端脚本源码被丢弃，同时保留脚本引用与坐标。
func TestDraftNodeConfig(t *testing.T) {
	raw, err := draftNodeConfig(`{"version":2,"hooks":{"approved":7},"script_snapshots":{"7":{"id":7,"source":"untrusted"}},"nodes":[{"id":"s","type":"START","x":12,"y":34},{"id":"e","type":"END"}],"edges":[{"from":"s","to":"e"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "untrusted") || strings.Contains(raw, "script_snapshots") {
		t.Fatal("不能保存客户端快照")
	}
	var graph *workflow.Graph
	graph, err = workflow.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Hooks.Approved != 7 || graph.Nodes["s"].X != 12 || graph.Nodes["s"].Y != 34 {
		t.Fatal("原始配置丢失")
	}
}
