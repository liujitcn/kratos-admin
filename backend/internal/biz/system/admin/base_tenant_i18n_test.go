package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestLocalizedTenantDefaultsUseAllLocales 验证新租户的系统默认名称和说明按创建语言初始化。
func TestLocalizedTenantDefaultsUseAllLocales(t *testing.T) {
	catalog, err := corei18n.NewI18n("tenant-default-i18n-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		locale string
		key    string
		want   string
	}{
		{locale: "zh-CN", key: "system.base.tenant.default_department_name", want: "默认部门"},
		{locale: "en-US", key: "system.base.tenant.default_department_name", want: "Default department"},
		{locale: "zh-TW", key: "system.base.tenant.default_admin_nickname", want: "管理員"},
		{locale: "ja-JP", key: "system.base.tenant.default_admin_nickname", want: "管理者"},
	}
	for _, test := range tests {
		if got := (&BaseTenantCase{catalog: catalog}).localizeTenantDefault(test.locale, test.key, "fallback"); got != test.want {
			t.Errorf("%s / %s = %q, want %q", test.locale, test.key, got, test.want)
		}
	}
}

// TestLocalizedTenantRoleTemplateTranslatesKnownDefaultsAndPreservesCustomText 验证角色模板只翻译已登记的固定源文。
func TestLocalizedTenantRoleTemplateTranslatesKnownDefaultsAndPreservesCustomText(t *testing.T) {
	catalog, err := corei18n.NewI18n("tenant-role-template-i18n-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	caseValue := &BaseTenantCase{catalog: catalog}
	if got, want := caseValue.localizeTenantTemplate("en-US", "租户管理员"), "Tenant administrator"; got != want {
		t.Fatalf("localized template role name = %q, want %q", got, want)
	}
	if got, want := caseValue.localizeTenantTemplate("en-US", "custom role name"), "custom role name"; got != want {
		t.Fatalf("custom role name = %q, want %q", got, want)
	}
}
