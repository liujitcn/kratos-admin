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
	// CacheExpire 表示运行配置缓存有效期。
	CacheExpire = runtimeconfig.CacheExpire
	// BaseLogFallbackFileName 表示日志入库回退文件的固定文件名。
	BaseLogFallbackFileName = runtimeconfig.BaseLogFallbackFileName
	// RedactedValue 表示管理端返回的敏感配置占位值。
	RedactedValue = runtimeconfig.RedactedValue
)

// BaseLogFallbackConfig 表示日志入库回退配置。
type BaseLogFallbackConfig = runtimeconfig.BaseLogFallbackConfig

// DefaultBaseLogFallbackConfig 返回日志入库回退配置默认值。
func DefaultBaseLogFallbackConfig() BaseLogFallbackConfig {
	return runtimeconfig.DefaultBaseLogFallbackConfig()
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
