package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/automation"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// Dispatch 扫描持久化计划并并行执行有限批次，任务记录决定跨实例领取权。
func (c *AiAutomationCase) Dispatch(ctx context.Context) error {
	now := time.Now()
	if err := c.recoverAutomationRuns(ctx, now); err != nil {
		return err
	}
	query := c.Query(ctx).AiAutomation
	rows, err := c.List(ctx, repository.Where(query.Status.Eq("active")), repository.Where(query.NextFireAt.Gt(0)), repository.Where(query.NextFireAt.Lte(now.UnixMilli())), repository.Order(query.NextFireAt.Asc()), repository.Limit(100))
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, err = c.dispatchAutomation(ctx, row.ID, now); err != nil {
			return err
		}
	}
	runQuery := c.runRepo.Query(ctx).AiAutomationRun
	var queued []*models.AiAutomationRun
	queued, err = c.runRepo.List(ctx, repository.Where(runQuery.Status.Eq("queued")), repository.Order(runQuery.ID.Asc()), repository.Limit(4))
	if err != nil {
		return err
	}
dispatch:
	for _, row := range queued {
		select {
		case c.workers <- struct{}{}:
			go func(id int64) {
				defer func() { <-c.workers }()
				if err := c.executeAutomationRun(ctx, id); err != nil {
					log.Error("AiAutomationRun", "error", err)
				}
			}(row.ID)
		default:
			break dispatch
		}
	}

	return c.notifyAutomationRuns(ctx)
}

// executeAutomationRun 领取后重建 owner 身份，再调用普通消息的同一业务入口。
func (c *AiAutomationCase) executeAutomationRun(ctx context.Context, id int64) error {
	run, err := c.claimAutomationRun(ctx, id, time.Now())
	if err != nil {
		return err
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	ctx, err = c.automationOwnerContext(ctx, run)
	if err != nil {
		return c.completeAutomationRun(context.WithoutCancel(ctx), run, "failed", "所有者权限或依赖已失效", "DEPENDENCY_UNAVAILABLE", false, time.Now())
	}
	form := &basev1.AiAutomationForm{}
	if err = json.Unmarshal([]byte(run.Configuration), form); err != nil {
		return c.completeAutomationRun(ctx, run, "failed", "执行配置损坏", "INVALID_CONFIGURATION", false, time.Now())
	}
	if form.Plugins == nil {
		form.Plugins = make([]*basev1.AiPluginSelection, 0)
	}
	if err = c.validateAutomationDependencies(ctx, form, run.Terminal); err != nil {
		return c.completeAutomationRun(ctx, run, "failed", "模型、知识库或插件账号不可用", "DEPENDENCY_UNAVAILABLE", false, time.Now())
	}
	var pool agent.ToolPool
	pool, err = c.pluginCase.ResolveToolPool(ctx, run.Terminal, form.Plugins)
	if err != nil {
		return c.finishAutomationResponse(context.WithoutCancel(ctx), run, nil, err)
	}
	run.ToolRevision = pool.Revision
	query := c.runRepo.Query(ctx).AiAutomationRun
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.LeaseVersion.Eq(run.LeaseVersion), query.Status.Eq("running"), query.CancelRequested.Eq(0)).UpdateSimple(query.ToolRevision.Value(pool.Revision))
	if err != nil {
		return c.finishAutomationResponse(context.WithoutCancel(ctx), run, nil, err)
	}
	if result.RowsAffected != 1 {
		return c.finishAutomationResponse(context.WithoutCancel(ctx), run, nil, errorsx.Conflict("执行已停止"))
	}
	ctx = automation.ContextWithExecution(ctx, automation.Execution{RunID: run.ID, LeaseVersion: run.LeaseVersion})
	ctx = agent.ContextWithToolPool(ctx, pool)
	ctx = agent.ContextWithToolExecutionObserver(ctx, c.automationToolObserver(run))
	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go c.heartbeatAutomation(ctx, run, cancel, heartbeatDone)
	var session *models.AiSession
	session, err = c.createAutomationSession(ctx, run, form)
	if err != nil {
		return c.finishAutomationResponse(context.WithoutCancel(ctx), run, nil, err)
	}
	var response *basev1.SendAiMessageResponse
	response, err = c.aiMessageCase.SendAiMessage(ctx, &basev1.SendAiMessageRequest{SessionId: strconv.FormatInt(session.ID, 10), Content: form.Prompt, ProviderId: form.ProviderId, ModelName: form.ModelName, KnowledgeBaseIds: form.KnowledgeBaseIds})
	finishCtx := context.WithoutCancel(ctx)
	if err != nil {
		return c.finishAutomationResponse(finishCtx, run, nil, err)
	}
	return c.finishAutomationResponse(finishCtx, run, response, nil)
}

