package runtimeconfig

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"buf.build/go/protovalidate"
	configv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/config/v1"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// BaseLogFallbackKey 表示日志入库回退配置键。
	BaseLogFallbackKey = "baseLogFallback"
	// CacheExpire 表示运行配置缓存有效期。
	CacheExpire = 100 * 365 * 24 * time.Hour
	// BaseLogFallbackFileName 表示日志入库回退文件的固定文件名。
	BaseLogFallbackFileName = "admin-log.jsonl"
	// BaseLogFallbackIntegrityKeyName 表示日志入库回退完整性密钥的固定名称。
	BaseLogFallbackIntegrityKeyName = "kratos-admin:base-log-fallback/integrity"
	// RedactedValue 表示管理端返回的敏感配置占位值。
	RedactedValue = "[REDACTED]"
)

// BaseLogFallbackConfig 表示日志入库回退配置。
type BaseLogFallbackConfig = configv1.BaseLogFallbackConfig

// ResolveBaseLogFallbackIntegrityKey 从运行时密钥服务派生日志回退完整性密钥。
func ResolveBaseLogFallbackIntegrityKey() (string, error) {
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return "", errors.New("日志入库回退完整性密钥为空且运行时密钥未初始化")
	}
	derived, err := keyValue.Derive(context.Background(), BaseLogFallbackIntegrityKeyName)
	if err != nil {
		return "", fmt.Errorf("派生日志入库回退完整性密钥失败: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(derived), nil
}

// BaseLogFallbackFilePath 根据配置目录返回日志入库回退文件路径。
func BaseLogFallbackFilePath(filePath string) string {
	return filepath.Join(filePath, BaseLogFallbackFileName)
}

// Definition 描述一个可被 Admin 管理的运行配置。
type Definition struct {
	// Key 是配置在数据库和缓存中的稳定键。
	Key string
	// New 创建一个不带默认值的 Proto 配置消息。
	New func() proto.Message
	// Default 创建一个带业务默认值的 Proto 配置消息。
	Default func() proto.Message
	// Owner 标识贡献该配置的模块。
	Owner string
	// NameKey 是配置名称的国际化消息键。
	NameKey string
	// DescriptionKey 是配置说明的国际化消息键。
	DescriptionKey string
	// SensitiveFields 是需要在管理端和缓存诊断中脱敏的 Proto 字段路径。
	SensitiveFields []string
}

// Registry 管理运行配置定义，并为配置读写提供稳定的 Proto 接缝。
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

var defaultRegistry = mustNewRegistry()

// NewRegistry 创建包含 Admin 内置配置的运行配置注册表。
func NewRegistry(values ...Definition) (*Registry, error) {
	registry := &Registry{definitions: make(map[string]Definition)}
	definitions := append(builtinDefinitions(), values...)
	var err error
	for _, definition := range definitions {
		err = registry.Register(definition)
		if err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// DefaultRegistry 返回进程内默认运行配置注册表。
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register 向进程内默认注册表追加一个外部模块的运行配置定义。
func Register(definition Definition) error {
	return defaultRegistry.Register(definition)
}

// Register 向注册表追加一个运行配置定义。
func (r *Registry) Register(definition Definition) error {
	if r == nil {
		return errors.New("运行配置注册表未配置")
	}
	if definition.Key == "" {
		return errors.New("运行配置键不能为空")
	}
	if definition.New == nil || definition.Default == nil {
		return fmt.Errorf("运行配置 %s 未配置 Proto 工厂", definition.Key)
	}
	message := definition.New()
	defaultMessage := definition.Default()
	if message == nil || defaultMessage == nil {
		return fmt.Errorf("运行配置 %s 的 Proto 工厂返回空消息", definition.Key)
	}
	if message.ProtoReflect().Descriptor().FullName() != defaultMessage.ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("运行配置 %s 的 Proto 工厂类型不一致", definition.Key)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[definition.Key]; exists {
		return fmt.Errorf("运行配置键已注册: %s", definition.Key)
	}
	r.definitions[definition.Key] = definition
	return nil
}

// Definitions 返回按配置键排序的运行配置定义快照。
func (r *Registry) Definitions() []Definition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		values = append(values, definition)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Key < values[j].Key })
	return values
}

// Keys 返回按配置键排序的运行配置键集合。
func (r *Registry) Keys() []string {
	definitions := r.Definitions()
	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		keys = append(keys, definition.Key)
	}
	return keys
}

// IsSupportedKey 判断配置键是否已注册。
func (r *Registry) IsSupportedKey(key string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.definitions[key]
	return exists
}

