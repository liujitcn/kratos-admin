package biz

import (
	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-kit/cache"
)

// incrementCacheRevision 递增共享缓存版本，缓存故障时记录日志并由短 TTL 兜底。
func incrementCacheRevision(store cache.Cache, revisionKey string) {
	if err := cache.IncrementRevision(store, revisionKey); err != nil {
		log.Error("increment cache revision", "key", revisionKey, "error", err)
	}
}
