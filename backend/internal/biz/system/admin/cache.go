package biz

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/cache"
)

const cacheQueryDefaultPageSize int64 = 20

// CacheCase 提供当前进程运行时缓存的只读查询能力。
type CacheCase struct {
	*biz.BaseCase
}

// NewCacheCase 创建运行时缓存查询业务实例。
func NewCacheCase(baseCase *biz.BaseCase) *CacheCase {
	return &CacheCase{BaseCase: baseCase}
}

// PageCache 分页查询当前进程缓存条目。
func (c *CacheCase) PageCache(_ context.Context, req *adminv1.PageCacheRequest) (*adminv1.PageCacheResponse, error) {
	entries, err := c.Cache.List()
	if err != nil {
		return nil, errorsx.Internal("查询缓存条目失败").WithCause(err)
	}
	keyword := strings.ToLower(req.GetKeyword())
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	filtered := make([]cache.Entry, 0, len(entries))
	for _, entry := range entries {
		if keyword == "" || strings.Contains(strings.ToLower(entry.Key), keyword) {
			filtered = append(filtered, entry)
		}
	}
	pageNum := req.GetPageNum()
	if pageNum < 1 {
		pageNum = 1
	}
	pageSize := req.GetPageSize()
	if pageSize < 1 {
		pageSize = cacheQueryDefaultPageSize
	}
	total := int64(len(filtered))
	start := (pageNum - 1) * pageSize
	if start >= total {
		return &adminv1.PageCacheResponse{CacheEntries: []*adminv1.CacheEntry{}, Total: int32(total)}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	items := make([]*adminv1.CacheEntry, 0, end-start)
	for _, entry := range filtered[start:end] {
		fields := make([]*adminv1.CacheField, 0, len(entry.Fields))
		fieldKeys := make([]string, 0, len(entry.Fields))
		for fieldKey := range entry.Fields {
			fieldKeys = append(fieldKeys, fieldKey)
		}
		sort.Strings(fieldKeys)
		for _, fieldKey := range fieldKeys {
			fields = append(fields, &adminv1.CacheField{Key: strings.ToValidUTF8(fieldKey, "\uFFFD"), Value: strings.ToValidUTF8(sanitizeCacheValue(entry.Key, entry.Fields[fieldKey]), "\uFFFD")})
		}
		expiresAt := ""
		if !entry.ExpiresAt.IsZero() {
			expiresAt = entry.ExpiresAt.Format(time.RFC3339Nano)
		}
		items = append(items, &adminv1.CacheEntry{
			Key:        strings.ToValidUTF8(entry.Key, "\uFFFD"),
			Type:       entry.Type,
			Value:      strings.ToValidUTF8(sanitizeCacheValue(entry.Key, entry.Value), "\uFFFD"),
			Fields:     fields,
			TtlSeconds: ttlSeconds(entry.TTL),
			ExpiresAt:  expiresAt,
			CreatedAt:  entry.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:  entry.UpdatedAt.Format(time.RFC3339Nano),
		})
	}
	return &adminv1.PageCacheResponse{CacheEntries: items, Total: int32(total)}, nil
}

func ttlSeconds(ttl time.Duration) int64 {
	if ttl < 0 {
		return -1
	}
	return int64(ttl.Seconds())
}

// sanitizeCacheValue 脱敏隐藏运行配置缓存中的凭据字段。
func sanitizeCacheValue(key, value string) string {
	if !strings.HasPrefix(key, "base-config:hidden:") {
		return value
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value
	}
	configKey := strings.TrimPrefix(key, "base-config:hidden:")
	fields := runtimeconfig.SensitiveFields(configKey)
	if len(fields) == 0 {
		fields = []string{"password", "integrity_key", "encryption_key"}
	}
	for _, field := range fields {
		if _, ok := payload[field]; ok {
			payload[field] = "[REDACTED]"
		}
	}
	sanitized, err := json.Marshal(payload)
	if err != nil {
		return value
	}
	return string(sanitized)
}
