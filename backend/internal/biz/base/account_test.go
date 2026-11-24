package biz

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/liujitcn/go-utils/crypto"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUpdateUserPasswordPostgres 验证自助改密的 PostgreSQL 语句只更新一次 updated_at。
func TestUpdateUserPasswordPostgres(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=test dbname=test sslmode=disable"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	const oldPassword = "Old-Passw0rd!"
	var oldHash string
	oldHash, err = crypto.Encrypt(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Callback().Query().After("gorm:query").Register("account_test:user", func(tx *gorm.DB) {
		*tx.Statement.Dest.(*models.BaseUser) = models.BaseUser{ID: 1, TenantID: 1, Password: oldHash, PasswordHistory: "[]"}
	}); err != nil {
		t.Fatal(err)
	}
	var updateSQL string
	if err = db.Callback().Update().After("gorm:update").Register("account_test:update", func(tx *gorm.DB) {
		updateSQL = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatal(err)
	}
	var provider *data.Data
	provider, err = data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	var cache *memory.Memory
	var cleanup func()
	cache, cleanup, err = memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	claims := (&authdata.UserTokenPayload{UserId: 1, TenantId: 1}).MakeAuthClaims()
	ctx := engine.ContextWithAuthClaims(t.Context(), claims)
	account := NewAccountCase(&biz.BaseCase{Cache: cache}, data.NewBaseUserRepository(provider), nil)
	if err = account.UpdateUserPassword(ctx, &basev1.UserPasswordForm{
		OldPwd: &commonv1.SecretCrypto{Text: oldPassword},
		NewPwd: &commonv1.SecretCrypto{Text: "New-Passw0rd!"},
	}); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(updateSQL, `"updated_at"=`); count != 1 {
		t.Fatalf("updated_at 赋值次数为 %d，期望 1", count)
	}
}
