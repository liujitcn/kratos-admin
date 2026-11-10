package base

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/service/base/v1"
	kitmcp "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestRegisterMCPAccessControl 验证生产注册入口会绑定 HTTP 处理器，并在权限仓储缺失时拒绝执行工具。
func TestRegisterMCPAccessControl(t *testing.T) {
	h := &biz.McpCase{}
	services := Services{Mcp: base.NewMcpService(h)}
	srv := kitmcp.NewHandler(kitmcp.WithJSONResponse(true), kitmcp.WithStreamableHTTPOptions(server.WithStateLess(true)))
	t.Cleanup(func() {
		if err := srv.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	services.RegisterMCP(srv)
	if h.Handler == nil {
		t.Fatal("生产注册入口未初始化 MCP 处理器和访问控制")
	}
	called := false
	srv.MCPServer().AddTool(mcp.NewTool("app_probe"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"app_probe","arguments":{}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	var response struct {
		Result mcp.CallToolResult `json:"result"`
	}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil || called || !response.Result.IsError {
		t.Fatalf("生产工具调用绕过了权限过滤: called=%v body=%s err=%v", called, recorder.Body.String(), err)
	}
}
