package locale

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

var (
	// Default 表示宿主配置的默认回退区域。
	Default          string
	supportedLocales []string
	supportedTags    []language.Tag
	matcher          language.Matcher
)

// Config 表示宿主提供的编译期语言区域配置。
type Config struct {
	// Default 表示宿主的默认回退语言。
	Default string
	// Supported 表示宿主编译期支持的语言集合。
	Supported []string
}

// Configure 配置当前宿主可用的编译期语言区域。
func Configure(config Config) error {
	if config.Default == "" {
		return fmt.Errorf("语言配置缺少默认语言")
	}
	if len(config.Supported) == 0 {
		return fmt.Errorf("语言配置缺少支持语言")
	}
	seen := make(map[string]struct{}, len(config.Supported))
	tags := make([]language.Tag, 0, len(config.Supported))
	var err error
	for _, value := range config.Supported {
		if value == "" {
			return fmt.Errorf("语言配置包含空语言代码")
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("语言配置包含重复语言代码 %s", value)
		}
		var tag language.Tag
		tag, err = language.Parse(value)
		if err != nil {
			return fmt.Errorf("解析语言代码 %s: %w", value, err)
		}
		seen[value] = struct{}{}
		tags = append(tags, tag)
	}
	if _, ok := seen[config.Default]; !ok {
		return fmt.Errorf("语言配置缺少默认语言 %s", config.Default)
	}
	Default = config.Default
	supportedLocales = append([]string(nil), config.Supported...)
	supportedTags = tags
	matcher = language.NewMatcher(tags)
	return nil
}

type contextKey struct{}

// Supported 返回编译期语言包白名单副本；运行时启用列表由 base_language 提供。
func Supported() []string {
	return append([]string(nil), supportedLocales...)
}

// NonDefault 返回除项目回退语言之外的区域副本。
func NonDefault() []string {
	if len(supportedLocales) == 0 {
		return nil
	}
	locales := make([]string, 0, len(supportedLocales)-1)
	for _, value := range supportedLocales {
		if value != Default {
			locales = append(locales, value)
		}
	}
	return locales
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
	if len(supportedLocales) == 0 {
		return Default
	}
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

// normalize 解析单个语言区域，并区分未知值与项目回退语言。
func normalize(value string) (string, bool) {
	if len(supportedLocales) == 0 {
		return "", false
	}
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
