package codegen

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/liujitcn/kratos-core/resource/i18n"
)

func TestGeneratedMenuI18nsReplacesMessagePlaceholders(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-test", fstest.MapFS{
		"en-US.json": &fstest.MapFile{Data: []byte(`{
  "common.resource.default": {"other": "{resource}"},
  "common.action.create_resource": {"other": "Create {resource}"}
}`)},
		"zh-CN.json": &fstest.MapFile{Data: []byte(`{
  "common.resource.default": {"other": "{resource}"},
  "common.action.create_resource": {"other": "新增{resource}"}
}`)},
	})
	if err != nil {
		t.Fatalf("创建测试国际化目录失败: %v", err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })

	table := &Table{
		TableComment:     "测试项目",
		BusinessName:     "test_item",
		I18NConfig:       map[string]LocaleConfig{"en-US": {Comment: "Test Item"}},
		PermissionPrefix: "app:test:item",
	}
	state := LocaleState{Current: "zh-CN", Primary: "zh-CN", Enabled: []string{"zh-CN", "en-US"}}

	i18ns := GeneratedMenuI18ns(table, nil, "", state)
	if got := i18ns["en-US"]; got != "Test Item" {
		t.Fatalf("页面菜单译文 = %q, want %q", got, "Test Item")
	}

	buttonI18ns := GeneratedMenuI18ns(table, nil, "create", state)
	if got := buttonI18ns["en-US"]; got != "Create Test Item" {
		t.Fatalf("新增按钮译文 = %q, want %q", got, "Create Test Item")
	}
}

func TestRenderGeneratedMenuSQLUsesLocalizedResourceName(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-sql-test", fstest.MapFS{
		"en-US.json": &fstest.MapFile{Data: []byte(`{
  "common.resource.default": {"other": "{resource}"}
}`)},
		"zh-CN.json": &fstest.MapFile{Data: []byte(`{
  "common.resource.default": {"other": "{resource}"}
}`)},
	})
	if err != nil {
		t.Fatalf("创建测试国际化目录失败: %v", err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })

	table := &Table{
		TableName_:       "test_item",
		TableComment:     "测试项目",
		BusinessName:     "test_item",
		PermissionPrefix: "app:test:item",
		ParentMenuID:     950,
		I18NConfig:       map[string]LocaleConfig{"en-US": {Comment: "Test Item"}},
	}
	state := LocaleState{Current: "zh-CN", Primary: "zh-CN", Enabled: []string{"zh-CN", "en-US"}}

	sql := RenderGeneratedMenuSQL(table, nil, nil, "app/test/item", "测试项目", state)
	if !strings.Contains(sql, "'Test Item'") {
		t.Fatalf("生成菜单 SQL 未包含英文资源名: %s", sql)
	}
	if strings.Contains(sql, "'{resource}'") {
		t.Fatalf("生成菜单 SQL 仍包含未替换资源占位符: %s", sql)
	}
}

func TestFrontendLocaleMessagesFallsBackToCurrentLocale(t *testing.T) {
	table := &Table{
		TableComment:     "项目管理",
		BusinessName:     "project",
		BusinessModule:   "system",
		PermissionPrefix: "system:project",
		I18NConfig: map[string]LocaleConfig{
			"en-US": {Comment: "Project Management"},
		},
	}
	columns := []*CodeGenColumn{
		{
			Name:    "project_code",
			Comment: "项目编码",
			I18NConfig: map[string]LocaleConfig{
				"en-US": {Comment: "Project Code"},
			},
		},
	}

	messages := FrontendLocaleMessages(table, columns, "ja-JP", "en-US", "zh-CN")
	if got := messages["system.project.resource"]; got != "Project Management" {
		t.Fatalf("缺失日文表描述 = %q, want %q", got, "Project Management")
	}
	if got := messages["system.project.field.project_code"]; got != "Project Code" {
		t.Fatalf("缺失日文字段描述 = %q, want %q", got, "Project Code")
	}
}
