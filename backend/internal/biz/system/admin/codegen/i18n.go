package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"

	"github.com/liujitcn/go-utils/stringcase"
)

// RequiredTranslationLocales 返回代码生成正式写入所需的非主语言。
func RequiredTranslationLocales() []string {
	return coreLocale.NonDefault()
}

// GeneratedFrontendLocales 返回生成页面需要同步的全部语言区域。
func GeneratedFrontendLocales() []string {
	return coreLocale.Supported()
}

// MissingTranslationFields 返回正式生成前尚未填写的表级和字段级翻译。
func MissingTranslationFields(table *Table, columns []*CodeGenColumn) []string {
	if table == nil {
		return []string{"表配置"}
	}
	missing := make([]string, 0)
	leftTreeEnabled := LeftTreeConfigFromTable(table).Enabled
	for _, localeValue := range RequiredTranslationLocales() {
		config := table.I18NConfig[localeValue]
		if config.Comment == "" {
			missing = append(missing, fmt.Sprintf("表描述（%s）", localeValue))
		}
		if leftTreeEnabled && config.LeftTreeComment == "" {
			missing = append(missing, fmt.Sprintf("左树描述（%s）", localeValue))
		}
		for _, column := range columns {
			if column != nil && column.I18NConfig[localeValue].Comment == "" {
				missing = append(missing, fmt.Sprintf("字段 %s（%s）", column.Name, localeValue))
			}
		}
	}
	sort.Strings(missing)
	return missing
}

// FrontendLocaleKeyPrefix 返回生成页面拥有的稳定语言键前缀。
func FrontendLocaleKeyPrefix(table *Table) string {
	if table == nil {
		return "generated.unknown"
	}
	parts := strings.Split(PermissionPrefix(table), ":")
	if len(parts) > 0 && parts[0] == table.BusinessModule {
		parts = parts[1:]
	}
	pathParts := []string{frontendLocalePath(table.BusinessModule)}
	for _, part := range parts {
		pathParts = append(pathParts, frontendLocalePath(part))
	}
	return strings.Join(pathParts, ".")
}

// FrontendLocaleMessages 构建单个生成页面在指定语言下拥有的全部固定文案。
func FrontendLocaleMessages(table *Table, columns []*CodeGenColumn, localeValue string) map[string]string {
	prefix := FrontendLocaleKeyPrefix(table)
	resource := localizedTableComment(table, localeValue)
	messages := localizedResourceMessages(prefix, resource, localeValue)
	for _, column := range columns {
		if column == nil {
			continue
		}
		messages[prefix+".field."+stringcase.ToSnakeCase(column.Name)] = localizedColumnComment(column, localeValue)
		if column.FormComponent == "password" {
			messages[prefix+".field.password_strength"] = localizedPasswordStrength(localeValue)
		}
		for _, option := range enabledCodeGenColumnOptions(column) {
			if option.SourceType != OptionSourceStatic {
				continue
			}
			for _, item := range parseCodeGenStaticOptions(option) {
				messages[frontendStaticOptionLocaleKey(table, column, item.Value)] = localizedGeneratedStaticLabel(item.Label, localeValue)
			}
		}
	}
	if LeftTreeConfigFromTable(table).Enabled {
		messages[prefix+".title.left_tree"] = localizedLeftTreeComment(table, localeValue)
	}
	return messages
}

// newFrontendLocalePreviewFiles 创建语言包的结构化合并预览。
func (c *renderer) newFrontendLocalePreviewFiles(table *Table, columns []*CodeGenColumn) []*systemadminv1.CodeGenPreviewFile {
	target := ProtoTargetForTable(table)
	files := make([]*systemadminv1.CodeGenPreviewFile, 0, len(GeneratedFrontendLocales()))
	for _, localeValue := range GeneratedFrontendLocales() {
		path := target.FrontendLocaleFilePath(localeValue)
		messages := FrontendLocaleMessages(table, columns, localeValue)
		files = append(files, c.newMergedFrontendLocalePreviewFile(path, FrontendLocaleKeyPrefix(table), messages))
	}
	return files
}

// renderFrontendLocaleMessages 把扁平语言包稳定渲染为 JSON。
func renderFrontendLocaleMessages(messages map[string]string) (string, error) {
	content, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return "", err
	}
	return string(content) + "\n", nil
}

