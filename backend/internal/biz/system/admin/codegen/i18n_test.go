package codegen

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	locales "github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/errorsx"
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

// TestProtoStatusMessagesLocalizeForEveryLocale 验证 Proto 检查表状态说明使用请求语言。
func TestProtoStatusMessagesLocalizeForEveryLocale(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-proto-status-test", locales.Assets())
	if err != nil {
		t.Fatal(err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })
	tests := []struct {
		locale string
		key    string
		want   string
	}{
		{locale: "zh-CN", key: "proto.route_unavailable", want: "无法推导接口路由"},
		{locale: "en-US", key: "proto.route_unavailable", want: "The API route cannot be determined"},
		{locale: "zh-TW", key: "proto.missing_select_generate", want: "缺少，可選擇生成"},
		{locale: "ja-JP", key: "proto.missing_select_generate", want: "不足しています。生成対象として選択できます"},
		{locale: "en-US", key: "common.exists", want: "Exists"},
	}
	for _, test := range tests {
		state := LocaleState{Current: test.locale, Primary: "zh-CN"}
		if got := Message(state, test.key, nil); got != test.want {
			t.Errorf("locale %s / %s = %q, want %q", test.locale, test.key, got, test.want)
		}
	}
}

// TestNoGoFilesToFormatMessageLocalizesForEveryLocale 验证无须格式化的进度提示使用当前语言。
func TestNoGoFilesToFormatMessageLocalizesForEveryLocale(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-no-go-files-to-format-test", locales.Assets())
	if err != nil {
		t.Fatal(err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "本次没有需要格式化的 Go 文件"},
		{locale: "en-US", want: "No changed Go files need formatting"},
		{locale: "zh-TW", want: "本次沒有需要格式化的 Go 檔案"},
		{locale: "ja-JP", want: "今回整形が必要な Go ファイルはありません"},
	}
	for _, test := range tests {
		state := LocaleState{Current: test.locale, Primary: "zh-CN"}
		if got := Message(state, "progress.no_go_files_to_format", nil); got != test.want {
			t.Errorf("locale %s formatting progress = %q, want %q", test.locale, got, test.want)
		}
	}
}

// TestRenderGeneratedMenuSQLUsesLocalizedResourceName 验证菜单脚本使用配置的语言资源名称。
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
		ParentMenuID:     95000000,
		I18NConfig:       map[string]LocaleConfig{"en-US": {Comment: "Test Item"}},
	}
	state := LocaleState{Current: "zh-CN", Primary: "zh-CN", Enabled: []string{"zh-CN", "en-US"}}

	var sql string
	sql, err = RenderGeneratedMenuSQL(table, nil, nil, "app/test/item", "测试项目", state)
	if err != nil {
		t.Fatal(err)
	}
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

// TestResolveCodeGenOutputPathsLocalizesInvalidAndDuplicateFields 验证路径错误仍标明具体冲突字段且随请求语言本地化。
func TestResolveCodeGenOutputPathsLocalizesInvalidAndDuplicateFields(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-output-path-test", locales.Assets())
	if err != nil {
		t.Fatal(err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })
	tests := []struct {
		locale        string
		invalid       string
		duplicatePath string
	}{
		{locale: "zh-CN", invalid: "Proto文件路径无效", duplicatePath: "后端Biz文件路径不能与Proto文件路径使用相同路径"},
		{locale: "en-US", invalid: "The Proto file path is invalid", duplicatePath: "Backend Biz file path cannot use the same path as Proto file path"},
		{locale: "zh-TW", invalid: "Proto 檔案路徑無效", duplicatePath: "後端Biz檔案路徑不能與Proto 檔案路徑使用相同路徑"},
		{locale: "ja-JP", invalid: "Protoファイルパスが無効です", duplicatePath: "バックエンドBizファイルパスはProtoファイルパスと同じパスを使用できません"},
	}
	for _, test := range tests {
		state := LocaleState{Current: test.locale, Primary: "zh-CN"}
		renderer := &renderer{localeState: state}
		table := &Table{EntityName: "User", BusinessModule: "demo", GenBackend: 1, GenFrontend: 1}
		_, err = renderer.resolveCodeGenOutputPaths(table, &adminv1.CodeGenOutputPaths{ProtoFilePath: "../invalid.proto"})
		if err == nil {
			t.Fatal("无效 Proto 路径必须返回错误")
		}
		localized := errors.FromError(i18n.LocalizeError(catalog, test.locale, "zh-CN", err))
		if localized.Message != test.invalid {
			t.Errorf("locale %s invalid path message = %q, want %q", test.locale, localized.Message, test.invalid)
		}
		_, err = renderer.resolveCodeGenOutputPaths(table, &adminv1.CodeGenOutputPaths{
			ProtoFilePath:      "same.go",
			BackendBizFilePath: "same.go",
		})
		if err == nil {
			t.Fatal("重复生成路径必须返回错误")
		}
		localized = errors.FromError(i18n.LocalizeError(catalog, test.locale, "zh-CN", err))
		if localized.Message != test.duplicatePath {
			t.Errorf("locale %s duplicate path message = %q, want %q", test.locale, localized.Message, test.duplicatePath)
		}
	}
}

