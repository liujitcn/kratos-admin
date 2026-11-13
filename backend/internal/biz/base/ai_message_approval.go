package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/automation"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/google/uuid"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const (
	aiRunRunning int32 = iota
	aiRunPending
	aiRunCompleted
	aiRunRejected
	aiRunExpired
	aiRunFailed
	aiApprovalLifetime = 10 * time.Minute
)

// DecideAiToolApproval 原子领取审批执行权，恢复原始调用；重复请求不能再次执行。
func (c *AiMessageCase) DecideAiToolApproval(ctx context.Context, req *basev1.DecideAiToolApprovalRequest) (*basev1.SendAiMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if session.AutomationRunID > 0 {
		var currentRun *models.AiAutomationRun
		currentRun, err = c.automationCase.ownedAutomationRun(ctx, strconv.FormatInt(session.AutomationRunID, 10))
		if err != nil {
			return nil, err
		}
		ctx, err = c.automationCase.automationOwnerContext(ctx, currentRun)
		if err != nil {
			return nil, err
		}
	}
	ctx, err = c.bindAiSessionTools(ctx, session)
	if err != nil {
		return nil, err
	}
	if err = c.validatePendingAiTools(ctx, session, message); err != nil {
		return nil, err
	}
	var automationRun *models.AiAutomationRun
	var run *models.AiToolApproval
	var state ai.RunState
	var calls []ai.ApprovalCall
	var now time.Time
	expired := false
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var txErr error
		if session.AutomationRunID > 0 {
			automationRun, txErr = c.automationCase.claimAutomationApproval(txCtx, session)
			if txErr != nil {
				return txErr
			}
		}
		if txErr = c.aiSessionCase.lockAiSession(txCtx, session); txErr != nil {
			return txErr
		}
		message, txErr = c.aiMessageRepo.FindByID(txCtx, message.ID)
		if txErr != nil {
			return txErr
		}
		output := ai.ParseOutputContent(message.OutputContent)
		if message.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) || output.Approval == nil || output.Approval.ID != req.GetApprovalId() {
			return errorsx.Conflict("审批已失效或已处理")
		}
		if txErr = c.ensureLastAiMessage(txCtx, session.TenantID, session.ID, message.ID); txErr != nil {
			return txErr
		}
		query := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		run, txErr = query.WithContext(txCtx).Where(query.ApprovalID.Eq(req.GetApprovalId()), query.TenantID.Eq(session.TenantID), query.SessionID.Eq(session.ID), query.MessageID.Eq(message.ID), query.UserID.Eq(session.UserID), query.Status.Eq(aiRunPending)).First()
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return errorsx.Conflict("审批已失效或已处理")
		}
		if txErr != nil {
			return txErr
		}
		if txErr = json.Unmarshal([]byte(run.RuntimeState), &state); txErr != nil {
			return errorsx.Internal("审批运行配置损坏").WithCause(txErr)
		}
		if txErr = json.Unmarshal([]byte(run.PendingCalls), &calls); txErr != nil {
			return errorsx.Internal("审批工具参数损坏").WithCause(txErr)
		}
		if state.Terminal != ai.NormalizeTerminalString(session.Terminal) || len(calls) == 0 {
			return errorsx.Conflict("审批运行状态无效")
		}
		now = time.Now().Truncate(time.Millisecond)
		expired = !now.Before(run.ExpiresAt)
		status := aiRunRunning
		if expired {
			status = aiRunExpired
		}
		var result gen.ResultInfo
		result, txErr = query.WithContext(txCtx).Where(query.ID.Eq(run.ID), query.Status.Eq(aiRunPending), query.ApprovalID.Eq(req.GetApprovalId())).UpdateSimple(query.Status.Value(status), query.ExpiresAt.Value(now.Add(aiApprovalLifetime)), query.UpdatedAt.Value(now))
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("审批已失效或已处理")
		}
		messageQuery := c.aiMessageRepo.Query(txCtx).AiMessage
		result, txErr = messageQuery.WithContext(txCtx).Where(messageQuery.ID.Eq(message.ID), messageQuery.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))).UpdateColumnSimple(messageQuery.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)), messageQuery.UpdatedAt.Value(now))
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	message.Status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)
	message.UpdatedAt = now
	if automationRun != nil {
		var stopApproval context.CancelFunc
		ctx, stopApproval = c.automationCase.bindAutomationApproval(ctx, automationRun)
		defer stopApproval()
	}
	startAt := time.Now()
	var reply *ai.Response
	status := int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
	if expired {
		reply = &ai.Response{Content: c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.approval.expired", nil, "Approval expired. No pending operation was executed.")}
		query := c.aiApprovalRepo.Query(ctx).AiToolApproval
		_, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID)).UpdateSimple(query.CheckpointData.Value([]byte{}))
	} else if !req.GetApproved() {
		// 拒绝后直接结束，避免模型在同一轮再次申请被拒绝的操作。
		reply = &ai.Response{Content: c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.approval.rejected", nil, "Rejected. No pending operation was executed.")}
		err = c.saveAiRuntime(ctx, run, reply, aiRunRejected)
	} else {
		reply, err = c.aiRuntime.Resume(ctx, strconv.FormatInt(run.ID, 10), state, calls, true)
		if err == nil {
			err = c.saveAiRuntime(ctx, run, reply, aiRunCompleted)
		} else {
			err = errors.Join(err, c.saveAiRuntime(ctx, run, nil, aiRunFailed))
		}
	}
	if err != nil {
		reply = c.buildAiFailedReply(ctx, reply, err, "", state.ModelName)
		status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)
	}
	// 前半轮的读取工具和 token 必须保留，等待确认的时间不计入执行耗时。
	previous := ai.ParseTokenUsage(message.Token)
	reply.Token.Input += previous.Input
	reply.Token.Output += previous.Output
	reply.Token.Cache += previous.Cache
	reply.Token.Total += previous.Total
	reply.Tools = append(ai.ParseTools(message.Tools), reply.Tools...)
	// 审批恢复的执行结果按业务规则附加结构化引导卡片（例如问数成功后的建卡入口）。
	if reply.BlocksJSON == "" {
		if blocks := c.aiRuntime.BuildResponseBlocks(reply.Tools); len(blocks) > 0 {
			if raw, marshalErr := json.Marshal(blocks); marshalErr == nil {
				reply.BlocksJSON = string(raw)
			}
		}
	}
	err = c.finishAiMessage(ctx, session, message, reply, time.Now(), message.FirstTokenMs, message.DurationMs+durationMilliseconds(startAt, time.Now()), status)
	if err != nil {
		return nil, err
	}
	response := &basev1.SendAiMessageResponse{Messages: []*basev1.AiMessage{c.ToDTO(message)}, Session: c.aiSessionCase.ToDTO(session)}
	if automationRun != nil {
		if expired {
			err = c.automationCase.completeAutomationRun(context.WithoutCancel(ctx), automationRun, "expired", "本次确认已过期", "APPROVAL_EXPIRED", false, time.Now())
		} else if !req.GetApproved() {
			err = c.automationCase.completeAutomationRun(context.WithoutCancel(ctx), automationRun, "canceled", "本人已拒绝本次操作", "APPROVAL_REJECTED", false, time.Now())
		} else {
			err = c.automationCase.finishAutomationResponse(context.WithoutCancel(ctx), automationRun, response, nil)
		}
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

// runAiRuntime 在会话锁内创建持久化运行记录，防止会话删除后留下孤儿快照。
func (c *AiMessageCase) runAiRuntime(ctx context.Context, session *models.AiSession, messageID int64, input ai.RuntimeInput, onDelta func(string)) (*ai.Response, error) {
	var run *models.AiToolApproval
	var err error
	ctx, err = c.bindAiSessionTools(ctx, session)
	if err != nil {
		return nil, err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.aiSessionCase.lockAiSession(txCtx, session); err != nil {
			return err
		}
		now := time.Now()
		run = &models.AiToolApproval{ApprovalID: uuid.NewString(), TenantID: session.TenantID, SessionID: session.ID, MessageID: messageID, UserID: session.UserID, CheckpointData: []byte{}, RuntimeState: "{}", PendingCalls: "[]", Status: aiRunRunning, ExpiresAt: now.Add(aiApprovalLifetime), CreatedAt: now, UpdatedAt: now}
		return c.aiApprovalRepo.Create(txCtx, run)
	})
	if err != nil {
		return nil, err
	}
	input.CheckPointID = strconv.FormatInt(run.ID, 10)
	var reply *ai.Response
	if onDelta == nil {
		reply, err = c.aiRuntime.Run(ctx, input)
	} else {
		reply, err = c.aiRuntime.RunStream(ctx, input, onDelta)
	}
	if err != nil {
		return reply, errors.Join(err, c.saveAiRuntime(ctx, run, nil, aiRunFailed))
	}
	if err = c.saveAiRuntime(ctx, run, reply, aiRunCompleted); err != nil {
		return nil, err
	}
	return reply, nil
}

