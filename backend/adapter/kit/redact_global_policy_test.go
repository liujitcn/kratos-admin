package kit

import (
	"context"
	"testing"
	"time"

	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// newDefaultTenantPolicyTestResolver 构造同时配置默认租户（全局）和普通租户策略的解析器。
func newDefaultTenantPolicyTestResolver(t *testing.T) *RedactPolicyResolver {
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
			// 默认租户策略即全局策略，覆盖 phone 和 id_code；租户2单独配置 phone 和 email。
			outputPolicyKey(1, operation, "system.admin.v1.BaseUser.phone"):   maskPolicy,
			outputPolicyKey(1, operation, "system.admin.v1.BaseUser.id_code"): fixedPolicy,
			outputPolicyKey(2, operation, "system.admin.v1.BaseUser.phone"):   fixedPolicy,
			outputPolicyKey(2, operation, "system.admin.v1.BaseUser.email"):   maskPolicy,
		},
		storagePolicies: map[string][]redact.StorageFieldPolicy{
			storagePolicyKey(1, kitgorm.DefaultClientName, "base_user"): {
				{ID: 11, TenantID: 1, TableName: "base_user", ColumnName: "phone", Rule: maskPolicy},
				{ID: 12, TenantID: 1, TableName: "base_user", ColumnName: "id_code", Rule: fixedPolicy},
			},
			storagePolicyKey(2, kitgorm.DefaultClientName, "base_user"): {
				{ID: 21, TenantID: 2, TableName: "base_user", ColumnName: "phone", Rule: fixedPolicy},
			},
		},
	}
}

// TestOutputPolicyDefaultTenantAppliesWithoutTenant 验证响应未携带租户编号时应用默认租户的全局策略。
func TestOutputPolicyDefaultTenantAppliesWithoutTenant(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 0, Phone: "13800138000", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "138****8000" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("无租户响应应回退默认租户策略: %+v", user)
	}
}

// TestOutputPolicyTenantOverridesDefaultTenant 验证租户策略覆盖同字段默认租户策略并保留其余全局策略。
func TestOutputPolicyTenantOverridesDefaultTenant(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 2, Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "XXXXXXXXXXX" || user.Email != "ali**********.com" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("租户策略应覆盖默认租户策略并补齐未覆盖字段: %+v", user)
	}
}

// TestOutputPolicyDefaultTenantDataUsesOwnPolicies 验证默认租户自己的响应数据应用默认租户策略。
func TestOutputPolicyDefaultTenantDataUsesOwnPolicies(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	user := &adminv1.BaseUser{TenantId: 1, Phone: "13800138000", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", user)
	if user.Phone != "138****8000" || user.IdCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("默认租户响应应使用默认租户策略: %+v", user)
	}
}

// TestListStoragePoliciesMergesDefaultAndTenant 验证入库策略按字段合并且租户覆盖默认租户策略。
func TestListStoragePoliciesMergesDefaultAndTenant(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 2, "base_user")
	if len(policies) != 2 {
		t.Fatalf("合并后应包含两个字段策略: %+v", policies)
	}
	ruleByID := make(map[int64]redact.StorageFieldPolicy, len(policies))
	for _, policy := range policies {
		ruleByID[policy.ID] = policy
	}
	// 租户2的 phone 策略覆盖默认租户策略。
	if ruleByID[21].Rule.RuleType != "FIXED_LENGTH" {
		t.Fatalf("租户 phone 策略应覆盖默认租户策略: %+v", ruleByID)
	}
	if _, ok := ruleByID[11]; ok {
		t.Fatalf("被覆盖的默认租户 phone 策略不应出现: %+v", ruleByID)
	}
	if _, ok := ruleByID[12]; !ok {
		t.Fatalf("未被覆盖的默认租户 id_code 策略应保留: %+v", ruleByID)
	}
}

// TestListStoragePoliciesDefaultOnlyWithoutTenant 验证租户为零时仅应用默认租户策略。
func TestListStoragePoliciesDefaultOnlyWithoutTenant(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 0, "base_user")
	if len(policies) != 2 {
		t.Fatalf("无租户应仅包含默认租户策略: %+v", policies)
	}
	for _, policy := range policies {
		if policy.TenantID != 1 {
			t.Fatalf("无租户不应返回其他租户策略: %+v", policy)
		}
	}
}

// TestListStoragePoliciesFallbackToDefaultTenant 验证未配置策略的租户回退默认租户策略。
func TestListStoragePoliciesFallbackToDefaultTenant(t *testing.T) {
	resolver := newDefaultTenantPolicyTestResolver(t)
	policies := resolver.ListStoragePolicies(context.Background(), 3, "base_user")
	if len(policies) != 2 {
		t.Fatalf("无策略租户应回退默认租户策略: %+v", policies)
	}
}

// TestMergeStoragePoliciesIgnoresColumnCase 验证字段覆盖匹配忽略列名大小写。
func TestMergeStoragePoliciesIgnoresColumnCase(t *testing.T) {
	global := []redact.StorageFieldPolicy{{ID: 1, TenantID: 1, ColumnName: "PHONE"}}
	tenant := []redact.StorageFieldPolicy{{ID: 2, TenantID: 2, ColumnName: "phone"}}
	merged := mergeStoragePolicies(global, tenant)
	if len(merged) != 1 || merged[0].ID != 2 {
		t.Fatalf("租户策略应按列名覆盖全局策略: %+v", merged)
	}
}