// automationOwnerContext 重新加载用户与当前角色、部门、租户，后台不保存 JWT。
func (c *AiAutomationCase) automationOwnerContext(ctx context.Context, run *models.AiAutomationRun) (context.Context, error) {
	user, err := c.userRepo.FindByID(ctx, run.OwnerUserID)
	if err != nil {
		return ctx, err
	}
	if user.TenantID != run.TenantID {
		return ctx, errorsx.PermissionDenied("执行所有者租户已变化")
	}
	var auth *data.UserTokenPayload
	auth, err = c.loginCase.buildAuthInfo(ctx, user)
	if err != nil {
		return ctx, err
	}
	return agent.ContextWithTerminal(engine.ContextWithAuthClaims(ctx, auth.MakeAuthClaims()), run.Terminal), nil
}

// validateAutomationDependencies 显式依赖失效直接拒绝，禁止静默更换模型或连接。
func (c *AiAutomationCase) validateAutomationDependencies(ctx context.Context, form *basev1.AiAutomationForm, terminal string) error {
	selection, err := c.aiMessageCase.aiRuntime.SelectModel(form.ProviderId, form.ModelName)
	if err != nil {
		return errorsx.InvalidArgument("固定模型不可用").WithCause(err)
	}
	if selection.ProviderID != form.ProviderId || selection.ModelName != form.ModelName {
		return errorsx.Conflict("固定模型已变化")
	}
	for _, id := range form.KnowledgeBaseIds {
		if _, err = c.aiMessageCase.aiKnowledgeEngine.GetKnowledge(ctx, id); err != nil {
			return errorsx.PermissionDenied("知识库已不可用").WithCause(err)
		}
	}
	selections := form.Plugins
	if selections == nil {
		selections = make([]*basev1.AiPluginSelection, 0)
	}
	_, err = c.pluginCase.ResolveToolPool(ctx, terminal, selections)
	return err
}

// createAutomationSession 在计划锁内登记独立冻结结果会话与执行引用。
func (c *AiAutomationCase) createAutomationSession(ctx context.Context, run *models.AiAutomationRun, form *basev1.AiAutomationForm) (*models.AiSession, error) {
	var session *models.AiSession
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		plan, err := c.lockAutomation(txCtx, run.AutomationID)
		if err != nil {
			return err
		}
		if plan.Revision != run.Revision || plan.ActiveRunID != run.ID {
			return errorsx.Conflict("计划或执行已失效")
		}
		if err = c.checkAutomationExecution(txCtx, run, false); err != nil {
			return err
		}
		plugins := plan.Plugins
		if plugins == "" {
			plugins = "[]"
		}
		session = &models.AiSession{TenantID: run.TenantID, UserID: run.OwnerUserID, Terminal: ai.TerminalAdmin, Title: "自动化 · " + form.Name, Summary: form.Prompt, Plugins: plugins, AutomationID: plan.ID, AutomationRunID: run.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if run.Terminal == "app" {
			session.Terminal = ai.TerminalApp
		}
		if err = c.aiSessionCase.Create(txCtx, session); err != nil {
			return err
		}
		query := c.runRepo.Query(txCtx).AiAutomationRun
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(run.ID), query.LeaseVersion.Eq(run.LeaseVersion)).UpdateSimple(query.SessionID.Value(session.ID))
		return err
	})
	if err == nil {
		run.SessionID = session.ID
	}
	return session, err
}

// checkAutomationExecution 每次模型或工具边界重核 fencing、任务版本与停止标记。
func (c *AiAutomationCase) checkAutomationExecution(ctx context.Context, run *models.AiAutomationRun, markTool bool) error {
	plan, err := c.FindByID(ctx, run.AutomationID)
	if err != nil {
		return err
	}
	if plan.Revision != run.Revision || plan.ActiveRunID != run.ID || plan.Status == "archived" {
		return errorsx.Conflict("自动化配置已变化")
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	if !markTool {
		_, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq("running"), query.LeaseVersion.Eq(run.LeaseVersion), query.LeaseUntil.Gt(time.Now().UnixMilli()), query.CancelRequested.Eq(0)).First()
		return err
	}
	updates := []field.AssignExpr{query.UpdatedAt.Value(time.Now())}
	if markTool {
		updates = append(updates, query.ToolStarted.Add(1))
	}
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq("running"), query.LeaseVersion.Eq(run.LeaseVersion), query.LeaseUntil.Gt(time.Now().UnixMilli()), query.CancelRequested.Eq(0)).UpdateSimple(updates...)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("自动化执行已停止或租约失效")
	}
	return nil
}