// saveAiRuntime 仅在运行有效期内保存中断或终止状态，每次中断分配新的确认编号。
func (c *AiMessageCase) saveAiRuntime(ctx context.Context, run *models.AiToolApproval, reply *ai.Response, status int32) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), aiMessagePersistTimeout)
	defer cancel()
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		if execution := automation.ExecutionFromContext(ctx); execution.RunID > 0 {
			if err := c.automationCase.lockAutomationExecution(ctx, execution, reply != nil && reply.Pending != nil); err != nil {
				return err
			}
		}

		query := c.aiApprovalRepo.Query(ctx).AiToolApproval
		now := time.Now()
		updates := []field.AssignExpr{query.Status.Value(status), query.CheckpointData.Value([]byte{}), query.UpdatedAt.Value(now)}
		var err error
		if reply != nil && reply.Pending != nil {
			var state []byte
			state, err = json.Marshal(reply.Pending.State)
			if err != nil {
				return err
			}
			var calls []byte
			calls, err = json.Marshal(reply.Pending.Calls)
			if err != nil {
				return err
			}
			run.ApprovalID = uuid.NewString()
			run.ExpiresAt = now.Add(aiApprovalLifetime)
			updates = []field.AssignExpr{query.Status.Value(aiRunPending), query.ApprovalID.Value(run.ApprovalID), query.RuntimeState.Value(string(state)), query.PendingCalls.Value(string(calls)), query.ExpiresAt.Value(run.ExpiresAt), query.UpdatedAt.Value(now)}
			reply.Approval = &ai.Approval{ID: run.ApprovalID, ExpiresAt: run.ExpiresAt}
			for _, call := range reply.Pending.Calls {
				reply.Approval.Calls = append(reply.Approval.Calls, call.ApprovalInfo)
			}
		}
		var result gen.ResultInfo
		result, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq(aiRunRunning), query.ExpiresAt.Gt(now)).UpdateSimple(updates...)
		if err != nil {
			return err
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("AI运行状态已变化，不能保存审批")
		}
		return nil
	})
}
