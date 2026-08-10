package _const

import (
	"strconv"
	"time"
)

const (
	// BASE_CONFIG_CACHE_PREFIX 表示按站点缓存配置快照的键前缀。
	BASE_CONFIG_CACHE_PREFIX = "config:site:"
	// BASE_CONFIG_CACHE_EXPIRE 表示配置缓存的有效期。
	BASE_CONFIG_CACHE_EXPIRE = 100 * 365 * 24 * time.Hour
	// BASE_CONFIG_KEY_OAUTH_AUTO_REGISTER 表示微信未绑定时是否自动注册用户。
	BASE_CONFIG_KEY_OAUTH_AUTO_REGISTER = "oauthAutoRegister"
)

// BaseConfigCacheKey 生成指定站点的配置缓存键。
func BaseConfigCacheKey(site int32) string {
	return BASE_CONFIG_CACHE_PREFIX + strconv.FormatInt(int64(site), 10)
}
