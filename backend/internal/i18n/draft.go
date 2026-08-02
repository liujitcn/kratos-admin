package i18n

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/liujitcn/go-utils/translator"
)

var (
	protectedTextPattern  = regexp.MustCompile("(?s)```.*?```|`[^`]+`|\\{\\{[^{}]+\\}\\}|\\$\\{[^{}]+\\}|\\{[A-Za-z_][A-Za-z0-9_.-]*\\}|%[sdv]|</?[^>]+>|https?://[^\\s<>()]+|[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}|/(?:api|events|mcp|v[0-9]+)/[A-Za-z0-9_./:{}-]+|(?i:kratos-admin)")
	protectedTokenPattern = regexp.MustCompile(`__KRATOS_I18N_TOKEN_[0-9]{3}__`)
)

// TranslateProtected 使用稳定哨兵保护结构化片段后调用单文本翻译器。
func TranslateProtected(ctx context.Context, provider translator.Translator, source, sourceLocale, targetLocale string) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("翻译草稿提供方未配置")
	}
	protectedSource, values := protectText(source)
	translated, err := provider.Translate(ctx, protectedSource, sourceLocale, targetLocale)
	if err != nil {
		return "", fmt.Errorf("生成翻译草稿: %w", err)
	}
	for index, value := range values {
		token := protectedToken(index)
		if strings.Count(translated, token) != 1 {
			return "", fmt.Errorf("翻译草稿哨兵 %s 数量不一致", token)
		}
		translated = strings.Replace(translated, token, value, 1)
	}
	if protectedTokenPattern.MatchString(translated) {
		return "", fmt.Errorf("翻译草稿包含未恢复哨兵")
	}
	return translated, nil
}

// protectText 将需要原样保留的片段替换为稳定哨兵。
func protectText(source string) (string, []string) {
	values := make([]string, 0)
	protected := protectedTextPattern.ReplaceAllStringFunc(source, func(value string) string {
		index := len(values)
		values = append(values, value)
		return protectedToken(index)
	})
	return protected, values
}

// protectedToken 生成固定宽度的翻译哨兵。
func protectedToken(index int) string {
	return fmt.Sprintf("__KRATOS_I18N_TOKEN_%03d__", index)
}
