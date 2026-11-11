package biz

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestParseXbergExtractResult 验证结构化结果和文本结果都只读取 Xberg 文档内容字段。
func TestParseXbergExtractResult(t *testing.T) {
	structured := &mcp.CallToolResult{StructuredContent: map[string]any{
		"results": []any{map[string]any{"content": "第一段"}, map[string]any{"content": "第二段"}},
		"summary": map[string]any{"results": 2},
	}}
	text := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"results":[{"content":"文本结果"}],"summary":{"results":1}}`}}}
	for _, test := range []struct {
		name   string
		result *mcp.CallToolResult
		want   string
	}{
		{name: "structured", result: structured, want: "第一段\n\n第二段"},
		{name: "text", result: text, want: "文本结果"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseXbergExtractResult(test.result)
			if err != nil {
				t.Fatalf("解析 Xberg 结果失败: %v", err)
			}
			if got != test.want {
				t.Fatalf("解析结果 = %q，期望 %q", got, test.want)
			}
		})
	}
}

// TestParseXbergExtractResultRejectsInvalid 验证错误、空文本和未知结果结构不会进入 RAG。
func TestParseXbergExtractResultRejectsInvalid(t *testing.T) {
	for _, result := range []*mcp.CallToolResult{
		{IsError: true},
		{StructuredContent: map[string]any{"results": []any{map[string]any{"content": ""}}, "summary": map[string]any{"results": 1}}},
		{StructuredContent: map[string]any{"value": "任意 JSON"}},
	} {
		if _, err := parseXbergExtractResult(result); err == nil {
			t.Fatal("无效 Xberg 结果应被拒绝")
		}
	}
}
