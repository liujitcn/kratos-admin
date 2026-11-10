package admin

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestBiSourceHostClientAccess 验证普通租户不能通过新配置或已有配置访问宿主客户端。
func TestBiSourceHostClientAccess(t *testing.T) {
	cards, _ := newBiWorkflowTestCases(t)
	c := cards.biQueryCase.biSourceCase
	c.GormClients = map[string]*gorm.Client{gorm.DefaultClientName: {}}
	ctx := biCardTestContext(2, 10, 0)
	legacy := &models.BiSource{TenantID: 2, Code: "legacy", SourceName: gorm.DefaultClientName, Status: 1, MaxRows: 100, SchemaJSON: `{"baseTable":"main.accounts","tables":[{"key":"main.accounts","name":"accounts"}],"measures":[{"code":"count","table":"main.accounts","column":"id"}]}`}
	err := c.sourceRepo.Create(ctx, legacy)
	if err != nil {
		t.Fatal(err)
	}
	form := &adminv1.BiSourceForm{Code: "bound", Name: "宿主连接", SourceName: gorm.DefaultClientName, MaxRows: 100}
	updated := &adminv1.BiSourceForm{Id: legacy.ID, Code: legacy.Code, Name: form.Name, SourceName: form.SourceName}
	for _, test := range []struct {
		name string
		run  func() error
	}{
		{name: "列出客户端", run: func() error {
			_, err = c.OptionBiSourceDatabase(ctx, &adminv1.OptionBiSourceDatabaseRequest{})
			return err
		}},
		{name: "创建绑定", run: func() error { _, err = c.CreateBiSource(ctx, &adminv1.CreateBiSourceRequest{Source: form}); return err }},
		{name: "更新绑定", run: func() error {
			_, err = c.UpdateBiSource(ctx, &adminv1.UpdateBiSourceRequest{Source: updated})
			return err
		}},
		{name: "测试绑定", run: func() error { _, err = c.TestBiSource(ctx, &adminv1.TestBiSourceRequest{Source: form}); return err }},
		{name: "发现旧配置", run: func() error {
			_, err = c.ListBiSourceTables(ctx, &adminv1.ListBiSourceTablesRequest{Id: legacy.ID})
			return err
		}},
		{name: "建模旧配置", run: func() error {
			_, err = c.SaveBiSourceModel(ctx, &adminv1.SaveBiSourceModelRequest{Id: legacy.ID, Model: testSourceModel()})
			return err
		}},
		{name: "执行旧配置", run: func() error {
			_, err = c.Execute(ctx, bi.Scope{TenantID: 2, UserID: 10}, bi.QuerySpec{SourceCode: legacy.Code})
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err = test.run(); errors.Code(err) != 403 {
				t.Fatalf("普通租户访问宿主客户端未被拒绝: %v", err)
			}
		})
	}
	defaultCtx := engine.ContextWithAuthClaims(t.Context(), (&data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode, UserId: 1, DataScope: gorm.DataScopeAll}).MakeAuthClaims())
	if _, err = c.OptionBiSourceDatabase(defaultCtx, &adminv1.OptionBiSourceDatabaseRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err = c.CreateBiSource(defaultCtx, &adminv1.CreateBiSourceRequest{Source: form}); err != nil {
		t.Fatal(err)
	}
	remote := &adminv1.BiSourceForm{Code: "remote", Name: "独立连接", Driver: "mysql", Host: "db.example.test", Port: 3306, DatabaseName: "analytics", Username: "reader", Password: &commonv1.SecretCrypto{Text: "test-password"}, MaxRows: 100}
	if _, err = c.CreateBiSource(ctx, &adminv1.CreateBiSourceRequest{Source: remote}); err != nil {
		t.Fatalf("普通租户的独立连接被拒绝: %v", err)
	}
}
