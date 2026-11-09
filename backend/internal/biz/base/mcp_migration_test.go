package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	authn "github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	kitmcp "github.com/liujitcn/kratos-kit/transport/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestMcpMigrationAccess 验证真实协议请求保留终端隔离、登录鉴权、禁用状态和只读边界。
func TestMcpMigrationAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcp.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stmt := &gorm.Statement{DB: db}
	if err = stmt.Parse(&models.BaseAPI{}); err != nil {
		t.Fatal(err)
	}
	stmt.Schema.FieldsByName["ID"].DataType = schema.Int
	if err = db.AutoMigrate(&models.BaseAPI{}); err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	apis := []*models.BaseAPI{
		{ID: 1, ToolName: "app_read", Method: "GET", Operation: "/app.v1.Example/Read", McpStatus: _const.STATUS_STATUS_ENABLE, ToolPrompts: `["读取提示"]`},
		{ID: 2, ToolName: "app_write", Method: "POST", McpStatus: _const.STATUS_STATUS_ENABLE},
		{ID: 3, ToolName: "system_admin_v1_read", Method: "GET", McpStatus: _const.STATUS_STATUS_ENABLE},
		{ID: 4, ToolName: "app_disabled", Method: "GET"},
		{ID: 5, ToolName: "base_v1_ai_message_service_read", Method: "GET", McpStatus: _const.STATUS_STATUS_ENABLE},
	}
	if err = db.Create(apis).Error; err != nil {
		t.Fatal(err)
	}
	gate := &mcpMigrationAuthorizer{t: t, allowed: true}
	h := &McpCase{baseAPIRepo: data.NewBaseAPIRepository(store), authorizer: gate}
	hooks := &server.Hooks{}
	hookCalled := false
	hooks.AddAfterListTools(func(context.Context, any, *mcp.ListToolsRequest, *mcp.ListToolsResult) { hookCalled = true })
	srv := kitmcp.NewHandler(kitmcp.WithJSONResponse(true), kitmcp.WithStreamableHTTPOptions(server.WithStateLess(true)), kitmcp.WithServerOptions(server.WithHooks(hooks)))
	t.Cleanup(func() {
		if err = srv.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	called := false
	for _, api := range apis {
		srv.MCPServer().AddTool(mcp.NewTool(api.ToolName), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			called = true
			return mcp.NewToolResultText("ok"), nil
		})
	}
	h.RegisterMCP(srv)
	claims := (&authdata.UserTokenPayload{TenantId: 1, TenantCode: "tenant-a", UserId: 2, RoleCode: "reader"}).MakeAuthClaims()
	ctx := authn.ContextWithAuthClaims(t.Context(), claims)
	var list mcp.ListToolsResult
	callMcpMigrationRequest(t, h, ctx, "app", "tools/list", map[string]any{}, &list)
	if !hookCalled {
		t.Fatal("已有 SDK 钩子被覆盖")
	}
	readFound := false
	for _, tool := range list.Tools {
		if tool.Name == "system_admin_v1_read" {
			t.Fatal("应用端泄漏管理端工具")
		}
		if tool.Name == "app_read" {
			readFound = tool.Description == "读取提示"
		}
	}
	if !readFound {
		t.Fatal("工具列表未应用数据库提示词")
	}
	for _, test := range []struct {
		name, terminal, tool string
		anonymous, denied    bool
		wantCall             bool
	}{
		{name: "允许读取", terminal: "app", tool: "app_read", wantCall: true},
		{name: "未登录", terminal: "app", tool: "app_read", anonymous: true},
		{name: "角色无权", terminal: "app", tool: "app_read", denied: true},
		{name: "终端隔离", terminal: "app", tool: "system_admin_v1_read"},
		{name: "写入拦截", terminal: "app", tool: "app_write"},
		{name: "工具禁用", terminal: "app", tool: "app_disabled"},
		{name: "审批入口封闭", terminal: "app", tool: "base_v1_ai_message_service_read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called = false
			gate.allowed = !test.denied
			callCtx := ctx
			if test.anonymous {
				callCtx = t.Context()
			}
			var result mcp.CallToolResult
			callMcpMigrationRequest(t, h, callCtx, test.terminal, "tools/call", map[string]any{"name": test.tool, "arguments": map[string]any{}}, &result)
			if called != test.wantCall || result.IsError == test.wantCall {
				t.Fatalf("权限边界错误: called=%v isError=%v", called, result.IsError)
			}
		})
	}
}

// TestMcpTerminalFallback 验证无请求头时保留上下文终端，有请求头时使用本次请求的终端。
func TestMcpTerminalFallback(t *testing.T) {
	ctx := context.WithValue(t.Context(), mcpTerminalContextKey{}, "app")
	if mcpTerminal(ctx, nil) != "app" || mcpTerminal(ctx, http.Header{mcpTerminalHeader: {"admin"}}) != "admin" {
		t.Fatal("终端上下文或请求头被忽略")
	}
}

type mcpMigrationAuthorizer struct {
	engine.Engine
	t       *testing.T
	allowed bool
}

// IsAuthorized 验证鉴权链路传入当前租户和角色。
func (g *mcpMigrationAuthorizer) IsAuthorized(ctx context.Context, subject engine.Subject, action engine.Action, resource engine.Resource, project engine.Project) (bool, error) {
	claims, ok := engine.AuthClaimsFromContext(ctx)
	if !ok || claims.Tenant == nil || *claims.Tenant != "tenant-a" || subject != "reader" {
		g.t.Error("鉴权链路丢失租户或角色")
	}
	return g.allowed, nil
}

// callMcpMigrationRequest 使用真实 HTTP JSON-RPC 请求验证 MCP 注册链路。
func callMcpMigrationRequest(t *testing.T, h *McpCase, ctx context.Context, terminal, method string, params, result any) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	req.Header.Set(mcpTerminalHeader, terminal)
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err = json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("响应不是 JSON: %s: %v", recorder.Body.String(), err)
	}
	if recorder.Code != http.StatusOK || len(response.Error) > 0 {
		t.Fatalf("协议请求失败: %d %s", recorder.Code, recorder.Body.String())
	}
	if err = json.Unmarshal(response.Result, result); err != nil {
		t.Fatal(err)
	}
}
