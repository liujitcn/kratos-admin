package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestLocalizedAiSessionTextUsesRequestLocale 验证 AI 会话默认标题和摘要覆盖全部内置语言。
func TestLocalizedAiSessionTextUsesRequestLocale(t *testing.T) {
	catalog, err := corei18n.NewI18n("ai-session-default-text-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		locale string
		key    string
		want   string
	}{
		{locale: "zh-CN", key: "base.ai.session.default_title", want: "新对话"},
		{locale: "en-US", key: "base.ai.session.default_title", want: "New conversation"},
		{locale: "zh-TW", key: "base.ai.session.branch_title", want: "分支對話"},
		{locale: "ja-JP", key: "base.ai.session.branch_title", want: "分岐会話"},
		{locale: "en-US", key: "base.ai.session.default_summary", want: "New conversation"},
	}
	for _, test := range tests {
		if got := localizedAiSessionText(catalog, test.locale, test.key, "fallback"); got != test.want {
			t.Errorf("%s / %s = %q, want %q", test.locale, test.key, got, test.want)
		}
	}
}
