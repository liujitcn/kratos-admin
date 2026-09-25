package i18n

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"
)

// TestLocalizeRedactRuleDisableConflicts 验证停用脱敏规则时返回具体的引用策略提示。
func TestLocalizeRedactRuleDisableConflicts(t *testing.T) {
	catalog, err := i18n.NewI18n("redact-rule-disable-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		messageKey string
		message    string
	}{
		{name: "入库策略引用", messageKey: "system.admin.base.redact_rule.disable.storage_policy_in_use", message: "已有启用的入库脱敏策略引用该规则，请先停用引用策略"},
		{name: "出库策略引用", messageKey: "system.admin.base.redact_rule.disable.output_policy_in_use", message: "已有启用的出库脱敏策略引用该规则，请先停用引用策略"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = errorsx.WithMessageKey(errorsx.ProtectedResourceConflict(test.message, "base_redact_rule"), test.messageKey, nil)
			localized := errors.FromError(i18n.LocalizeError(catalog, "zh-CN", "zh-CN", err))
			if localized.Message != test.message {
				t.Fatalf("本地化冲突消息 = %q, want %q", localized.Message, test.message)
			}
		})
	}
}

// TestLocalizeGeneratedDeleteHasChildrenConflict 验证生成的子资源冲突提示支持全部内置语言。
func TestLocalizeGeneratedDeleteHasChildrenConflict(t *testing.T) {
	catalog, err := i18n.NewI18n("generated-delete-conflict-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	message := "删除{{.Parent}}失败，下面有{{.Child}}"
	err = errorsx.WithMessageKey(
		errorsx.HasChildrenConflict(message, "base_category", "base_category"),
		"system.code.gen.error.has_children",
		map[string]string{"Parent": "分类", "Child": "分类"},
	)
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "删除分类失败，下面有分类"},
		{locale: "en-US", want: "Cannot delete 分类 because it still contains 分类 records."},
		{locale: "zh-TW", want: "刪除分类失敗，其下仍有分类。"},
		{locale: "ja-JP", want: "分类を削除できません。配下に分类が残っています。"},
	}
	for _, test := range tests {
		localized := errors.FromError(i18n.LocalizeError(catalog, test.locale, "zh-CN", err))
		if localized.Message != test.want {
			t.Errorf("locale %s: localized conflict message = %q, want %q", test.locale, localized.Message, test.want)
		}
	}
}

// TestLocalizeThirdAccountConflictMessages 验证三种唯一索引冲突按当前语言说明具体绑定关系。
func TestLocalizeThirdAccountConflictMessages(t *testing.T) {
	catalog, err := i18n.NewI18n("third-account-conflict-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		messageKey   string
		message      string
		translations map[string]string
	}{
		{
			messageKey: "base.third_account.error.binding_exists",
			message:    "三方账号绑定关系已存在",
			translations: map[string]string{
				"zh-CN": "三方账号绑定关系已存在",
				"en-US": "This third-party account binding already exists",
				"zh-TW": "第三方帳號綁定關係已存在",
				"ja-JP": "サードパーティアカウントの連携はすでに存在します",
			},
		},
		{
			messageKey: "base.third_account.error.current_user_provider_bound",
			message:    "当前用户已绑定该登录方式",
			translations: map[string]string{
				"zh-CN": "当前用户已绑定该登录方式",
				"en-US": "The current user has already linked this sign-in method",
				"zh-TW": "目前使用者已綁定此登入方式",
				"ja-JP": "現在のユーザーはこのログイン方法をすでに連携しています",
			},
		},
		{
			messageKey: "base.third_account.error.account_bound_to_other_user",
			message:    "三方账号已被其他用户绑定",
			translations: map[string]string{
				"zh-CN": "三方账号已被其他用户绑定",
				"en-US": "This third-party account is already linked to another user",
				"zh-TW": "此第三方帳號已被其他使用者綁定",
				"ja-JP": "このサードパーティアカウントは別のユーザーに連携されています",
			},
		},
	}
	locales := []string{"zh-CN", "en-US", "zh-TW", "ja-JP"}
	for _, test := range tests {
		err = errorsx.WithMessageKey(errorsx.UniqueConflict(test.message, "base_third_account", "", ""), test.messageKey, nil)
		for _, locale := range locales {
			localized := errors.FromError(i18n.LocalizeError(catalog, locale, "zh-CN", err))
			if localized.Message != test.translations[locale] {
				t.Errorf("%s / %s = %q, want %q", test.messageKey, locale, localized.Message, test.translations[locale])
			}
		}
	}
}

