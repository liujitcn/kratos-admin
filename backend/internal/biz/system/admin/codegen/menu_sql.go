package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

const generatedMenuSQLFileName = "default_data.up.sql"

// RenderGeneratedMenuSQL 渲染当前代码生成对象的菜单和按钮权限脚本。
func RenderGeneratedMenuSQL(table *Table, columns []*CodeGenColumn, methods []*Proto, resourcePath string, tableComment string, localeState LocaleState) string {
	pageSpec, buttonSpecs := MenuSpecs(table, columns, methods, resourcePath, tableComment, localeState)
	page := pageSpec.Menu
	var builder strings.Builder
	builder.WriteString("-- 代码生成菜单权限脚本，请勿手工修改。\n")
	builder.WriteString("-- 重新执行代码生成会覆盖本表菜单权限片段，执行还原会恢复数据库中的生成前状态。\n\n")
	builder.WriteString("SET @codegen_parent_menu_id = ")
	builder.WriteString(strconv.FormatInt(page.ParentID, 10))
	builder.WriteString(";\n")
	writeMenuUpsertSQL(&builder, "page", page, "@codegen_parent_menu_id", "type = 2")
	builder.WriteString("SET @codegen_page_menu_id = (SELECT `id` FROM `base_menu` WHERE `type` = 2 AND (`path` = ")
	builder.WriteString(sqlString(page.Path))
	builder.WriteString(" OR `name` = ")
	builder.WriteString(sqlString(page.Name))
	builder.WriteString(" OR `component` = ")
	builder.WriteString(sqlString(page.Component))
	builder.WriteString(") ORDER BY `id` LIMIT 1);\n")
	writeMenuI18nSQL(&builder, "@codegen_page_menu_id", pageSpec, localeState)
	for index, buttonSpec := range buttonSpecs {
		button := buttonSpec.Menu
		varName := fmt.Sprintf("@codegen_button_menu_id_%d", index+1)
		writeMenuUpsertSQL(&builder, fmt.Sprintf("button_%d", index+1), button, "@codegen_page_menu_id", "type = 3")
		builder.WriteString("SET ")
		builder.WriteString(varName)
		builder.WriteString(" = (SELECT `id` FROM `base_menu` WHERE `parent_id` = @codegen_page_menu_id AND `type` = 3 AND (`path` = ")
		builder.WriteString(sqlString(button.Path))
		builder.WriteString(" OR `api` = ")
		builder.WriteString(sqlString(button.API))
		builder.WriteString(") ORDER BY `id` LIMIT 1);\n")
		writeMenuI18nSQL(&builder, varName, buttonSpec, localeState)
	}
	writeStaleStatusMenuSQL(&builder, table, buttonSpecs)
	builder.WriteString("\n-- 代码生成菜单权限脚本结束。\n")
	return builder.String()
}

// writeMenuI18nSQL 写入启用的非主语言菜单译文，已有记录一律保留。
func writeMenuI18nSQL(builder *strings.Builder, menuIDExpression string, spec CodeGenMenuSpec, localeState LocaleState) {
	for _, localeValue := range RequiredI18nLocales(localeState) {
		title := spec.I18ns[localeValue]
		if title == "" {
			continue
		}
		builder.WriteString("INSERT IGNORE INTO `base_i18n` (`target_type`, `target_id`, `locale`, `name`)\n")
		builder.WriteString("SELECT ")
		builder.WriteString(strconv.FormatInt(int64(_const.I18N_TARGET_TYPE_BASE_MENU_META_TITLE), 10))
		builder.WriteString(", ")
		builder.WriteString(menuIDExpression)
		builder.WriteString(", ")
		builder.WriteString(sqlString(localeValue))
		builder.WriteString(", ")
		builder.WriteString(sqlString(title))
		builder.WriteString("\n")
		builder.WriteString("WHERE ")
		builder.WriteString(menuIDExpression)
		builder.WriteString(" IS NOT NULL AND EXISTS (SELECT 1 FROM `base_language` WHERE `language_code` = ")
		builder.WriteString(sqlString(localeValue))
		builder.WriteString(" AND `is_primary` = 0 AND `status` = 1 AND `deleted_at` = 0) AND NOT EXISTS (SELECT 1 FROM `base_i18n` WHERE `target_type` = ")
		builder.WriteString(strconv.FormatInt(int64(_const.I18N_TARGET_TYPE_BASE_MENU_META_TITLE), 10))
		builder.WriteString(" AND `target_id` = ")
		builder.WriteString(menuIDExpression)
		builder.WriteString(" AND `locale` = ")
		builder.WriteString(sqlString(localeValue))
		builder.WriteString(");\n")
	}
}

