package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestPaginateAiPluginTools 验证工具快照分页边界和未分页兼容行为。
func TestPaginateAiPluginTools(t *testing.T) {
	items := []*models.AiPluginTool{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}
	tests := []struct {
		name     string
		pageNum  int64
		pageSize int64
		want     []int64
	}{
		{name: "first page", pageNum: 1, pageSize: 2, want: []int64{1, 2}},
		{name: "middle page", pageNum: 2, pageSize: 2, want: []int64{3, 4}},
		{name: "last page", pageNum: 3, pageSize: 2, want: []int64{5}},
		{name: "zero page uses first page", pageNum: 0, pageSize: 2, want: []int64{1, 2}},
		{name: "out of range", pageNum: 4, pageSize: 2},
		{name: "zero size keeps all", pageNum: 1, pageSize: 0, want: []int64{1, 2, 3, 4, 5}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := paginateAiPluginTools(items, test.pageNum, test.pageSize)
			if len(got) != len(test.want) {
				t.Fatalf("got %d items, want %d", len(got), len(test.want))
			}
			for index, want := range test.want {
				if got[index].ID != want {
					t.Fatalf("item %d got ID %d, want %d", index, got[index].ID, want)
				}
			}
		})
	}
}

// TestInferMCPToolEffect 根据工具名称和描述推断默认读写效果。
func TestInferMCPToolEffect(t *testing.T) {
	for _, test := range []struct {
		name        string
		description string
		want        string
	}{
		{name: "get_version", description: "Returns the service version", want: "read"},
		{name: "cache_clear", description: "清除远端缓存", want: "write"},
		{name: "convert_to_markdown", description: "将文档转换为 Markdown", want: "write"},
		{name: "list_formats", description: "列出支持的格式", want: "read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := inferMCPToolEffect(test.name, test.description); got != test.want {
				t.Fatalf("effect = %q, want %q", got, test.want)
			}
		})
	}
}
