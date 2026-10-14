package biz

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/liujitcn/go-utils/crypto"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	passwordPolicy "github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestOptionBaseTenantScopesOrdinaryTenant 验证普通租户的租户选项只返回当前租户，默认租户仍可查看全部租户。
func TestOptionBaseTenantScopesOrdinaryTenant(t *testing.T) {
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
		if err := connection.Close(); err != nil {
			t.Error(err)
		}
	})
	err = db.AutoMigrate(&models.BaseTenant{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tenant := range []*models.BaseTenant{
		{ID: 1, Code: kitgorm.DefaultTenantCode, Name: "默认租户", Status: 1},
		{ID: 3, Code: "tenant-3", Name: "租户三", Status: 1},
		{ID: 4, Code: "tenant-4", Name: "租户四", Status: 1},
	} {
		if err = db.Create(tenant).Error; err != nil {
			t.Fatal(err)
		}
	}
	store, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	caseValue := &BaseTenantCase{BaseCase: &biz.BaseCase{}, BaseTenantRepository: data.NewBaseTenantRepository(store)}
	ordinaryCtx := engine.ContextWithAuthClaims(context.Background(), (&authdata.UserTokenPayload{TenantId: 4, TenantCode: "tenant-4"}).MakeAuthClaims())
	options, err := caseValue.OptionBaseTenant(ordinaryCtx, &adminv1.OptionBaseTenantRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.List) != 1 || options.List[0].Value != 4 {
		t.Fatalf("普通租户选项 = %+v, want only tenant 4", options.List)
	}
	defaultCtx := engine.ContextWithAuthClaims(context.Background(), (&authdata.UserTokenPayload{TenantId: 1, TenantCode: kitgorm.DefaultTenantCode}).MakeAuthClaims())
	options, err = caseValue.OptionBaseTenant(defaultCtx, &adminv1.OptionBaseTenantRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.List) != 3 {
		t.Fatalf("默认租户选项数量 = %d, want 3", len(options.List))
	}
}

// TestBaseTenantIDUsesDatabaseAutoIncrement 验证租户 ID 由数据库自增且不依赖租户编号。
func TestBaseTenantIDUsesDatabaseAutoIncrement(t *testing.T) {
	modelSchema, err := schema.Parse(&models.BaseTenant{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	idField, ok := modelSchema.FieldsByDBName["id"]
	if !ok || !idField.PrimaryKey || !idField.AutoIncrement {
		t.Fatalf("tenant ID field metadata = %+v, want primary key with auto increment", idField)
	}
}

// TestBuildTenantAdminCredentialsUsesGlobalPassword 验证已配置初始化密码时直接复用密码哈希。
func TestBuildTenantAdminCredentialsUsesGlobalPassword(t *testing.T) {
	configuredHash := "$2a$10$configured-password-hash"
	password, initialPassword, err := buildTenantAdminCredentials(loginpolicy.PasswordConfig{InitialPasswordHash: configuredHash})
	if err != nil {
		t.Fatalf("buildTenantAdminCredentials() error = %v", err)
	}
	if password != configuredHash {
		t.Fatalf("password = %q, want configured hash", password)
	}
	if initialPassword != "" {
		t.Fatalf("initialPassword = %q, want empty", initialPassword)
	}
}

// TestBuildTenantAdminCredentialsGeneratesUsablePassword 验证未配置初始化密码时生成可登录且符合策略的随机密码。
func TestBuildTenantAdminCredentialsGeneratesUsablePassword(t *testing.T) {
	config := loginpolicy.PasswordConfig{MinLength: 24, MinComplexityClasses: 4}
	password, initialPassword, err := buildTenantAdminCredentials(config)
	if err != nil {
		t.Fatalf("buildTenantAdminCredentials() error = %v", err)
	}
	if initialPassword == "" {
		t.Fatal("initialPassword is empty")
	}
	if len([]rune(initialPassword)) != int(config.MinLength) {
		t.Fatalf("generated password length = %d, want %d", len([]rune(initialPassword)), config.MinLength)
	}
	if err = passwordPolicy.ValidateComplexity(initialPassword, config); err != nil {
		t.Fatalf("generated password does not satisfy policy: %v", err)
	}
	if err = crypto.Verify(initialPassword, password); err != nil {
		t.Fatalf("generated password does not match hash: %v", err)
	}
}
