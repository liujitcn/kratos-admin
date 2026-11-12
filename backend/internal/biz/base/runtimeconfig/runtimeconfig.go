package runtimeconfig

import (
	"errors"

	"github.com/liujitcn/kratos-admin/backend/pkg/runtimeconfig"
	"github.com/liujitcn/kratos-kit/cache"
	"google.golang.org/protobuf/proto"
)

const (
	// BaseLogFallbackKey 表示日志入库回退配置键。
	BaseLogFallbackKey = runtimeconfig.BaseLogFallbackKey
	// SecurityProxyKey 表示反向代理安全配置键。
	SecurityProxyKey = runtimeconfig.SecurityProxyKey
	// SecurityScriptKey 表示脚本沙箱安全配置键。
	SecurityScriptKey = runtimeconfig.SecurityScriptKey
	// SecurityOutboundKey 表示出站网络安全配置键。
	SecurityOutboundKey = runtimeconfig.SecurityOutboundKey
	// SecurityIdentityKey 表示账号安全配置键。
	SecurityIdentityKey = runtimeconfig.SecurityIdentityKey
	// SecurityMigrationKey 表示一次性安全迁移配置键。
	SecurityMigrationKey = runtimeconfig.SecurityMigrationKey
	// CacheExpire 表示运行配置缓存有效期。
	CacheExpire = runtimeconfig.CacheExpire
	// BaseLogFallbackFileName 表示日志入库回退文件的固定文件名。
	BaseLogFallbackFileName = runtimeconfig.BaseLogFallbackFileName
	// RedactedValue 表示管理端返回的敏感配置占位值。
	RedactedValue = runtimeconfig.RedactedValue
)

// BaseLogFallbackConfig 表示日志入库回退配置。
type BaseLogFallbackConfig = runtimeconfig.BaseLogFallbackConfig

// SecurityProxyConfig 表示反向代理安全配置。
type SecurityProxyConfig = runtimeconfig.SecurityProxyConfig

// SecurityScriptConfig 表示脚本沙箱资源和出站白名单配置。
type SecurityScriptConfig = runtimeconfig.SecurityScriptConfig

// SecurityOutboundConfig 表示公共和 AI 出站白名单配置。
type SecurityOutboundConfig = runtimeconfig.SecurityOutboundConfig

// SecurityKeyValue 表示安全配置中的键值项。
type SecurityKeyValue = runtimeconfig.SecurityKeyValue

// SecurityIdentityConfig 表示账号与保留用户名安全配置。
type SecurityIdentityConfig = runtimeconfig.SecurityIdentityConfig

// SecurityMigrationConfig 表示一次性安全迁移配置。
type SecurityMigrationConfig = runtimeconfig.SecurityMigrationConfig

// DefaultBaseLogFallbackConfig 返回日志入库回退配置默认值。
func DefaultBaseLogFallbackConfig() BaseLogFallbackConfig {
	return runtimeconfig.DefaultBaseLogFallbackConfig()
}

// DefaultSecurityProxyConfig 返回反向代理安全配置默认值。
func DefaultSecurityProxyConfig() SecurityProxyConfig {
	return runtimeconfig.DefaultSecurityProxyConfig()
}

// DefaultSecurityScriptConfig 返回脚本沙箱安全配置默认值。
func DefaultSecurityScriptConfig() SecurityScriptConfig {
	return runtimeconfig.DefaultSecurityScriptConfig()
}

// DefaultSecurityOutboundConfig 返回出站网络安全配置默认值。
func DefaultSecurityOutboundConfig() SecurityOutboundConfig {
	return runtimeconfig.DefaultSecurityOutboundConfig()
}

// DefaultSecurityIdentityConfig 返回账号安全配置默认值。
func DefaultSecurityIdentityConfig() SecurityIdentityConfig {
	return runtimeconfig.DefaultSecurityIdentityConfig()
}

// DefaultSecurityMigrationConfig 返回一次性安全迁移配置默认值。
func DefaultSecurityMigrationConfig() SecurityMigrationConfig {
	return runtimeconfig.DefaultSecurityMigrationConfig()
}

// ResolveBaseLogFallbackIntegrityKey 从运行时密钥服务派生日志回退完整性密钥。
func ResolveBaseLogFallbackIntegrityKey() (string, error) {
	return runtimeconfig.ResolveBaseLogFallbackIntegrityKey()
}

// BaseLogFallbackFilePath 根据配置目录返回日志入库回退文件路径。
func BaseLogFallbackFilePath(filePath string) string {
	return runtimeconfig.BaseLogFallbackFilePath(filePath)
}

// CacheKey 返回运行配置缓存键。
func CacheKey(key string) string {
	return runtimeconfig.CacheKey(key)
}

// IsSupportedKey 判断配置键是否由系统运行配置模块管理。
func IsSupportedKey(key string) bool {
	return runtimeconfig.IsSupportedKey(key)
}

// SensitiveFields 返回指定运行配置需要脱敏的字段路径。
func SensitiveFields(key string) []string {
	return runtimeconfig.SensitiveFields(key)
}

// Keys 返回全部受支持的系统运行配置键。
func Keys() []string {
	return runtimeconfig.Keys()
}

// DefaultJSON 返回配置键对应的默认 JSON。
func DefaultJSON(key string) (string, error) {
	return runtimeconfig.DefaultJSON(key)
}

// CanonicalJSON 规范化指定配置键的 ProtoJSON。
func CanonicalJSON(key, value string) (string, error) {
	return runtimeconfig.CanonicalJSON(key, value)
}

// RedactJSON 将运行配置中的敏感字段替换为管理端占位值。
func RedactJSON(key, value string) (string, error) {
	return runtimeconfig.RedactJSON(key, value)
}

// MergeSensitiveJSON 将管理端提交的敏感字段占位值替换为当前已保存值。
func MergeSensitiveJSON(key, current, incoming string) (string, error) {
	return runtimeconfig.MergeSensitiveJSON(key, current, incoming)
}

// ValidateJSON 校验配置 JSON 的结构和值域。
func ValidateJSON(key, value string) error {
	return runtimeconfig.ValidateJSON(key, value)
}

// LoadJSON 从缓存读取并反序列化指定配置。
func LoadJSON(store cache.Cache, key string, target any) error {
	message, ok := target.(proto.Message)
	if !ok {
		return errors.New("运行配置目标消息类型无效")
	}
	return runtimeconfig.LoadJSON(store, key, message)
}

// SaveJSON 校验并保存指定配置的缓存值。
func SaveJSON(store cache.Cache, key, value string) error {
	return runtimeconfig.SaveJSON(store, key, value)
}
