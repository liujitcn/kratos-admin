package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
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
func NewChatClient(modelCfg *modelconfig.ModelConfig) *ChatClient {
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
	client.name = modelCfg.ModelName
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
func NewAssistantClient(modelCfg *modelconfig.ModelConfig) *AssistantClient {
	return NewAssistantClientWithName(modelCfg, "")
}

// NewAssistantClientWithName 创建带展示名称的 AI 助手模型客户端。
func NewAssistantClientWithName(modelCfg *modelconfig.ModelConfig, name string) *AssistantClient {
	client := &AssistantClient{apiType: resolveAssistantAPIType(modelCfg)}
	// AI 未配置完整时保持空客户端，避免服务启动阶段因为可选能力缺失而失败。
	if !aiModelConfigured(modelCfg) {
		return client
	}
	var agenticModel model.AgenticModel
	var err error
	// Responses 协议能力更强，但部分网关只实现了聊天补全协议，按部署配置选择。
	if client.apiType == modelconfig.APITypeResponses {
		agenticModel, err = newResponsesModel(context.Background(), modelCfg)
	} else {
		agenticModel, err = newChatModel(context.Background(), modelCfg, nil)
	}
	if err != nil {
		return client
	}
	client.name = name
	if client.name == "" {
		client.name = modelCfg.ModelName
	}
	client.AgenticModel = agenticModel
	return client
}

// AssistantClient 表示 AI 助手专用模型客户端，按配置选择 Chat Completions 或 Responses 协议。
type AssistantClient struct {
	model.AgenticModel
	name    string
	apiType modelconfig.APIType
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
func (c *AssistantClient) APIType() modelconfig.APIType {
	if c == nil {
		return modelconfig.APITypeUnspecified
	}
	return c.apiType
}

// SupportsResponsesServerTools 判断当前协议是否支持 Responses 服务端工具。
func (c *AssistantClient) SupportsResponsesServerTools() bool {
	return c != nil && c.apiType == modelconfig.APITypeResponses
}

// resolveAssistantAPIType 解析 AI 助手使用的 API 协议类型，未配置时默认聊天补全。
func resolveAssistantAPIType(modelCfg *modelconfig.ModelConfig) modelconfig.APIType {
	if modelCfg != nil && modelCfg.APIType == modelconfig.APITypeResponses {
		return modelconfig.APITypeResponses
	}
	// 聊天补全协议是所有 OpenAI 兼容网关的公共子集，作为默认值最稳妥。
	return modelconfig.APITypeChatCompletions
}

// aiModelConfigured 判断模型客户端配置是否完整。
func aiModelConfigured(modelCfg *modelconfig.ModelConfig) bool {
	// 模型名称是云模型和本地模型共同需要的最小配置。
	if modelCfg == nil || modelCfg.ModelName == "" {
		return false
	}
	// 不同模型来源需要校验的启动参数不同，保持在这里集中判断。
	switch modelCfg.Provider {
	case modelconfig.ProviderOpenAICompatible:
		return modelCfg.APIKey != ""
	case modelconfig.ProviderOllama:
		return modelCfg.ResolvedBaseURL() != ""
	default:
		// 未知模型类型不启用 Agent，避免启动后调用到不明确的模型提供商。
		return false
	}
}

// newChatModel 根据配置创建 Chat Completions AgenticModel。
func newChatModel(ctx context.Context, cfg *modelconfig.ModelConfig, mutate func(*agenticopenai.ChatConfig)) (model.AgenticModel, error) {
	if cfg == nil {
		return nil, errors.New("ai model config is nil")
	}
	config := &agenticopenai.ChatConfig{Model: cfg.ModelName}
	switch cfg.Provider {
	case modelconfig.ProviderOpenAICompatible, modelconfig.ProviderOllama:
		config.APIKey = cfg.ResolvedAPIKey()
		config.BaseURL = cfg.ResolvedBaseURL()
	default:
		return nil, fmt.Errorf("unsupported ai model provider: %s", cfg.Provider)
	}
	if cfg.TimeoutSeconds > 0 {
		config.Timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	if cfg.Temperature > 0 {
		value := cfg.Temperature
		config.Temperature = &value
	}
	if cfg.MaxTokens > 0 {
		value := int(cfg.MaxTokens)
		// 当前模型配置使用 max_tokens，而 SDK 配置项只有 max_completion_tokens。
		config.ExtraFields = map[string]any{"max_tokens": value}
	}
	if mutate != nil {
		mutate(config)
	}
	return agenticopenai.NewChatModel(ctx, config)
}

// newResponsesModel 根据配置创建 Responses AgenticModel。
func newResponsesModel(ctx context.Context, cfg *modelconfig.ModelConfig) (model.AgenticModel, error) {
	if cfg == nil {
		return nil, errors.New("ai model config is nil")
	}
	config := &agenticopenai.ResponsesConfig{Model: cfg.ModelName}
	switch cfg.Provider {
	case modelconfig.ProviderOpenAICompatible, modelconfig.ProviderOllama:
		config.APIKey = cfg.ResolvedAPIKey()
		config.BaseURL = cfg.ResolvedBaseURL()
	default:
		return nil, fmt.Errorf("unsupported ai model provider: %s", cfg.Provider)
	}
	if cfg.TimeoutSeconds > 0 {
		value := time.Duration(cfg.TimeoutSeconds) * time.Second
		config.Timeout = &value
	}
	if cfg.MaxRetries > 0 {
		value := int(cfg.MaxRetries)
		config.MaxRetries = &value
	}
	if cfg.Temperature > 0 {
		value := cfg.Temperature
		config.Temperature = &value
	}
	if cfg.MaxTokens > 0 {
		value := int(cfg.MaxTokens)
		config.MaxTokens = &value
	}
	return agenticopenai.NewResponsesModel(ctx, config)
}

// TestConnection 使用固定短提示验证模型接口是否可用。
func TestConnection(ctx context.Context, cfg *modelconfig.ModelConfig) error {
	if cfg == nil {
		return errors.New("ai model config is nil")
	}
	var client model.AgenticModel
	var err error
	if cfg.APIType == modelconfig.APITypeResponses {
		client, err = newResponsesModel(ctx, cfg)
	} else {
		client, err = newChatModel(ctx, cfg, nil)
	}
	if err != nil {
		return err
	}
	_, err = client.Generate(ctx, []*schema.AgenticMessage{schema.UserAgenticMessage("Reply with OK.")})
	return err
}
