package biz

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/jackc/pgx/v5/pgconn"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestDatabaseConnectionChecksHostPool 验证真实探测、宿主连接池所有权和默认租户限制。
func TestDatabaseConnectionChecksHostPool(t *testing.T) {
	client, _ := newSecurityData(t)
	pool, err := client.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	database := NewDatabaseCase(&biz.BaseCase{GormClients: map[string]*gorm.Client{"default": client}}, nil)
	req := &adminv1.TestConnectionRequest{SourceName: "default"}

	ctx := databaseTestContext(t.Context(), "TEST")
	if _, err = database.TestConnection(ctx, req); errors.Code(err) != 403 {
		t.Fatalf("普通租户测试宿主连接: %v", err)
	}
	ctx = databaseTestContext(t.Context(), gorm.DefaultTenantCode)
	var result *adminv1.TestConnectionResponse
	result, err = database.TestConnection(ctx, req)
	if err != nil || !result.GetSuccess() {
		t.Fatalf("宿主连接应可用: %v, %v", result, err)
	}
	if err = pool.PingContext(ctx); err != nil {
		t.Fatalf("测试不应关闭宿主连接池: %v", err)
	}
	if err = pool.Close(); err != nil {
		t.Fatal(err)
	}
	result, err = database.TestConnection(ctx, req)
	if err != nil || result.GetSuccess() {
		t.Fatalf("已关闭的宿主连接不能通过测试: %v, %v", result, err)
	}
}

// TestIsPostgresUniqueConstraint 验证只有指定的唯一约束错误会被识别。
func TestIsPostgresUniqueConstraint(t *testing.T) {
	const constraint = "unique_base_login_policy"
	err := fmt.Errorf("写入登录策略: %w", &pgconn.PgError{Code: "23505", ConstraintName: constraint})
	if !isPostgresUniqueConstraint(err, constraint) {
		t.Fatal("应识别包装后的业务唯一约束冲突")
	}
	for _, err := range []error{
		&pgconn.PgError{Code: "23505", ConstraintName: "base_login_policy_pkey"},
		&pgconn.PgError{Code: "23502", ConstraintName: constraint},
	} {
		if isPostgresUniqueConstraint(err, constraint) {
			t.Fatalf("不应将该错误识别为业务唯一约束冲突: %v", err)
		}
	}
}

// databaseTestContext 构建指定租户的可信测试身份。
func databaseTestContext(ctx context.Context, tenant string) context.Context {
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: tenant, UserId: 1, DataScope: gorm.DataScopeAll}
	return engine.ContextWithAuthClaims(ctx, identity.MakeAuthClaims())
}