// TestLocalizeOAuthProviderValidationMessages 验证自定义 Proto 校验 ID 在全部语言下保留具体字段和规则提示。
func TestLocalizeOAuthProviderValidationMessages(t *testing.T) {
	catalog, err := i18n.NewI18n("oauth-provider-validation-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	locales := []string{"zh-CN", "en-US", "zh-TW", "ja-JP"}
	tests := []struct {
		ruleID       string
		source       string
		translations map[string]string
	}{
		{
			ruleID: "system.admin.base.oauth_provider.get.id.required", source: "OAuth登录方式ID不能为空",
			translations: map[string]string{"zh-CN": "OAuth登录方式ID不能为空", "en-US": "OAuth provider ID is required", "zh-TW": "OAuth登入方式ID不可為空", "ja-JP": "OAuthログイン方式IDを指定してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.delete.id.required", source: "OAuth登录方式ID列表不能为空",
			translations: map[string]string{"zh-CN": "OAuth登录方式ID列表不能为空", "en-US": "At least one OAuth provider ID is required", "zh-TW": "OAuth登入方式ID清單不可為空", "ja-JP": "OAuthログイン方式IDを1件以上指定してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.set_status.id.required", source: "OAuth登录方式ID不能为空",
			translations: map[string]string{"zh-CN": "OAuth登录方式ID不能为空", "en-US": "OAuth provider ID is required", "zh-TW": "OAuth登入方式ID不可為空", "ja-JP": "OAuthログイン方式IDを指定してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.provider.required", source: "Provider标识格式无效",
			translations: map[string]string{"zh-CN": "Provider标识格式无效", "en-US": "The Provider identifier is invalid", "zh-TW": "Provider識別格式無效", "ja-JP": "Provider識別子の形式が無効です"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.name.required", source: "登录方式名称不能为空且不能超过50个字符",
			translations: map[string]string{"zh-CN": "登录方式名称不能为空且不能超过50个字符", "en-US": "The sign-in method name is required and must not exceed 50 characters", "zh-TW": "登入方式名稱不可為空且不得超過50個字元", "ja-JP": "ログイン方式名を入力してください（50文字以内）"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.description.length", source: "登录方式提示语不能超过255个字符",
			translations: map[string]string{"zh-CN": "登录方式提示语不能超过255个字符", "en-US": "The sign-in method hint must not exceed 255 characters", "zh-TW": "登入方式提示語不得超過255個字元", "ja-JP": "ログイン方式のヒントは255文字以内で入力してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.icon.required", source: "图标不能为空且不能超过255个字符",
			translations: map[string]string{"zh-CN": "图标不能为空且不能超过255个字符", "en-US": "The icon is required and must not exceed 255 characters", "zh-TW": "圖示不可為空且不得超過255個字元", "ja-JP": "アイコンを入力してください（255文字以内）"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.client_id.required", source: "第三方应用标识不能为空且不能超过255个字符",
			translations: map[string]string{"zh-CN": "第三方应用标识不能为空且不能超过255个字符", "en-US": "The third-party application ID is required and must not exceed 255 characters", "zh-TW": "第三方應用程式識別不可為空且不得超過255個字元", "ja-JP": "サードパーティアプリケーションIDを入力してください（255文字以内）"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.client_secret.length", source: "第三方应用密钥不能超过512个字符",
			translations: map[string]string{"zh-CN": "第三方应用密钥不能超过512个字符", "en-US": "The third-party application secret must not exceed 512 characters", "zh-TW": "第三方應用程式密鑰不得超過512個字元", "ja-JP": "サードパーティアプリケーションのシークレットは512文字以内で入力してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.redirect_uri.length", source: "OAuth回调地址不能超过512个字符",
			translations: map[string]string{"zh-CN": "OAuth回调地址不能超过512个字符", "en-US": "The OAuth redirect URI must not exceed 512 characters", "zh-TW": "OAuth回呼網址不得超過512個字元", "ja-JP": "OAuthリダイレクトURIは512文字以内で入力してください"},
		},
		{
			ruleID: "system.admin.base.oauth_provider.form.scopes.unique", source: "OAuth Scope不能重复",
			translations: map[string]string{"zh-CN": "OAuth Scope不能重复", "en-US": "OAuth scopes must not contain duplicates", "zh-TW": "OAuth Scope不可重複", "ja-JP": "OAuth Scopeに重複がないようにしてください"},
		},
	}
	for _, test := range tests {
		t.Run(test.ruleID, func(t *testing.T) {
			validationErr := errorsx.WithMessageKey(errorsx.InvalidArgument(test.source), test.ruleID, map[string]string{"Field": "provider"})
			for _, locale := range locales {
				localized := errors.FromError(i18n.LocalizeError(catalog, locale, "zh-CN", validationErr))
				if localized.Message != test.translations[locale] {
					t.Errorf("locale %s: validation message = %q, want %q", locale, localized.Message, test.translations[locale])
				}
			}
		})
	}
}

// TestLocalizeAgentToolDisabledMessage 验证 Agent 面向用户的错误消息按请求语言渲染参数。
func TestLocalizeAgentToolDisabledMessage(t *testing.T) {
	catalog, err := i18n.NewI18n("agent-tool-disabled-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	args := map[string]any{"Name": "search_public"}
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "Agent 工具 search_public 已被禁用，无法继续调用。"},
		{locale: "en-US", want: "Agent tool search_public is disabled and cannot be called."},
		{locale: "zh-TW", want: "Agent 工具 search_public 已停用，無法繼續呼叫。"},
		{locale: "ja-JP", want: "Agent ツール search_public は無効のため呼び出しを続行できません。"},
	}
	for _, test := range tests {
		if got := catalog.Localize(test.locale, "zh-CN", "base.ai.tool.disabled", args, "fallback"); got != test.want {
			t.Errorf("locale %s: localized message = %q, want %q", test.locale, got, test.want)
		}
	}
}

// TestLocalizeSchedulerExecutionConflict 验证 Core 调度器冲突原文按请求语言返回。
func TestLocalizeSchedulerExecutionConflict(t *testing.T) {
	catalog, err := i18n.NewI18n("scheduler-execution-conflict-test", Assets())
	if err != nil {
		t.Fatal(err)
	}
	message := "任务正在其他实例执行"
	conflict := errorsx.WithMessageKey(errorsx.Conflict(message), errorsx.MessageKey(message), nil)
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "任务正在其他实例执行"},
		{locale: "en-US", want: "The task is already running on another instance"},
		{locale: "zh-TW", want: "任務正在其他執行個體執行"},
		{locale: "ja-JP", want: "タスクは別のインスタンスで実行中です"},
	}
	for _, test := range tests {
		localized := errors.FromError(i18n.LocalizeError(catalog, test.locale, "zh-CN", conflict))
		if localized.Message != test.want {
			t.Errorf("locale %s: conflict message = %q, want %q", test.locale, localized.Message, test.want)
		}
	}
}