// Definition 返回指定配置键的定义。
func (r *Registry) Definition(key string) (Definition, error) {
	if r == nil {
		return Definition{}, errors.New("运行配置注册表未配置")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, exists := r.definitions[key]
	if !exists {
		return Definition{}, fmt.Errorf("不支持的系统配置键: %s", key)
	}
	return definition, nil
}

// SensitiveFields 返回指定配置需要脱敏的字段路径。
func (r *Registry) SensitiveFields(key string) ([]string, error) {
	definition, err := r.Definition(key)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), definition.SensitiveFields...), nil
}

// RedactJSON 将运行配置中的敏感字段替换为管理端占位值。
func (r *Registry) RedactJSON(key, value string) (string, error) {
	definition, err := r.Definition(key)
	if err != nil {
		return "", err
	}
	var payload map[string]json.RawMessage
	err = json.Unmarshal([]byte(value), &payload)
	if err != nil || payload == nil {
		if err == nil {
			err = errors.New("系统配置 JSON 必须是对象")
		}
		return "", fmt.Errorf("解析系统配置 JSON 失败: %w", err)
	}
	for _, fieldPath := range definition.SensitiveFields {
		parts := strings.Split(fieldPath, ".")
		if _, exists := getJSONField(payload, parts); !exists {
			continue
		}
		err = setJSONField(payload, parts, json.RawMessage(`"`+RedactedValue+`"`))
		if err != nil {
			return "", fmt.Errorf("脱敏系统配置 JSON 失败: %w", err)
		}
	}
	var data []byte
	data, err = json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("生成脱敏系统配置 JSON 失败: %w", err)
	}
	return string(data), nil
}

// MergeSensitiveJSON 将管理端提交的敏感字段占位值替换为当前已保存值。
func (r *Registry) MergeSensitiveJSON(key, current, incoming string) (string, error) {
	definition, err := r.Definition(key)
	if err != nil {
		return "", err
	}
	var currentPayload map[string]json.RawMessage
	err = json.Unmarshal([]byte(current), &currentPayload)
	if err != nil || currentPayload == nil {
		if err == nil {
			err = errors.New("当前系统配置 JSON 必须是对象")
		}
		return "", fmt.Errorf("解析当前系统配置 JSON 失败: %w", err)
	}
	var incomingPayload map[string]json.RawMessage
	err = json.Unmarshal([]byte(incoming), &incomingPayload)
	if err != nil || incomingPayload == nil {
		if err == nil {
			err = errors.New("提交的系统配置 JSON 必须是对象")
		}
		return "", fmt.Errorf("解析提交系统配置 JSON 失败: %w", err)
	}
	for _, fieldPath := range definition.SensitiveFields {
		parts := strings.Split(fieldPath, ".")
		currentValue, exists := getJSONField(currentPayload, parts)
		if !exists {
			incomingValue, incomingExists := getJSONField(incomingPayload, parts)
			if incomingExists && isRedactedJSONValue(incomingValue) {
				err = setJSONField(incomingPayload, parts, json.RawMessage(`""`))
				if err != nil {
					return "", fmt.Errorf("清理敏感系统配置 JSON 失败: %w", err)
				}
			}
			continue
		}
		incomingValue, incomingExists := getJSONField(incomingPayload, parts)
		if !incomingExists || isRedactedJSONValue(incomingValue) {
			err = setJSONField(incomingPayload, parts, currentValue)
			if err != nil {
				return "", fmt.Errorf("恢复敏感系统配置 JSON 失败: %w", err)
			}
		}
	}
	var data []byte
	data, err = json.Marshal(incomingPayload)
	if err != nil {
		return "", fmt.Errorf("生成合并系统配置 JSON 失败: %w", err)
	}
	return string(data), nil
}

// CacheKey 返回运行配置缓存键。
func CacheKey(key string) string {
	return "base-config:hidden:" + key
}

// DefaultJSON 返回配置键对应的默认 ProtoJSON。
func (r *Registry) DefaultJSON(key string) (string, error) {
	definition, err := r.Definition(key)
	if err != nil {
		return "", err
	}
	return marshalJSON(definition.Default())
}

// ValidateJSON 解析并校验指定配置的 ProtoJSON。
func (r *Registry) ValidateJSON(key, value string) error {
	_, err := r.decodeJSON(key, value)
	return err
}

// CanonicalJSON 将配置 JSON 规范化为带默认字段的 ProtoJSON。
func (r *Registry) CanonicalJSON(key, value string) (string, error) {
	message, err := r.decodeJSON(key, value)
	if err != nil {
		return "", err
	}
	return marshalJSON(message)
}

