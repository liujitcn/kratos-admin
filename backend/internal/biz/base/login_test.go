package biz

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoginFailureKeyTenantAndClientIsolation(t *testing.T) {
	key := loginFailureKey("tenant-a", "alice", "192.0.2.1")
	if key != loginFailureKey("tenant-a", "alice", "192.0.2.1") {
		t.Fatal("login failure key is not deterministic")
	}
	if key == loginFailureKey("tenant-b", "alice", "192.0.2.1") {
		t.Fatal("login failure key is not tenant isolated")
	}
	if key == loginFailureKey("tenant-a", "alice", "192.0.2.2") {
		t.Fatal("login failure key is not client isolated")
	}
}

func TestLoginFailurePolicyLocksAndClears(t *testing.T) {
	cache, _, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	loginCase := &LoginCase{BaseCase: &biz.BaseCase{Cache: cache}}
	ctx := context.Background()
	policySet := loginpolicy.PolicySet{Policies: []loginpolicy.Policy{{
		ScopeType:           loginpolicy.ScopeGlobal,
		Status:              loginpolicy.StatusEnable,
		MaxFailedAttempts:   3,
		LockDurationMinutes: 15,
	}}}
	for attempt := 0; attempt < 3; attempt++ {
		if err = loginCase.recordLoginFailure(ctx, "tenant-a", "alice", policySet, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice", policySet, 1, 1); err == nil {
		t.Fatal("expected login policy to lock the account")
	}
	if err = loginCase.clearLoginFailures(ctx, "tenant-a", "alice"); err != nil {
		t.Fatal(err)
	}
	if err = loginCase.checkLoginPolicy(ctx, "tenant-a", "alice", policySet, 1, 1); err != nil {
		t.Fatalf("expected cleared account to be available: %v", err)
	}
}

// TestFindUserByPasswordUsesTenantCode 验证同名账号登录时按请求租户查询用户。
func TestFindUserByPasswordUsesTenantCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	err = db.AutoMigrate(&models.BaseTenant{}, &models.BaseUser{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.BaseTenant{ID: 1, Code: kitgorm.DefaultTenantCode, Name: "默认租户", Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.BaseTenant{ID: 2, Code: "1000", Name: "普通租户", Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	var defaultPasswordHash string
	defaultPasswordHash, err = crypto.Encrypt("DefaultPassword1!")
	if err != nil {
		t.Fatal(err)
	}
	var tenantPasswordHash string
	tenantPasswordHash, err = crypto.Encrypt("TenantPassword1!")
	if err != nil {
		t.Fatal(err)
	}
	users := []*models.BaseUser{
		{ID: 1, TenantID: 1, UserName: "admin", Password: defaultPasswordHash, Status: 1},
		{ID: 2, TenantID: 2, UserName: "admin", Password: tenantPasswordHash, Status: 1},
	}
	for _, user := range users {
		if err = db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	var store *memory.Memory
	var cleanup func()
	store, cleanup, err = memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	dataStore, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	baseCase := &biz.BaseCase{Cache: store}
	baseUserCase := NewBaseUserCase(baseCase, data.NewBaseUserRepository(dataStore))
	loginCase := &LoginCase{
		BaseCase:       baseCase,
		baseUserCase:   baseUserCase,
		baseTenantRepo: data.NewBaseTenantRepository(dataStore),
	}
	password := &commonv1.SecretCrypto{Text: "TenantPassword1!"}
	user, err := loginCase.FindUserByPassword(context.Background(), "1000", "admin", password)
	if err != nil {
		t.Fatalf("普通租户同名管理员登录失败: %v", err)
	}
	if user.ID != 2 || user.TenantID != 2 {
		t.Fatalf("命中了错误租户账号: %+v", user)
	}
}

// TestGetAuthInfoByRefreshTokenEmpty 验证空刷新令牌不会访问缓存。
func TestGetAuthInfoByRefreshTokenEmpty(t *testing.T) {
	loginCase := &LoginCase{}
	if _, err := loginCase.getAuthInfoByRefreshToken(""); err == nil {
		t.Fatal("空刷新令牌应直接返回未认证错误")
	}
}

// TestGetAuthInfoByRefreshTokenMissing 验证缓存缺失仍按认证失效处理。
func TestGetAuthInfoByRefreshTokenMissing(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	loginCase := &LoginCase{BaseCase: &biz.BaseCase{Cache: store}}
	_, err = loginCase.getAuthInfoByRefreshToken("missing-refresh-token")
	if !errors.IsUnauthorized(err) {
		t.Fatalf("缓存缺失错误 = %v, want unauthorized", err)
	}
}