// newGeneratedMenuSQLPreviewFile 创建固定初始化版本 SQL 的菜单权限预览文件。
func (c *renderer) newGeneratedMenuSQLPreviewFile(table *Table, content string) *adminv1.CodeGenPreviewFile {
	path, err := nextGeneratedMenuSQLPath(c.migrationVersion)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{Action: "skip", Content: content, Message: err.Error()}
	}
	_, err = SafeRepoFilePath(path)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: err.Error()}
	}
	var current []byte
	current, err = c.readRepoFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: err.Error()}
		}
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "create",
			Content: generatedMenuSQLBlock(table, content) + "\n",
			Message: Message(c.localeState, "preview.menu_sql_create", map[string]string{"path": path}),
		}
	}
	var merged string
	merged, err = mergeGeneratedMenuSQLAtPath(string(current), table, content, path)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: string(current),
			Exists:  true,
			Message: err.Error(),
		}
	}
	if string(current) == merged {
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: merged,
			Exists:  true,
			Message: Message(c.localeState, "preview.menu_sql_unchanged", map[string]string{"path": path}),
		}
	}
	return &adminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  "update",
		Content: merged,
		Exists:  true,
		Message: Message(c.localeState, "preview.menu_sql_update", map[string]string{"path": path}),
	}
}

// mergeGeneratedMenuSQLAtPath 在指定迁移脚本中替换或追加指定表的菜单权限片段。
func mergeGeneratedMenuSQLAtPath(existing string, table *Table, content string, path string) (string, error) {
	if table == nil {
		return existing, fmt.Errorf("代码生成表不能为空，无法写入菜单 SQL")
	}
	beginMarker := fmt.Sprintf("-- CODEGEN_MENU_BEGIN table=%s", table.TableName_)
	endMarker := fmt.Sprintf("-- CODEGEN_MENU_END table=%s", table.TableName_)
	beginIndex := strings.Index(existing, beginMarker)
	endIndex := strings.Index(existing, endMarker)
	if beginIndex < 0 && endIndex >= 0 {
		return existing, fmt.Errorf("%s 中表%s的菜单 SQL 结束标记缺少开始标记", path, table.TableName_)
	}
	block := generatedMenuSQLBlock(table, content)
	if beginIndex >= 0 {
		contentStart := beginIndex + len(beginMarker)
		relativeEndIndex := strings.Index(existing[contentStart:], endMarker)
		if relativeEndIndex < 0 {
			return existing, fmt.Errorf("%s 中表%s的菜单 SQL 标记不完整", path, table.TableName_)
		}
		endIndex = contentStart + relativeEndIndex + len(endMarker)
		return existing[:beginIndex] + block + existing[endIndex:], nil
	}
	if existing == "" {
		return block + "\n", nil
	}
	separator := "\n"
	if !strings.HasSuffix(existing, "\n") {
		separator = "\n\n"
	}
	return existing + separator + block + "\n", nil
}

// generatedMenuSQLBlock 返回带表级标记的菜单权限 SQL 片段。
func generatedMenuSQLBlock(table *Table, content string) string {
	if table == nil {
		return strings.TrimRight(content, "\r\n")
	}
	beginMarker := fmt.Sprintf("-- CODEGEN_MENU_BEGIN table=%s", table.TableName_)
	endMarker := fmt.Sprintf("-- CODEGEN_MENU_END table=%s", table.TableName_)
	return beginMarker + "\n" + strings.TrimRight(content, "\r\n") + "\n" + endMarker
}

