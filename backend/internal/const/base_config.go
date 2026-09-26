package _const

import (
	"strconv"
	"time"
)

const (
	// BASE_CONFIG_APP_CACHE_PREFIX 表示应用端配置快照的缓存键前缀。
	BASE_CONFIG_APP_CACHE_PREFIX = "app:config:site:"
	// BASE_CONFIG_ADMIN_CACHE_PREFIX 表示管理端配置快照的缓存键前缀。
	BASE_CONFIG_ADMIN_CACHE_PREFIX = "admin:config:site:"
	// BASE_CONFIG_CACHE_EXPIRE 表示配置缓存的有效期。
	BASE_CONFIG_CACHE_EXPIRE = 24 * time.Hour
	// BASE_CONFIG_KEY_OAUTH_AUTO_REGISTER 表示微信未绑定时是否自动注册用户。
	BASE_CONFIG_KEY_OAUTH_AUTO_REGISTER = "oauthAutoRegister"
	// BASE_CONFIG_KEY_SECURITY_MFA_POLICY 表示全局多因素认证策略。
	BASE_CONFIG_KEY_SECURITY_MFA_POLICY = "securityMfaPolicy"
	// BASE_CONFIG_KEY_SECURITY_MFA_METHOD 表示全局多因素认证方式。
	BASE_CONFIG_KEY_SECURITY_MFA_METHOD = "securityMfaMethod"
)

// BaseConfigCacheKey 生成指定站点的配置缓存键。
func BaseConfigCacheKey(site int32) string {
	prefix := BASE_CONFIG_ADMIN_CACHE_PREFIX
	if site == BASE_CONFIG_SITE_APP {
		prefix = BASE_CONFIG_APP_CACHE_PREFIX
	}
	return prefix + strconv.FormatInt(int64(site), 10)
}
