package biz

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/automation"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
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
	var automationRun *models.AiAutomationRun
	var run *models.AiToolApproval
	var state ai.RunState
	var calls []ai.ApprovalCall
	var output ai.OutputContentPayload
	decision := ""
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
		output = ai.ParseOutputContent(message.OutputContent)
		if message.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) || output.Approval == nil || output.Approval.ID != req.GetApprovalId() {
			latest := latestAiApprovalOutcome(output.BlocksJSON)
			if session.AutomationRunID == 0 && message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS) && latest.ApprovalID == req.GetApprovalId() && latest.State == "expired" {
				decision = "expired"
				return nil
			}
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
		now := time.Now().Truncate(time.Millisecond)
		if !now.Before(run.ExpiresAt) {
			decision = "expired"
			return c.closeAiPendingApproval(txCtx, session, message, run, output, aiRunExpired)
		}
		if !req.GetApproved() {
			decision = "rejected"
			return c.closeAiPendingApproval(txCtx, session, message, run, output, aiRunRejected)
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
		if c.aiPluginCase != nil {
			permissionCtx := agent.ContextWithTerminal(txCtx, state.Terminal)
			var pool agent.ToolPool
			pool, txErr = c.resolveAiSessionToolPool(permissionCtx, session, state.Terminal)
			if txErr != nil {
				return txErr
			}
			txCtx = agent.ContextWithToolPool(permissionCtx, pool)
			ctx = agent.ContextWithToolPool(agent.ContextWithTerminal(ctx, state.Terminal), pool)
		}
		if txErr = c.aiRuntime.ValidateApproval(txCtx, state, calls); txErr != nil {
			return errorsx.PermissionDenied("工具配置或当前用户权限已变化，请重新发起").WithCause(txErr)
		}
		now = time.Now().Truncate(time.Millisecond)
		if !now.Before(run.ExpiresAt) {
			decision = "expired"
			return c.closeAiPendingApproval(txCtx, session, message, run, output, aiRunExpired)
		}
		var result gen.ResultInfo
		result, txErr = query.WithContext(txCtx).Where(query.ID.Eq(run.ID), query.Status.Eq(aiRunPending), query.ApprovalID.Eq(req.GetApprovalId())).UpdateSimple(query.Status.Value(aiRunRunning), query.ExpiresAt.Value(now.Add(aiApprovalLifetime)))
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("审批已失效或已处理")
		}
		messageQuery := c.aiMessageRepo.Query(txCtx).AiMessage
		// 先落库确认决策；进程中断时保留未知结果，不能让客户端重放已发出的调用。
		executingOutput := output
		executingOutput.Approval = nil
		executingOutput.BlocksJSON = ai.AppendReplyBlocks(output.BlocksJSON, dto.AiApprovalOutcomeBlock{Type: "approval_outcome", ApprovalID: run.ApprovalID, State: "approved", Execution: "unknown"})
		result, txErr = messageQuery.WithContext(txCtx).Where(messageQuery.ID.Eq(message.ID), messageQuery.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))).UpdateSimple(
			messageQuery.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)),
			messageQuery.OutputContent.Value(ai.MarshalOutputContentPayload(executingOutput)),
		)
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
		}
		// 使用落库时间作为生成版本，兼容数据库时间列的精度。
		message, txErr = c.aiMessageRepo.FindByID(txCtx, message.ID)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	if decision != "" {
		if automationRun != nil {
			if decision == "expired" {
				err = c.automationCase.completeAutomationRun(context.WithoutCancel(ctx), automationRun, "expired", "本次确认已过期", "APPROVAL_EXPIRED", false, time.Now())
			} else {
				err = c.automationCase.completeAutomationRun(context.WithoutCancel(ctx), automationRun, "canceled", "本人已拒绝本次操作", "APPROVAL_REJECTED", false, time.Now())
			}
			if err != nil {
				return nil, err
			}
		}
		return &basev1.SendAiMessageResponse{Messages: []*basev1.AiMessage{c.ToDTO(message)}, Session: c.aiSessionCase.ToDTO(session)}, nil
	}
	var observer agent.ToolExecutionObserver
	if automationRun != nil {
		var stopApproval context.CancelFunc
		ctx, stopApproval = c.automationCase.bindAutomationApproval(ctx, automationRun)
		defer stopApproval()
		observer = c.automationCase.automationToolObserver(automationRun)
	}
	ctx, _, err = c.startAiMessageEvents(ctx, session, message, nil)
	if err != nil {
		return nil, err
	}
	callback.EmitActivity(ctx, callback.Activity{ID: "reasoning:0", Phase: "reasoning", Status: "running"})
	startAt := time.Now()
	var reply *ai.Response
	var execution string
	reply, execution, err = c.resumeAiApproval(ctx, run, state, calls, observer)
	status := int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
	if err == nil {
		err = c.saveAiRuntime(ctx, run, reply, aiRunCompleted)
	} else {
		err = errors.Join(err, c.saveAiRuntime(ctx, run, nil, aiRunFailed))
	}
	if err != nil {
		reply = c.buildAiFailedReply(ctx, reply, err, "", state.ModelName)
		status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)
	} else if output.Fallback {
		reply.Fallback = true
		reply.FallbackReason = output.FallbackReason
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
	blocks := previousAiApprovalBlocks(output.BlocksJSON)
	blocks = append(blocks, dto.AiApprovalOutcomeBlock{Type: "approval_outcome", ApprovalID: req.GetApprovalId(), State: "approved", Execution: execution})
	reply.BlocksJSON = ai.AppendReplyBlocks(reply.BlocksJSON, blocks...)
	err = c.finishAiMessage(ctx, session, message, reply, message.FirstTokenMs, message.DurationMs+durationMilliseconds(startAt, time.Now()), status)
	if err != nil {
		return nil, err
	}
	response := &basev1.SendAiMessageResponse{Messages: []*basev1.AiMessage{c.ToDTO(message)}, Session: c.aiSessionCase.ToDTO(session)}
	if automationRun != nil {
		err = c.automationCase.finishAutomationResponse(context.WithoutCancel(ctx), automationRun, response, nil)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

// resumeAiApproval 汇总恢复中真正执行的审批工具结果，忽略模型和读取工具完成状态。
func (c *AiMessageCase) resumeAiApproval(ctx context.Context, run *models.AiToolApproval, state ai.RunState, calls []ai.ApprovalCall, observer agent.ToolExecutionObserver) (*ai.Response, string, error) {
	var mu sync.Mutex
	outcomes := make(map[string]string, len(calls))
	after := observer.After
	observer.After = func(callCtx context.Context, name string, cause error) error {
		var err error
		if after != nil {
			err = after(callCtx, name, cause)
		}
		target, valid, decision := compose.GetResumeContext[tool.ApprovalDecision](callCtx)
		callID := compose.GetToolCallID(callCtx)
		if target && valid && decision.Approved && decision.Name == name && callID != "" {
			outcome := "succeeded"
			_, networkError := errors.AsType[net.Error](cause)
			if err != nil || networkError || errors.Is(cause, middleware.ErrToolOutcomeUnknown) || errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
				outcome = "unknown"
			} else if cause != nil {
				outcome = "failed"
			}
			mu.Lock()
			outcomes[callID] = outcome
			mu.Unlock()
		}
		return err
	}
	ctx = agent.ContextWithToolExecutionObserver(ctx, observer)
	reply, err := c.aiRuntime.Resume(ctx, strconv.FormatInt(run.ID, 10), state, calls, true)
	mu.Lock()
	defer mu.Unlock()
	if len(outcomes) != len(calls) {
		return reply, "unknown", err
	}
	execution := "succeeded"
	for _, outcome := range outcomes {
		if outcome == "unknown" {
			return reply, outcome, err
		}
		if outcome == "failed" {
			execution = outcome
		}
	}
	return reply, execution, err
}

// closeAiPendingApproval 在会话锁内一起作废检查点和等待消息，拒绝及过期均不恢复运行。
func (c *AiMessageCase) closeAiPendingApproval(ctx context.Context, session *models.AiSession, message *models.AiMessage, run *models.AiToolApproval, output ai.OutputContentPayload, status int32) error {
	query := c.aiApprovalRepo.Query(ctx).AiToolApproval
	result, err := query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq(aiRunPending)).UpdateSimple(query.Status.Value(status), query.CheckpointData.Value([]byte{}))
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("审批已失效或已处理")
	}
	block := dto.AiApprovalOutcomeBlock{Type: "approval_outcome", ApprovalID: run.ApprovalID, State: "rejected", Execution: "not_run"}
	output.Content = c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.approval.rejected", nil, "Rejected. No pending operation was executed.")
	if status == aiRunExpired {
		block.State = "expired"
		block.Retryable = session.AutomationRunID == 0 && canRegenerateAiApproval(output.BlocksJSON)
		output.Content = c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.approval.expired", nil, "Approval expired. No pending operation was executed.")
	}
	output.Approval = nil
	output.BlocksJSON = ai.AppendReplyBlocks(output.BlocksJSON, block)
	messageQuery := c.aiMessageRepo.Query(ctx).AiMessage
	result, err = messageQuery.WithContext(ctx).Where(messageQuery.ID.Eq(message.ID), messageQuery.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))).UpdateSimple(
		messageQuery.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)),
		messageQuery.OutputContent.Value(ai.MarshalOutputContentPayload(output)),
	)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("消息状态已变化，请刷新后重试")
	}
	if status == aiRunRejected {
		if err = c.aiSessionCase.RefreshSessionUpdatedAt(ctx, session); err != nil {
			return err
		}
	}
	var saved *models.AiMessage
	saved, err = c.aiMessageRepo.FindByID(ctx, message.ID)
	if err != nil {
		return err
	}
	*message = *saved
	return nil
}

