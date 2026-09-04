package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/liujitcn/go-utils/stringcase"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/resource/i18n"
)

const codegenMessagePrefix = "system.code.gen."

var codegenCatalogValue *i18n.I18n

// SetCatalog 设置代码生成器使用的统一国际化目录。
func SetCatalog(catalog *i18n.I18n) {
	codegenCatalogValue = catalog
}

// LocaleState 描述代码生成使用的数据库语言状态。
type LocaleState struct {
	// Current 是当前请求使用的语言。
	Current string
	// Enabled 是数据库中启用的语言。
	Enabled []string
	// Primary 是数据库中的主语言。
	Primary string
}

// Message 返回代码生成器当前语言的固定文案。
func Message(state LocaleState, key string, values map[string]string) string {
	return localizeCodegen(state.Current, state.Primary, "message."+key, values, key)
}

// RequiredI18nLocales 返回代码生成正式写入所需的非主语言。
func RequiredI18nLocales(state LocaleState) []string {
	result := make([]string, 0, len(state.Enabled))
	for _, localeValue := range state.Enabled {
		if localeValue != state.Primary {
			result = append(result, localeValue)
		}
	}
	return result
}

// GeneratedFrontendLocales 返回生成页面需要同步的全部语言区域。
func GeneratedFrontendLocales(state LocaleState) []string {
	return append([]string(nil), state.Enabled...)
}

