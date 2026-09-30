package kit_test

import (
	"context"
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// staticKey 测试用固定派生密钥。
type staticKey struct{}

// Derive 按用途返回固定的 32 字节测试密钥。
func (staticKey) Derive(_ context.Context, _ string) ([]byte, error) {
	return make([]byte, 32), nil
}

// newSecretFieldTestDB 创建启用密钥字段回调的内存数据库。
func newSecretFieldTestDB(t *testing.T) (*gorm.DB, *kit.SecretFieldRuntime) {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=Local&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库: %v", err)
	}
	if err = db.AutoMigrate(&models.AiProvider{}, &models.BaseOauthProvider{}, &models.BaseMessageProvider{}); err != nil {
		t.Fatalf("建表: %v", err)
	}
	runtime, err := kit.BindSecretFieldStorage(db)
	if err != nil {
		t.Fatalf("绑定密钥字段回调: %v", err)
	}
	if err = runtime.Backfill(context.Background()); err != nil {
		t.Fatalf("回填存量数据: %v", err)
	}
	return db, runtime
}

func TestSecretFieldStorageRoundTrip(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, _ := newSecretFieldTestDB(t)
	const apiKey = "sk-live-abcdef123456"
	item := &models.AiProvider{Name: "openai", BaseURL: "https://api.openai.com", APIKey: apiKey}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("写入记录: %v", err)
	}
	var stored string
	if err := db.Table("ai_provider").Select("api_key").Where("name = 'openai'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取库内密文: %v", err)
	}
	if !strings.HasPrefix(stored, "sbox:v1:") || stored == apiKey {
		t.Fatalf("库内应为信封密文: %q", stored)
	}
	var loaded models.AiProvider
	if err := db.Select("id", "name", "api_key").Where("name = 'openai'").First(&loaded).Error; err != nil {
		t.Fatalf("读取记录: %v", err)
	}
	if loaded.APIKey != apiKey {
		t.Fatalf("读回明文不符: %q", loaded.APIKey)
	}
}

func TestSecretFieldStorageBackfill(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, runtime := newSecretFieldTestDB(t)
	if err := db.Exec("INSERT INTO ai_provider (id, provider, name, base_url, api_key, config, sort, status, created_at, updated_at, created_by, updated_by, deleted_at) VALUES (1, 'openai_compatible', 'legacy', 'https://legacy', 'sk-legacy-key', '{}', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, 1, 0)").Error; err != nil {
		t.Fatalf("插入存量明文: %v", err)
	}
	if err := runtime.Backfill(context.Background()); err != nil {
		t.Fatalf("回填: %v", err)
	}
	var stored string
	if err := db.Table("ai_provider").Select("api_key").Where("name = 'legacy'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取回填结果: %v", err)
	}
	if !strings.HasPrefix(stored, "sbox:v1:") {
		t.Fatalf("存量明文应已加密: %q", stored)
	}
	var loaded models.AiProvider
	if err := db.Select("id", "name", "api_key").Where("name = 'legacy'").First(&loaded).Error; err != nil {
		t.Fatalf("读取回填记录: %v", err)
	}
	if loaded.APIKey != "sk-legacy-key" {
		t.Fatalf("回填读回明文不符: %q", loaded.APIKey)
	}
}

func TestBackfillBypassesBusinessCallbacks(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, runtime := newSecretFieldTestDB(t)
	// 模拟脱敏存储回调：任何以 map 提交的 ai_provider 更新一律报错（零实体拒绝）。
	if err := db.Callback().Update().Before("gorm:update").Register("test:redact-guard", func(tx *gorm.DB) {
		if tx.Statement.Table == "ai_provider" {
			if _, ok := tx.Statement.Dest.(map[string]interface{}); ok {
				panic("受保护表 ai_provider 的敏感字段更新必须使用带租户ID的实体模型")
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO ai_provider (id, provider, name, base_url, api_key, config, sort, status, created_at, updated_at, created_by, updated_by, deleted_at) VALUES (2, 'openai_compatible', 'legacy2', 'https://legacy2', 'sk-legacy-plain', '{}', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, 1, 0)").Error; err != nil {
		t.Fatalf("插入存量明文: %v", err)
	}
	if err := runtime.Backfill(context.Background()); err != nil {
		t.Fatalf("回填不应触发业务回调: %v", err)
	}
	var stored string
	if err := db.Table("ai_provider").Select("api_key").Where("name = 'legacy2'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取回填结果: %v", err)
	}
	if !strings.HasPrefix(stored, "sbox:v1:") {
		t.Fatalf("存量明文应已加密: %q", stored)
	}
}

func TestBackfillSkipsProtectedTable(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, runtime := newSecretFieldTestDB(t)
	runtime.SetProtectedTableChecker(func(table string) bool { return table == "ai_provider" })
	if err := db.Exec("INSERT INTO ai_provider (id, provider, name, base_url, api_key, config, sort, status, created_at, updated_at, created_by, updated_by, deleted_at) VALUES (3, 'openai_compatible', 'guarded', 'https://guarded', 'sk-guard-plain', '{}', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, 1, 0)").Error; err != nil {
		t.Fatalf("插入存量明文: %v", err)
	}
	if err := runtime.Backfill(context.Background()); err != nil {
		t.Fatalf("受保护表回填应跳过: %v", err)
	}
	var stored string
	if err := db.Table("ai_provider").Select("api_key").Where("name = 'guarded'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取结果: %v", err)
	}
	if stored != "sk-guard-plain" {
		t.Fatalf("受保护表存量应保持原样交由脱敏策略处理: %q", stored)
	}
}
