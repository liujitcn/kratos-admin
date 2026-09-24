package i18n

import (
	"fmt"
	"net/url"
)

// EncodeMessage 编码供异步任务和持久化记录使用的稳定国际化消息标记。
func EncodeMessage(key string, params map[string]string) string {
	message := "__I18N__:" + key
	if len(params) == 0 {
		return message
	}
	values := make(url.Values, len(params))
	for name, value := range params {
		values.Set(name, value)
	}
	return message + "?" + values.Encode()
}

// WrapMessageError 在保留内部错误 cause 的同时为任务日志添加稳定国际化消息标记。
func WrapMessageError(key string, cause error) error {
	if cause == nil {
		return nil
	}
	return fmt.Errorf("%s\n%w", EncodeMessage(key, nil), cause)
}