// automationToolObserver 在审批通过后持久化调用中状态，失联或错误不能自动重放。
func (c *AiAutomationCase) automationToolObserver(run *models.AiAutomationRun) agent.ToolExecutionObserver {
	return agent.ToolExecutionObserver{BeforeModel: func(ctx context.Context) error {
		err := c.checkAutomationExecution(ctx, run, false)
		if err != nil {
			return fmt.Errorf("自动化不能继续模型调用: %w: %w", context.Canceled, err)
		}
		return nil
	}, Before: func(ctx context.Context, _ string) error {
		ownerCtx, err := c.automationOwnerContext(ctx, run)
		if err != nil {
			return err
		}
		var original, current *data.UserTokenPayload
		original, err = c.GetAuthInfo(ctx)
		if err != nil {
			return err
		}
		current, err = c.GetAuthInfo(ownerCtx)
		if err != nil {
			return err
		}
		if original.RoleId != current.RoleId || original.RoleCode != current.RoleCode || original.DataScope != current.DataScope || original.DeptId != current.DeptId {
			return errorsx.PermissionDenied("执行身份权限已变化，请重新运行")
		}
		return c.checkAutomationExecution(ctx, run, true)
	}}
}

// heartbeatAutomation 定期续租，丢失租约立即停止后续调用。
func (c *AiAutomationCase) heartbeatAutomation(ctx context.Context, run *models.AiAutomationRun, cancel context.CancelFunc, done <-chan struct{}) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			query := c.runRepo.Query(ctx).AiAutomationRun
			now := time.Now()
			result, err := query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq("running"), query.LeaseVersion.Eq(run.LeaseVersion), query.LeaseUntil.Gt(now.UnixMilli())).UpdateSimple(query.LeaseUntil.Value(now.Add(automationLeaseLifetime).UnixMilli()))
			if err != nil || result.RowsAffected != 1 {
				cancel()
				return
			}
		}
	}
}

// finishAutomationResponse 持久化等待或终态；停止与未知结果不能被成功回复覆盖。
func (c *AiAutomationCase) finishAutomationResponse(ctx context.Context, run *models.AiAutomationRun, response *basev1.SendAiMessageResponse, cause error) error {
	current, err := c.runRepo.FindByID(ctx, run.ID)
	if err != nil {
		return err
	}
	if current.LeaseVersion != run.LeaseVersion || current.Status != "running" {
		return errorsx.Conflict("执行已被恢复流程终止")
	}
	status := "succeeded"
	code := ""
	summary := "本次执行已完成，请查看结果"
	if cause != nil {
		status = "failed"
		code = "EXECUTION_FAILED"
		summary = "本次执行失败，请查看记录"
	}
	if response != nil && len(response.Messages) > 0 {
		message := response.Messages[len(response.Messages)-1]
		if message.Status == basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL {
			status = "waiting_approval"
			summary = "等待本人确认本次工具操作"
		}
		if message.Status == basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED {
			status = "failed"
			code = "EXECUTION_FAILED"
			summary = "本次执行失败，请查看记录"
		}
	}
	if current.CancelRequested != 0 {
		status = "canceled"
		summary = "本次执行已停止"
	}
	// 先确认工具结果已随消息持久化，才能解除未知标记；进程中途退出始终保留标记。
	if response != nil && len(response.Messages) > 0 {
		message := response.Messages[len(response.Messages)-1]
		known := len(message.Tools) > 0 && (message.Status == basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS || message.Status == basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL)
		for _, call := range message.Tools {
			if call.Status != "success" {
				known = false
			}
		}
		if known {
			query := c.runRepo.Query(ctx).AiAutomationRun
			_, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.LeaseVersion.Eq(run.LeaseVersion), query.Status.Eq("running"), query.LeaseUntil.Gt(time.Now().UnixMilli())).UpdateSimple(query.ToolStarted.Value(0))
			if err != nil {
				return err
			}
			current.ToolStarted = 0
		}
	}
	unknown := current.ToolStarted > 0
	if unknown {
		status = "unknown"
		code = "TOOL_RESULT_UNKNOWN"
		summary = "工具已发出但结果未确认，请核对业务结果"
	}
	return c.completeAutomationRun(ctx, run, status, summary, code, unknown, time.Now())
}

