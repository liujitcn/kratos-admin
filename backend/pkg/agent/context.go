package agent

import "context"

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
