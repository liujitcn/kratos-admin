package automation

import "context"

// Execution 绑定服务端领取的自动化执行及 fencing 版本。
type Execution struct {
	RunID        int64
	LeaseVersion int64
}

type executionKey struct{}

// ContextWithExecution 仅供后台 worker 与原子审批领取后绑定执行来源。
func ContextWithExecution(ctx context.Context, execution Execution) context.Context {
	return context.WithValue(ctx, executionKey{}, execution)
}

// ExecutionFromContext 读取冻结执行来源，普通请求没有此上下文。
func ExecutionFromContext(ctx context.Context) Execution {
	value, _ := ctx.Value(executionKey{}).(Execution)
	return value
}