// recoverAutomationRuns 回收失效租约和过期待确认，不重放已发出的工具。
func (c *AiAutomationCase) recoverAutomationRuns(ctx context.Context, now time.Time) error {
	query := c.runRepo.Query(ctx).AiAutomationRun
	rows, err := c.runRepo.List(ctx, repository.Where(query.Status.In("running", "waiting_approval")), repository.Order(query.ID.Asc()), repository.Limit(200))
	if err != nil {
		return err
	}
	for _, run := range rows {
		expired := run.Status == "running" && run.LeaseUntil <= now.UnixMilli()
		if run.Status == "waiting_approval" {
			approval := c.aiMessageCase.aiApprovalRepo.Query(ctx).AiToolApproval
			var pending int64
			pending, err = c.aiMessageCase.aiApprovalRepo.Count(ctx, repository.Where(approval.SessionID.Eq(run.SessionID)), repository.Where(approval.Status.Eq(aiRunPending)), repository.Where(approval.ExpiresAt.Gt(now)))
			if err != nil {
				return err
			}
			expired = pending == 0
		}
		if !expired {
			continue
		}
		err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
			plan, err := c.lockAutomation(txCtx, run.AutomationID)
			if err != nil {
				return err
			}
			current, err := c.runRepo.FindByID(txCtx, run.ID)
			if err != nil {
				return err
			}
			if current.Status != run.Status || current.LeaseVersion != run.LeaseVersion || (current.Status == "running" && current.LeaseUntil > now.UnixMilli()) {
				return nil
			}
			status := "expired"
			unknown := int32(0)
			code := "EXECUTION_EXPIRED"
			summary := "执行已过期；未确认结果的外部操作请人工核对"
			if current.CancelRequested != 0 && current.ToolStarted == 0 {
				status = "canceled"
				code = ""
				summary = "本次执行已停止"
			} else if current.ToolStarted > 0 {
				status = "unknown"
				unknown = 1
				code = "TOOL_RESULT_UNKNOWN"
			}
			runQuery := c.runRepo.Query(txCtx).AiAutomationRun
			_, err = runQuery.WithContext(txCtx).Where(runQuery.ID.Eq(run.ID)).UpdateSimple(runQuery.Status.Value(status), runQuery.ResultUnknown.Value(unknown), runQuery.ErrorCode.Value(code), runQuery.Summary.Value(summary), runQuery.LeaseVersion.Value(current.LeaseVersion+1), runQuery.LeaseUntil.Value(0), runQuery.FinishedAt.Value(now.UnixMilli()), runQuery.Notified.Value(0), runQuery.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
			if run.SessionID > 0 {
				approval := c.aiMessageCase.aiApprovalRepo.Query(txCtx).AiToolApproval
				_, err = approval.WithContext(txCtx).Where(approval.SessionID.Eq(run.SessionID), approval.Status.In(aiRunPending, aiRunRunning)).UpdateSimple(approval.Status.Value(aiRunExpired), approval.CheckpointData.Value([]byte{}), approval.UpdatedAt.Value(now))
				if err != nil {
					return err
				}
			}
			if err = c.closeAutomationMessages(txCtx, run, summary); err != nil {
				return err
			}
			planQuery := c.Query(txCtx).AiAutomation
			_, err = planQuery.WithContext(txCtx).Where(planQuery.ID.Eq(plan.ID), planQuery.ActiveRunID.Eq(run.ID)).UpdateSimple(planQuery.ActiveRunID.Value(0), planQuery.UpdatedAt.Value(now))
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// notifyAutomationRuns 独立重试站内提醒，唯一事件键保证不会重新执行任务。
func (c *AiAutomationCase) notifyAutomationRuns(ctx context.Context) error {
	query := c.runRepo.Query(ctx).AiAutomationRun
	rows, err := c.runRepo.List(ctx, repository.Where(query.Status.NotIn("queued", "running")), repository.Where(query.Notified.Eq(0)), repository.Order(query.ID.Asc()), repository.Limit(100))
	if err != nil {
		return err
	}
	for _, run := range rows {
		actionParams, err := json.Marshal(map[string]string{"automation_id": strconv.FormatInt(run.AutomationID, 10), "run_id": strconv.FormatInt(run.ID, 10), "session_id": strconv.FormatInt(run.SessionID, 10)})
		if err != nil {
			return err
		}
		_, err = notification.Publish(ctx, notification.Message{TenantID: run.TenantID, CategoryCode: "ai_automation", Title: "AI自动化执行更新", Content: run.Summary, ActionType: basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY, ActionTarget: "AI", ActionParams: string(actionParams), Audiences: []notification.Audience{{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER, ID: run.OwnerUserID}}, Source: "ai_automation", IdempotencyKey: fmt.Sprintf("ai:%d:%s", run.ID, run.Status)})
		if err != nil {
			log.Error("AiAutomationNotification", "run", run.ID, "error", err)
			continue
		}
		_, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq(run.Status)).UpdateSimple(query.Notified.Value(1))
		if err != nil {
			return err
		}
	}
	return nil
}
