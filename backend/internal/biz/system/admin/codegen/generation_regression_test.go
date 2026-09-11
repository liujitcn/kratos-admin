package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMenuSQLUsesInitialMigration 验证已执行初始化版本后仍更新固定版本脚本。
func TestMenuSQLUsesInitialMigration(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	err := os.MkdirAll(filepath.Join(root, "backend/migration/assets/v0.0.1/mysql"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	path, err = nextGeneratedMenuSQLPath("v0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if path != "backend/migration/assets/v0.0.1/mysql/default_data.up.sql" || !isGeneratedMenuSQLPath(path) {
		t.Fatalf("迁移路径错误: %s", path)
	}
}

// TestMergeFrontendObjectPropertyOrder 验证旧页属性顺序不同时嵌套扩展不会破坏源码。
func TestMergeFrontendObjectPropertyOrder(t *testing.T) {
	candidate := `{ prop: "project_code", props: { placeholder: t("common.validation.required_input", { field: t("project_code") }) }, search: { el: "input" } }`
	existing := `{ prop: "project_code", search: { el: "input", custom: true }, props: { clearable: true } }`
	result, ok := mergeFrontendObject(candidate, existing)
	if !ok || !strings.Contains(result, `placeholder: t("common.validation.required_input", { field: t("project_code") })`) || !strings.Contains(result, "clearable: true") || !strings.Contains(result, "custom: true") {
		t.Fatalf("合并损坏: %s", result)
	}
}

// TestMenuRouteMatchesPage 验证菜单、SQL 与前端模块注册的组件键一致。
func TestMenuRouteMatchesPage(t *testing.T) {
	table := &Table{EntityName: "TenantProject", BusinessModule: "system", ModulePath: "tenant/project", ParentMenuID: 20010000}
	renderer := &renderer{}
	paths, err := renderer.resolveCodeGenOutputPaths(table, nil)
	if err != nil {
		t.Fatal(err)
	}
	component := FrontendPageComponentPath(paths.GetFrontendPageFilePath())
	page, _ := MenuSpecs(table, nil, nil, component, "租户项目", LocaleState{})
	if page.Menu.Component != "system/tenant/project/index" || page.Menu.Path != "/system/tenant/project" {
		t.Fatalf("菜单与页面不一致: %#v, %s", page.Menu, paths.GetFrontendPageFilePath())
	}
	sql := RenderGeneratedMenuSQL(table, nil, nil, component, "租户项目", LocaleState{})
	if !strings.Contains(sql, sqlString(page.Menu.Component)) || !strings.Contains(sql, sqlString(page.Menu.Path)) || !strings.Contains(sql, "INSERT INTO `base_menu` (`id`, `parent_id`") {
		t.Fatal("SQL 与在线菜单契约不一致")
	}
}

// TestMenuSQLMergePreservesOtherTables 验证重复生成只替换本表片段并保留初始化数据。
func TestMenuSQLMergePreservesOtherTables(t *testing.T) {
	table := &Table{TableName_: "tenant_project"}
	original := "-- 原有初始化内容\n" + generatedMenuSQLBlock(&Table{TableName_: "another"}, "SELECT 1;")
	first, err := mergeGeneratedMenuSQLAtPath(original, table, "SELECT 2;", "default_data.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	var second string
	second, err = mergeGeneratedMenuSQLAtPath(first, table, "SELECT 3;", "default_data.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(second, original) || strings.Contains(second, "SELECT 2;") || strings.Count(second, "table=tenant_project") != 2 {
		t.Fatal(second)
	}
}

// TestFrontendPageActionTypes 验证按钮权限不会错位写入 TypeScript 类型。
func TestFrontendPageActionTypes(t *testing.T) {
	c := &renderer{}
	table := &Table{EntityName: "TenantProject", BusinessModule: "system", BusinessName: "租户项目", ModulePath: "tenant/project"}
	paths, err := c.resolveCodeGenOutputPaths(table, nil)
	if err != nil {
		t.Fatal(err)
	}
	if paths.GetFrontendApiFilePath() != "frontend/admin/packages/modules/system/src/api/system/admin/v1/tenant_project.ts" {
		t.Fatal(paths.GetFrontendApiFilePath())
	}
	page := c.renderFrontendPageFile(table, nil, nil, paths)
	if !strings.Contains(page, "@liujitcn/kratos-admin-system/api/system/admin/v1/tenant_project") {
		t.Fatal("页面 API 导入未保留完整协议目录")
	}
	for _, expected := range []string{"handleDelete(scope.row as TenantProject)", "handleDelete(scope.selectedList as TenantProject[])", `BUTTONS.value["` + PermissionPrefix(table) + `:delete"]`, `BUTTONS.value["` + PermissionPrefix(table) + `:create"]`} {
		if !strings.Contains(page, expected) {
			t.Fatalf("生成页面缺少 %s", expected)
		}
	}
	if strings.Contains(page, "%!") {
		t.Fatal("模板参数未匹配")
	}
}
