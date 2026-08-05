package dto

import basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

// AiStreamEvent 表示聊天专用 SSE 事件名称。
type AiStreamEvent string

const (
	// AiStreamEventDelta 表示助手回复增量。
	AiStreamEventDelta AiStreamEvent = "delta"
	// AiStreamEventFinish 表示助手回复完成。
	AiStreamEventFinish AiStreamEvent = "finish"
	// AiStreamEventError 表示助手回复异常。
	AiStreamEventError AiStreamEvent = "error"
)

// AiStreamPayload 表示聊天专用 SSE 事件负载。
type AiStreamPayload struct {
	SessionID string              `json:"session_id"`
	MessageID string              `json:"message_id"`
	Delta     string              `json:"delta,omitempty"`
	Messages  []*basev1.AiMessage `json:"messages,omitempty"`
	Session   *basev1.AiSession   `json:"session,omitempty"`
}

// AiStreamEmitter 定义聊天专用 SSE 事件发送能力。
type AiStreamEmitter interface {
	EmitAiStream(event AiStreamEvent, payload AiStreamPayload) error
}
