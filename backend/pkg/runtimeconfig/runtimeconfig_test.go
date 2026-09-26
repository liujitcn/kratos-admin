package runtimeconfig

import (
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/config/v1"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestCacheKeyUsesAdminNamespace 验证运行配置使用管理端缓存命名空间。
func TestCacheKeyUsesAdminNamespace(t *testing.T) {
	if key := CacheKey(BaseLogFallbackKey); key != "admin:config:hidden:baseLogFallback" {
		t.Fatalf("CacheKey() = %q; want admin namespace", key)
	}
}

// TestSaveAndLoadJSONUsesAdminCacheKey 验证运行配置只通过管理端缓存键保存和读取。
func TestSaveAndLoadJSONUsesAdminCacheKey(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	value, err := DefaultJSON(BaseLogFallbackKey)
	if err != nil {
		t.Fatal(err)
	}
	if err = SaveJSON(store, BaseLogFallbackKey, value); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Get(CacheKey(BaseLogFallbackKey)); err != nil {
		t.Fatalf("new runtime config cache key was not populated: %v", err)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Key == CacheKey(BaseLogFallbackKey) {
			found = true
			if entry.TTL < 365*24*time.Hour {
				t.Fatalf("runtime config cache TTL = %s; want at least one year", entry.TTL)
			}
			break
		}
	}
	if !found {
		t.Fatal("runtime config cache entry was not listed")
	}
	if _, err = store.Get("base-config:hidden:" + BaseLogFallbackKey); err == nil {
		t.Fatal("legacy runtime config cache key was populated")
	}
	target := &configv1.BaseLogFallbackConfig{}
	if err = LoadJSON(store, BaseLogFallbackKey, target); err != nil {
		t.Fatal(err)
	}
	defaultConfig := DefaultBaseLogFallbackConfig()
	if target.GetFilePath() != defaultConfig.GetFilePath() {
		t.Fatalf("loaded file path = %q; want %q", target.GetFilePath(), defaultConfig.GetFilePath())
	}
}

// TestLoadJSONDoesNotReadLegacyCacheKey 验证运行配置不会回退读取旧缓存键。
func TestLoadJSONDoesNotReadLegacyCacheKey(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	value, err := DefaultJSON(BaseLogFallbackKey)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Set("base-config:hidden:"+BaseLogFallbackKey, value, CacheExpire); err != nil {
		t.Fatal(err)
	}
	if err = LoadJSON(store, BaseLogFallbackKey, &configv1.BaseLogFallbackConfig{}); err == nil {
		t.Fatal("LoadJSON() unexpectedly read the legacy cache key")
	}
}
