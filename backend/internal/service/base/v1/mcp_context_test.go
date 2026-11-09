package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-kit/auth"
	authn "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
)

// TestMcpHTTPMiddlewareContext 验证 MCP 转发保留 Kratos 中间件注入的认证与请求上下文。
func TestMcpHTTPMiddlewareContext(t *testing.T) {
	type middlewareKey struct{}
	called := false
	h := &biz.McpCase{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		info, err := auth.FromContext(r.Context())
		if err != nil || info.TenantCode != "tenant-a" || info.RoleCode != "reader" || r.Context().Value(middlewareKey{}) != "保留" {
			t.Errorf("MCP 转发丢失中间件上下文: info=%v err=%v", info, err)
		}
		if r.Header.Get("X-Admin-Mcp-Terminal") != "app" {
			t.Error("MCP 转发丢失终端标识")
		}
		w.WriteHeader(http.StatusNoContent)
	})}
	server := kratoshttp.NewServer(kratoshttp.Middleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			claims := (&authdata.UserTokenPayload{TenantId: 1, TenantCode: "tenant-a", UserId: 2, RoleCode: "reader"}).MakeAuthClaims()
			ctx = authn.ContextWithAuthClaims(ctx, claims)
			return next(context.WithValue(ctx, middlewareKey{}, "保留"), req)
		}
	}))
	RegisterMcpServiceHTTPServer(server, NewMcpService(h))
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/mcp/app", nil))
	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("MCP HTTP 路由未调用处理器: %d %s", recorder.Code, recorder.Body.String())
	}
}
