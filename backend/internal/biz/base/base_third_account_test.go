package biz

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBaseThirdAccountTenantIsolation 验证三方账号写入租户并按访问身份隔离查询。
func TestBaseThirdAccountTenantIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseThirdAccount{}); err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	accountCase := NewBaseThirdAccountCase(&biz.BaseCase{}, data.NewBaseThirdAccountRepository(store))
	if err = accountCase.CreateBinding(context.Background(), 2, 20, "wechat", "tenant-two"); err != nil {
		t.Fatal(err)
	}
	var saved models.BaseThirdAccount
	if err = db.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TenantID != 2 || saved.UserID != 20 {
		t.Fatalf("三方账号归属 = (%d, %d)，期望 (2, 20)", saved.TenantID, saved.UserID)
	}
	if err = accountCase.CreateBinding(context.Background(), 3, 30, "wechat", "tenant-two"); err == nil {
		t.Fatal("同一三方账号不应绑定到其他租户")
	}
	var accounts []models.BaseThirdAccount
	if err = db.Find(&accounts).Error; err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 {
		t.Fatalf("重复绑定后记录数 = %d，期望 1", len(accounts))
	}

	tenantIdentity := &authdata.UserTokenPayload{TenantId: 1, TenantCode: "tenant-one", UserId: 10}
	tenantCtx := engine.ContextWithAuthClaims(context.Background(), tenantIdentity.MakeAuthClaims())
	if _, err = accountCase.FindAuthorizedUserProvider(tenantCtx, 20, "wechat"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("普通租户查询其他租户账号错误 = %v，期望记录不存在", err)
	}

	defaultIdentity := &authdata.UserTokenPayload{TenantId: 1, TenantCode: kitgorm.DefaultTenantCode, UserId: 1}
	defaultCtx := engine.ContextWithAuthClaims(context.Background(), defaultIdentity.MakeAuthClaims())
	var account *models.BaseThirdAccount
	account, err = accountCase.FindAuthorizedUserProvider(defaultCtx, 20, "wechat")
	if err != nil {
		t.Fatal(err)
	}
	if account.TenantID != 2 || account.UserID != 20 {
		t.Fatalf("平台租户查询账号归属 = (%d, %d)，期望 (2, 20)", account.TenantID, account.UserID)
	}
}
