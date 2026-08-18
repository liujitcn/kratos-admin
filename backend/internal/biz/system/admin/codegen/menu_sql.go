package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

const (
	generatedMenuSQLFileName      = "default_data.up.sql"
	initialMigrationVersionName   = "v0.0.1"
	migrationVersionFormatSemVer  = "semver"
	migrationVersionFormatNumeric = "numeric"
)

var generatedMigrationVersionPattern = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)(?:[-.]([0-9]{14}))?$`)

type generatedMigrationVersion struct {
	name      string
	format    string
	major     uint64
	minor     uint64
	patch     uint64
	timestamp uint64
	width     int
}

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
	writeMenuTranslationSQL(&builder, "@codegen_page_menu_id", pageSpec, localeState)
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
		writeMenuTranslationSQL(&builder, varName, buttonSpec, localeState)
	}
	writeStaleStatusMenuSQL(&builder, table, buttonSpecs)
	builder.WriteString("\n-- 代码生成菜单权限脚本结束。\n")
	return builder.String()
}

// writeMenuTranslationSQL 写入启用的非主语言菜单译文，已有记录一律保留。
func writeMenuTranslationSQL(builder *strings.Builder, menuIDExpression string, spec CodeGenMenuSpec, localeState LocaleState) {
	for _, localeValue := range RequiredTranslationLocales(localeState) {
		title := spec.Translations[localeValue]
		if title == "" {
			continue
		}
		builder.WriteString("INSERT IGNORE INTO `base_i18n` (`target_type`, `target_id`, `locale`, `name`)\n")
		builder.WriteString("SELECT ")
		builder.WriteString(strconv.FormatInt(int64(_const.TRANSLATION_TARGET_TYPE_BASE_MENU), 10))
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
		builder.WriteString(strconv.FormatInt(int64(_const.TRANSLATION_TARGET_TYPE_BASE_MENU), 10))
		builder.WriteString(" AND `target_id` = ")
		builder.WriteString(menuIDExpression)
		builder.WriteString(" AND `locale` = ")
		builder.WriteString(sqlString(localeValue))
		builder.WriteString(");\n")
	}
}

// newGeneratedMenuSQLPreviewFile 创建新版本迁移 SQL 的菜单权限预览文件。
func (c *renderer) newGeneratedMenuSQLPreviewFile(table *Table, content string) *systemadminv1.CodeGenPreviewFile {
	path, err := nextGeneratedMenuSQLPath(c.migrationVersion)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Action: "skip", Content: content, Message: err.Error()}
	}
	_, err = SafeRepoFilePath(path)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: err.Error()}
	}
	var current []byte
	current, err = c.readRepoFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: content, Message: err.Error()}
		}
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "create",
			Content: generatedMenuSQLBlock(table, content) + "\n",
			Message: fmt.Sprintf("将新增 %s", path),
		}
	}
	var merged string
	merged, err = mergeGeneratedMenuSQLAtPath(string(current), table, content, path)
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
			Message: fmt.Sprintf("%s 中的菜单和按钮权限 SQL 无需更新", path),
		}
	}
	return &systemadminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  "update",
		Content: merged,
		Exists:  true,
		Message: fmt.Sprintf("将更新 %s 中的菜单和按钮权限 SQL", path),
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

// nextGeneratedMenuSQLPath 返回可复用或下一版本的菜单 SQL 路径。
func nextGeneratedMenuSQLPath(appliedVersion string) (string, error) {
	relativeDir, directory, err := findMigrationAssetsDirectory()
	if err != nil {
		return "", err
	}
	var entries []os.DirEntry
	entries, err = os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("读取迁移资源目录%s失败: %w", relativeDir, err)
	}
	var pendingPath string
	var found bool
	pendingPath, found, err = findPendingGeneratedMenuSQLPath(directory, relativeDir, entries, appliedVersion)
	if err != nil {
		return "", err
	}
	if found {
		return pendingPath, nil
	}
	var versionName string
	versionName, err = nextGeneratedMigrationVersion(entries)
	if err != nil {
		return "", err
	}
	if appliedVersion != "" {
		var databaseVersionName string
		var ok bool
		databaseVersionName, ok, err = nextMigrationVersionAfter(appliedVersion)
		if err != nil {
			return "", err
		}
		if ok && generatedMigrationVersionGreater(databaseVersionName, versionName) {
			versionName = databaseVersionName
		}
	}
	return filepath.ToSlash(filepath.Join(relativeDir, versionName, generatedMenuSQLFileName)), nil
}

// findPendingGeneratedMenuSQLPath 查找尚未成功执行且已有代码生成标记的迁移脚本。
func findPendingGeneratedMenuSQLPath(
	directory string,
	relativeDir string,
	entries []os.DirEntry,
	appliedVersion string,
) (string, bool, error) {
	var latest generatedMigrationVersion
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version, ok := parseGeneratedMigrationVersion(entry.Name())
		if !ok || isGeneratedMigrationVersionApplied(entry.Name(), appliedVersion) {
			continue
		}
		path := filepath.Join(directory, entry.Name(), generatedMenuSQLFileName)
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", false, err
		}
		if !strings.Contains(string(content), "-- CODEGEN_MENU_BEGIN table=") ||
			!strings.Contains(string(content), "-- CODEGEN_MENU_END table=") {
			continue
		}
		if !found || compareGeneratedMigrationVersion(version, latest) < 0 {
			latest = version
			found = true
		}
	}
	if !found {
		return "", false, nil
	}
	return filepath.ToSlash(filepath.Join(relativeDir, latest.name, generatedMenuSQLFileName)), true, nil
}

// generatedMigrationVersionGreater 判断左侧迁移版本是否大于右侧版本。
func generatedMigrationVersionGreater(leftName string, rightName string) bool {
	left, leftOK := parseGeneratedMigrationVersion(leftName)
	right, rightOK := parseGeneratedMigrationVersion(rightName)
	return leftOK && rightOK && compareGeneratedMigrationVersion(left, right) > 0
}

// isGeneratedMigrationVersionApplied 判断迁移版本是否已经成功执行。
func isGeneratedMigrationVersionApplied(versionName string, appliedVersion string) bool {
	if appliedVersion == "" {
		return false
	}
	version, versionOK := parseGeneratedMigrationVersion(versionName)
	applied, appliedOK := parseGeneratedMigrationVersion(appliedVersion)
	if !versionOK || !appliedOK {
		return versionName == appliedVersion
	}
	return compareGeneratedMigrationVersion(version, applied) <= 0
}

// findMigrationAssetsDirectory 查找当前仓库中的迁移资源目录。
func findMigrationAssetsDirectory() (string, string, error) {
	root, err := filepath.Abs(repoRoot())
	if err != nil {
		return "", "", err
	}
	candidates := []string{filepath.Join(root, "migration", "assets", "mysql")}
	var entries []os.DirEntry
	entries, err = os.ReadDir(root)
	if err != nil {
		return "", "", fmt.Errorf("读取仓库目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(root, entry.Name(), "migration", "assets", "mysql"))
		}
	}
	for _, candidate := range candidates {
		var info os.FileInfo
		info, err = os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", "", err
		}
		if !info.IsDir() {
			continue
		}
		var relativeDir string
		relativeDir, err = filepath.Rel(root, candidate)
		if err != nil {
			return "", "", err
		}
		return filepath.ToSlash(relativeDir), candidate, nil
	}
	return "", "", fmt.Errorf("未找到 migration/assets/mysql 迁移资源目录")
}

// nextGeneratedMigrationVersion 返回迁移资源目录下的下一版本名称。
func nextGeneratedMigrationVersion(entries []os.DirEntry) (string, error) {
	var latest generatedMigrationVersion
	hasLatest := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version, ok := parseGeneratedMigrationVersion(entry.Name())
		if !ok {
			continue
		}
		if !hasLatest || compareGeneratedMigrationVersion(version, latest) > 0 {
			latest = version
			hasLatest = true
		}
	}
	if !hasLatest {
		return initialMigrationVersionName, nil
	}
	return incrementGeneratedMigrationVersion(latest)
}

// nextMigrationVersionAfter 返回指定迁移版本的下一版本名称。
func nextMigrationVersionAfter(versionName string) (string, bool, error) {
	version, ok := parseGeneratedMigrationVersion(versionName)
	if !ok {
		return "", false, nil
	}
	nextVersion, err := incrementGeneratedMigrationVersion(version)
	if err != nil {
		return "", false, err
	}
	return nextVersion, true, nil
}

// incrementGeneratedMigrationVersion 将迁移版本的补丁号递增。
func incrementGeneratedMigrationVersion(latest generatedMigrationVersion) (string, error) {
	if latest.patch == ^uint64(0) {
		return "", fmt.Errorf("迁移版本%s无法继续递增", latest.name)
	}
	latest.patch++
	if latest.format == migrationVersionFormatNumeric {
		versionName := strconv.FormatUint(latest.patch, 10)
		if len(versionName) < latest.width {
			versionName = strings.Repeat("0", latest.width-len(versionName)) + versionName
		}
		return versionName, nil
	}
	return fmt.Sprintf("v%d.%d.%d", latest.major, latest.minor, latest.patch), nil
}

// parseGeneratedMigrationVersion 解析代码生成器支持的迁移版本目录名。
func parseGeneratedMigrationVersion(name string) (generatedMigrationVersion, bool) {
	matches := generatedMigrationVersionPattern.FindStringSubmatch(name)
	if len(matches) == 5 {
		version := generatedMigrationVersion{name: name, format: migrationVersionFormatSemVer}
		var err error
		version.major, err = strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			return generatedMigrationVersion{}, false
		}
		version.minor, err = strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			return generatedMigrationVersion{}, false
		}
		version.patch, err = strconv.ParseUint(matches[3], 10, 64)
		if err != nil {
			return generatedMigrationVersion{}, false
		}
		if matches[4] != "" {
			version.timestamp, err = strconv.ParseUint(matches[4], 10, 64)
			if err != nil {
				return generatedMigrationVersion{}, false
			}
		}
		return version, true
	}
	patch, err := strconv.ParseUint(name, 10, 64)
	if err != nil {
		return generatedMigrationVersion{}, false
	}
	return generatedMigrationVersion{name: name, format: migrationVersionFormatNumeric, patch: patch, width: len(name)}, true
}

// compareGeneratedMigrationVersion 按主、次、补丁版本号比较迁移版本。
func compareGeneratedMigrationVersion(left generatedMigrationVersion, right generatedMigrationVersion) int {
	if left.major != right.major {
		if left.major < right.major {
			return -1
		}
		return 1
	}
	if left.minor != right.minor {
		if left.minor < right.minor {
			return -1
		}
		return 1
	}
	if left.patch < right.patch {
		return -1
	}
	if left.patch > right.patch {
		return 1
	}
	if left.timestamp < right.timestamp {
		return -1
	}
	if left.timestamp > right.timestamp {
		return 1
	}
	return 0
}

// isGeneratedMenuSQLPath 判断路径是否是代码生成器使用的迁移菜单 SQL 文件。
func isGeneratedMenuSQLPath(path string) bool {
	normalizedPath := filepath.ToSlash(filepath.Clean(path))
	if filepath.Base(normalizedPath) != generatedMenuSQLFileName {
		return false
	}
	versionDirectory := filepath.Dir(normalizedPath)
	if _, ok := parseGeneratedMigrationVersion(filepath.Base(versionDirectory)); !ok {
		return false
	}
	assetsDirectory := filepath.Dir(versionDirectory)
	return filepath.Base(assetsDirectory) == "mysql" &&
		filepath.Base(filepath.Dir(assetsDirectory)) == "assets" &&
		filepath.Base(filepath.Dir(filepath.Dir(assetsDirectory))) == "migration"
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
