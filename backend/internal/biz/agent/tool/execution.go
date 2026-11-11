package tool

import "context"

// ExecutionObserver 在模型与实际工具边界核实后台状态；等待审批不标记工具已发出。
type ExecutionObserver struct {
	// BeforeModel 在模型续跑前核实后台执行是否仍有效。
	BeforeModel func(context.Context) error
	Before      func(context.Context, string) error
	After       func(context.Context, string, error) error
}

type executionObserverKey struct{}

// ContextWithExecutionObserver 绑定后台执行租约观察者，不保存到 checkpoint。
func ContextWithExecutionObserver(ctx context.Context, observer ExecutionObserver) context.Context {
	return context.WithValue(ctx, executionObserverKey{}, observer)
}

// CheckModelExecution 在每次模型调用前核实后台租约和停止状态。
func CheckModelExecution(ctx context.Context) error {
	observer, ok := ctx.Value(executionObserverKey{}).(ExecutionObserver)
	if ok && observer.BeforeModel != nil {
		return observer.BeforeModel(ctx)
	}
	return nil
}

// beforeExecution 只在实际发出工具前执行，等待审批不能标记为已发出。
func beforeExecution(ctx context.Context, name string) error {
	observer, ok := ctx.Value(executionObserverKey{}).(ExecutionObserver)
	if ok && observer.Before != nil {
		return observer.Before(ctx, name)
	}
	return nil
}

// afterExecution 持久化明确工具结果，未知结果由恢复流程阻止重放。
func afterExecution(ctx context.Context, name string, cause error) error {
	observer, ok := ctx.Value(executionObserverKey{}).(ExecutionObserver)
	if ok && observer.After != nil {
		return observer.After(ctx, name, cause)
	}
	return nil
}
