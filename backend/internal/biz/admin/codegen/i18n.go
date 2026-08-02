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

// RequiredTranslationLocales 返回代码生成正式写入所需的非默认语言。
func RequiredTranslationLocales() []string {
	return []string{coreLocale.EnUS, coreLocale.JaJP}
}

// GeneratedFrontendLocales 返回生成页面需要同步的全部语言区域。
func GeneratedFrontendLocales() []string {
	return []string{coreLocale.ZhCN, coreLocale.EnUS, coreLocale.JaJP}
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
	parts = append([]string{table.BusinessModule}, parts...)
	return strings.Join(parts, ".")
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
		messages[prefix+".field."+stringcase.ToCamelCase(column.Name)] = localizedColumnComment(column, localeValue)
		if column.FormComponent == "password" {
			messages[prefix+".field.passwordStrength"] = localizedPasswordStrength(localeValue)
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
		messages[prefix+".title.leftTree"] = localizedLeftTreeComment(table, localeValue)
	}
	return messages
}

// newFrontendLocalePreviewFiles 创建三语语言包的结构化合并预览。
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

// GeneratedMenuTranslations 返回页面或按钮菜单的英语和日语标题。
func GeneratedMenuTranslations(table *Table, column *CodeGenColumn, action string) map[string]string {
	translations := make(map[string]string, len(RequiredTranslationLocales()))
	for _, localeValue := range RequiredTranslationLocales() {
		resource := localizedTableComment(table, localeValue)
		title := resource
		switch action {
		case "create":
			title = "Create " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を新規作成"
			}
		case "update":
			title = "Edit " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を編集"
			}
		case "delete":
			title = "Delete " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を削除"
			}
		case "status":
			fieldName := localizedColumnComment(column, localeValue)
			title = "Set " + fieldName
			if localeValue == coreLocale.JaJP {
				title = fieldName + "を設定"
			}
		}
		translations[localeValue] = title
	}
	return translations
}

