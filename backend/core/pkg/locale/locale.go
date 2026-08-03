package locale

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

var (
	// Default 表示编译期语言包的默认回退区域，由 manifest 提供。
	Default          string
	supportedLocales []string
	supportedTags    []language.Tag
	matcher          language.Matcher
)

//go:embed manifest.json
var localeManifest []byte

type localeManifestConfig struct {
	Default string   `json:"default"`
	Locales []string `json:"locales"`
}

func init() {
	var config localeManifestConfig
	var err error
	err = json.Unmarshal(localeManifest, &config)
	if err != nil {
		panic(fmt.Errorf("解析语言 manifest: %w", err))
	}
	if config.Default == "" {
		panic(fmt.Errorf("语言 manifest 缺少默认语言"))
	}
	Default = config.Default
	seen := make(map[string]struct{}, len(config.Locales))
	for _, value := range config.Locales {
		if value == "" {
			panic(fmt.Errorf("语言 manifest 包含空语言代码"))
		}
		if _, ok := seen[value]; ok {
			panic(fmt.Errorf("语言 manifest 包含重复语言代码 %s", value))
		}
		var tag language.Tag
		tag, err = language.Parse(value)
		if err != nil {
			panic(fmt.Errorf("解析语言代码 %s: %w", value, err))
		}
		seen[value] = struct{}{}
		supportedLocales = append(supportedLocales, value)
		supportedTags = append(supportedTags, tag)
	}
	if _, ok := seen[Default]; !ok {
		panic(fmt.Errorf("语言 manifest 缺少默认语言 %s", Default))
	}
	matcher = language.NewMatcher(supportedTags)
}

type contextKey struct{}

// Supported 返回编译期语言包白名单副本；运行时启用列表由 base_language 提供。
func Supported() []string {
	return append([]string(nil), supportedLocales...)
}

// NonDefault 返回除项目回退语言之外的区域副本。
func NonDefault() []string {
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