// LoadJSON 从缓存读取并反序列化指定配置。
func (r *Registry) LoadJSON(store cache.Cache, key string, target proto.Message) error {
	if store == nil {
		return errors.New("运行配置缓存未配置")
	}
	if target == nil {
		return errors.New("运行配置目标消息未配置")
	}
	value, err := store.Get(CacheKey(key))
	if err != nil {
		return err
	}
	canonical, err := r.CanonicalJSON(key, value)
	if err != nil {
		return err
	}
	definition, err := r.Definition(key)
	if err != nil {
		return err
	}
	if target.ProtoReflect().Descriptor().FullName() != definition.New().ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("运行配置 %s 的目标消息类型不匹配", key)
	}
	proto.Reset(target)
	options := protojson.UnmarshalOptions{DiscardUnknown: false}
	if err = options.Unmarshal([]byte(canonical), target); err != nil {
		return fmt.Errorf("解析系统配置缓存失败: %w", err)
	}
	return nil
}

// SaveJSON 校验并保存指定配置的规范化 ProtoJSON。
func (r *Registry) SaveJSON(store cache.Cache, key, value string) error {
	if store == nil {
		return errors.New("运行配置缓存未配置")
	}
	canonical, err := r.CanonicalJSON(key, value)
	if err != nil {
		return err
	}
	return store.Set(CacheKey(key), canonical, CacheExpire)
}

// DefaultJSON 使用进程内默认注册表生成默认 ProtoJSON。
func DefaultJSON(key string) (string, error) {
	return defaultRegistry.DefaultJSON(key)
}

// ValidateJSON 使用进程内默认注册表校验配置 ProtoJSON。
func ValidateJSON(key, value string) error {
	return defaultRegistry.ValidateJSON(key, value)
}

// LoadJSON 使用进程内默认注册表从缓存加载配置。
func LoadJSON(store cache.Cache, key string, target proto.Message) error {
	return defaultRegistry.LoadJSON(store, key, target)
}

// SaveJSON 使用进程内默认注册表保存配置 ProtoJSON。
func SaveJSON(store cache.Cache, key, value string) error {
	return defaultRegistry.SaveJSON(store, key, value)
}

// Keys 返回进程内默认注册表的配置键集合。
func Keys() []string {
	return defaultRegistry.Keys()
}

// IsSupportedKey 判断配置键是否由进程内默认注册表管理。
func IsSupportedKey(key string) bool {
	return defaultRegistry.IsSupportedKey(key)
}

// SensitiveFields 返回进程内默认注册表的敏感字段路径。
func SensitiveFields(key string) []string {
	fields, err := defaultRegistry.SensitiveFields(key)
	if err != nil {
		return nil
	}
	return fields
}

// RedactJSON 使用进程内默认注册表脱敏运行配置 JSON。
func RedactJSON(key, value string) (string, error) {
	return defaultRegistry.RedactJSON(key, value)
}

// MergeSensitiveJSON 使用进程内默认注册表恢复敏感配置占位值。
func MergeSensitiveJSON(key, current, incoming string) (string, error) {
	return defaultRegistry.MergeSensitiveJSON(key, current, incoming)
}

// DefaultBaseLogFallbackConfig 返回日志入库回退配置默认值。
func DefaultBaseLogFallbackConfig() BaseLogFallbackConfig {
	return BaseLogFallbackConfig{FilePath: "./logs/base-log-fallback"}
}

func (r *Registry) decodeJSON(key, value string) (proto.Message, error) {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) == "null" {
		return nil, errors.New("系统配置 JSON 不能为空")
	}
	definition, err := r.Definition(key)
	if err != nil {
		return nil, err
	}
	message := definition.Default()
	defaults, err := marshalJSON(message)
	if err != nil {
		return nil, err
	}
	var merged []byte
	merged, err = mergeJSONObjects([]byte(defaults), []byte(value))
	if err != nil {
		return nil, fmt.Errorf("解析系统配置 JSON 失败: %w", err)
	}
	options := protojson.UnmarshalOptions{DiscardUnknown: false}
	if err = options.Unmarshal(merged, message); err != nil {
		return nil, fmt.Errorf("解析系统配置 JSON 失败: %w", err)
	}
	if err = protovalidate.Validate(message); err != nil {
		return nil, err
	}
	return message, nil
}

func marshalJSON(message proto.Message) (string, error) {
	if message == nil {
		return "", errors.New("运行配置 Proto 消息未配置")
	}
	options := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	payload, err := options.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("生成系统配置 JSON 失败: %w", err)
	}
	return string(payload), nil
}

