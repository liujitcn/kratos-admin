package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/notify"
	"github.com/liujitcn/kratos-kit/notify/dingtalk"
	"github.com/liujitcn/kratos-kit/notify/email"
	"github.com/liujitcn/kratos-kit/notify/feishu"
	"github.com/liujitcn/kratos-kit/notify/sms/aliyun"
	"github.com/liujitcn/kratos-kit/notify/sms/huawei"
	"github.com/liujitcn/kratos-kit/notify/sms/tencent"
	"github.com/liujitcn/kratos-kit/notify/webhook"
	"github.com/liujitcn/kratos-kit/notify/wechat"
	"github.com/liujitcn/kratos-kit/notify/wechatwork"
)

// ProviderManager 管理从数据库加载的消息 Provider 发送器。
type ProviderManager struct {
	cache cache.Cache
}

// NewProviderManager 创建消息 Provider 运行时管理器。
func NewProviderManager(baseCase *biz.BaseCase) *ProviderManager {
	return &ProviderManager{cache: baseCase.Cache}
}

// Send 按数据库 Provider ID 发送一条外部通知。
func (m *ProviderManager) Send(ctx context.Context, provider *models.BaseMessageProvider, message notify.Message) (*notify.Receipt, error) {
	sender, err := newProviderSender(provider, m.cache)
	if err != nil {
		return nil, fmt.Errorf("构造消息 Provider %d(%s) 失败: %w", provider.ID, provider.Provider, err)
	}
	manager, err := notify.NewManager()
	if err != nil {
		return nil, err
	}
	if err = manager.Register(sender); err != nil {
		return nil, err
	}
	return manager.Send(ctx, strconv.FormatInt(provider.ID, 10), message)
}

// ValidateProviderConfig 校验消息 Provider 配置是否可以创建发送器。
func ValidateProviderConfig(provider *models.BaseMessageProvider, sharedCache cache.Cache) error {
	_, err := newProviderSender(provider, sharedCache)
	return err
}

