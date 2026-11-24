package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gen"
	"gorm.io/gorm"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/errorsx"
)

type aiExecutionContextKey struct{}

// ListAiMessageEvent 只读补读当前用户消息的最新执行，摘要缓存和检查点永不对外返回。
func (c *AiMessageCase) ListAiMessageEvent(ctx context.Context, req *basev1.ListAiMessageEventRequest) (*basev1.ListAiMessageEventResponse, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}
	var messageID int64
	messageID, err = strconv.ParseInt(req.GetMessageId(), 10, 64)
	if err != nil || messageID <= 0 {
		return nil, errorsx.InvalidArgument("消息编号不合法")
	}
	var message *models.AiMessage
	var after int64
	if req.GetAfterId() != "" {
		after, err = strconv.ParseInt(req.GetAfterId(), 10, 64)
		if err != nil || after < 0 {
			return nil, errorsx.InvalidArgument("事件游标不合法")
		}
	}
	var response *basev1.ListAiMessageEventResponse
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		// 与消息终态和新运行共享会话锁，避免把旧快照和新运行事件拼成错误状态。
		if err = c.aiSessionCase.lockAiSession(txCtx, session); err != nil {
			return err
		}
		messageQuery := c.aiMessageRepo.Query(txCtx).AiMessage
		message, err = messageQuery.WithContext(txCtx).Where(messageQuery.ID.Eq(messageID), messageQuery.SessionID.Eq(session.ID), messageQuery.TenantID.Eq(session.TenantID), messageQuery.UserID.Eq(session.UserID)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("消息不存在")
		}
		if err != nil {
			return err
		}
		response = &basev1.ListAiMessageEventResponse{Events: []*basev1.AiMessageEvent{}, NextCursor: req.GetAfterId(), Messages: []*basev1.AiMessage{c.ToDTO(message)}, Session: c.aiSessionCase.ToDTO(session)}
		query := c.aiMessageRepo.Query(txCtx).AiMessageEvent
		conditions := []gen.Condition{query.TenantID.Eq(session.TenantID), query.UserID.Eq(session.UserID), query.SessionID.Eq(session.ID), query.MessageID.Eq(message.ID)}
		var latest *models.AiMessageEvent
		latest, err = query.WithContext(txCtx).Where(conditions...).Where(query.EventType.Eq("run_start")).Order(query.ID.Desc()).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		response.RunId = latest.RunID
		var items []*models.AiMessageEvent
		items, err = query.WithContext(txCtx).Where(conditions...).Where(query.RunID.Eq(latest.RunID), query.ID.Gt(after), query.EventType.In("activity", "finish", "approval_required", "error")).Order(query.ID.Asc()).Limit(101).Find()
		if err != nil {
			return err
		}
		response.HasMore = len(items) > 100
		if response.HasMore {
			items = items[:100]
		}
		for _, item := range items {
			id := strconv.FormatInt(item.ID, 10)
			response.Events = append(response.Events, &basev1.AiMessageEvent{Id: id, RunId: item.RunID, Event: item.EventType, Payload: item.Payload})
			response.NextCursor = id
		}
		return nil
	})
	return response, err
}

// startAiMessageEvents 先持久化执行边界再运行模型；活动持久化失败会取消后续模型和工具。
func (c *AiMessageCase) startAiMessageEvents(ctx context.Context, session *models.AiSession, message *models.AiMessage, emitter dto.AiStreamEmitter) (context.Context, dto.AiStreamEmitter, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	execution := &dto.AiExecution{RunID: uuid.NewString(), Session: session, Message: message, Generation: message.UpdatedAt, Cancel: cancel}
	ctx = context.WithValue(ctx, aiExecutionContextKey{}, execution)
	err := c.recordAiMessageEvent(ctx, execution, "run_start", dto.AiStreamPayload{})
	if err != nil {
		cancel(err)
		return ctx, emitter, err
	}
	wrapped := dto.AiStreamEmitterFunc(func(event dto.AiStreamEvent, payload dto.AiStreamPayload) error {
		execution.Mutex.Lock()
		defer execution.Mutex.Unlock()
		payload.RunID = execution.RunID
		if event == dto.AiStreamEventActivity {
			var eventErr error
			eventErr = c.recordAiMessageEvent(ctx, execution, "activity", payload)
			payload.EventID = execution.LastEventID
			if eventErr != nil {
				cancel(eventErr)
				return eventErr
			}
		} else if event != dto.AiStreamEventDelta {
			if execution.TerminalID == "" {
				return errorsx.Conflict("消息终态尚未持久化，请通过只读接口核实")
			}
			payload.EventID = execution.TerminalID
		}
		if emitter != nil {
			return emitter.EmitAiStream(event, payload)
		}
		return nil
	})
	ctx = callback.WithActivityEmitter(ctx, func(activity callback.Activity) {
		err := wrapped.EmitAiStream(dto.AiStreamEventActivity, dto.AiStreamPayload{SessionID: strconv.FormatInt(session.ID, 10), MessageID: strconv.FormatInt(message.ID, 10), Activity: &dto.AiStreamActivity{ID: activity.ID, Phase: activity.Phase, Status: activity.Status, Summary: activity.Summary, ToolName: activity.ToolName, ToolTitle: activity.ToolTitle}})
		if err != nil {
			cancel(err)
		}
	})
	return ctx, wrapped, nil
}