// expireAiPendingApprovals 清理会话内真实到期的审批，仅按完整属主边界处理相同编号的等待消息。
func (c *AiMessageCase) expireAiPendingApprovals(ctx context.Context, session *models.AiSession, now time.Time) error {
	query := c.aiApprovalRepo.Query(ctx).AiToolApproval
	runs, err := query.WithContext(ctx).Where(query.TenantID.Eq(session.TenantID), query.UserID.Eq(session.UserID), query.SessionID.Eq(session.ID), query.Status.Eq(aiRunPending), query.ExpiresAt.Lte(now)).Find()
	if err != nil {
		return err
	}
	messageQuery := c.aiMessageRepo.Query(ctx).AiMessage
	for _, run := range runs {
		var message *models.AiMessage
		message, err = messageQuery.WithContext(ctx).Where(messageQuery.ID.Eq(run.MessageID), messageQuery.TenantID.Eq(session.TenantID), messageQuery.UserID.Eq(session.UserID), messageQuery.SessionID.Eq(session.ID), messageQuery.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		output := ai.ParseOutputContent(message.OutputContent)
		if output.Approval != nil && output.Approval.ID == run.ApprovalID {
			if err = c.closeAiPendingApproval(ctx, session, message, run, output, aiRunExpired); err != nil {
				return err
			}
		}
	}
	return nil
}

// runAiRuntime 在会话锁内创建运行记录，并按会话类型选择模型执行策略。
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
		run = &models.AiToolApproval{ApprovalID: uuid.NewString(), TenantID: session.TenantID, SessionID: session.ID, MessageID: messageID, UserID: session.UserID, CheckpointData: []byte{}, RuntimeState: "{}", PendingCalls: "[]", Status: aiRunRunning, ExpiresAt: now.Add(aiApprovalLifetime), CreatedAt: now}
		return c.aiApprovalRepo.Create(txCtx, run)
	})
	if err != nil {
		return nil, err
	}
	input.CheckPointID = strconv.FormatInt(run.ID, 10)
	var reply *ai.Response
	if session.Terminal == ai.TerminalAdmin && session.AutomationRunID == 0 {
		reply, err = c.aiRuntime.RunChat(ctx, input, onDelta)
	} else if onDelta == nil {
		reply, err = c.aiRuntime.Run(ctx, input)
	} else {
		reply, err = c.aiRuntime.RunStream(ctx, input, onDelta)
	}
	if err != nil {
		return reply, errors.Join(err, c.saveAiRuntime(ctx, run, nil, aiRunFailed))
	}
	if err = c.saveAiRuntime(ctx, run, reply, aiRunCompleted); err != nil {
		return reply, err
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
		updates := []field.AssignExpr{query.Status.Value(status), query.CheckpointData.Value([]byte{})}
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
			updates = []field.AssignExpr{query.Status.Value(aiRunPending), query.ApprovalID.Value(run.ApprovalID), query.RuntimeState.Value(string(state)), query.PendingCalls.Value(string(calls)), query.ExpiresAt.Value(run.ExpiresAt)}
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

// previousAiApprovalBlocks 续接时保留原知识资料和之前已决定的审批。
func previousAiApprovalBlocks(raw string) []any {
	var blocks []json.RawMessage
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
		return nil
	}
	var result []any
	for _, rawBlock := range blocks {
		var block dto.AiApprovalOutcomeBlock
		if err := json.Unmarshal(rawBlock, &block); err == nil && (block.Type == "approval_outcome" || block.Type == "knowledge_sources") {
			result = append(result, rawBlock)
		}
	}
	return result
}

