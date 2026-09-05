package kit

import (
	"context"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-kit/redact"
)

// TestBaseUserOutputPolicies 验证用户列表、分页和详情接口按精确 operation 脱敏。
func TestBaseUserOutputPolicies(t *testing.T) {
	phonePolicy, err := redact.NewFieldPolicy(redact.PolicyModeApplyRule, "MASK", `{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var emailPolicy redact.FieldPolicy
	emailPolicy, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "EMAIL", `{"email":{"keep_local_first":2,"mask_domain":false,"mask_char":"*"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var idCodePolicy redact.FieldPolicy
	idCodePolicy, err = redact.NewFieldPolicy(redact.PolicyModeApplyRule, "FIXED_LENGTH", `{"fixed_length":{"char":"X"}}`)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewRedactPolicyResolver(nil, nil, nil)
	resolver.loadedAt = time.Now()
	resolver.outputPolicies = make(map[string]redact.FieldPolicy)
	for _, operation := range []string{
		"/system.admin.v1.BaseUserService/ListBaseUser",
		"/system.admin.v1.BaseUserService/PageBaseUser",
	} {
		resolver.outputPolicies[outputPolicyKey(operation, "system.admin.v1.BaseUser.phone")] = phonePolicy
		resolver.outputPolicies[outputPolicyKey(operation, "system.admin.v1.BaseUser.email")] = emailPolicy
		resolver.outputPolicies[outputPolicyKey(operation, "system.admin.v1.BaseUser.id_code")] = idCodePolicy
	}
	detailOperation := "/system.admin.v1.BaseUserService/GetBaseUser"
	resolver.outputPolicies[outputPolicyKey(detailOperation, "system.admin.v1.BaseUserForm.phone")] = phonePolicy
	resolver.outputPolicies[outputPolicyKey(detailOperation, "system.admin.v1.BaseUserForm.email")] = emailPolicy
	resolver.outputPolicies[outputPolicyKey(detailOperation, "system.admin.v1.BaseUserForm.id_code")] = idCodePolicy

	listUser := &adminv1.BaseUser{Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	listResponse := &adminv1.ListBaseUserResponse{BaseUsers: []*adminv1.BaseUser{listUser}}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/ListBaseUser", listResponse)
	assertBaseUserRedacted(t, listUser.Phone, listUser.Email, listUser.IdCode)

	pageUser := &adminv1.BaseUser{Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	pageResponse := &adminv1.PageBaseUserResponse{BaseUsers: []*adminv1.BaseUser{pageUser}}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/PageBaseUser", pageResponse)
	assertBaseUserRedacted(t, pageUser.Phone, pageUser.Email, pageUser.IdCode)

	detail := &adminv1.BaseUserForm{Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, detailOperation, detail)
	assertBaseUserRedacted(t, detail.Phone, detail.Email, detail.IdCode)
}

// TestUnconfiguredBaseUserOperationKeepsOriginalValue 验证未配置接口不会命中隐式字段策略。
func TestUnconfiguredBaseUserOperationKeepsOriginalValue(t *testing.T) {
	resolver := NewRedactPolicyResolver(nil, nil, nil)
	resolver.loadedAt = time.Now()
	user := &adminv1.BaseUser{Phone: "13800138000", Email: "alice@example.com", IdCode: "411381199401282014"}
	applyOutputPolicy(resolver, "/system.admin.v1.BaseUserService/OptionBaseUser", user)
	if user.Phone != "13800138000" || user.Email != "alice@example.com" || user.IdCode != "411381199401282014" {
		t.Fatalf("未配置接口不应脱敏: %+v", user)
	}
}

// applyOutputPolicy 使用响应方向和指定接口执行动态脱敏。
func applyOutputPolicy(resolver *RedactPolicyResolver, operation string, message any) {
	ctx := redact.WithDirection(redact.WithOperation(context.Background(), operation), redact.DirectionResponse)
	redact.ApplyWith(ctx, resolver, message)
}

// assertBaseUserRedacted 校验用户三个敏感字段的预期输出。
func assertBaseUserRedacted(t *testing.T, phone, email, idCode string) {
	t.Helper()
	if phone != "138****8000" || email != "al***@example.com" || idCode != "XXXXXXXXXXXXXXXXXX" {
		t.Fatalf("用户敏感字段脱敏错误: phone=%q email=%q id_code=%q", phone, email, idCode)
	}
}
