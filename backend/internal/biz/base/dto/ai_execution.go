package dto

import (
	"context"
	"sync"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// AiExecution 保存当前请求的执行关联，互斥锁保证同一运行事件与SSE发送顺序一致。
type AiExecution struct {
	// Mutex 串行化本轮回调与终态写入。
	Mutex sync.Mutex
	// RunID 关联单次执行，不复用审批编号。
	RunID string
	// Session 保存已验证归属的会话。
	Session *models.AiSession
	// Message 保存当前消息快照。
	Message *models.AiMessage
	// Generation 冻结本次生成版本，拒绝旧请求写入。
	Generation time.Time
	// LastEventID 是最近一次已提交事件的编号。
	LastEventID string
	// TerminalID 是与消息同事务落库的终态编号。
	TerminalID string
	// Cancel 在持久化失败时阻止后续外部调用。
	Cancel context.CancelCauseFunc
}

// AiStreamEmitterFunc 适配现有SSE发送函数，不改变HTTP取消语义。
type AiStreamEmitterFunc func(AiStreamEvent, AiStreamPayload) error

// EmitAiStream 发送一条关联当前执行的事件。
func (f AiStreamEmitterFunc) EmitAiStream(event AiStreamEvent, payload AiStreamPayload) error {
	return f(event, payload)
}
