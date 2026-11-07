package kit

import (
	"context"
	"errors"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestRevealStorageFieldLimitsResources 验证受控明文出口不能扩展到任意已加密表。
func TestRevealStorageFieldLimitsResources(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(&models.AiProvider{}, &models.OauthClient{})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := newFieldCipher(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("AES_GCM", "provider-secret")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.AiProvider{ID: 1, APIKey: encrypted}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.OauthClient{ID: 1, TenantID: 2, ClientSecret: encrypted}).Error
	if err != nil {
		t.Fatal(err)
	}
	resolver := &RedactPolicyResolver{defaultDB: db, fieldCipher: cipher, storagePolicies: map[string][]redact.StorageFieldPolicy{}}
	for table, column := range map[string]string{"ai_provider": "api_key", "oauth_client": "client_secret"} {
		resolver.storagePolicies[storagePolicyKey(defaultTenantID, kitgorm.DefaultClientName, table)] = []redact.StorageFieldPolicy{{
			ColumnName: column,
			Rule:       redact.FieldPolicy{EncryptAlgorithm: "AES_GCM", RuleType: "ENCRYPT"},
		}}
	}
	text, err := resolver.RevealStorageField(context.Background(), "ai_provider", "api_key", 1)
	if err != nil || text != "provider-secret" {
		t.Fatalf("已授权供应商密钥读取失败: %q, %v", text, err)
	}
	_, err = resolver.RevealStorageField(context.Background(), "oauth_client", "client_secret", 1)
	if !errors.Is(err, ErrUnprotectedStorageField) {
		t.Fatalf("任意加密表应拒绝明文读取，实际 %v", err)
	}
}