// newProviderSender 将数据库 Provider 配置转换为 notify Sender。
func newProviderSender(provider *models.BaseMessageProvider, sharedCache cache.Cache) (notify.Sender, error) {
	config, err := providerConfig(provider.Config)
	if err != nil {
		return nil, err
	}
	name := strconv.FormatInt(provider.ID, 10)
	switch strings.ToLower(provider.Provider) {
	case "email":
		return email.New(email.Config{
			Name: name, Host: stringValue(config, "host"), Port: intValue(config, "port"),
			Username: provider.ClientID, Password: provider.ClientSecret, From: stringValue(config, "from"),
			TLSMode: email.TLSMode(stringValue(config, "tls_mode")), Timeout: timeoutValue(config),
		})
	case "webhook":
		return webhook.New(webhook.Config{
			Name: name, URL: firstString(provider.ClientID, stringValue(config, "url")), Secret: provider.ClientSecret,
			SignatureHeader: stringValue(config, "signature_header"), Headers: stringMapValue(config, "headers"),
			Timeout: timeoutValue(config), AllowHTTP: boolValue(config, "allow_http"), AllowPrivateNetwork: boolValue(config, "allow_private_network"),
		})
	case "dingtalk":
		if stringValue(config, "implementation") == "group" {
			return dingtalk.NewGroupSender(dingtalk.GroupConfig{Name: name, URL: provider.ClientID, Secret: provider.ClientSecret, AllowHTTP: boolValue(config, "allow_http"), AllowPrivateNetwork: boolValue(config, "allow_private_network"), Timeout: timeoutValue(config)})
		}
		return dingtalk.NewAppSender(dingtalk.Config{Name: name, AppKey: provider.ClientID, AppSecret: provider.ClientSecret, AgentID: int64Value(config, "agent_id"), BaseURL: stringValue(config, "base_url"), AllowHTTP: boolValue(config, "allow_http"), Timeout: timeoutValue(config), Cache: sharedCache})
	case "feishu":
		if stringValue(config, "implementation") == "group" {
			return feishu.NewGroupSender(feishu.GroupConfig{Name: name, URL: provider.ClientID, Secret: provider.ClientSecret, AllowHTTP: boolValue(config, "allow_http"), AllowPrivateNetwork: boolValue(config, "allow_private_network"), Timeout: timeoutValue(config)})
		}
		return feishu.NewAppSender(feishu.Config{Name: name, AppID: provider.ClientID, AppSecret: provider.ClientSecret, ReceiveIDType: stringValue(config, "receive_id_type"), BaseURL: stringValue(config, "base_url"), AllowHTTP: boolValue(config, "allow_http"), Timeout: timeoutValue(config), Cache: sharedCache})
	case "wechat":
		return wechat.NewOfficialSender(wechat.Config{Name: name, AppID: provider.ClientID, AppSecret: provider.ClientSecret, BaseURL: stringValue(config, "base_url"), AllowHTTP: boolValue(config, "allow_http"), Timeout: timeoutValue(config), Cache: sharedCache})
	case "wechatwork":
		if stringValue(config, "implementation") == "group" {
			return wechatwork.NewGroupSender(wechatwork.GroupConfig{Name: name, URL: provider.ClientID, AllowHTTP: boolValue(config, "allow_http"), AllowPrivateNetwork: boolValue(config, "allow_private_network"), Timeout: timeoutValue(config)})
		}
		return wechatwork.NewAppSender(wechatwork.AppConfig{Name: name, CorpID: provider.ClientID, CorpSecret: provider.ClientSecret, AgentID: intValue(config, "agent_id"), BaseURL: stringValue(config, "base_url"), AllowHTTP: boolValue(config, "allow_http"), Timeout: timeoutValue(config), Cache: sharedCache})
	case "sms":
		switch strings.ToLower(stringValue(config, "implementation")) {
		case "tencent":
			return tencent.New(tencent.Config{Name: name, SecretID: provider.ClientID, SecretKey: provider.ClientSecret, SDKAppID: stringValue(config, "sdk_app_id"), SignName: stringValue(config, "sign_name"), Region: stringValue(config, "region"), Endpoint: stringValue(config, "endpoint")})
		case "huawei":
			return huawei.New(huawei.Config{Name: name, AppKey: provider.ClientID, AppSecret: provider.ClientSecret, Region: stringValue(config, "region"), Endpoint: stringValue(config, "endpoint"), From: stringValue(config, "from"), Signature: stringValue(config, "signature"), Timeout: timeoutValue(config)})
		default:
			return aliyun.New(aliyun.Config{Name: name, AccessKeyID: provider.ClientID, AccessKeySecret: provider.ClientSecret, SignName: stringValue(config, "sign_name"), Region: stringValue(config, "region"), Endpoint: stringValue(config, "endpoint")})
		}
	default:
		return nil, fmt.Errorf("不支持的消息 Provider: %s", provider.Provider)
	}
}

// providerConfig 解析 Provider 扩展配置对象。
func providerConfig(raw string) (map[string]any, error) {
	values := make(map[string]any)
	if raw == "" {
		return values, nil
	}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("Provider 扩展配置必须是 JSON 对象: %w", err)
	}
	return values, nil
}

// stringValue 读取字符串扩展配置。
func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

// firstString 返回第一个非空字符串。
func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// intValue 读取整数扩展配置。
func intValue(values map[string]any, key string) int {
	return int(int64Value(values, key))
}

// int64Value 读取 int64 扩展配置。
func int64Value(values map[string]any, key string) int64 {
	switch value := values[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

// boolValue 读取布尔扩展配置。
func boolValue(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

// stringMapValue 读取字符串 Map 扩展配置。
func stringMapValue(values map[string]any, key string) map[string]string {
	result := make(map[string]string)
	value, _ := values[key].(map[string]any)
	for itemKey, itemValue := range value {
		if stringValue, ok := itemValue.(string); ok {
			result[itemKey] = stringValue
		}
	}
	return result
}

// timeoutValue 读取秒数或 duration 字符串扩展配置。
func timeoutValue(values map[string]any) time.Duration {
	if seconds := int64Value(values, "timeout_seconds"); seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if raw := stringValue(values, "timeout"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err == nil {
			return value
		}
	}
	return 0
}