// nextGeneratedMenuSQLPath 返回项目固定初始化版本的菜单脚本路径。
func nextGeneratedMenuSQLPath(_ string) (string, error) {
	path := "backend/migration/assets/v0.0.1/mysql/" + generatedMenuSQLFileName
	info, err := os.Stat(filepath.Join(repoRoot(), filepath.Dir(path)))
	if err != nil {
		return "", fmt.Errorf("读取初始化迁移目录失败: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("初始化迁移路径不是目录: %s", filepath.Dir(path))
	}
	return path, nil
}

// isGeneratedMenuSQLPath 判断是否为项目初始化版本的菜单脚本。
func isGeneratedMenuSQLPath(path string) bool {
	return filepath.ToSlash(filepath.Clean(path)) == "backend/migration/assets/v0.0.1/mysql/"+generatedMenuSQLFileName
}

// writeMenuUpsertSQL 写入单个菜单的幂等插入和更新语句。
func writeMenuUpsertSQL(builder *strings.Builder, label string, menu *models.BaseMenu, parentExpression string, typeCondition string) {
	if menu == nil {
		return
	}
	builder.WriteString("-- ")
	builder.WriteString(label)
	builder.WriteString("\n")
	// 与在线菜单分配规则一致，使用父级下第一个未占用的层级编号。
	builder.WriteString("SET @codegen_new_menu_id = (SELECT MIN(candidate.id) FROM (SELECT CASE\n")
	fmt.Fprintf(builder, "WHEN %s %% 1000000 = 0 THEN %s + seq.n * 10000\n", parentExpression, parentExpression)
	fmt.Fprintf(builder, "WHEN %s %% 10000 = 0 THEN %s + seq.n * 100\n", parentExpression, parentExpression)
	fmt.Fprintf(builder, "WHEN %s %% 100 = 0 THEN %s + seq.n\n", parentExpression, parentExpression)
	fmt.Fprintf(builder, "WHEN (%s %% 100) * 10 + seq.n <= 99 THEN FLOOR(%s / 100) * 100 + (%s %% 100) * 10 + seq.n\n", parentExpression, parentExpression, parentExpression)
	builder.WriteString("END AS id FROM (")
	for sequence := 1; sequence <= 99; sequence++ {
		if sequence > 1 {
			builder.WriteString(" UNION ALL ")
		}
		fmt.Fprintf(builder, "SELECT %d AS n", sequence)
	}
	builder.WriteString(") AS seq) AS candidate LEFT JOIN base_menu AS used ON used.id = candidate.id WHERE used.id IS NULL);\n")
	builder.WriteString("INSERT INTO `base_menu` (`id`, `parent_id`, `type`, `path`, `name`, `component`, `redirect`, `meta`, `api`, `sort`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)\n")
	builder.WriteString("SELECT @codegen_new_menu_id, ")
	builder.WriteString(parentExpression)
	builder.WriteString(", ")
	builder.WriteString(strconv.FormatInt(int64(menu.Type), 10))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.Path))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.Name))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.Component))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.Redirect))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.Meta))
	builder.WriteString(", ")
	builder.WriteString(sqlString(menu.API))
	builder.WriteString(", ")
	builder.WriteString(strconv.FormatInt(int64(menu.Sort), 10))
	builder.WriteString(", ")
	builder.WriteString(strconv.FormatInt(int64(menu.Status), 10))
	builder.WriteString(", 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0\n")
	builder.WriteString("WHERE @codegen_new_menu_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM `base_menu` WHERE ")
	builder.WriteString(typeCondition)
	if menu.Type == 2 {
		builder.WriteString(" AND (`path` = ")
		builder.WriteString(sqlString(menu.Path))
		builder.WriteString(" OR `name` = ")
		builder.WriteString(sqlString(menu.Name))
		builder.WriteString(" OR `component` = ")
		builder.WriteString(sqlString(menu.Component))
		builder.WriteString(")")
	} else {
		builder.WriteString(" AND `parent_id` = ")
		builder.WriteString(parentExpression)
		builder.WriteString(" AND (`path` = ")
		builder.WriteString(sqlString(menu.Path))
		builder.WriteString(" OR `api` = ")
		builder.WriteString(sqlString(menu.API))
		builder.WriteString(")")
	}
	builder.WriteString(");\n")
	builder.WriteString("UPDATE `base_menu` SET `parent_id` = ")
	builder.WriteString(parentExpression)
	builder.WriteString(", `type` = ")
	builder.WriteString(strconv.FormatInt(int64(menu.Type), 10))
	builder.WriteString(", `path` = ")
	builder.WriteString(sqlString(menu.Path))
	builder.WriteString(", `name` = ")
	builder.WriteString(sqlString(menu.Name))
	builder.WriteString(", `component` = ")
	builder.WriteString(sqlString(menu.Component))
	builder.WriteString(", `redirect` = ")
	builder.WriteString(sqlString(menu.Redirect))
	builder.WriteString(", `meta` = ")
	builder.WriteString(sqlString(menu.Meta))
	builder.WriteString(", `api` = ")
	builder.WriteString(sqlString(menu.API))
	builder.WriteString(", `sort` = ")
	builder.WriteString(strconv.FormatInt(int64(menu.Sort), 10))
	builder.WriteString(", `status` = ")
	builder.WriteString(strconv.FormatInt(int64(menu.Status), 10))
	builder.WriteString(" WHERE `id` = (SELECT `id` FROM (SELECT `id` FROM `base_menu` WHERE ")
	builder.WriteString(typeCondition)
	if menu.Type == 2 {
		builder.WriteString(" AND (`path` = ")
		builder.WriteString(sqlString(menu.Path))
		builder.WriteString(" OR `name` = ")
		builder.WriteString(sqlString(menu.Name))
		builder.WriteString(" OR `component` = ")
		builder.WriteString(sqlString(menu.Component))
		builder.WriteString(")")
	} else {
		builder.WriteString(" AND `parent_id` = ")
		builder.WriteString(parentExpression)
		builder.WriteString(" AND (`path` = ")
		builder.WriteString(sqlString(menu.Path))
		builder.WriteString(" OR `api` = ")
		builder.WriteString(sqlString(menu.API))
		builder.WriteString(")")
	}
	builder.WriteString(" ORDER BY `id` LIMIT 1) AS `codegen_target_menu`);\n")
}

