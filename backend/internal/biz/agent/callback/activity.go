package callback

import "context"

type activityEmitterContextKey struct{}

// Activity 表示 AI 助手运行中的一条可展示活动。
type Activity struct {
	// ID 是活动在当前运行中的稳定编号。
	ID string
	// Phase 是活动阶段，例如 reasoning、tool 或 answering。
	Phase string
	// Status 是活动状态，例如 running、completed 或 failed。
	Status string
	// Summary 是面向用户的摘要，不包含原始推理内容。
	Summary string
	// ToolName 是工具的内部名称。
	ToolName string
	// ToolTitle 是工具的展示名称。
	ToolTitle string
}

// ActivityEmitter 接收 AI 助手运行中的可展示活动。
type ActivityEmitter func(Activity)

// WithActivityEmitter 将活动发送器写入当前运行上下文。
func WithActivityEmitter(ctx context.Context, emitter ActivityEmitter) context.Context {
	if emitter == nil {
		return ctx
	}
	return context.WithValue(ctx, activityEmitterContextKey{}, emitter)
}

// EmitActivity 向当前运行的活动发送器发送一条活动。
func EmitActivity(ctx context.Context, activity Activity) {
	if ctx == nil {
		return
	}
	emitter, _ := ctx.Value(activityEmitterContextKey{}).(ActivityEmitter)
	if emitter != nil {
		emitter(activity)
	}
}
