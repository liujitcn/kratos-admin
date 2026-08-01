package module

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

const (
	// AITerminalApp 表示应用端 AI 终端值。
	AITerminalApp int32 = 1
	// AITerminalAdmin 表示管理端 AI 终端值。
	AITerminalAdmin int32 = 2
)

// AIRuntime 表示可供业务固定流程使用的 AI 运行时能力。
type AIRuntime interface {
	// InvokeTool 按工具名调用当前终端已启用的 Agent 工具。
	InvokeTool(context.Context, string, string, string) (*AIToolInvokeResult, error)
	// Model 返回当前使用的模型名称。
	Model() string
}

// AIResponse 表示 AI 助手单轮回复结果。
type AIResponse struct {
	// Content 回复正文，面向前端展示。
	Content string `json:"content"`
	// Token 本次调用 token 消耗。
	Token AITokenUsage `json:"token"`
	// Tools 本轮回复实际使用的工具列表。
	Tools []AIToolUsage `json:"tools"`
	// Source 回复来源，例如 llm 或 fallback。
	Source string `json:"source"`
	// Model 使用的模型名称，便于前端展示和排障。
	Model string `json:"model"`
	// Fallback 标记本次回复是否由本地兜底逻辑生成。
	Fallback bool `json:"fallback"`
	// FallbackReason 记录触发降级的底层错误信息，仅用于排障和后台展示。
	FallbackReason string `json:"fallbackReason"`
	// Flow 记录本次回复所属的模块固定流程。
	Flow string `json:"flow"`
	// Step 记录本次回复所属的流程步骤。
	Step string `json:"step"`
	// BlocksJSON 记录模块定义的结构化内容 JSON。
	BlocksJSON string `json:"blocksJson"`
}

// AITokenUsage 表示 AI 助手单轮真实 token 使用量。
type AITokenUsage struct {
	// Input 输入 token 数。
	Input int32 `json:"input"`
	// Output 输出 token 数。
	Output int32 `json:"output"`
	// Cache 命中缓存 token 数。
	Cache int32 `json:"cache"`
	// Total 总 token 数。
	Total int32 `json:"total"`
}

// AIToolUsage 表示 AI 助手单轮回复涉及的工具。
type AIToolUsage struct {
	// Type 工具类型，例如 function 或 server。
	Type string `json:"type"`
	// Name 工具名称，用于排障和唯一识别。
	Name string `json:"name"`
	// Title 工具展示名称，优先使用生成工具描述。
	Title string `json:"title"`
	// Status 工具状态，例如 success、error。
	Status string `json:"status"`
	// Input 工具原始入参 JSON，用于后台展开排障。
	Input string `json:"input"`
	// Output 工具原始出参 JSON，用于后台展开排障。
	Output string `json:"output"`
}

// AIToolInvokeResult 表示直接调用 Agent 工具后的结果。
type AIToolInvokeResult struct {
	// Output 工具原始输出 JSON。
	Output string
	// Usage 本次工具调用的后台展示记录。
	Usage AIToolUsage
}

// AIFixedFlowProvider 提供模块私有的固定流程、入口校验和快捷入口。
type AIFixedFlowProvider interface {
	// FlowNames 返回该模块声明的稳定流程标识，用于在启动阶段检测冲突。
	FlowNames() []string
	// GenerateFixedFlowReply 尝试生成固定流程回复；handled 为 false 时继续由通用聊天处理。
	GenerateFixedFlowReply(context.Context, AIRuntime, int32, string, *basev1.AiAction) (*AIResponse, bool, error)
	// IsFixedFlowEntryAction 判断动作是否为该模块流程的入口。
	IsFixedFlowEntryAction(int32, string, string) bool
	// FixedFlowShortcuts 返回当前终端可展示的模块快捷入口。
	FixedFlowShortcuts(int32, map[string]bool) []*basev1.AiShortcut
}

// AIFixedFlowContributor 表示可向 Admin AI 助手贡献固定流程的扩展模块。
type AIFixedFlowContributor interface {
	// AIFixedFlowProviders 返回模块提供的固定流程。
	AIFixedFlowProviders() []AIFixedFlowProvider
}

// NormalizeAITerminalString 将 AI 终端枚举转换为稳定字符串。
func NormalizeAITerminalString(terminal basev1.Terminal) string {
	if terminal == basev1.Terminal_TERMINAL_APP {
		return "app"
	}
	return "admin"
}
