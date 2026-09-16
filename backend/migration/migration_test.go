package migration

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestTenantRoleIncludesRoleManagement 验证租户管理员模板包含角色管理页面和全部操作权限。
func TestTenantRoleIncludesRoleManagement(t *testing.T) {
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
	for _, id := range []int64{30000000, 30030000, 30030100, 30030200, 30030300, 30030400, 30030500} {
		if !slices.Contains(menus, id) {
			t.Errorf("租户管理员缺少角色管理菜单权限 %d", id)
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
