package agent

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

type terminalContextKey struct{}

type assistantClientContextKey struct{}

// ContextWithTerminal 将工具调用所属终端写入上下文。
func ContextWithTerminal(ctx context.Context, terminal string) context.Context {
	return context.WithValue(ctx, terminalContextKey{}, terminal)
}

// TerminalFromContext 读取工具调用所属终端，未设置时返回空字符串。
func TerminalFromContext(ctx context.Context) string {
	terminal, _ := ctx.Value(terminalContextKey{}).(string)
	return terminal
}

// ContextWithAssistantClient 将本轮对话已选定的模型客户端传给业务工具。
func ContextWithAssistantClient(ctx context.Context, client *AssistantClient) context.Context {
	return context.WithValue(ctx, assistantClientContextKey{}, client)
}

// AssistantClientFromContext 读取本轮对话的模型客户端，未设置时返回 nil。
func AssistantClientFromContext(ctx context.Context) *AssistantClient {
	client, _ := ctx.Value(assistantClientContextKey{}).(*AssistantClient)
	return client
}

// ToolPool 是当前身份和会话已解析的请求工具池；空 Tools 表示明确禁用工具。
type ToolPool struct {
	// Tools 是本次请求绑定的执行器。
	Tools []Tool
	// Checker 在模型调用与实际执行前核实当前权限。
	Checker ToolAccessChecker
	// Revision 绑定插件、策略、连接及契约的修订快照。
	Revision string
}

type toolPoolContextKey struct{}

// ContextWithToolPool 绑定请求工具池并复制工具切片，防止调用方后续修改集合。
func ContextWithToolPool(ctx context.Context, pool ToolPool) context.Context {
	pool.Tools = append([]Tool(nil), pool.Tools...)
	return context.WithValue(ctx, toolPoolContextKey{}, pool)
}

// requestToolPool 读取显式请求池，未绑定时沿用静态登记和权限配置。
func (r *Runtime) requestToolPool(ctx context.Context, terminal string) ToolPool {
	pool, ok := ctx.Value(toolPoolContextKey{}).(ToolPool)
	if ok {
		return pool
	}
	return ToolPool{Tools: r.RegisteredTools(terminal), Checker: r.toolGate}
}

// ToolExecutionObserver 在模型与工具真实调用边界核实后台租约和停止状态。
type ToolExecutionObserver = tool.ExecutionObserver

// ContextWithToolExecutionObserver 绑定实际执行观察者，审批等待不触发。
func ContextWithToolExecutionObserver(ctx context.Context, observer ToolExecutionObserver) context.Context {
	return tool.ContextWithExecutionObserver(ctx, observer)
}
