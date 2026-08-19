package config

import (
	"errors"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/oauth"
)

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
