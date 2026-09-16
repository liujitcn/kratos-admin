package server

import (
	"context"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/server/middleware"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"google.golang.org/protobuf/proto"
)

// TestTenantScopeMiddlewareFillsBaseUserForm 验证用户新增嵌套表单会在 Proto 校验前补全租户。
func TestTenantScopeMiddlewareFillsBaseUserForm(t *testing.T) {
	request := &adminv1.CreateBaseUserRequest{BaseUser: &adminv1.BaseUserForm{}}
	ctx := engine.ContextWithAuthClaims(context.Background(), (&data.UserTokenPayload{
		TenantId:   1000,
		TenantCode: "tenant-1000",
	}).MakeAuthClaims())

	handler := middleware.NewTenantScopeMiddleware()(func(_ context.Context, req any) (any, error) {
		return req, nil
	})
	_, err := handler(ctx, request)
	if err != nil {
		t.Fatalf("租户补全失败: %v", err)
	}
	if request.GetBaseUser().GetTenantId() != 1000 {
		t.Fatalf("用户表单租户ID = %d, want 1000", request.GetBaseUser().GetTenantId())
	}
}

// TestTenantScopeMiddlewareRejectsExplicitZeroTenant 验证可选查询字段显式传零不会被当成省略。
func TestTenantScopeMiddlewareRejectsExplicitZeroTenant(t *testing.T) {
	request := &adminv1.PageBaseUserRequest{TenantId: proto.Int64(0)}
	ctx := engine.ContextWithAuthClaims(context.Background(), (&data.UserTokenPayload{
		TenantId:   1000,
		TenantCode: "tenant-1000",
	}).MakeAuthClaims())

	handler := middleware.NewTenantScopeMiddleware()(func(_ context.Context, req any) (any, error) {
		return req, nil
	})
	if _, err := handler(ctx, request); err == nil {
		t.Fatal("显式零租户ID未被拒绝")
	}
}
