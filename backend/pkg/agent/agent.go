// Package agent 提供可被其他 Backend 模块复用的 AI Agent 公开 API。
//
// 业务模块可以直接使用 Eino 工具、模型客户端和 Runtime；不需要依赖
// internal/biz 下的实现路径。工具既可以在创建 Runtime 时传入，也可以在
// 启动后通过 Runtime.RegisterTool 动态追加。
package agent

import (
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// Tool 是 Eino 可执行工具接口。
type Tool = tool.InvokableTool

// ResponsesClient 是 Responses API 模型客户端。
type ResponsesClient = model.ResponsesClient

// RuntimeConfig 是公开 Runtime 的初始化配置。
type RuntimeConfig struct {
	// Client 是 Responses API 模型客户端。
	Client *ResponsesClient
	// Checker 是可选的工具权限检查器；为 nil 时全部注册工具默认启用。
	Checker ToolAccessChecker
	// AdminTools 是管理端工具集合。
	AdminTools []Tool
	// AppTools 是应用端工具集合。
	AppTools []Tool
}

// NewRuntime 创建可被外部模块复用的 AI Runtime。
func NewRuntime(config RuntimeConfig) *Runtime {
	return newRuntime(config.Client, config.Checker, config.AdminTools, config.AppTools)
}

// NewRuntimeWithTools 创建只使用管理端工具集合的 Runtime。
func NewRuntimeWithTools(client *ResponsesClient, tools ...Tool) *Runtime {
	return NewRuntime(RuntimeConfig{Client: client, AdminTools: tools})
}

// NewResponsesClient 根据 Backend AI 模型配置创建 Responses 客户端。
func NewResponsesClient(modelConfig *configv1.AI_Model) *ResponsesClient {
	return model.NewResponsesClient(modelConfig)
}

// InferTool 根据输入结构自动生成 Eino 工具 schema 和执行器。
func InferTool[T, D any](name string, description string, fn utils.InvokeFunc[T, D], options ...utils.Option) (Tool, error) {
	return utils.InferTool(name, description, fn, options...)
}
