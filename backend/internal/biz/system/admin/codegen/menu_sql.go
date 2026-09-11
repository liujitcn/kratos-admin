package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

const generatedMenuSQLFileName = "default_data.up.sql"

// RenderGeneratedMenuSQL 使用生成时的菜单快照输出固定编号插入和当前角色授权语句。
func RenderGeneratedMenuSQL(table *Table, columns []*CodeGenColumn, methods []*Proto, resourcePath string, tableComment string, localeState LocaleState) (string, error) {
	pageSpec, buttonSpecs := MenuSpecs(table, columns, methods, resourcePath, tableComment, localeState)
	state := table.MenuSQLState
	if state == nil {
		state = &MenuSQLState{}
	}
	specs := append([]CodeGenMenuSpec{pageSpec}, buttonSpecs...)
	var builder strings.Builder
	builder.WriteString("-- 代码生成菜单权限脚本，请勿手工修改。\n-- 菜单编号取自生成时数据库，角色授权保留已有菜单。\n\n")
	for index, spec := range specs {
		if index > 0 {
			spec.Menu.ParentID = pageSpec.Menu.ID
		}
		err := state.resolveMenu(spec.Menu)
		if err != nil {
			return "", err
		}
		menu := spec.Menu
		fmt.Fprintf(&builder, "INSERT INTO `base_menu` (`id`, `parent_id`, `type`, `path`, `name`, `component`, `redirect`, `meta`, `api`, `sort`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)\nVALUES (%d, %d, %d, %s, %s, %s, %s, %s, %s, %d, %d, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)\nON DUPLICATE KEY UPDATE `parent_id` = VALUES(`parent_id`), `type` = VALUES(`type`), `path` = VALUES(`path`), `name` = VALUES(`name`), `component` = VALUES(`component`), `redirect` = VALUES(`redirect`), `meta` = VALUES(`meta`), `api` = VALUES(`api`), `sort` = VALUES(`sort`), `status` = VALUES(`status`);\n", menu.ID, menu.ParentID, menu.Type, sqlString(menu.Path), sqlString(menu.Name), sqlString(menu.Component), sqlString(menu.Redirect), sqlString(menu.Meta), sqlString(menu.API), menu.Sort, menu.Status)
		for _, locale := range RequiredI18nLocales(localeState) {
			if title := spec.I18ns[locale]; title != "" {
				fmt.Fprintf(&builder, "INSERT IGNORE INTO `base_i18n` (`target_type`, `target_id`, `locale`, `name`) VALUES (%d, %d, %s, %s);\n", _const.I18N_TARGET_TYPE_BASE_MENU_META_TITLE, menu.ID, sqlString(locale), sqlString(title))
			}
		}
	}
	if state.RoleID > 0 {
		ids := []int64{pageSpec.Menu.ParentID}
		for _, spec := range specs {
			ids = append(ids, spec.Menu.ID)
		}
		for _, id := range ids {
			fmt.Fprintf(&builder, "UPDATE `base_role` SET `menus` = JSON_ARRAY_APPEND(COALESCE(`menus`, JSON_ARRAY()), '$', %d) WHERE `id` = %d AND NOT JSON_CONTAINS(COALESCE(`menus`, JSON_ARRAY()), '%d');\n", id, state.RoleID, id)
		}
	}
	return builder.String(), nil
}

// resolveMenu 复用现有菜单编号，并为批次新菜单预留首个未占用层级编号。
func (s *MenuSQLState) resolveMenu(menu *models.BaseMenu) error {
	used := make(map[int64]bool, len(s.Menus))
	for index, existing := range s.Menus {
		used[existing.ID] = true
		if existing.DeletedAt != 0 || existing.Type != menu.Type {
			continue
		}
		matches := existing.ParentID == menu.ParentID && (existing.Path == menu.Path || existing.API == menu.API)
		if menu.Type == _const.BASE_MENU_TYPE_MENU {
			matches = existing.Path == menu.Path || existing.Name == menu.Name || existing.Component == menu.Component
		}
		if matches {
			if existing.ParentID != menu.ParentID {
				return fmt.Errorf("已生成菜单不能更换父级: %s", menu.Path)
			}
			menu.ID = existing.ID
			s.Menus[index] = menu
			return nil
		}
	}
	for sequence := int64(1); sequence <= 99; sequence++ {
		var id int64
		switch {
		case menu.ParentID < 10000000 || menu.ParentID > 99999999:
			return fmt.Errorf("父级菜单编号无效: %d", menu.ParentID)
		case menu.ParentID%1000000 == 0:
			id = menu.ParentID + sequence*10000
		case menu.ParentID%10000 == 0:
			id = menu.ParentID + sequence*100
		case menu.ParentID%100 == 0:
			id = menu.ParentID + sequence
		default:
			suffix := menu.ParentID%100*10 + sequence
			if suffix > 99 {
				continue
			}
			id = menu.ParentID/100*100 + suffix
		}
		if !used[id] {
			menu.ID = id
			s.Menus = append(s.Menus, menu)
			return nil
		}
	}
	return fmt.Errorf("父级菜单 %d 的子编号已用完", menu.ParentID)
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

// sqlString 将文本安全编码为 MySQL 字符串字面量。
func sqlString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "''")
	return "'" + value + "'"
}
