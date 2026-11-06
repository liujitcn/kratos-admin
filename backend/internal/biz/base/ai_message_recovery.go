package biz

import (
	"context"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// expireAiGeneratingMessages 在属主会话锁内终止过期生成，释放会话但不重放业务操作。
func (c *AiMessageCase) expireAiGeneratingMessages(ctx context.Context, session *models.AiSession) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		if err = c.aiSessionCase.lockAiSession(txCtx, session); err != nil {
			return err
		}
		now := time.Now().Truncate(time.Millisecond)
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var list []*models.AiMessage
		list, err = query.WithContext(txCtx).Where(query.TenantID.Eq(session.TenantID), query.SessionID.Eq(session.ID), query.UserID.Eq(session.UserID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)), query.UpdatedAt.Lte(now.Add(-aiApprovalLifetime))).Find()
		if err != nil {
			return err
		}
		approval := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		for _, item := range list {
			runs := approval.WithContext(txCtx).Where(approval.TenantID.Eq(session.TenantID), approval.SessionID.Eq(session.ID), approval.UserID.Eq(session.UserID), approval.MessageID.Eq(item.ID), approval.Status.In(aiRunRunning, aiRunPending))
			var active int64
			active, err = runs.WithContext(txCtx).Where(approval.ExpiresAt.Gt(now)).Count()
			if err != nil {
				return err
			}
			if active > 0 {
				continue
			}
			_, err = runs.WithContext(txCtx).Where(approval.ExpiresAt.Lte(now)).UpdateColumnSimple(approval.Status.Value(aiRunExpired), approval.CheckpointData.Value([]byte{}), approval.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
			output := ai.MarshalReplyContent(&ai.Response{Content: c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.error.run_expired", nil, "Generation expired. This request will not resume automatically. Check the business result before retrying."), Fallback: true})
			_, err = query.WithContext(txCtx).Where(query.ID.Eq(item.ID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING))).UpdateColumnSimple(query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)), query.OutputContent.Value(output), query.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
		}
		return nil
	})
}
