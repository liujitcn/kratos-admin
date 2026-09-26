package _const

const (
	// NOTIFICATION_CATEGORY_CACHE_REVISION_KEY 表示共享消息分类缓存版本键。
	NOTIFICATION_CATEGORY_CACHE_REVISION_KEY = "shared:notification:category:revision"
)

// NotificationCategoryCacheKey 生成指定版本的共享消息分类缓存键。
func NotificationCategoryCacheKey(revision string) string {
	return "shared:notification:categories:" + revision
}
