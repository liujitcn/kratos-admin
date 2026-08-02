// Package i18n 提供后端错误消息目录与本地化能力。
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"

	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFiles embed.FS

// Catalog 表示只读的后端错误消息目录。
type Catalog struct {
	bundle *goi18n.Bundle
}

var defaultCatalog = mustLoadCatalog()

// DefaultCatalog 返回项目默认错误消息目录。
func DefaultCatalog() *Catalog {
	return defaultCatalog
}

// Localize 按语言区域和消息键渲染错误消息。
func (c *Catalog) Localize(localeValue, messageKey string, messageArgs map[string]any, fallback string) string {
	if c == nil || c.bundle == nil || messageKey == "" {
		return fallbackMessage(fallback)
	}
	localizer := goi18n.NewLocalizer(c.bundle, coreLocale.Normalize(localeValue))
	message, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
	if err == nil && message != "" {
		return message
	}
	localizer = goi18n.NewLocalizer(c.bundle, coreLocale.Default)
	message, err = localizer.Localize(&goi18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
	if err == nil && message != "" {
		return message
	}
	return fallbackMessage(fallback)
}

// mustLoadCatalog 加载嵌入式三语错误目录，目录损坏属于启动期不可恢复错误。
func mustLoadCatalog() *Catalog {
	bundle := goi18n.NewBundle(language.SimplifiedChinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	files, err := fs.Glob(localeFiles, "locales/*.json")
	if err != nil {
		panic(fmt.Errorf("扫描后端国际化目录: %w", err))
	}
	for _, name := range files {
		var data []byte
		data, err = localeFiles.ReadFile(name)
		if err != nil {
			panic(fmt.Errorf("读取后端国际化目录 %s: %w", name, err))
		}
		_, err = bundle.ParseMessageFileBytes(data, name)
		if err != nil {
			panic(fmt.Errorf("解析后端国际化目录 %s: %w", name, err))
		}
	}
	return &Catalog{bundle: bundle}
}

// fallbackMessage 返回安全的目录缺失兜底消息。
func fallbackMessage(fallback string) string {
	if fallback != "" {
		return fallback
	}
	return "系统内部错误"
}
