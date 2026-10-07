package kit

import (
	"context"
	"testing"
	"time"

	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// newGlobalPolicyTestResolver 构造同时配置全局和租户策略的解析器。
func newGlobalPolicyTestResolver(t *testing.T) *RedactPolicyResolver {
	t.Helper()
	maskPolicy, err := redact.NewFieldPolicy(redact.PolicyModeApplyRule, "MASK", `{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var fixedPolicy redact.FieldPolicy
	fixedPolicy, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "FIXED_LENGTH", `{"fixed_length":{"char":"X"}}`)
	if err != nil {
		t.Fatal(err)
	}
	operation := "/system.admin.v1.BaseUserService/ListBaseUser"
	return &RedactPolicyResolver{
		loadedAt: time.Now(),
		outputPolicies: map[string]redact.FieldPolicy{
			// 全局策略覆盖 phone 和 id_code，租户1单独配置 phone 和 email。
			outputPolicyKey(0, operation, "system.admin.v1.BaseUser.phone"):   maskPolicy,
			outputPolicyKey(0, operation, "system.admin.v1.BaseUser.id_code"): fixedPolicy,
			outputPolicyKey(1, operation, "system.admin.v1.BaseUser.phone"):   fixedPolicy,
			outputPolicyKey(1, operation, "system.admin.v1.BaseUser.email"):   maskPolicy,
		},
		storagePolicies: map[string][]redact.StorageFieldPolicy{
			storagePolicyKey(0, kitgorm.DefaultClientName, "base_user"): {
				{ID: 11, TenantID: 0, TableName: "base_user", ColumnName: "phone", Rule: maskPolicy},
				{ID: 12, TenantID: 0, TableName: "base_user", ColumnName: "id_code", Rule: fixedPolicy},
			},
			storagePolicyKey(1, kitgorm.DefaultClientName, "base_user"): {
				{ID: 21, TenantID: 1, TableName: "base_user", ColumnName: "phone", Rule: fixedPolicy},
			},
		},
	}
}

// TestOutputPolicyFallsBackToGlobalWithoutTenant 验证响应未携带租户编号时应用全局策略。
func TestOutputPolicyFallsBackToGlobalWithoutTenant(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 0, Phone: "13800138000", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "138****8000" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("无租户响应应回退全局策略: %+v", user)
	}
}

// TestOutputPolicyTenantOverridesGlobal 验证租户策略覆盖同字段全局策略并保留其余全局策略。
func TestOutputPolicyTenantOverridesGlobal(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 1, Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "XXXXXXXXXXX" || user.Email != "ali**********.com" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("租户策略应覆盖全局并补齐未覆盖字段: %+v", user)
	}
}

// TestOutputPolicyFallbackForTenantWithoutPolicies 验证未配置策略的租户回退全局策略。
func TestOutputPolicyFallbackForTenantWithoutPolicies(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 2, Phone: "13800138000", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "138****8000" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("无策略租户应回退全局策略: %+v", user)
	}
}

// TestListStoragePoliciesMergesGlobalAndTenant 验证入库策略按字段合并且租户覆盖全局。
func TestListStoragePoliciesMergesGlobalAndTenant(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 1, "base_user")
	if len(policies) != 2 {
		t.Fatalf("合并后应包含两个字段策略: %+v", policies)
	}
	ruleByID := make(map[int64]redact.StorageFieldPolicy, len(policies))
	for _, policy := range policies {
		ruleByID[policy.ID] = policy
	}
	// 租户1的 phone 策略覆盖全局策略。
	if ruleByID[21].Rule.RuleType != "FIXED_LENGTH" {
		t.Fatalf("租户 phone 策略应覆盖全局策略: %+v", ruleByID)
	}
	if _, ok := ruleByID[11]; ok {
		t.Fatalf("被覆盖的全局 phone 策略不应出现: %+v", ruleByID)
	}
	if _, ok := ruleByID[12]; !ok {
		t.Fatalf("未被覆盖的全局 id_code 策略应保留: %+v", ruleByID)
	}
}

// TestListStoragePoliciesGlobalOnlyWithoutTenant 验证租户为零时仅应用全局入库策略。
func TestListStoragePoliciesGlobalOnlyWithoutTenant(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 0, "base_user")
	if len(policies) != 2 {
		t.Fatalf("无租户应仅包含全局策略: %+v", policies)
	}
	for _, policy := range policies {
		if policy.TenantID != 0 {
			t.Fatalf("无租户不应返回租户策略: %+v", policy)
		}
	}
}

// TestListStoragePoliciesFallbackToGlobal 验证未配置策略的租户回退全局入库策略。
func TestListStoragePoliciesFallbackToGlobal(t *testing.T) {
	resolver := newGlobalPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 2, "base_user")
	if len(policies) != 2 {
		t.Fatalf("无策略租户应回退全局策略: %+v", policies)
	}
}

// TestMergeStoragePoliciesIgnoresColumnCase 验证字段覆盖匹配忽略列名大小写。
func TestMergeStoragePoliciesIgnoresColumnCase(t *testing.T) {
	global := []redact.StorageFieldPolicy{{ID: 1, TenantID: 0, ColumnName: "PHONE"}}
	tenant := []redact.StorageFieldPolicy{{ID: 2, TenantID: 1, ColumnName: "phone"}}
	merged := mergeStoragePolicies(global, tenant)
	if len(merged) != 1 || merged[0].ID != 2 {
		t.Fatalf("租户策略应按列名覆盖全局策略: %+v", merged)
	}
}
