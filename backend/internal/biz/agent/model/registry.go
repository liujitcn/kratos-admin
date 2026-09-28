package model

import (
	"errors"
	"sync"

	"github.com/liujitcn/kratos-kit/ai/model"
)

// ProviderModel 表示供应商下的一个可用模型。
type ProviderModel struct {
	// ProviderID 是数据库中的供应商编号。
	ProviderID int64
	// ProviderName 是供应商展示名称。
	ProviderName string
	// DisplayName 是模型在聊天前端的展示名称，为空时由前端按模型名映射。
	DisplayName string
	// Config 是当前模型的运行参数。
	Config *model.ModelConfig
}

// ModelOption 表示可以提供给聊天前端的供应商和模型名称。
type ModelOption struct {
	// ProviderID 是数据库中的供应商编号。
	ProviderID int64
	// ProviderName 是供应商展示名称。
	ProviderName string
	// ModelName 是供应商支持的模型名称。
	ModelName string
	// DisplayName 是模型在聊天前端的展示名称，为空时由前端按模型名映射。
	DisplayName string
}

type registeredModel struct {
	option ModelOption
	client *AssistantClient
}

// AssistantClientResolver 定义 AI 助手模型的选择能力。
type AssistantClientResolver interface {
	ResolveAssistantClient(int64, string) (*AssistantClient, error)
	DefaultAssistantClient() *AssistantClient
	HasEnabledAssistantClient() bool
}

// Registry 保存数据库配置对应的可用模型客户端快照。
type Registry struct {
	mu     sync.RWMutex
	models []registeredModel
}

// NewRegistry 创建空的 AI 模型注册表。
func NewRegistry() *Registry {
	return &Registry{models: []registeredModel{}}
}

// Replace 根据已启用的供应商配置重建模型客户端快照。
func (r *Registry) Replace(values []ProviderModel) {
	models := make([]registeredModel, 0, len(values))
	for _, value := range values {
		if value.Config == nil {
			continue
		}
		name := value.ProviderName + " / " + value.Config.ModelName
		client := NewAssistantClientWithName(value.Config, name)
		if !client.Enabled() {
			continue
		}
		models = append(models, registeredModel{
			option: ModelOption{
				ProviderID:   value.ProviderID,
				ProviderName: value.ProviderName,
				ModelName:    value.Config.ModelName,
				DisplayName:  value.DisplayName,
			},
			client: client,
		})
	}
	r.mu.Lock()
	r.models = models
	r.mu.Unlock()
}

// ResolveAssistantClient 根据供应商和模型名称解析模型客户端；未指定时使用首个可用模型。
func (r *Registry) ResolveAssistantClient(providerID int64, modelName string) (*AssistantClient, error) {
	if providerID == 0 && modelName == "" {
		client := r.DefaultAssistantClient()
		if client == nil {
			return nil, errors.New("AI助手没有已启用的模型")
		}
		return client, nil
	}
	if providerID <= 0 || modelName == "" {
		return nil, errors.New("AI供应商和模型选择不完整")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.models {
		if item.option.ProviderID == providerID && item.option.ModelName == modelName {
			return item.client, nil
		}
	}
	return nil, errors.New("AI供应商或模型未启用")
}

// DefaultAssistantClient 返回当前快照中的首个可用模型客户端。
func (r *Registry) DefaultAssistantClient() *AssistantClient {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.models) == 0 {
		return nil
	}
	return r.models[0].client
}

// HasEnabledAssistantClient 判断当前快照中是否存在可用模型。
func (r *Registry) HasEnabledAssistantClient() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.models) > 0
}

// ModelOptions 返回当前快照的供应商和模型名称列表。
func (r *Registry) ModelOptions() []ModelOption {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	options := make([]ModelOption, 0, len(r.models))
	for _, item := range r.models {
		options = append(options, item.option)
	}
	return options
}
