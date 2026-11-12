package dto

import basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

// AiStreamEvent 表示聊天专用 SSE 事件名称。
type AiStreamEvent string

const (
	// AiStreamEventDelta 表示助手回复增量。
	AiStreamEventDelta AiStreamEvent = "delta"
	// AiStreamEventFinish 表示助手回复完成。
	AiStreamEventFinish AiStreamEvent = "finish"
	// AiStreamEventApprovalRequired 表示工具等待当前用户确认。
	AiStreamEventApprovalRequired AiStreamEvent = "approval_required"
	// AiStreamEventError 表示助手回复异常。
	AiStreamEventError AiStreamEvent = "error"
	// AiStreamEventActivity 表示助手运行活动更新。
	AiStreamEventActivity AiStreamEvent = "activity"
)

// AiStreamActivity 表示一条面向用户的助手运行活动。
type AiStreamActivity struct {
	// ID 是活动在当前运行中的稳定编号。
	ID string `json:"id"`
	// Phase 是活动阶段，例如 reasoning、tool 或 answering。
	Phase string `json:"phase"`
	// Status 是活动状态，例如 running、completed、failed 或 waiting_approval。
	Status string `json:"status"`
	// Summary 是面向用户的摘要。
	Summary string `json:"summary,omitempty"`
	// ToolName 是工具内部名称。
	ToolName string `json:"tool_name,omitempty"`
	// ToolTitle 是工具展示名称。
	ToolTitle string `json:"tool_title,omitempty"`
}

// AiStreamPayload 表示聊天专用 SSE 事件负载。
type AiStreamPayload struct {
	SessionID string `json:"session_id"`
	MessageID string `json:"message_id"`
	Delta     string `json:"delta,omitempty"`
	// Activity 是助手运行中的活动更新。
	Activity *AiStreamActivity   `json:"activity,omitempty"`
	Messages []*basev1.AiMessage `json:"messages,omitempty"`
	Session  *basev1.AiSession   `json:"session,omitempty"`
}

// AiStreamEmitter 定义聊天专用 SSE 事件发送能力。
type AiStreamEmitter interface {
	EmitAiStream(event AiStreamEvent, payload AiStreamPayload) error
}
