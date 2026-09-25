package biz

import (
	"testing"
	"testing/fstest"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/errorsx"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestLocalizeLogReasonTranslatesKnownSourcesAndKeepsDynamicData 验证日志固定错误按查看者语言翻译，动态诊断内容保留。
func TestLocalizeLogReasonTranslatesKnownSourcesAndKeepsDynamicData(t *testing.T) {
	catalog, err := corei18n.NewI18n("log-reason-test", fstest.MapFS{
		"en-US.json": &fstest.MapFile{Data: []byte(`{"legacy.error.login":{"other":"Invalid username or password"},"common.error.internal":{"other":"Internal error"}}`)},
		"zh-CN.json": &fstest.MapFile{Data: []byte(`{"legacy.error.login":{"other":"用户名或密码错误"}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := localizeLogReason(catalog, "en-US", "", "用户名或密码错误"), "Invalid username or password"; got != want {
		t.Fatalf("localized log reason = %q, want %q", got, want)
	}
	if got, want := localizeLogReason(catalog, "en-US", "", "driver connection 172.18.0.4:3306 refused"), "driver connection 172.18.0.4:3306 refused"; got != want {
		t.Fatalf("dynamic log reason = %q, want %q", got, want)
	}
	if got, want := localizeLogReason(catalog, "en-US", errorsx.ReasonInternalError, "驱动连接 172.18.0.4:3306 被拒绝"), "Internal error"; got != want {
		t.Fatalf("unknown internal log reason = %q, want %q", got, want)
	}
}

// TestLocalizeLoginPolicyDefaultReasons 验证登录策略固定拦截原因按查看者语言显示。
func TestLocalizeLoginPolicyDefaultReasons(t *testing.T) {
	catalog, err := corei18n.NewI18n("login-policy-reason-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	reasons := []string{
		"登录 IP 命中黑名单",
		"登录 MAC 命中黑名单",
		"登录地区命中黑名单",
		"当前时间不允许登录",
		"登录设备命中黑名单",
		"登录来源命中黑名单",
		"登录 IP 不在白名单",
		"登录 MAC 不在白名单",
		"登录地区不在白名单",
		"当前时间不在允许窗口",
		"登录设备不在白名单",
		"登录来源不在白名单",
	}
	for _, reason := range reasons {
		for _, locale := range []string{"en-US", "zh-TW", "ja-JP"} {
			if got := localizeLogReason(catalog, locale, "", reason); got == reason {
				t.Errorf("locale %s returned untranslated policy reason %q", locale, reason)
			}
		}
	}
}
