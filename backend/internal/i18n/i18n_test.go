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
