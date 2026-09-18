package model

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// AgenticModel 表示当前项目使用的 Eino Agentic 模型接口。
type AgenticModel = model.AgenticModel

// Option 表示 Eino 模型调用选项。
type Option = model.Option

// ChatClient 表示评论审核与摘要专用聊天模型客户端。
type ChatClient struct {
	model.AgenticModel
	name string
}

// NewChatClient 创建评论审核与摘要专用聊天模型客户端。
func NewChatClient(modelCfg *configv1.AI_Model) *ChatClient {
	client := &ChatClient{}
	// AI 未配置完整时保持空客户端，业务层会通过 Enabled 判断并走降级路径。
	if !aiModelConfigured(modelCfg) {
		return client
	}
	agenticModel, err := newChatModel(context.Background(), modelCfg, func(modelConfig *agenticopenai.ChatConfig) {
		// 评论结构化输出不依赖采样温度，交给服务端使用模型默认值。
		modelConfig.Temperature = nil
	})
	if err != nil {
		panic(fmt.Errorf("创建评论智能体模型失败: %w", err))
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

// Name 返回当前结构化聊天模型名称。
func (c *ChatClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// NewAssistantClient 创建 AI 助手专用模型客户端。
func NewAssistantClient(modelCfg *configv1.AI_Model) *AssistantClient {
	client := &AssistantClient{apiType: resolveAssistantAPIType(modelCfg)}
	// AI 未配置完整时保持空客户端，避免服务启动阶段因为可选能力缺失而失败。
	if !aiModelConfigured(modelCfg) {
		return client
	}
	var agenticModel model.AgenticModel
	var err error
	// Responses 协议能力更强，但部分网关只实现了聊天补全协议，按部署配置选择。
	if client.apiType == configv1.AI_Model_API_TYPE_RESPONSES {
		agenticModel, err = newResponsesModel(context.Background(), modelCfg)
	} else {
		agenticModel, err = newChatModel(context.Background(), modelCfg, nil)
	}
	if err != nil {
		return client
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

// AssistantClient 表示 AI 助手专用模型客户端，按配置选择 Chat Completions 或 Responses 协议。
type AssistantClient struct {
	model.AgenticModel
	name    string
	apiType configv1.AI_Model_APIType
}

// Enabled 判断 AI 助手模型客户端是否可用。
func (c *AssistantClient) Enabled() bool {
	return c != nil && c.AgenticModel != nil
}

// Name 返回当前 AI 助手模型名称。
func (c *AssistantClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// APIType 返回当前模型客户端使用的 API 协议类型。
func (c *AssistantClient) APIType() configv1.AI_Model_APIType {
	if c == nil {
		return configv1.AI_Model_API_TYPE_UNSPECIFIED
	}
	return c.apiType
}

// SupportsResponsesServerTools 判断当前协议是否支持 Responses 服务端工具。
func (c *AssistantClient) SupportsResponsesServerTools() bool {
	return c != nil && c.apiType == configv1.AI_Model_API_TYPE_RESPONSES
}

// resolveAssistantAPIType 解析 AI 助手使用的 API 协议类型，未配置时默认聊天补全。
func resolveAssistantAPIType(modelCfg *configv1.AI_Model) configv1.AI_Model_APIType {
	if modelCfg != nil && modelCfg.GetApiType() == configv1.AI_Model_API_TYPE_RESPONSES {
		return configv1.AI_Model_API_TYPE_RESPONSES
	}
	// 聊天补全协议是所有 OpenAI 兼容网关的公共子集，作为默认值最稳妥。
	return configv1.AI_Model_API_TYPE_CHAT_COMPLETIONS
}

// aiModelConfigured 判断大模型启动配置是否完整。
func aiModelConfigured(modelCfg *configv1.AI_Model) bool {
	// 模型名称是云模型和本地模型共同需要的最小配置。
	if modelCfg == nil || modelCfg.GetModelName() == "" {
		return false
	}
	// 不同模型来源需要校验的启动参数不同，保持在这里集中判断。
	switch modelCfg.GetType() {
	case configv1.AI_Model_CLOUD_MODEL:
		cloud := modelCfg.GetCloud()
		return cloud != nil && cloud.GetApiKey() != ""
	case configv1.AI_Model_LOCAL_MODEL:
		return modelCfg.GetLocal() != nil
	default:
		// 未知模型类型不启用 Agent，避免启动后调用到不明确的模型提供商。
		return false
	}
}

const (
	defaultLocalHost = "localhost"
	defaultLocalPort = 11434
	localBaseScheme  = "http"
)

// newChatModel 根据配置创建 Chat Completions AgenticModel。
func newChatModel(ctx context.Context, cfg *configv1.AI_Model, mutate func(*agenticopenai.ChatConfig)) (model.AgenticModel, error) {
	if cfg == nil {
		return nil, errors.New("ai model config is nil")
	}
	config := &agenticopenai.ChatConfig{Model: cfg.GetModelName()}
	switch cfg.GetType() {
	case configv1.AI_Model_CLOUD_MODEL:
		cloud := cfg.GetCloud()
		if cloud == nil {
			return nil, errors.New("ai cloud config is nil")
		}
		config.APIKey = cloud.GetApiKey()
		config.BaseURL = cloud.GetBaseUrl()
	case configv1.AI_Model_LOCAL_MODEL:
		local := cfg.GetLocal()
		if local == nil {
			return nil, errors.New("ai local config is nil")
		}
		config.APIKey = "ollama"
		config.BaseURL = localBaseURL(local)
	default:
		return nil, fmt.Errorf("unsupported ai model type: %v", cfg.GetType())
	}
	if cfg.GetTimeoutSeconds() > 0 {
		config.Timeout = time.Duration(cfg.GetTimeoutSeconds()) * time.Second
	}
	if cfg.GetTemperature() > 0 {
		value := cfg.GetTemperature()
		config.Temperature = &value
	}
	if cfg.GetMaxTokens() > 0 {
		value := int(cfg.GetMaxTokens())
		config.MaxCompletionTokens = &value
	}
	if mutate != nil {
		mutate(config)
	}
	return agenticopenai.NewChatModel(ctx, config)
}

// newResponsesModel 根据配置创建 Responses AgenticModel。
func newResponsesModel(ctx context.Context, cfg *configv1.AI_Model) (model.AgenticModel, error) {
	if cfg == nil {
		return nil, errors.New("ai model config is nil")
	}
	config := &agenticopenai.ResponsesConfig{Model: cfg.GetModelName()}
	switch cfg.GetType() {
	case configv1.AI_Model_CLOUD_MODEL:
		cloud := cfg.GetCloud()
		if cloud == nil {
			return nil, errors.New("ai cloud config is nil")
		}
		config.APIKey = cloud.GetApiKey()
		config.BaseURL = cloud.GetBaseUrl()
	case configv1.AI_Model_LOCAL_MODEL:
		local := cfg.GetLocal()
		if local == nil {
			return nil, errors.New("ai local config is nil")
		}
		config.APIKey = "ollama"
		config.BaseURL = localBaseURL(local)
	default:
		return nil, fmt.Errorf("unsupported ai model type: %v", cfg.GetType())
	}
	if cfg.GetTimeoutSeconds() > 0 {
		value := time.Duration(cfg.GetTimeoutSeconds()) * time.Second
		config.Timeout = &value
	}
	if cfg.GetMaxRetries() > 0 {
		value := int(cfg.GetMaxRetries())
		config.MaxRetries = &value
	}
	if cfg.GetTemperature() > 0 {
		value := cfg.GetTemperature()
		config.Temperature = &value
	}
	if cfg.GetMaxTokens() > 0 {
		value := int(cfg.GetMaxTokens())
		config.MaxTokens = &value
	}
	return agenticopenai.NewResponsesModel(ctx, config)
}

// localBaseURL 根据 Ollama 配置生成 OpenAI 兼容地址。
func localBaseURL(local *configv1.AI_Model_LocalConfig) string {
	host := local.GetHost()
	if host == "" {
		host = defaultLocalHost
	}
	port := local.GetPort()
	if port == 0 {
		port = defaultLocalPort
	}
	return (&url.URL{
		Scheme: localBaseScheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(int(port))),
		Path:   "/v1",
	}).String()
}