// MissingI18nFields 返回正式生成前尚未填写的表级和字段级翻译。
func MissingI18nFields(table *Table, columns []*CodeGenColumn, state LocaleState) []string {
	if table == nil {
		return []string{Message(state, "missing.table_config", nil)}
	}
	missing := make([]string, 0)
	leftTreeEnabled := LeftTreeConfigFromTable(table).Enabled
	for _, localeValue := range RequiredI18nLocales(state) {
		config := table.I18NConfig[localeValue]
		if config.Comment == "" {
			missing = append(missing, Message(state, "missing.table_comment", map[string]string{"locale": localeValue}))
		}
		if leftTreeEnabled && config.LeftTreeComment == "" {
			missing = append(missing, Message(state, "missing.left_tree_comment", map[string]string{"locale": localeValue}))
		}
		for _, column := range columns {
			// 主键和软删除字段不在字段配置页展示，不能要求用户补齐不可编辑内容。
			if column == nil || column.IsPrimary == 1 || column.Name == "deleted_at" {
				continue
			}
			if column.I18NConfig[localeValue].Comment == "" {
				missing = append(missing, Message(state, "missing.field_comment", map[string]string{"field": column.Name, "locale": localeValue}))
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
func FrontendLocaleMessages(table *Table, columns []*CodeGenColumn, localeValue string, primaryLocale string) map[string]string {
	prefix := FrontendLocaleKeyPrefix(table)
	resource := localizedTableComment(table, localeValue, primaryLocale)
	messages := localizedResourceMessages(prefix, resource)
	for _, column := range columns {
		if column == nil {
			continue
		}
		messages[prefix+".field."+stringcase.ToSnakeCase(column.Name)] = localizedColumnComment(column, localeValue, primaryLocale)
		if column.FormComponent == "password" {
			messages[prefix+".field.password_strength"] = localizedPasswordStrength(localeValue, primaryLocale)
		}
		for _, option := range enabledCodeGenColumnOptions(column) {
			if option.SourceType != OptionSourceStatic {
				continue
			}
			for _, item := range parseCodeGenStaticOptions(option) {
				messages[frontendStaticOptionLocaleKey(table, column, item.Value)] = localizedGeneratedStaticLabel(item.Label, localeValue, primaryLocale)
			}
		}
	}
	if LeftTreeConfigFromTable(table).Enabled {
		messages[prefix+".title.left_tree"] = localizedLeftTreeComment(table, localeValue, primaryLocale)
	}
	return messages
}

// newFrontendLocalePreviewFiles 创建语言包的结构化合并预览。
func (c *renderer) newFrontendLocalePreviewFiles(table *Table, columns []*CodeGenColumn, state LocaleState) []*adminv1.CodeGenPreviewFile {
	target := ProtoTargetForTable(table)
	files := make([]*adminv1.CodeGenPreviewFile, 0, len(GeneratedFrontendLocales(state)))
	for _, localeValue := range GeneratedFrontendLocales(state) {
		path := target.FrontendLocaleFilePath(localeValue)
		messages := FrontendLocaleMessages(table, columns, localeValue, state.Primary)
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

// GeneratedMenuI18ns 返回页面或按钮菜单的非主语言标题。
func GeneratedMenuI18ns(table *Table, column *CodeGenColumn, action string, state LocaleState) map[string]string {
	i18ns := make(map[string]string, len(RequiredI18nLocales(state)))
	for _, localeValue := range RequiredI18nLocales(state) {
		resource := localizedTableComment(table, localeValue, state.Primary)
		values := map[string]string{"resource": resource}
		if action == "status" {
			values["field"] = localizedColumnComment(column, localeValue, state.Primary)
		}
		messageKey := map[string]string{
			"default": "common.resource.default",
			"create":  "common.action.create_resource",
			"update":  "common.action.edit_resource",
			"delete":  "common.action.delete_resource",
			"status":  "common.action.set_field_status",
		}[action]
		if messageKey == "" {
			messageKey = "common.resource.default"
		}
		i18ns[localeValue] = localizeCodegen(localeValue, state.Primary, messageKey, values, messageKey)
	}
	return i18ns
}

func localizedResourceMessages(prefix string, resource string) map[string]string {
	return map[string]string{prefix + ".resource": resource}
}

func localizedTableComment(table *Table, localeValue string, primaryLocale string) string {
	if table == nil {
		return ""
	}
	if localeValue == primaryLocale {
		return DefaultString(table.TableComment, table.BusinessName)
	}
	return DefaultString(table.I18NConfig[localeValue].Comment, DefaultString(table.TableComment, table.BusinessName))
}

func localizedLeftTreeComment(table *Table, localeValue string, primaryLocale string) string {
	config := LeftTreeConfigFromTable(table)
	if localeValue == primaryLocale {
		return DefaultString(config.Comment, localizedTableComment(table, localeValue, primaryLocale))
	}
	return DefaultString(table.I18NConfig[localeValue].LeftTreeComment, DefaultString(config.Comment, localizedTableComment(table, localeValue, primaryLocale)))
}

func localizedColumnComment(column *CodeGenColumn, localeValue string, primaryLocale string) string {
	if column == nil {
		return ""
	}
	if localeValue == primaryLocale {
		return DefaultString(column.Comment, column.Name)
	}
	return DefaultString(column.I18NConfig[localeValue].Comment, DefaultString(column.Comment, column.Name))
}

func localizedPasswordStrength(localeValue string, primaryLocale string) string {
	return localizeCodegen(localeValue, primaryLocale, "password_strength", nil, "password_strength")
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

func localizedGeneratedStaticLabel(label string, localeValue string, primaryLocale string) string {
	messageKey, ok := codegenStaticMessageKeys[label]
	if !ok {
		return label
	}
	return localizeCodegen(localeValue, primaryLocale, messageKey, nil, label)
}

var codegenStaticMessageKeys = map[string]string{
	"启用": "common.status.enabled",
	"啟用": "common.status.enabled",
	"禁用": "common.status.disabled",
	"是":  "common.value.yes",
	"否":  "common.value.no",
	"男":  "common.value.male",
	"女":  "common.value.female",
}

// localizeCodegen 从统一后端目录读取代码生成器文案并执行占位符替换。
func localizeCodegen(localeValue string, primaryLocale string, key string, values map[string]string, fallback string) string {
	if codegenCatalogValue == nil {
		return replaceCodegenMessagePlaceholders(fallback, values)
	}
	args := make(map[string]any, len(values))
	for name, value := range values {
		args[name] = value
	}
	localized := codegenCatalogValue.Localize(localeValue, primaryLocale, fullCodegenMessageKey(key), args, fallback)
	return replaceCodegenMessagePlaceholders(localized, values)
}

// replaceCodegenMessagePlaceholders 替换代码生成器文案中的兼容占位符。
func replaceCodegenMessagePlaceholders(message string, values map[string]string) string {
	for name, value := range values {
		message = strings.ReplaceAll(message, "{"+name+"}", value)
		message = strings.ReplaceAll(message, "{{."+name+"}}", value)
	}
	return message
}

func fullCodegenMessageKey(key string) string {
	if strings.HasPrefix(key, "common.") || strings.HasPrefix(key, codegenMessagePrefix) {
		return key
	}
	return codegenMessagePrefix + key
}
