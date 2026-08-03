package codegen

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
)

// codegenLocaleCatalog 描述代码生成器使用的单语言模板和术语。
type codegenLocaleCatalog struct {
	Menu             map[string]string `json:"menu"`
	Resource         map[string]string `json:"resource"`
	PasswordStrength string            `json:"passwordStrength"`
	Static           map[string]string `json:"static"`
}

//go:embed locales/catalog.json
var codegenLocaleCatalogFile []byte

var codegenLocaleCatalogs = loadCodegenLocaleCatalogs()

// loadCodegenLocaleCatalogs 加载代码生成器语言目录。
func loadCodegenLocaleCatalogs() map[string]codegenLocaleCatalog {
	catalogs := make(map[string]codegenLocaleCatalog)
	if err := json.Unmarshal(codegenLocaleCatalogFile, &catalogs); err != nil {
		panic(fmt.Errorf("解析代码生成器语言目录: %w", err))
	}
	for _, localeValue := range coreLocale.Supported() {
		catalog, ok := catalogs[localeValue]
		if !ok || catalog.Menu == nil || catalog.Resource == nil {
			panic(fmt.Errorf("代码生成器语言目录缺少 %s", localeValue))
		}
	}
	return catalogs
}

// codegenCatalog 返回指定语言目录，未知语言回退默认语言。
func codegenCatalog(localeValue string) codegenLocaleCatalog {
	if catalog, ok := codegenLocaleCatalogs[localeValue]; ok {
		return catalog
	}
	return codegenLocaleCatalogs[coreLocale.Default]
}

// renderCodegenTemplate 替换代码生成器模板中的稳定占位符。
func renderCodegenTemplate(template string, values map[string]string) string {
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "{"+key+"}", value)
	}
	return strings.NewReplacer(replacements...).Replace(template)
}
