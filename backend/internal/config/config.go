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

const configSecretKeyName = "kratos-kit:config"
const redactStorageKeyName = "kratos-admin:redact/storage"

// ParseAIModel 提取本地 AI 模型配置。
func ParseAIModel(cfg *configv1.Bootstrap) (*configv1.AI_Model, error) {
	if cfg == nil || cfg.GetAi() == nil {
		return nil, nil
	}
	return cfg.GetAi().GetModel(), nil
}

// NewOAuthManager 创建由数据库动态维护的 OAuth 管理器。
func NewOAuthManager() (*oauth.Manager, error) {
	return oauth.NewManager(nil)
}

// NewRuntimeFieldCipher 创建存储脱敏运行时字段加密器。
func NewRuntimeFieldCipher() (*FieldCipher, error) {
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return nil, errors.New("配置加密密钥为空且运行时密钥未初始化")
	}
	derived, err := keyValue.Derive(context.Background(), configSecretKeyName)
	if err != nil {
		return nil, err
	}
	return NewFieldCipher(derived)
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
