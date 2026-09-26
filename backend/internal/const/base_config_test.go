package _const

import "testing"

// TestCacheKeysUseTerminalAndSharedNamespaces 验证缓存键按终端归属或共享范围使用命名空间。
func TestCacheKeysUseTerminalAndSharedNamespaces(t *testing.T) {
	if key := BaseConfigCacheKey(BASE_CONFIG_SITE_SYSTEM); key != "admin:config:site:1" {
		t.Fatalf("system config cache key = %q; want admin namespace", key)
	}
	if key := BaseConfigCacheKey(BASE_CONFIG_SITE_ADMIN); key != "admin:config:site:2" {
		t.Fatalf("admin config cache key = %q; want admin namespace", key)
	}
	if key := BaseConfigCacheKey(BASE_CONFIG_SITE_APP); key != "app:config:site:3" {
		t.Fatalf("app config cache key = %q; want app namespace", key)
	}
	if key := AppDictCacheKey("4", "status"); key != "app:dict:data:4:status" {
		t.Fatalf("app dict cache key = %q", key)
	}
	if key := AppMenuCacheKey("4"); key != "app:menu:tree:4" {
		t.Fatalf("app menu cache key = %q", key)
	}
	if key := NotificationCategoryCacheKey("4"); key != "shared:notification:categories:4" {
		t.Fatalf("shared notification category cache key = %q", key)
	}
}