// recordAiMessageEvent 串行校验当前生成版本并持久化关键事件，旧请求不能向新执行追加内容。
func (c *AiMessageCase) recordAiMessageEvent(ctx context.Context, execution *dto.AiExecution, event string, payload any) error {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), aiMessagePersistTimeout)
	defer cancel()
	var item *models.AiMessageEvent
	err := c.tx.Transaction(persistCtx, func(txCtx context.Context) error {
		var err error
		if err = c.aiSessionCase.lockAiSession(txCtx, execution.Session); err != nil {
			return err
		}
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var current *models.AiMessage
		current, err = query.WithContext(txCtx).Where(query.ID.Eq(execution.Message.ID), query.TenantID.Eq(execution.Session.TenantID), query.UserID.Eq(execution.Session.UserID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)), query.UpdatedAt.Eq(execution.Generation)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.Conflict("消息执行已失效")
		}
		if err != nil {
			return err
		}
		item, err = c.createAiMessageEvent(txCtx, execution.Session, current.ID, execution.RunID, event, payload)
		return err
	})
	if err == nil {
		execution.LastEventID = strconv.FormatInt(item.ID, 10)
	}
	return err
}

// createAiMessageEvent 在调用方持有的会话事务内追加事件，数据库ID即稳定补读游标。
func (c *AiMessageCase) createAiMessageEvent(ctx context.Context, session *models.AiSession, messageID int64, runID, event string, payload any) (*models.AiMessageEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	item := &models.AiMessageEvent{TenantID: session.TenantID, UserID: session.UserID, SessionID: session.ID, MessageID: messageID, RunID: runID, EventType: event, Payload: string(raw), CreatedAt: time.Now()}
	err = c.aiMessageRepo.Query(ctx).AiMessageEvent.WithContext(ctx).Create(item)
	return item, err
}

// finishAiMessageEvent 将终态与消息放入同一事务，客户端断开后仍可用只读接口恢复。
func (c *AiMessageCase) finishAiMessageEvent(ctx context.Context, session *models.AiSession, message *models.AiMessage) error {
	execution, _ := ctx.Value(aiExecutionContextKey{}).(*dto.AiExecution)
	if execution == nil {
		return nil
	}
	event := dto.AiStreamEventFinish
	if message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED) {
		event = dto.AiStreamEventError
	}
	if message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) {
		event = dto.AiStreamEventApprovalRequired
	}
	payload := dto.AiStreamPayload{SessionID: strconv.FormatInt(session.ID, 10), MessageID: strconv.FormatInt(message.ID, 10), RunID: execution.RunID, Messages: []*basev1.AiMessage{c.ToDTO(message)}, Session: c.aiSessionCase.ToDTO(session)}
	item, err := c.createAiMessageEvent(ctx, session, message.ID, execution.RunID, string(event), payload)
	if err != nil {
		return err
	}
	execution.TerminalID = strconv.FormatInt(item.ID, 10)
	return nil
}

// loadAiContextSummary 读取同会话的候选摘要；来源与权限修订仍由运行时重新验证。
func (c *AiMessageCase) loadAiContextSummary(ctx context.Context, session *models.AiSession) (*agent.ContextSummary, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessageEvent
	item, err := query.WithContext(ctx).Where(query.TenantID.Eq(session.TenantID), query.UserID.Eq(session.UserID), query.SessionID.Eq(session.ID), query.EventType.Eq("context_summary")).Order(query.ID.Desc()).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var summary agent.ContextSummary
	if err = json.Unmarshal([]byte(item.Payload), &summary); err != nil {
		return nil, errorsx.Internal("上下文摘要缓存损坏").WithCause(err)
	}
	return &summary, nil
}
