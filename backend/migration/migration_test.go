package migration

import (
	"io/fs"
	"strings"
	"testing"
)

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
