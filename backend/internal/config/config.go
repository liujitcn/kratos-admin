package config

import (
	"context"
	"encoding/base64"
	"errors"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/oauth"
	"github.com/liujitcn/kratos-kit/redact"
	"github.com/liujitcn/kratos-kit/sdk"
)

const redactStorageKeyName = "kratos-admin:redact/storage"

// ParseAIModel 提取本地 AI 模型配置。
func ParseAIModel(cfg *configv1.Bootstrap) (*configv1.AI_Model, error) {
	if cfg == nil || cfg.GetAi() == nil {
		return nil, errors.New("ai相关配置为空")
	}
	return cfg.GetAi().GetModel(), nil
}

// ParseOAuthManager 根据 Admin 启动配置创建 OAuth 管理器。
func ParseOAuthManager(cfg *configv1.Bootstrap) (*oauth.Manager, error) {
	return oauth.NewManager(cfg.GetOauth())
}

// ParseMfaConfig 提取多因素认证配置。
func ParseMfaConfig(cfg *configv1.Bootstrap) *configv1.Mfa {
	if cfg == nil {
		return nil
	}
	return cfg.GetMfa()
}

// NewRedactStorageProtector 创建敏感字段旁表加密保护器。
func NewRedactStorageProtector() (*redact.StorageProtector, error) {
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return nil, errors.New("脱敏存储密钥为空且运行时密钥未初始化")
	}
	derived, err := keyValue.Derive(context.Background(), redactStorageKeyName)
	if err != nil {
		return nil, err
	}
	return redact.NewStorageProtector(base64.RawStdEncoding.EncodeToString(derived))
}
