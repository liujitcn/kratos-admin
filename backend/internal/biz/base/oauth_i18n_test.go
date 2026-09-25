package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/errorsx"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestLocalizeOAuthRedirectErrorMessagePreservesCallbackDetail 验证 OAuth 回调业务错误按语言本地化。
func TestLocalizeOAuthRedirectErrorMessagePreservesCallbackDetail(t *testing.T) {
	catalog, err := corei18n.NewI18n("oauth-callback-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		key          string
		source       string
		translations map[string]string
	}{
		{
			name:   "unbound account",
			key:    "base.oauth.callback.account_unbound",
			source: "三方账号未绑定，请先使用账号密码登录后到个人中心绑定",
			translations: map[string]string{
				"zh-CN": "三方账号未绑定，请先使用账号密码登录后到个人中心绑定",
				"en-US": "This third-party account is not linked. Sign in with your account and password, then link it from your profile.",
				"zh-TW": "第三方帳號尚未綁定，請先使用帳號密碼登入，再至個人中心綁定",
				"ja-JP": "このサードパーティアカウントは未連携です。ユーザー名とパスワードでログインしてから、プロフィールで連携してください。",
			},
		},
		{
			name:   "provider already bound",
			key:    "base.oauth.binding_callback.current_user_provider_bound",
			source: "当前用户已绑定该登录方式",
			translations: map[string]string{
				"zh-CN": "当前用户已绑定该登录方式",
				"en-US": "The current user has already linked this sign-in method",
				"zh-TW": "目前使用者已綁定此登入方式",
				"ja-JP": "現在のユーザーはこのログイン方法をすでに連携しています",
			},
		},
	}
	locales := []string{"zh-CN", "en-US", "zh-TW", "ja-JP"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callbackErr := newOauthCallbackError(test.key, test.source)
			for _, locale := range locales {
				if got := localizeOAuthRedirectErrorMessage(catalog, locale, "zh-CN", callbackErr); got != test.translations[locale] {
					t.Errorf("locale %s: callback message = %q, want %q", locale, got, test.translations[locale])
				}
			}
		})
	}
	conflict := errorsx.UniqueConflict("手机号已被占用", "base_user", "phone", "unique_base_user_phone")
	if got := localizeOAuthRedirectErrorMessage(catalog, "en-US", "zh-CN", conflict); got != "The phone number is already in use" {
		t.Errorf("localized typed conflict = %q", got)
	}
	internal := errorsx.Internal("internal oauth failure detail")
	if got := localizeOAuthRedirectErrorMessage(catalog, "en-US", "zh-CN", internal); got != "internal oauth failure detail" {
		t.Errorf("internal error detail changed = %q", got)
	}
}
