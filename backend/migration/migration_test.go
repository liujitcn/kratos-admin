package migration

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestTenantRoleIncludesRoleAndProjectManagement 验证租户管理员模板包含角色管理和项目授权权限。
func TestTenantRoleIncludesRoleAndProjectManagement(t *testing.T) {
	content, err := fs.ReadFile(Assets(), "v0.0.1/mysql/default_data.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile("(?m)^INSERT IGNORE INTO `base_role`.*'tenant', 1, '(\\[[^']*\\])'")
	match := pattern.FindSubmatch(content)
	if len(match) != 2 {
		t.Fatal("未找到租户管理员权限模板")
	}
	var menus []int64
	err = json.Unmarshal(match[1], &menus)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{30000000, 30030000, 30030100, 30030200, 30030300, 30030400, 30030500, 20040100, 20040200, 20040201, 20040202, 20040203} {
		if !slices.Contains(menus, id) {
			t.Errorf("租户管理员缺少菜单权限 %d", id)
		}
	}
}

// TestSecurityDefaultsMatchMigrationDocumentation 验证登录策略和在线会话的默认权限与迁移说明一致。
func TestSecurityDefaultsMatchMigrationDocumentation(t *testing.T) {
	content, err := fs.ReadFile(Assets(), "v0.0.1/mysql/default_data.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	adminMatch := regexp.MustCompile("(?m)^INSERT IGNORE INTO `base_role`.*'admin', 1, '(\\[[^']*\\])'").FindSubmatch(content)
	if len(adminMatch) != 2 {
		t.Fatal("未找到管理员权限模板")
	}
	var adminMenus []int64
	err = json.Unmarshal(adminMatch[1], &adminMenus)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{91030100, 91030101, 91030102, 91030103, 91030104} {
		if slices.Contains(adminMenus, id) {
			t.Errorf("平台管理员不应拥有登录策略菜单权限 %d", id)
		}
	}
	for _, id := range []int64{91030200, 91030201} {
		if !slices.Contains(adminMenus, id) {
			t.Errorf("平台管理员缺少在线会话菜单权限 %d", id)
		}
	}
	if !strings.Contains(string(content), "INSERT IGNORE INTO `base_login_policy`") {
		t.Fatal("默认迁移未创建全局登录策略记录")
	}
	documentation, err := fs.ReadFile(Assets(), "v0.0.1/mysql/README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"登录策略菜单及其服务、按钮权限只授予平台超级管理员", "在线会话菜单及下线权限授予平台管理员和租户管理员", "默认创建一条全局登录策略记录"} {
		if !strings.Contains(string(documentation), phrase) {
			t.Errorf("迁移说明缺少安全默认语义: %s", phrase)
		}
	}
}

// TestAssetsUsesEmbeddedResourcesByDefault 验证默认资源不受宿主工作目录影响。
func TestAssetsUsesEmbeddedResourcesByDefault(t *testing.T) {
	content, err := fs.ReadFile(Assets(), "v0.0.1/mysql/default_data.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "base_language") {
		t.Fatal("默认迁移资源未读取 Admin 初始化脚本")
	}
}
