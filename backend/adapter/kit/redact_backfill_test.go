package kit

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// TestBackfillRoundTrip 验证真实主旁表事务、短姓名加密、历史回填及重复执行不改变密文。
func TestBackfillRoundTrip(t *testing.T) {
	driver.Opens["security-sqlite"] = sqlite.Open
	registered := []any{new(models.BaseUser), new(models.BaseTenant), new(models.BaseRedactStorageValue)}
	client, cleanup, err := kitgorm.NewGormClient(&configv1.Data_Database{Driver: "security-sqlite", Source: filepath.Join(t.TempDir(), "privacy.db")}, kitgorm.WithMigrateModels(registered...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	for _, model := range registered {
		statement := &gorm.Statement{DB: client.DB}
		err = statement.Parse(model)
		if err != nil {
			t.Fatal(err)
		}
		statement.Schema.FieldsByName["ID"].DataType = schema.Int
	}
	err = client.AutoMigrate(registered...)
	if err != nil {
		t.Fatal(err)
	}
	legacy := &models.BaseUser{ID: 10, TenantID: 1, UserName: "legacy", UserCode: "legacy", Phone: "19999911111", Status: 1}
	err = client.DB.Create(legacy).Error
	if err != nil {
		t.Fatal(err)
	}
	tenant := &models.BaseTenant{ID: 2, Code: "tenant-two", Name: "测试租户", ContactName: "张", ContactPhone: "18888822222", Status: 1}
	err = client.DB.Create(tenant).Error
	if err != nil {
		t.Fatal(err)
	}
	var resolver *RedactPolicyResolver
	resolver, err = NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	var phone, name redact.FieldPolicy
	phone, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "MASK", `{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}`)
	if err != nil {
		t.Fatal(err)
	}
	name, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "MASK", `{"mask":{"keep_first":1,"keep_last":0,"mask_char":"*"}}`)
	if err != nil {
		t.Fatal(err)
	}
	resolver.storagePolicies[storagePolicyKey(1, kitgorm.DefaultClientName, "base_user")] = []redact.StorageFieldPolicy{{ID: 2001, TenantID: 1, TableName: "base_user", ColumnName: "phone", Rule: phone}}
	resolver.storagePolicies[storagePolicyKey(1, kitgorm.DefaultClientName, "base_tenant")] = []redact.StorageFieldPolicy{{ID: 2011, TenantID: 1, TableName: "base_tenant", ColumnName: "contact_name", Rule: name}, {ID: 2012, TenantID: 1, TableName: "base_tenant", ColumnName: "contact_phone", Rule: phone}}
	var protector *redact.StorageProtector
	protector, err = redact.NewStorageProtector("local-storage-test-key")
	if err != nil {
		t.Fatal(err)
	}
	resolver.fieldCipher, err = newFieldCipher(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	resolver.runtime = newStorageRuntime(resolver.store, resolver, protector, resolver.fieldCipher)
	err = resolver.runtime.registerCallbacks(client.DB)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	count, err = resolver.Backfill(t.Context(), registered...)
	if err != nil || count != 2 {
		t.Fatalf("历史数据回填失败: %d, %v", count, err)
	}
	raw := context.WithValue(t.Context(), rawStorageReadKey{}, true)
	var stored models.BaseUser
	err = client.DB.WithContext(raw).Where(clause.Eq{Column: clause.Column{Name: "id"}, Value: legacy.ID}).First(&stored).Error
	if err != nil || stored.Phone == legacy.Phone || !strings.Contains(stored.Phone, "*") {
		t.Fatalf("主表仍保存原文: %v", err)
	}
	var values []*models.BaseRedactStorageValue
	err = client.DB.Find(&values).Error
	if err != nil || len(values) != 3 {
		t.Fatalf("旁表原文不完整: %d, %v", len(values), err)
	}
	firstCipher := bytes.Clone(values[0].Ciphertext)
	count, err = resolver.Backfill(t.Context(), registered...)
	if err != nil || count != 0 {
		t.Fatalf("回填不幂等: %d, %v", count, err)
	}
	err = client.DB.Find(&values).Error
	if err != nil || !bytes.Equal(firstCipher, values[0].Ciphertext) {
		t.Fatalf("重复回填改写了密文: %v", err)
	}
	err = client.DB.Where(clause.Eq{Column: clause.Column{Name: "id"}, Value: legacy.ID}).First(&stored).Error
	if err != nil || stored.Phone != legacy.Phone {
		t.Fatalf("正常业务无法读取原文: %v", err)
	}
	identity := &authdata.UserTokenPayload{UserId: 9, TenantId: 2, RoleCode: "operator"}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	var original string
	original, err = resolver.RevealStorageField(ctx, "base_tenant", "contact_phone", 2)
	if err != nil || original != tenant.ContactPhone {
		t.Fatalf("授权租户不能查看旁表原文: %v", err)
	}
	host := &securityHostCredential{ID: 20, TenantID: 2, ProjectID: 7, Secret: "business-secret"}
	err = client.AutoMigrate(new(securityHostCredential))
	if err != nil {
		t.Fatal(err)
	}
	err = resolver.RegisterStorageModels(t.Context(), new(securityHostCredential))
	if err != nil {
		t.Fatal(err)
	}
	var encryption redact.FieldPolicy
	encryption, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "ENCRYPT", `{"encrypt":{"algorithm":"AES_GCM"}}`)
	if err != nil {
		t.Fatal(err)
	}
	resolver.storagePolicies[storagePolicyKey(2, kitgorm.DefaultClientName, "security_host_credential")] = []redact.StorageFieldPolicy{{ID: 2021, TenantID: 2, TableName: "security_host_credential", ColumnName: "secret", Rule: encryption}}
	err = client.DB.WithContext(ctx).Create(host).Error
	if err != nil {
		t.Fatal(err)
	}
	allowedProjects := map[int64][]int64{2: {7}}
	resolver.SetProjectScopeLoader(func(context.Context) (kitgorm.ProjectScope, error) {
		return kitgorm.ProjectScope{Tenants: allowedProjects}, nil
	})
	original, err = resolver.RevealStorageField(ctx, "security_host_credential", "secret", 20)
	if err != nil || original != "business-secret" {
		t.Fatalf("宿主注册凭据原文查看回归: %v", err)
	}
	identity.TenantId, identity.TenantCode = 1, kitgorm.DefaultTenantCode
	ctx = engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	original, err = resolver.RevealStorageField(ctx, "security_host_credential", "secret", 20)
	if err != nil || original != "business-secret" {
		t.Fatalf("明确授权的跨租户项目原文查看回归: %v", err)
	}
	allowedProjects = map[int64][]int64{2: {8}}
	_, err = resolver.RevealStorageField(ctx, "security_host_credential", "secret", 20)
	if err == nil {
		t.Fatal("项目授权撤销后仍能查看原文")
	}
	identity.TenantId = 3
	identity.TenantCode = "tenant-three"
	ctx = engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	_, err = resolver.RevealStorageField(ctx, "base_tenant", "contact_phone", 2)
	if err == nil {
		t.Fatal("跨租户原文查看未拒绝")
	}
	_, err = resolver.RevealStorageField(ctx, "base_user;drop table base_user", "phone", 10)
	if err == nil {
		t.Fatal("未注册模型被接受")
	}
}

// securityHostCredential 模拟宿主注册的凭据表，不依赖任何具体业务仓库。
type securityHostCredential struct {
	ID        int64 `gorm:"primaryKey"`
	TenantID  int64
	ProjectID int64
	Secret    string
}