// latestAiApprovalOutcome 返回最新的持久审批决定，旧编号不能影响下一项审批。
func latestAiApprovalOutcome(raw string) dto.AiApprovalOutcomeBlock {
	blocks := parseAiApprovalOutcomes(raw)
	if len(blocks) > 0 {
		return blocks[len(blocks)-1]
	}
	return dto.AiApprovalOutcomeBlock{}
}

// canRegenerateAiApproval 避免重新生成已批准或外部结果未知的操作。
func canRegenerateAiApproval(raw string) bool {
	var replyErrors []dto.AiReplyErrorBlock
	if err := json.Unmarshal([]byte(raw), &replyErrors); err == nil && slices.ContainsFunc(replyErrors, func(block dto.AiReplyErrorBlock) bool {
		return block.Type == "reply_error" && block.Code == "TOOL_OUTCOME_UNKNOWN"
	}) {
		return false
	}
	return !slices.ContainsFunc(parseAiApprovalOutcomes(raw), func(block dto.AiApprovalOutcomeBlock) bool {
		return block.State == "approved"
	})
}

// parseAiApprovalOutcomes 解析同一轮回复中的全部审批决定，忽略其他业务块结构。
func parseAiApprovalOutcomes(raw string) []dto.AiApprovalOutcomeBlock {
	var blocks []json.RawMessage
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
		return nil
	}
	var outcomes []dto.AiApprovalOutcomeBlock
	for _, rawBlock := range blocks {
		var block dto.AiApprovalOutcomeBlock
		if err := json.Unmarshal(rawBlock, &block); err == nil && block.Type == "approval_outcome" {
			outcomes = append(outcomes, block)
		}
	}
	return outcomes
}
