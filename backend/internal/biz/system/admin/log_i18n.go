package biz

import (
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"
)

// localizeLogReason 仅翻译日志目录中已登记的固定错误文案，保留未知的动态诊断文本。
func localizeLogReason(catalog *i18n.I18n, locale, reasonCode, reason string) string {
	if catalog == nil || reason == "" {
		return reason
	}
	messageKey, ok := catalog.KeyForSource(reason)
	if !ok {
		if reasonCode == errorsx.ReasonInternalError {
			return catalog.Localize(locale, "zh-CN", "common.error.internal", nil, reason)
		}
		return reason
	}
	return catalog.Localize(locale, "zh-CN", messageKey, nil, reason)
}
