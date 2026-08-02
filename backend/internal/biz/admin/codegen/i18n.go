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

// newFrontendLocalePreviewFiles 创建七语语言包的结构化合并预览。
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
		resource := localizedTableComment(table, localeValue)
		title := resource
		switch action {
		case "create":
			title = "Create " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を新規作成"
			} else if localeValue == coreLocale.ZhTW {
				title = "新增" + resource
			} else if localeValue == coreLocale.KoKR {
				title = resource + " 생성"
			} else if localeValue == coreLocale.FrFR {
				title = "Créer " + resource
			} else if localeValue == coreLocale.EsES {
				title = "Crear " + resource
			}
		case "update":
			title = "Edit " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を編集"
			} else if localeValue == coreLocale.ZhTW {
				title = "編輯" + resource
			} else if localeValue == coreLocale.KoKR {
				title = resource + " 편집"
			} else if localeValue == coreLocale.FrFR {
				title = "Modifier " + resource
			} else if localeValue == coreLocale.EsES {
				title = "Editar " + resource
			}
		case "delete":
			title = "Delete " + resource
			if localeValue == coreLocale.JaJP {
				title = resource + "を削除"
			} else if localeValue == coreLocale.ZhTW {
				title = "刪除" + resource
			} else if localeValue == coreLocale.KoKR {
				title = resource + " 삭제"
			} else if localeValue == coreLocale.FrFR {
				title = "Supprimer " + resource
			} else if localeValue == coreLocale.EsES {
				title = "Eliminar " + resource
			}
		case "status":
			fieldName := localizedColumnComment(column, localeValue)
			title = "Set " + fieldName
			if localeValue == coreLocale.JaJP {
				title = fieldName + "を設定"
			} else if localeValue == coreLocale.ZhTW {
				title = "設定" + fieldName
			} else if localeValue == coreLocale.KoKR {
				title = fieldName + " 설정"
			} else if localeValue == coreLocale.FrFR {
				title = "Définir " + fieldName
			} else if localeValue == coreLocale.EsES {
				title = "Establecer " + fieldName
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
	if localeValue == coreLocale.ZhTW {
		messages[prefix+".placeholder.input"] = "請輸入{field}"
		messages[prefix+".placeholder.select"] = "請選擇{field}"
		messages[prefix+".validation.required"] = "{field}不能為空"
		messages[prefix+".validation.maxLength"] = "{field}不能超過 {max} 個字元"
		messages[prefix+".title.create"] = "新增" + resource
		messages[prefix+".title.edit"] = "編輯" + resource
		messages[prefix+".message.createSuccess"] = "新增" + resource + "成功"
		messages[prefix+".message.updateSuccess"] = "修改" + resource + "成功"
		messages[prefix+".message.deleteSuccess"] = "刪除" + resource + "成功"
		messages[prefix+".message.deleteCanceled"] = "已取消刪除" + resource
		messages[prefix+".message.selectDelete"] = "請勾選刪除項目"
		messages[prefix+".message.statusSuccess"] = "{action}成功"
		messages[prefix+".dialog.deleteSingle"] = "是否確定刪除" + resource + "？"
		messages[prefix+".dialog.deleteBatch"] = "確認刪除已選取的" + resource + "嗎？"
		messages[prefix+".dialog.confirmStatus"] = "是否確定{action}" + resource + "？"
		messages[prefix+".status.enabled"] = "啟用"
		messages[prefix+".status.disabled"] = "停用"
		messages[prefix+".value.topLevel"] = "頂級節點"
	}
	if localeValue == coreLocale.KoKR {
		messages[prefix+".placeholder.input"] = "{field} 입력"
		messages[prefix+".placeholder.select"] = "{field} 선택"
		messages[prefix+".validation.required"] = "{field}은(는) 필수입니다"
		messages[prefix+".validation.maxLength"] = "{field}은(는) {max}자를 초과할 수 없습니다"
		messages[prefix+".title.create"] = resource + " 생성"
		messages[prefix+".title.edit"] = resource + " 편집"
		messages[prefix+".message.createSuccess"] = resource + " 생성 성공"
		messages[prefix+".message.updateSuccess"] = resource + " 수정 성공"
		messages[prefix+".message.deleteSuccess"] = resource + " 삭제 성공"
		messages[prefix+".message.deleteCanceled"] = "삭제가 취소되었습니다"
		messages[prefix+".message.selectDelete"] = "삭제할 항목을 선택하세요"
		messages[prefix+".message.statusSuccess"] = "{action} 성공"
		messages[prefix+".dialog.deleteSingle"] = "이 " + resource + "을(를) 삭제하시겠습니까?"
		messages[prefix+".dialog.deleteBatch"] = "선택한 " + resource + "을(를) 삭제하시겠습니까?"
		messages[prefix+".dialog.confirmStatus"] = resource + "을(를) {action}하시겠습니까?"
		messages[prefix+".status.enabled"] = "사용"
		messages[prefix+".status.disabled"] = "사용 안 함"
		messages[prefix+".value.topLevel"] = "최상위 노드"
	}
	if localeValue == coreLocale.FrFR {
		messages[prefix+".placeholder.input"] = "Saisissez {field}"
		messages[prefix+".placeholder.select"] = "Sélectionnez {field}"
		messages[prefix+".validation.required"] = "{field} est obligatoire"
		messages[prefix+".validation.maxLength"] = "{field} ne peut pas dépasser {max} caractères"
		messages[prefix+".title.create"] = "Créer " + resource
		messages[prefix+".title.edit"] = "Modifier " + resource
		messages[prefix+".message.createSuccess"] = resource + " créé"
		messages[prefix+".message.updateSuccess"] = resource + " modifié"
		messages[prefix+".message.deleteSuccess"] = resource + " supprimé"
		messages[prefix+".message.deleteCanceled"] = "Suppression annulée"
		messages[prefix+".message.selectDelete"] = "Sélectionnez les éléments à supprimer"
		messages[prefix+".message.statusSuccess"] = "{action} réussi"
		messages[prefix+".dialog.deleteSingle"] = "Supprimer ce " + resource + " ?"
		messages[prefix+".dialog.deleteBatch"] = "Supprimer les " + resource + " sélectionnés ?"
		messages[prefix+".dialog.confirmStatus"] = "{action} ce " + resource + " ?"
		messages[prefix+".status.enabled"] = "Activer"
		messages[prefix+".status.disabled"] = "Désactiver"
		messages[prefix+".value.topLevel"] = "Niveau supérieur"
	}
	if localeValue == coreLocale.EsES {
		messages[prefix+".placeholder.input"] = "Introduzca {field}"
		messages[prefix+".placeholder.select"] = "Seleccione {field}"
		messages[prefix+".validation.required"] = "{field} es obligatorio"
		messages[prefix+".validation.maxLength"] = "{field} no puede superar {max} caracteres"
		messages[prefix+".title.create"] = "Crear " + resource
		messages[prefix+".title.edit"] = "Editar " + resource
		messages[prefix+".message.createSuccess"] = resource + " creado"
		messages[prefix+".message.updateSuccess"] = resource + " actualizado"
		messages[prefix+".message.deleteSuccess"] = resource + " eliminado"
		messages[prefix+".message.deleteCanceled"] = "Eliminación cancelada"
		messages[prefix+".message.selectDelete"] = "Seleccione los elementos que desea eliminar"
		messages[prefix+".message.statusSuccess"] = "{action} correctamente"
		messages[prefix+".dialog.deleteSingle"] = "¿Eliminar este " + resource + "?"
		messages[prefix+".dialog.deleteBatch"] = "¿Eliminar los " + resource + " seleccionados?"
		messages[prefix+".dialog.confirmStatus"] = "¿{action} este " + resource + "?"
		messages[prefix+".status.enabled"] = "Activar"
		messages[prefix+".status.disabled"] = "Desactivar"
		messages[prefix+".value.topLevel"] = "Nivel superior"
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
	case coreLocale.ZhTW:
		return "強度提示"
	case coreLocale.KoKR:
		return "비밀번호 강도"
	case coreLocale.FrFR:
		return "Force du mot de passe"
	case coreLocale.EsES:
		return "Fortaleza de la contraseña"
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
		"启用": {coreLocale.ZhTW: "啟用", coreLocale.EnUS: "Enabled", coreLocale.JaJP: "有効", coreLocale.KoKR: "사용", coreLocale.FrFR: "Activé", coreLocale.EsES: "Activado"},
		"禁用": {coreLocale.ZhTW: "停用", coreLocale.EnUS: "Disabled", coreLocale.JaJP: "無効", coreLocale.KoKR: "사용 안 함", coreLocale.FrFR: "Désactivé", coreLocale.EsES: "Desactivado"},
		"是":  {coreLocale.ZhTW: "是", coreLocale.EnUS: "Yes", coreLocale.JaJP: "はい", coreLocale.KoKR: "예", coreLocale.FrFR: "Oui", coreLocale.EsES: "Sí"},
		"否":  {coreLocale.ZhTW: "否", coreLocale.EnUS: "No", coreLocale.JaJP: "いいえ", coreLocale.KoKR: "아니요", coreLocale.FrFR: "Non", coreLocale.EsES: "No"},
		"男":  {coreLocale.ZhTW: "男", coreLocale.EnUS: "Male", coreLocale.JaJP: "男性", coreLocale.KoKR: "남성", coreLocale.FrFR: "Homme", coreLocale.EsES: "Hombre"},
		"女":  {coreLocale.ZhTW: "女", coreLocale.EnUS: "Female", coreLocale.JaJP: "女性", coreLocale.KoKR: "여성", coreLocale.FrFR: "Femme", coreLocale.EsES: "Mujer"},
	}
	if translated := translations[label][localeValue]; translated != "" {
		return translated
	}
	return label
}
