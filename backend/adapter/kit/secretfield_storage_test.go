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

// newSecretConfigTestDB 创建启用敏感配置加密回调的内存数据库。
func newSecretConfigTestDB(t *testing.T) (*gorm.DB, *kit.SecretFieldRuntime) {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=Local&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库: %v", err)
	}
	if err = db.AutoMigrate(&models.BaseConfig{}); err != nil {
		t.Fatalf("建表: %v", err)
	}
	runtime, err := kit.BindSecretFieldStorage(db)
	if err != nil {
		t.Fatalf("绑定敏感配置加密回调: %v", err)
	}
	return db, runtime
}

func TestSecretConfigWholeValueRoundTrip(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, runtime := newSecretConfigTestDB(t)
	kit.RegisterSecretConfigKeys("oauth.github.client_secret")
	const secret = "gh-secret-value"
	item := &models.BaseConfig{Key: "oauth.github.client_secret", Value: secret}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("写入配置: %v", err)
	}
	var stored string
	if err := db.Table("base_config").Select("value").Where("`key` = 'oauth.github.client_secret'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取库内密文: %v", err)
	}
	if !strings.HasPrefix(stored, "sbox:v1:") || stored == secret {
		t.Fatalf("库内应为信封密文: %q", stored)
	}
	plaintext, err := runtime.DecryptConfigValue(stored)
	if err != nil {
		t.Fatalf("解密配置值: %v", err)
	}
	if plaintext != secret {
		t.Fatalf("解密明文不符: %q", plaintext)
	}
}

func TestSecretConfigPlainTextKeyUntouched(t *testing.T) {
	sdk.Runtime.SetKey(staticKey{})
	t.Cleanup(func() { sdk.Runtime.SetKey(nil) })
	db, _ := newSecretConfigTestDB(t)
	const value = "plain-config-value"
	item := &models.BaseConfig{Key: "oauth.generic.redirect", Value: value}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("写入配置: %v", err)
	}
	var stored string
	if err := db.Table("base_config").Select("value").Where("`key` = 'oauth.generic.redirect'").Row().Scan(&stored); err != nil {
		t.Fatalf("读取配置: %v", err)
	}
	if stored != value {
		t.Fatalf("未注册 key 应保持明文: %q", stored)
	}
}