func localizedResourceMessages(prefix string, resource string, localeValue string) map[string]string {
	messages := map[string]string{
		prefix + ".placeholder.input":      "请输入{field}",
		prefix + ".placeholder.select":     "请选择{field}",
		prefix + ".validation.required":    "{field}不能为空",
		prefix + ".validation.maxLength":   "{field}不能超过 {max} 个字符",
		prefix + ".title.create":           "新增" + resource,
		prefix + ".title.edit":             "编辑" + resource,
		prefix + ".message.createSuccess":  "新增" + resource + "成功",
		prefix + ".message.updateSuccess":  "修改" + resource + "成功",
		prefix + ".message.deleteSuccess":  "删除" + resource + "成功",
		prefix + ".message.deleteCanceled": "已取消删除" + resource,
		prefix + ".message.selectDelete":   "请勾选删除项",
		prefix + ".message.statusSuccess":  "{action}成功",
		prefix + ".dialog.deleteSingle":    "是否确定删除" + resource + "？",
		prefix + ".dialog.deleteBatch":     "确认删除已选中的" + resource + "吗？",
		prefix + ".dialog.confirmStatus":   "是否确定{action}" + resource + "？",
		prefix + ".status.enabled":         "启用",
		prefix + ".status.disabled":        "禁用",
		prefix + ".value.topLevel":         "顶级节点",
	}
	if localeValue == coreLocale.EnUS {
		messages[prefix+".placeholder.input"] = "Enter {field}"
		messages[prefix+".placeholder.select"] = "Select {field}"
		messages[prefix+".validation.required"] = "{field} is required"
		messages[prefix+".validation.maxLength"] = "{field} cannot exceed {max} characters"
		messages[prefix+".title.create"] = "Create " + resource
		messages[prefix+".title.edit"] = "Edit " + resource
		messages[prefix+".message.createSuccess"] = resource + " created"
		messages[prefix+".message.updateSuccess"] = resource + " updated"
		messages[prefix+".message.deleteSuccess"] = resource + " deleted"
		messages[prefix+".message.deleteCanceled"] = "Deletion canceled"
		messages[prefix+".message.selectDelete"] = "Select records to delete"
		messages[prefix+".message.statusSuccess"] = "{action} succeeded"
		messages[prefix+".dialog.deleteSingle"] = "Delete this " + resource + "?"
		messages[prefix+".dialog.deleteBatch"] = "Delete the selected " + resource + " records?"
		messages[prefix+".dialog.confirmStatus"] = "{action} this " + resource + "?"
		messages[prefix+".status.enabled"] = "Enable"
		messages[prefix+".status.disabled"] = "Disable"
		messages[prefix+".value.topLevel"] = "Top level"
	}
	if localeValue == coreLocale.JaJP {
		messages[prefix+".placeholder.input"] = "{field}を入力してください"
		messages[prefix+".placeholder.select"] = "{field}を選択してください"
		messages[prefix+".validation.required"] = "{field}は必須です"
		messages[prefix+".validation.maxLength"] = "{field}は {max} 文字以内で入力してください"
		messages[prefix+".title.create"] = resource + "を新規作成"
		messages[prefix+".title.edit"] = resource + "を編集"
		messages[prefix+".message.createSuccess"] = resource + "を作成しました"
		messages[prefix+".message.updateSuccess"] = resource + "を更新しました"
		messages[prefix+".message.deleteSuccess"] = resource + "を削除しました"
		messages[prefix+".message.deleteCanceled"] = "削除をキャンセルしました"
		messages[prefix+".message.selectDelete"] = "削除する項目を選択してください"
		messages[prefix+".message.statusSuccess"] = "{action}しました"
		messages[prefix+".dialog.deleteSingle"] = resource + "を削除しますか？"
		messages[prefix+".dialog.deleteBatch"] = "選択した" + resource + "を削除しますか？"
		messages[prefix+".dialog.confirmStatus"] = resource + "を{action}しますか？"
		messages[prefix+".status.enabled"] = "有効化"
		messages[prefix+".status.disabled"] = "無効化"
		messages[prefix+".value.topLevel"] = "最上位ノード"
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
	switch localeValue {
	case coreLocale.EnUS:
		return "Password strength"
	case coreLocale.JaJP:
		return "パスワード強度"
	default:
		return "强度提示"
	}
}

// frontendStaticOptionLocaleKey 返回静态选项值对应的稳定语言键。
func frontendStaticOptionLocaleKey(table *Table, column *CodeGenColumn, value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		content = []byte(fmt.Sprint(value))
	}
	digest := sha256.Sum256(content)
	return FrontendLocaleKeyPrefix(table) + ".option." + stringcase.ToCamelCase(column.Name) + ".value" + hex.EncodeToString(digest[:6])
}

func parseCodeGenStaticOptions(option CodeGenColumnOptionConfig) []CodeGenStaticOption {
	var options []CodeGenStaticOption
	if json.Unmarshal([]byte(option.SourceValue), &options) != nil {
		return nil
	}
	return options
}

func localizedGeneratedStaticLabel(label string, localeValue string) string {
	translations := map[string]map[string]string{
		"启用": {coreLocale.EnUS: "Enabled", coreLocale.JaJP: "有効"},
		"禁用": {coreLocale.EnUS: "Disabled", coreLocale.JaJP: "無効"},
		"是":  {coreLocale.EnUS: "Yes", coreLocale.JaJP: "はい"},
		"否":  {coreLocale.EnUS: "No", coreLocale.JaJP: "いいえ"},
		"男":  {coreLocale.EnUS: "Male", coreLocale.JaJP: "男性"},
		"女":  {coreLocale.EnUS: "Female", coreLocale.JaJP: "女性"},
	}
	if translated := translations[label][localeValue]; translated != "" {
		return translated
	}
	return label
}
