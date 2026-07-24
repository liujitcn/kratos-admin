package codegen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"
)

const generatedMenuSQLPath = "sql/default-data.sql"

// GeneratedMenuSQLPath 返回代码生成菜单 SQL 文件路径。
func GeneratedMenuSQLPath(table *Table) string {
	if table == nil {
		return ""
	}
	return generatedMenuSQLPath
}

// RenderGeneratedMenuSQL 渲染当前代码生成对象的菜单和按钮权限脚本。
func RenderGeneratedMenuSQL(table *Table, columns []*CodeGenColumn, methods []*Proto, resourcePath string, tableComment string) string {
	pageSpec, buttonSpecs := MenuSpecs(table, columns, methods, resourcePath, tableComment)
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
	}
	writeStaleStatusMenuSQL(&builder, table, buttonSpecs)
	builder.WriteString("\n-- 代码生成菜单权限脚本结束。\n")
	return builder.String()
}

// newGeneratedMenuSQLPreviewFile 创建更新默认初始化 SQL 的菜单权限预览文件。
func (c *renderer) newGeneratedMenuSQLPreviewFile(table *Table, content string) *systemadminv1.CodeGenPreviewFile {
	path := GeneratedMenuSQLPath(table)
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: pathErr.Error()}
	}
	current, err := c.readRepoFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: err.Error()}
		}
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: content,
			Message: "sql/default-data.sql 不存在，无法写入菜单和按钮权限 SQL",
		}
	}
	var merged string
	merged, err = mergeGeneratedMenuSQL(string(current), table, content)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: string(current),
			Exists:  true,
			Message: err.Error(),
		}
	}
	if string(current) == merged {
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: merged,
			Exists:  true,
			Message: "sql/default-data.sql 中的菜单和按钮权限 SQL 无需更新",
		}
	}
	return &systemadminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  "update",
		Content: merged,
		Exists:  true,
		Message: "将更新 sql/default-data.sql 中的菜单和按钮权限 SQL",
	}
}

// mergeGeneratedMenuSQL 在默认初始化脚本中替换或追加指定表的菜单权限片段。
func mergeGeneratedMenuSQL(existing string, table *Table, content string) (string, error) {
	if table == nil {
		return existing, fmt.Errorf("代码生成表不能为空，无法写入菜单 SQL")
	}
	beginMarker := fmt.Sprintf("-- CODEGEN_MENU_BEGIN table=%s", table.TableName_)
	endMarker := fmt.Sprintf("-- CODEGEN_MENU_END table=%s", table.TableName_)
	beginIndex := strings.Index(existing, beginMarker)
	endIndex := strings.Index(existing, endMarker)
	if beginIndex < 0 && endIndex >= 0 {
		return existing, fmt.Errorf("sql/default-data.sql 中表%s的菜单 SQL 结束标记缺少开始标记", table.TableName_)
	}
	block := beginMarker + "\n" + strings.TrimRight(content, "\r\n") + "\n" + endMarker
	if beginIndex >= 0 {
		contentStart := beginIndex + len(beginMarker)
		relativeEndIndex := strings.Index(existing[contentStart:], endMarker)
		if relativeEndIndex < 0 {
			return existing, fmt.Errorf("sql/default-data.sql 中表%s的菜单 SQL 标记不完整", table.TableName_)
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

// writeMenuUpsertSQL 写入单个菜单的幂等插入和更新语句。
func writeMenuUpsertSQL(builder *strings.Builder, label string, menu *models.BaseMenu, parentExpression string, typeCondition string) {
	if menu == nil {
		return
	}
	builder.WriteString("-- ")
	builder.WriteString(label)
	builder.WriteString("\n")
	builder.WriteString("INSERT INTO `base_menu` (`parent_id`, `type`, `path`, `name`, `component`, `redirect`, `meta`, `api`, `sort`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)\n")
	builder.WriteString("SELECT ")
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
	builder.WriteString("WHERE NOT EXISTS (SELECT 1 FROM `base_menu` WHERE ")
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