// writeStaleStatusMenuSQL 写入停用本轮不再需要的状态按钮语句。
func writeStaleStatusMenuSQL(builder *strings.Builder, table *Table, buttonSpecs []CodeGenMenuSpec) {
	if table == nil {
		return
	}
	expectedPaths := make([]string, 0, len(buttonSpecs))
	for _, buttonSpec := range buttonSpecs {
		if buttonSpec.Menu != nil {
			expectedPaths = append(expectedPaths, buttonSpec.Menu.Path)
		}
	}
	builder.WriteString("\nUPDATE `base_menu` SET `status` = 2, `api` = '[]'\n")
	builder.WriteString("WHERE `parent_id` = @codegen_page_menu_id AND `type` = 3\n")
	builder.WriteString("  AND (`path` = ")
	builder.WriteString(sqlString(PermissionPrefix(table) + ":status"))
	builder.WriteString(" OR `path` LIKE ")
	builder.WriteString(sqlString(PermissionPrefix(table) + ":status:%"))
	builder.WriteString(" OR `api` LIKE ")
	builder.WriteString(sqlString("%" + GeneratedRPCServicePath(table, table.EntityName) + "/Set%"))
	builder.WriteString(")")
	if len(expectedPaths) == 0 {
		builder.WriteString(";\n")
		return
	}
	builder.WriteString(" AND `path` NOT IN (")
	for index, path := range expectedPaths {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(sqlString(path))
	}
	builder.WriteString(");\n")
}

// sqlString 将文本安全编码为 MySQL 字符串字面量。
func sqlString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "''")
	return "'" + value + "'"
}
