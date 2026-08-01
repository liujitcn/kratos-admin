package kratosadmin

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/event"
	"google.golang.org/grpc"
)

// RuntimeReadyContributor 表示需要接收 Backend 进程内 gRPC 连接的扩展模块。
type RuntimeReadyContributor interface {
	// RuntimeReady 在 Backend 运行时完成装配后绑定进程内 gRPC 连接。
	RuntimeReady(grpc.ClientConnInterface) error
}

// UserSubscriber 接收基础用户数据变更通知。
type UserSubscriber = event.UserSubscriber

// UserSubscriberContributor 表示可订阅基础用户变更的扩展模块。
type UserSubscriberContributor interface {
	// UserSubscribers 返回模块提供的用户变更订阅者。
	UserSubscribers() []UserSubscriber
}

// AIRuntime 表示 Backend AI 助手运行时。
type AIRuntime = ai.Runtime

// AIResponse 表示 AI 助手单轮回复结果。
type AIResponse = ai.Response

// AIToolUsage 表示 AI 助手单轮回复涉及的工具。
type AIToolUsage = ai.ToolUsage

// AIToolInvokeResult 表示一次 AI 工具调用结果。
type AIToolInvokeResult = ai.ToolInvokeResult

// AIFixedFlowProvider 提供模块私有的固定流程、入口校验和快捷入口。
type AIFixedFlowProvider = ai.FixedFlowProvider

// AIFixedFlowContributor 表示可向 Backend AI 助手贡献固定流程的扩展模块。
type AIFixedFlowContributor interface {
	// AIFixedFlowProviders 返回模块提供的固定流程。
	AIFixedFlowProviders() []AIFixedFlowProvider
}

const (
	// AITerminalApp 表示应用端 AI 终端值。
	AITerminalApp = ai.TerminalApp
	// AITerminalAdmin 表示管理端 AI 终端值。
	AITerminalAdmin = ai.TerminalAdmin
)

// NormalizeAITerminalString 将 AI 终端编号转换为稳定字符串。
func NormalizeAITerminalString(terminal int32) string {
	return ai.NormalizeTerminalString(terminal)
}
