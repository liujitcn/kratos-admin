package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Catalog 表示由宿主语言文件构建的只读消息目录。
type Catalog struct {
	bundle        *goi18n.Bundle
	defaultLocale string
	sourceKeys    map[string]string
}

// NewCatalog 从宿主提供的文件系统加载语言消息目录。
func NewCatalog(files fs.FS, supportedLocales []string, defaultLocale string) (*Catalog, error) {
	if files == nil {
		return nil, fmt.Errorf("国际化文件系统未配置")
	}
	if defaultLocale == "" {
		return nil, fmt.Errorf("国际化默认语言未配置")
	}
	if len(supportedLocales) == 0 {
		return nil, fmt.Errorf("国际化支持语言未配置")
	}
	var err error
	var defaultTag language.Tag
	defaultTag, err = language.Parse(defaultLocale)
	if err != nil {
		return nil, fmt.Errorf("解析国际化默认语言 %s: %w", defaultLocale, err)
	}
	bundle := goi18n.NewBundle(defaultTag)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	seen := make(map[string]struct{}, len(supportedLocales))
	sourceKeys := make(map[string]string)
	for _, localeValue := range supportedLocales {
		if localeValue == "" {
			return nil, fmt.Errorf("国际化目录包含空语言代码")
		}
		if _, ok := seen[localeValue]; ok {
			return nil, fmt.Errorf("国际化目录包含重复语言代码 %s", localeValue)
		}
		_, err = language.Parse(localeValue)
		if err != nil {
			return nil, fmt.Errorf("解析国际化语言 %s: %w", localeValue, err)
		}
		seen[localeValue] = struct{}{}
		name := localeValue + ".json"
		var data []byte
		data, err = fs.ReadFile(files, name)
		if err != nil {
			return nil, fmt.Errorf("读取国际化目录 %s: %w", name, err)
		}
		_, err = bundle.ParseMessageFileBytes(data, name)
		if err != nil {
			return nil, fmt.Errorf("解析国际化目录 %s: %w", name, err)
		}
		if localeValue == defaultLocale {
			var messages map[string]struct {
				Other string `json:"other"`
			}
			err = json.Unmarshal(data, &messages)
			if err != nil {
				return nil, fmt.Errorf("解析国际化源文 %s: %w", name, err)
			}
			for messageKey, message := range messages {
				if message.Other == "" {
					continue
				}
				if currentKey, ok := sourceKeys[message.Other]; !ok || messageKey < currentKey {
					sourceKeys[message.Other] = messageKey
				}
			}
		}
	}
	if _, ok := seen[defaultLocale]; !ok {
		return nil, fmt.Errorf("国际化目录缺少默认语言 %s", defaultLocale)
	}
	return &Catalog{bundle: bundle, defaultLocale: defaultLocale, sourceKeys: sourceKeys}, nil
}

// KeyForSource 返回默认语言中与源文完全匹配的消息键。
func (c *Catalog) KeyForSource(source string) (string, bool) {
	if c == nil || source == "" {
		return "", false
	}
	key, ok := c.sourceKeys[source]
	return key, ok
}

// HasMessage 判断消息目录中是否存在指定消息键。
func (c *Catalog) HasMessage(messageKey string) bool {
	if c == nil || c.bundle == nil || messageKey == "" {
		return false
	}
	localizer := goi18n.NewLocalizer(c.bundle, c.defaultLocale)
	message, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: messageKey})
	return err == nil && message != ""
}

// Localize 按语言区域和消息键渲染消息，缺少译文时回退默认语言和安全文本。
func (c *Catalog) Localize(localeValue, messageKey string, messageArgs map[string]any, fallback string) string {
	if c == nil || c.bundle == nil || messageKey == "" {
		return fallbackMessage(fallback)
	}
	localizer := goi18n.NewLocalizer(c.bundle, localeValue)
	message, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
	if err == nil && message != "" {
		return message
	}
	localizer = goi18n.NewLocalizer(c.bundle, c.defaultLocale)
	message, err = localizer.Localize(&goi18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
	if err == nil && message != "" {
		return message
	}
	return fallbackMessage(fallback)
}

// fallbackMessage 返回安全的目录缺失兜底消息。
func fallbackMessage(fallback string) string {
	if fallback != "" {
		return fallback
	}
	return "系统内部错误"
}