// mergeFrontendLocaleMessages 只替换当前生成对象拥有的语言键。
func mergeFrontendLocaleMessages(content string, prefix string, owned map[string]string) (string, error) {
	messages := make(map[string]string)
	err := json.Unmarshal([]byte(content), &messages)
	if err != nil {
		return "", fmt.Errorf("解析现有语言包失败: %w", err)
	}
	ownedPrefix := prefix + "."
	for key := range messages {
		if strings.HasPrefix(key, ownedPrefix) {
			delete(messages, key)
		}
	}
	for key, value := range owned {
		messages[key] = value
	}
	return renderFrontendLocaleMessages(messages)
}

// LocalizedTableComment 返回指定语言的业务名称。
func LocalizedTableComment(table *Table, localeValue string) string {
	return localizedTableComment(table, localeValue)
}

// LocalizedColumnComment 返回指定语言的字段名称。
func LocalizedColumnComment(column *CodeGenColumn, localeValue string) string {
	return localizedColumnComment(column, localeValue)
}

// GeneratedMenuTranslations 返回页面或按钮菜单的非主语言标题。
func GeneratedMenuTranslations(table *Table, column *CodeGenColumn, action string) map[string]string {
	translations := make(map[string]string, len(RequiredTranslationLocales()))
	for _, localeValue := range RequiredTranslationLocales() {
		catalog := codegenCatalog(localeValue)
		resource := localizedTableComment(table, localeValue)
		values := map[string]string{"resource": resource}
		if action == "status" {
			values["field"] = localizedColumnComment(column, localeValue)
		}
		template := catalog.Menu[action]
		if template == "" {
			template = catalog.Menu["default"]
		}
		translations[localeValue] = renderCodegenTemplate(template, values)
	}
	return translations
}

func localizedResourceMessages(prefix string, resource string, localeValue string) map[string]string {
	catalog := codegenCatalog(localeValue)
	messages := make(map[string]string, len(catalog.Resource))
	for key, template := range catalog.Resource {
		messages[prefix+"."+key] = renderCodegenTemplate(template, map[string]string{"resource": resource})
	}
	return messages
}

func localizedTableComment(table *Table, localeValue string) string {
	if table == nil {
		return ""
	}
	if localeValue == coreLocale.Default {
		return DefaultString(table.TableComment, table.BusinessName)
	}
	return DefaultString(table.I18NConfig[localeValue].Comment, DefaultString(table.TableComment, table.BusinessName))
}

func localizedLeftTreeComment(table *Table, localeValue string) string {
	config := LeftTreeConfigFromTable(table)
	if localeValue == coreLocale.Default {
		return DefaultString(config.Comment, localizedTableComment(table, localeValue))
	}
	return DefaultString(table.I18NConfig[localeValue].LeftTreeComment, DefaultString(config.Comment, localizedTableComment(table, localeValue)))
}

func localizedColumnComment(column *CodeGenColumn, localeValue string) string {
	if column == nil {
		return ""
	}
	if localeValue == coreLocale.Default {
		return DefaultString(column.Comment, column.Name)
	}
	return DefaultString(column.I18NConfig[localeValue].Comment, DefaultString(column.Comment, column.Name))
}

func localizedPasswordStrength(localeValue string) string {
	catalog := codegenCatalog(localeValue)
	if catalog.PasswordStrength != "" {
		return catalog.PasswordStrength
	}
	return codegenCatalog(coreLocale.Default).PasswordStrength
}

// frontendStaticOptionLocaleKey 返回静态选项值对应的稳定语言键。
func frontendStaticOptionLocaleKey(table *Table, column *CodeGenColumn, value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		content = []byte(fmt.Sprint(value))
	}
	digest := sha256.Sum256(content)
	return FrontendLocaleKeyPrefix(table) + ".option." + stringcase.ToSnakeCase(column.Name) + ".value" + hex.EncodeToString(digest[:6])
}

func frontendLocalePath(value string) string {
	return strings.ReplaceAll(stringcase.ToSnakeCase(value), "_", ".")
}

func parseCodeGenStaticOptions(option CodeGenColumnOptionConfig) []CodeGenStaticOption {
	var options []CodeGenStaticOption
	if json.Unmarshal([]byte(option.SourceValue), &options) != nil {
		return nil
	}
	return options
}

func localizedGeneratedStaticLabel(label string, localeValue string) string {
	if translated := codegenCatalog(localeValue).Static[label]; translated != "" {
		return translated
	}
	return label
}
