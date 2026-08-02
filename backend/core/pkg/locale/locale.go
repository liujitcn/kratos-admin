// Package locale 提供项目统一的语言区域解析与上下文传递能力。
package locale

import (
	"context"
	"strings"

	"golang.org/x/text/language"
)

const (
	// ZhCN 表示简体中文语言区域。
	ZhCN = "zh-CN"
	// EnUS 表示美式英语语言区域。
	EnUS = "en-US"
	// JaJP 表示日语语言区域。
	JaJP = "ja-JP"
	// Default 表示项目默认语言区域。
	Default = ZhCN
)

var (
	supportedLocales = []string{ZhCN, EnUS, JaJP}
	supportedTags    = []language.Tag{language.SimplifiedChinese, language.AmericanEnglish, language.Japanese}
	matcher          = language.NewMatcher(supportedTags)
)

type contextKey struct{}

// Supported 返回支持的语言区域副本。
func Supported() []string {
	return append([]string(nil), supportedLocales...)
}

// IsSupported 判断语言区域是否属于项目白名单。
func IsSupported(value string) bool {
	_, ok := normalize(value)
	return ok
}

// Normalize 将单个语言区域规范化为项目白名单值。
func Normalize(value string) string {
	normalized, ok := normalize(value)
	if !ok {
		return Default
	}
	return normalized
}

// ResolveAcceptLanguage 按请求头权重解析首个受支持语言区域。
func ResolveAcceptLanguage(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", "-")
	if value == "" {
		return Default
	}
	tags, _, err := language.ParseAcceptLanguage(value)
	if err != nil || len(tags) == 0 {
		return Default
	}
	_, index, confidence := matcher.Match(tags...)
	if confidence == language.No {
		return Default
	}
	return supportedLocales[index]
}

// WithContext 将规范语言区域写入上下文。
func WithContext(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, contextKey{}, Normalize(value))
}

// FromContext 从上下文读取规范语言区域。
func FromContext(ctx context.Context) string {
	value, ok := ctx.Value(contextKey{}).(string)
	if !ok {
		return Default
	}
	return Normalize(value)
}

// normalize 解析单个语言区域，并区分未知值与默认语言。
func normalize(value string) (string, bool) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", "-")
	if value == "" {
		return "", false
	}
	tag, err := language.Parse(value)
	if err != nil {
		return "", false
	}
	_, index, confidence := matcher.Match(tag)
	if confidence == language.No {
		return "", false
	}
	return supportedLocales[index], true
}