// TestResolveCodeGenOutputPathsLocalizesFieldLabels 验证重复路径错误以当前语言保留两个冲突目标。
func TestResolveCodeGenOutputPathsLocalizesFieldLabels(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-output-path-duplicate-test", locales.Assets())
	if err != nil {
		t.Fatal(err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "后端Biz文件路径不能与Proto文件路径使用相同路径"},
		{locale: "en-US", want: "Backend Biz file path cannot use the same path as Proto file path"},
		{locale: "zh-TW", want: "後端Biz檔案路徑不能與Proto 檔案路徑使用相同路徑"},
		{locale: "ja-JP", want: "バックエンドBizファイルパスはProtoファイルパスと同じパスを使用できません"},
	}
	for _, test := range tests {
		state := LocaleState{Current: test.locale, Primary: "zh-CN"}
		renderer := &renderer{localeState: state}
		table := &Table{EntityName: "User", BusinessModule: "demo", GenBackend: 1, GenFrontend: 1}
		_, err = renderer.resolveCodeGenOutputPaths(table, &adminv1.CodeGenOutputPaths{
			ProtoFilePath:      "same.go",
			BackendBizFilePath: "same.go",
		})
		if err == nil {
			t.Fatal("重复生成路径必须返回错误")
		}
		localized := errors.FromError(i18n.LocalizeError(catalog, test.locale, "zh-CN", err))
		if localized.Message != test.want {
			t.Errorf("locale %s duplicate path message = %q, want %q", test.locale, localized.Message, test.want)
		}
	}
}

// TestFailureRemarkLocalizesStructuredErrorsAndHidesUnclassifiedDetails 验证任务错误摘要遵循语言状态且不泄露未分类底层错误。
func TestFailureRemarkLocalizesStructuredErrorsAndHidesUnclassifiedDetails(t *testing.T) {
	catalog, err := i18n.NewI18n("codegen-failure-remark-test", locales.Assets())
	if err != nil {
		t.Fatal(err)
	}
	SetCatalog(catalog)
	t.Cleanup(func() { SetCatalog(nil) })

	state := LocaleState{Current: "en-US", Primary: "zh-CN"}
	conflict := errorsx.WithMessageKey(
		errorsx.Conflict("批量生成定义冲突"),
		"system.code.gen.error.batch.definition_conflict",
		map[string]string{"Previous": "users", "Current": "roles", "Key": "rpc.proto:UserService.Create"},
	)
	if got, want := FailureRemark(state, conflict), "Tables users and roles define different content for rpc.proto:UserService.Create"; got != want {
		t.Fatalf("localized failure remark = %q, want %q", got, want)
	}
	if got, want := FailureRemark(state, fmt.Errorf("数据库连接失败: secret")), "The operation failed. Review diagnostic output or service logs for details."; got != want {
		t.Fatalf("unclassified failure remark = %q, want %q", got, want)
	}
}
