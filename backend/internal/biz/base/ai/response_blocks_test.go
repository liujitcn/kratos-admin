package ai

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// TestBuildResponseBlocksCombinesRules 验证同一工具的多个编排规则会按注册顺序合并输出。
func TestBuildResponseBlocksCombinesRules(t *testing.T) {
	toolName := "response_blocks_test"
	RegisterResponseBlockRule(ResponseBlockRule{
		ToolName: toolName,
		Build: func(agent.ToolUsage) []map[string]any {
			return []map[string]any{{"type": "first"}}
		},
	})
	RegisterResponseBlockRule(ResponseBlockRule{
		ToolName: toolName,
		Build: func(agent.ToolUsage) []map[string]any {
			return []map[string]any{{"type": "second"}}
		},
	})

	blocks := buildResponseBlocks([]agent.ToolUsage{{Name: toolName, Status: "success"}})
	if len(blocks) != 2 || blocks[0]["type"] != "first" || blocks[1]["type"] != "second" {
		t.Fatalf("结构化编排块未按规则顺序合并: %v", blocks)
	}
}
