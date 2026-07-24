package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	componentsModel "github.com/cloudwego/eino/components/model"
	aiEino "github.com/liujitcn/kratos-kit/ai/eino"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// AgenticModel 表示当前项目使用的 Eino Agentic 模型接口。
type AgenticModel = componentsModel.AgenticModel

// Option 表示 Eino 模型调用选项。
type Option = componentsModel.Option

// ChatClient 表示一次性结构化任务聊天模型客户端。
type ChatClient struct {
	componentsModel.AgenticModel
	name string
}

// NewChatClient 创建一次性结构化任务聊天模型客户端。
func NewChatClient(modelCfg *bootstrapConfigv1.AI_Model) *ChatClient {
	client := &ChatClient{}
	// AI 未配置完整时保持空客户端，业务层会通过 Enabled 判断并走降级路径。
	if !aiModelConfigured(modelCfg) {
		return client
	}
	agenticModel, err := aiEino.NewChatModel(
		context.Background(),
		modelCfg,
		aiEino.WithChatConfigMutator(func(modelConfig *agenticopenai.ChatConfig) {
			// 结构化输出不依赖采样温度，交给服务端使用模型默认值。
			modelConfig.Temperature = nil
		}),
	)
	if err != nil {
		panic(fmt.Errorf("创建结构化任务模型失败: %w", err))
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

// NewResponsesClient 创建 AI 助手专用 Responses 模型客户端。
func NewResponsesClient(modelCfg *bootstrapConfigv1.AI_Model) *ResponsesClient {
	client := &ResponsesClient{}
	// AI 未配置完整时保持空客户端，避免服务启动阶段因为可选能力缺失而失败。
	if !aiModelConfigured(modelCfg) {
		return client
	}
	agenticModel, err := aiEino.NewResponsesModel(context.Background(), modelCfg)
	if err != nil {
		panic(fmt.Errorf("创建 AI 助手 Responses 模型失败: %w", err))
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

// ResponsesClient 表示 AI 助手专用 Responses 模型客户端。
type ResponsesClient struct {
	componentsModel.AgenticModel
	name string
}

// Name 返回当前聊天模型名称。
func (c *ChatClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Enabled 判断 Responses 模型客户端是否可用。
func (c *ResponsesClient) Enabled() bool {
	return c != nil && c.AgenticModel != nil
}

// Name 返回当前 Responses 模型名称。
func (c *ResponsesClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// aiModelConfigured 判断大模型启动配置是否完整。
func aiModelConfigured(modelCfg *bootstrapConfigv1.AI_Model) bool {
	// 模型名称是云模型和本地模型共同需要的最小配置。
	if modelCfg == nil || modelCfg.GetModelName() == "" {
		return false
	}
	// 不同模型来源需要校验的启动参数不同，保持在这里集中判断。
	switch modelCfg.GetType() {
	case bootstrapConfigv1.AI_Model_CLOUD_MODEL:
		cloud := modelCfg.GetCloud()
		return cloud != nil && cloud.GetApiKey() != ""
	case bootstrapConfigv1.AI_Model_LOCAL_MODEL:
		return modelCfg.GetLocal() != nil
	default:
		// 未知模型类型不启用 Agent，避免启动后调用到不明确的模型提供商。
		return false
	}
}
