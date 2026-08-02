package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"

	"github.com/liujitcn/go-utils/translator"
)

var (
	protoValidationTokenPattern = regexp.MustCompile(`\b(id|message):\s*"((?:\\.|[^"\\])*)"`)
	templatePlaceholderPattern  = regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}`)
)

// CatalogMessage 描述 go-i18n JSON 中的单条消息。
type CatalogMessage struct {
	Other string `json:"other"`
}

// CatalogCheckResult 汇总国际化目录检查结果。
type CatalogCheckResult struct {
	SourceCount int
	LocaleCount int
}

// DraftResult 汇总错误目录缺失草稿数量。
type DraftResult struct {
	MissingByLocale map[string]int
	Written         int
}

// CheckCatalogFiles 扫描 Proto、Go 源码和三语目录并校验完整性。
func CheckCatalogFiles(backendRoot string) (*CatalogCheckResult, error) {
	sources, explicitKeys, err := collectCatalogSources(backendRoot)
	if err != nil {
		return nil, err
	}
	var catalogs map[string]map[string]CatalogMessage
	catalogs, err = readCatalogs(backendRoot)
	if err != nil {
		return nil, err
	}
	zhCatalog := catalogs[coreLocale.ZhCN]
	for key, source := range sources {
		message, ok := zhCatalog[key]
		if !ok {
			return nil, fmt.Errorf("zh-CN 缺少消息键 %s", key)
		}
		if message.Other != source {
			return nil, fmt.Errorf("zh-CN 消息键 %s 与 Proto 源文不一致", key)
		}
	}
	for key := range explicitKeys {
		if _, ok := zhCatalog[key]; !ok {
			return nil, fmt.Errorf("zh-CN 缺少 Go 显式消息键 %s", key)
		}
	}
	referenceKeys := sortedCatalogKeys(zhCatalog)
	for _, localeValue := range coreLocale.Supported() {
		catalog := catalogs[localeValue]
		keys := sortedCatalogKeys(catalog)
		if !slices.Equal(keys, referenceKeys) {
			return nil, fmt.Errorf("%s 与 zh-CN 的消息键集合不一致", localeValue)
		}
		for _, key := range referenceKeys {
			if !slices.Equal(messagePlaceholders(catalog[key].Other), messagePlaceholders(zhCatalog[key].Other)) {
				return nil, fmt.Errorf("%s 的消息键 %s 占位符集合不一致", localeValue, key)
			}
		}
	}
	return &CatalogCheckResult{SourceCount: len(sources), LocaleCount: len(referenceKeys)}, nil
}

// DraftCatalogFiles 报告或补齐错误目录中缺失的英日译文。
func DraftCatalogFiles(ctx context.Context, backendRoot string, provider translator.Translator, write bool) (*DraftResult, error) {
	sources, _, err := collectCatalogSources(backendRoot)
	if err != nil {
		return nil, err
	}
	var catalogs map[string]map[string]CatalogMessage
	catalogs, err = readCatalogs(backendRoot)
	if err != nil {
		return nil, err
	}
	result := &DraftResult{MissingByLocale: make(map[string]int)}
	for key, source := range sources {
		if _, ok := catalogs[coreLocale.ZhCN][key]; !ok {
			result.MissingByLocale[coreLocale.ZhCN]++
			if write {
				catalogs[coreLocale.ZhCN][key] = CatalogMessage{Other: source}
			}
		}
	}
	translations := make(map[string]string)
	for _, localeValue := range []string{coreLocale.EnUS, coreLocale.JaJP} {
		for key, source := range sources {
			if _, ok := catalogs[localeValue][key]; ok {
				continue
			}
			result.MissingByLocale[localeValue]++
			if !write {
				continue
			}
			cacheKey := localeValue + "\x00" + source
			translated := translations[cacheKey]
			if translated == "" {
				translated, err = TranslateProtected(ctx, provider, source, coreLocale.ZhCN, localeValue)
				if err != nil {
					return nil, fmt.Errorf("翻译消息键 %s 到 %s: %w", key, localeValue, err)
				}
				translations[cacheKey] = translated
			}
			catalogs[localeValue][key] = CatalogMessage{Other: translated}
			result.Written++
		}
	}
	if !write {
		return result, nil
	}
	for _, localeValue := range coreLocale.Supported() {
		if err = writeCatalog(backendRoot, localeValue, catalogs[localeValue]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// collectCatalogSources 收集 Proto 校验消息和 Go 显式消息键。
func collectCatalogSources(backendRoot string) (map[string]string, map[string]struct{}, error) {
	sources := make(map[string]string)
	err := filepath.WalkDir(filepath.Join(backendRoot, "api", "proto"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}
		return collectProtoCatalogSources(path, sources)
	})
	if err != nil {
		return nil, nil, err
	}
	var explicitKeys map[string]struct{}
	explicitKeys, err = collectGoMessageKeys(backendRoot)
	if err != nil {
		return nil, nil, err
	}
	return sources, explicitKeys, nil
}

// collectProtoCatalogSources 收集单个 Proto 文件中的校验消息。
func collectProtoCatalogSources(path string, sources map[string]string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(content)
	matches := protoValidationTokenPattern.FindAllStringSubmatchIndex(text, -1)
	pendingID := ""
	pendingLine := 0
	for _, match := range matches {
		tokenType := text[match[2]:match[3]]
		var tokenValue string
		tokenValue, err = strconv.Unquote(`"` + text[match[4]:match[5]] + `"`)
		if err != nil {
			return fmt.Errorf("解析 %s 国际化校验标记: %w", path, err)
		}
		if tokenType == "id" {
			if pendingID != "" {
				return fmt.Errorf("%s:%d 的校验规则 %s 缺少 message", path, pendingLine, pendingID)
			}
			pendingID = tokenValue
			pendingLine = strings.Count(text[:match[0]], "\n") + 1
			continue
		}
		if pendingID == "" {
			continue
		}
		if previous, ok := sources[pendingID]; ok && previous != tokenValue {
			return fmt.Errorf("消息键 %s 存在不同中文源文", pendingID)
		}
		sources[pendingID] = tokenValue
		pendingID = ""
		pendingLine = 0
	}
	if pendingID != "" {
		return fmt.Errorf("%s:%d 的校验规则 %s 缺少 message", path, pendingLine, pendingID)
	}
	return nil
}

// collectGoMessageKeys 收集 Go 源码中显式声明的消息键。
func collectGoMessageKeys(backendRoot string) (map[string]struct{}, error) {
	keys := make(map[string]struct{})
	var err error
	err = filepath.WalkDir(backendRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		var parsed *ast.File
		parsed, err = parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "WithMessageKey" {
				return true
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			var key string
			key, err = strconv.Unquote(literal.Value)
			if err == nil && key != "" {
				keys[key] = struct{}{}
			}
			return true
		})
		return nil
	})
	return keys, err
}

// readCatalogs 读取全部受支持语言的错误目录。
func readCatalogs(backendRoot string) (map[string]map[string]CatalogMessage, error) {
	catalogs := make(map[string]map[string]CatalogMessage)
	for _, localeValue := range coreLocale.Supported() {
		path := catalogPath(backendRoot, localeValue)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		catalog := make(map[string]CatalogMessage)
		if err = json.Unmarshal(content, &catalog); err != nil {
			return nil, fmt.Errorf("解析 %s: %w", path, err)
		}
		catalogs[localeValue] = catalog
	}
	return catalogs, nil
}

// writeCatalog 按稳定格式写入单个语言的错误目录。
func writeCatalog(backendRoot string, localeValue string, catalog map[string]CatalogMessage) error {
	content, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	path := catalogPath(backendRoot, localeValue)
	if err = os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	return nil
}

// catalogPath 返回指定语言的错误目录路径。
func catalogPath(backendRoot string, localeValue string) string {
	return filepath.Join(backendRoot, "internal", "i18n", "locales", localeValue+".json")
}

// sortedCatalogKeys 返回按字典序排列的消息键。
func sortedCatalogKeys(catalog map[string]CatalogMessage) []string {
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// messagePlaceholders 返回消息模板中规范化后的占位符集合。
func messagePlaceholders(message string) []string {
	matches := templatePlaceholderPattern.FindAllStringSubmatch(message, -1)
	placeholders := make([]string, 0, len(matches))
	for _, match := range matches {
		placeholders = append(placeholders, strings.ToLower(match[1]))
	}
	slices.Sort(placeholders)
	return placeholders
}
