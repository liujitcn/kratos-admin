package agent

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// Tool 是 Eino 可执行工具接口。
type Tool = tool.InvokableTool

// AssistantClient 是 AI 助手模型客户端，按配置选择 Chat Completions 或 Responses 协议。
type AssistantClient = model.AssistantClient

// MessageLocalizer 按请求语言渲染 Agent 面向用户的消息。
type MessageLocalizer func(ctx context.Context, key string, args map[string]any, fallback string) string

// RuntimeConfig 是公开 Runtime 的初始化配置。
type RuntimeConfig struct {
	// Client 是 AI 助手模型客户端。
	Client *AssistantClient
	// Checker 是可选的工具权限检查器；为 nil 时全部注册工具默认启用。
	Checker ToolAccessChecker
	// AdminTools 是管理端工具集合。
	AdminTools []Tool
	// AppTools 是应用端工具集合。
	AppTools []Tool
	// LocalizeMessage 是可选消息本地化器；未提供时使用消息的英文兜底文案。
	LocalizeMessage MessageLocalizer
}

// NewRuntime 创建可被外部模块复用的 AI Runtime。
func NewRuntime(config RuntimeConfig) *Runtime {
	return newRuntime(config.Client, config.Checker, config.AdminTools, config.AppTools, config.LocalizeMessage)
}

// NewRuntimeWithTools 创建只使用管理端工具集合的 Runtime。
func NewRuntimeWithTools(client *AssistantClient, tools ...Tool) *Runtime {
	return NewRuntime(RuntimeConfig{Client: client, AdminTools: tools})
}

// NewAssistantClient 根据 Backend AI 模型配置创建 AI 助手模型客户端。
func NewAssistantClient(modelConfig *configv1.AI_Model) *AssistantClient {
	return model.NewAssistantClient(modelConfig)
}

// InferTool 根据输入结构自动生成 Eino 工具 schema 和执行器。
func InferTool[T, D any](name string, description string, fn utils.InvokeFunc[T, D], options ...utils.Option) (Tool, error) {
	return utils.InferTool(name, description, fn, options...)
}
