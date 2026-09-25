package biz

import (
	"context"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestMcpToolErrorsUseRequestLocale 验证 MCP 工具级错误按请求语言本地化。
func TestMcpToolErrorsUseRequestLocale(t *testing.T) {
	catalog, err := corei18n.NewI18n("mcp-error-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	ctx := biz.WithLocale(context.Background(), "en-US")
	mcpCase := &McpCase{catalog: catalog}
	message := mcpCase.localizeMcpMessage(ctx, "system.mcp.tool_not_registered", map[string]any{"Name": "search"}, "fallback")
	if want := "MCP tool search is not registered."; message != want {
		t.Fatalf("localized MCP message = %q, want %q", message, want)
	}
	structured := errorsx.WithMessageKey(errorsx.Internal("MCP工具执行失败"), "system.mcp.tool_check_failed", nil)
	if got, want := mcpCase.localizeMcpError(ctx, structured), "Could not verify MCP tool access. Check the service logs."; got != want {
		t.Fatalf("localized MCP error = %q, want %q", got, want)
	}
}