func mergeJSONObjects(defaults, overrides []byte) ([]byte, error) {
	var defaultObject map[string]json.RawMessage
	err := json.Unmarshal(defaults, &defaultObject)
	if err != nil {
		return nil, fmt.Errorf("默认配置不是 JSON 对象: %w", err)
	}
	var overrideObject map[string]json.RawMessage
	err = json.Unmarshal(overrides, &overrideObject)
	if err != nil {
		return nil, err
	}
	if overrideObject == nil {
		return nil, errors.New("系统配置 JSON 必须是对象")
	}
	for key, value := range overrideObject {
		canonicalKey := key
		if _, exists := defaultObject[canonicalKey]; !exists {
			for candidate := range defaultObject {
				if lowerCamelFieldName(candidate) == key {
					canonicalKey = candidate
					delete(defaultObject, candidate)
					break
				}
			}
		}
		defaultValue, exists := defaultObject[canonicalKey]
		if exists {
			var merged json.RawMessage
			var ok bool
			merged, ok, err = mergeNestedJSONObjects(defaultValue, value)
			if err != nil {
				return nil, err
			}
			if ok {
				defaultObject[canonicalKey] = merged
				continue
			}
		}
		defaultObject[canonicalKey] = value
	}
	return json.Marshal(defaultObject)
}

func lowerCamelFieldName(value string) string {
	parts := strings.Split(value, "_")
	if len(parts) == 1 {
		return value
	}
	var builder strings.Builder
	builder.WriteString(parts[0])
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func mergeNestedJSONObjects(defaults, overrides json.RawMessage) (json.RawMessage, bool, error) {
	var defaultObject map[string]json.RawMessage
	var err error
	err = json.Unmarshal(defaults, &defaultObject)
	if err != nil || defaultObject == nil {
		return nil, false, nil
	}
	var overrideObject map[string]json.RawMessage
	err = json.Unmarshal(overrides, &overrideObject)
	if err != nil || overrideObject == nil {
		return nil, false, nil
	}
	defaultJSON, err := json.Marshal(defaultObject)
	if err != nil {
		return nil, false, err
	}
	overrideJSON, err := json.Marshal(overrideObject)
	if err != nil {
		return nil, false, err
	}
	var merged []byte
	merged, err = mergeJSONObjects(defaultJSON, overrideJSON)
	if err != nil {
		return nil, false, err
	}
	return merged, true, nil
}

func getJSONField(object map[string]json.RawMessage, parts []string) (json.RawMessage, bool) {
	if len(parts) == 0 {
		return nil, false
	}
	value, exists := object[parts[0]]
	if !exists || len(parts) == 1 {
		return value, exists
	}
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(value, &nested); err != nil || nested == nil {
		return nil, false
	}
	return getJSONField(nested, parts[1:])
}

func setJSONField(object map[string]json.RawMessage, parts []string, value json.RawMessage) error {
	if len(parts) == 0 {
		return errors.New("系统配置敏感字段路径为空")
	}
	if len(parts) == 1 {
		object[parts[0]] = value
		return nil
	}
	existing, exists := object[parts[0]]
	if !exists {
		nested := make(map[string]json.RawMessage)
		err := setJSONField(nested, parts[1:], value)
		if err != nil {
			return err
		}
		var nestedValue []byte
		nestedValue, err = json.Marshal(nested)
		if err != nil {
			return err
		}
		object[parts[0]] = nestedValue
		return nil
	}
	var nested map[string]json.RawMessage
	err := json.Unmarshal(existing, &nested)
	if err != nil || nested == nil {
		return nil
	}
	err = setJSONField(nested, parts[1:], value)
	if err != nil {
		return err
	}
	var nestedValue []byte
	nestedValue, err = json.Marshal(nested)
	if err != nil {
		return err
	}
	object[parts[0]] = nestedValue
	return nil
}

func isRedactedJSONValue(value json.RawMessage) bool {
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return false
	}
	return text == RedactedValue
}

func builtinDefinitions() []Definition {
	return []Definition{
		{
			Key:            BaseLogFallbackKey,
			New:            func() proto.Message { return new(BaseLogFallbackConfig) },
			Default:        func() proto.Message { value := DefaultBaseLogFallbackConfig(); return &value },
			Owner:          "admin",
			NameKey:        "system.base.runtime_config.base_log_fallback.title",
			DescriptionKey: "system.base.runtime_config.base_log_fallback.description",
		},
	}
}

func mustNewRegistry() *Registry {
	registry, err := NewRegistry()
	if err != nil {
		panic(err)
	}
	return registry
}
