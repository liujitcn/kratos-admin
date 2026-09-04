package biz

import (
	"strings"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

func TestPageCachePaginatesAndReturnsMetadata(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err = store.Set("cache:a", "a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = store.Set("cache:b", "b", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = store.Set("cache:c", "c", time.Minute); err != nil {
		t.Fatal(err)
	}
	cacheCase := NewCacheCase(&biz.BaseCase{Cache: store})
	response, err := cacheCase.PageCache(nil, &adminv1.PageCacheRequest{PageNum: 2, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetTotal() != 3 || len(response.GetCacheEntries()) != 1 {
		t.Fatalf("PageCache() returned total=%d entries=%d, want 3/1", response.GetTotal(), len(response.GetCacheEntries()))
	}
	entry := response.GetCacheEntries()[0]
	if entry.GetKey() != "cache:c" || entry.GetValue() != "c" || entry.GetTtlSeconds() <= 0 || entry.GetExpiresAt() == "" || entry.GetCreatedAt() == "" || entry.GetUpdatedAt() == "" {
		t.Fatalf("PageCache() returned incomplete entry: %+v", entry)
	}
}

// TestSanitizeCacheValueMasksRuntimeConfigSecrets 验证缓存查询不会泄露运行配置凭据。
func TestSanitizeCacheValueMasksRuntimeConfigSecrets(t *testing.T) {
	value := `{"file_path":"./logs/base-log-fallback","password":"hmac"}`
	sanitized := sanitizeCacheValue("base-config:hidden:baseLogFallback", value)
	if strings.Contains(sanitized, "secret") || strings.Contains(sanitized, "hmac") {
		t.Fatalf("runtime config secret leaked: %s", sanitized)
	}
	if !strings.Contains(sanitized, "[REDACTED]") {
		t.Fatalf("runtime config secret was not masked: %s", sanitized)
	}
}
