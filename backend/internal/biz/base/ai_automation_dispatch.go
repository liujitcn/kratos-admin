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
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const automationLeaseLifetime = 90 * time.Second

// enqueueAutomation 在计划行锁内实现手动幂等和同一任务单活动执行。
func (c *AiAutomationCase) enqueueAutomation(ctx context.Context, id, revision int64, key, terminal string, slot int64, now time.Time) (*models.AiAutomationRun, error) {
	var run *models.AiAutomationRun
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		row, err := c.lockAutomation(txCtx, id)
		if err != nil {
			return err
		}
		query := c.runRepo.Query(txCtx).AiAutomationRun
		run, err = query.WithContext(txCtx).Where(query.AutomationID.Eq(id), query.RequestKey.Eq(key)).First()
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if row.Revision != revision || row.Status == "archived" {
			return errorsx.Conflict("计划已变化，请刷新后重试")
		}
		if row.ActiveRunID != 0 {
			return errorsx.Conflict("此计划仍有执行或待确认事项").WithMetadata(map[string]string{"active_run_id": strconv.FormatInt(row.ActiveRunID, 10)})
		}
		run, err = c.newAutomationRun(txCtx, row, key, terminal, slot, "queued", "", now)
		if err != nil {
			return err
		}
		planQuery := c.Query(txCtx).AiAutomation
		_, err = planQuery.WithContext(txCtx).Where(planQuery.ID.Eq(id)).UpdateSimple(planQuery.ActiveRunID.Value(run.ID), planQuery.UpdatedAt.Value(now))
		return err
	})
	return run, err
}

// dispatchAutomation 原子领取到期时隙，错过或上轮占用均保留跳过记录。
func (c *AiAutomationCase) dispatchAutomation(ctx context.Context, id int64, now time.Time) (bool, error) {
	claimed := false
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		row, err := c.lockAutomation(txCtx, id)
		if err != nil {
			return err
		}
		if row.Status != "active" || row.NextFireAt <= 0 || row.NextFireAt > now.UnixMilli() {
			return nil
		}
		slot := row.NextFireAt
		var scheduleTimes []time.Time
		var form *basev1.AiAutomationForm
		form, err = automationConfiguration(row)
		if err != nil {
			return err
		}
		next := int64(0)
		status := row.Status
		if form.Schedule.Frequency == "once" {
			status = "paused"
		} else {
			scheduleTimes, err = automation.Preview(form.Schedule, now, 1)
			if err != nil {
				return err
			}
			next = scheduleTimes[0].UnixMilli()
		}
		runStatus := "queued"
		reason := ""
		if now.UnixMilli()-slot >= time.Minute.Milliseconds() {
			runStatus = "skipped"
			reason = "MISSED_SCHEDULE"
		} else if row.ActiveRunID != 0 {
			runStatus = "skipped"
			reason = "PREVIOUS_RUN_ACTIVE"
		}
		var run *models.AiAutomationRun
		run, err = c.newAutomationRun(txCtx, row, "scheduled:"+strconv.FormatInt(slot, 10), row.Terminal, slot, runStatus, reason, now)
		if err != nil {
			return err
		}
		query := c.Query(txCtx).AiAutomation
		updates := []field.AssignExpr{query.NextFireAt.Value(next), query.LastScheduledAt.Value(slot), query.Status.Value(status), query.UpdatedAt.Value(now)}
		if runStatus == "queued" {
			updates = append(updates, query.ActiveRunID.Value(run.ID))
			claimed = true
		}
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(id)).UpdateSimple(updates...)
		return err
	})
	return claimed, err
}

// newAutomationRun 保存不可变配置及统一唯一请求键，不存凭据。
func (c *AiAutomationCase) newAutomationRun(ctx context.Context, row *models.AiAutomation, key, terminal string, slot int64, status, reason string, now time.Time) (*models.AiAutomationRun, error) {
	form, err := automationConfiguration(row)
	if err != nil {
		return nil, err
	}
	var configuration []byte
	configuration, err = json.Marshal(form)
	if err != nil {
		return nil, err
	}
	trigger := "manual"
	if slot > 0 {
		trigger = "scheduled"
	}
	run := &models.AiAutomationRun{AutomationID: row.ID, TenantID: row.TenantID, OwnerUserID: row.OwnerUserID, Revision: row.Revision, Configuration: string(configuration), Terminal: terminal, Trigger: trigger, ScheduledFor: slot, RequestKey: key, Status: status, ErrorCode: reason, CreatedAt: now, UpdatedAt: now, CreatedBy: row.OwnerUserID, UpdatedBy: row.OwnerUserID}
	if status == "skipped" {
		run.FinishedAt = now.UnixMilli()
		run.Summary = "本次时隙已跳过"
	}
	if err = c.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// claimAutomationRun 在计划锁内领取排队执行，避免与停止或编辑交错释放名额。
func (c *AiAutomationCase) claimAutomationRun(ctx context.Context, id int64, now time.Time) (*models.AiAutomationRun, error) {
	row, err := c.runRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		plan, err := c.lockAutomation(txCtx, row.AutomationID)
		if err != nil {
			return err
		}
		row, err = c.runRepo.FindByID(txCtx, id)
		if err != nil {
			return err
		}
		if plan.ActiveRunID != id || plan.Revision != row.Revision || plan.Status == "archived" {
			return errorsx.Conflict("计划或执行已失效")
		}
		query := c.runRepo.Query(txCtx).AiAutomationRun
		var result gen.ResultInfo
		result, err = query.WithContext(txCtx).Where(query.ID.Eq(id), query.Status.Eq("queued"), query.LeaseVersion.Eq(row.LeaseVersion), query.CancelRequested.Eq(0)).UpdateSimple(query.Status.Value("running"), query.LeaseVersion.Value(row.LeaseVersion+1), query.LeaseUntil.Value(now.Add(automationLeaseLifetime).UnixMilli()), query.StartedAt.Value(now.UnixMilli()), query.UpdatedAt.Value(now))
		if err != nil {
			return err
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("执行已由其他实例领取或停止")
		}
		row.Status = "running"
		row.LeaseVersion++
		row.LeaseUntil = now.Add(automationLeaseLifetime).UnixMilli()
		row.StartedAt = now.UnixMilli()
		return nil
	})
	return row, err
}

// completeAutomationRun 在计划锁内复查停止和 fencing，保存等待或终态并按需释放名额。
func (c *AiAutomationCase) completeAutomationRun(ctx context.Context, run *models.AiAutomationRun, status, summary, code string, unknown bool, now time.Time) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		_, err := c.lockAutomation(txCtx, run.AutomationID)
		if err != nil {
			return err
		}
		query := c.runRepo.Query(txCtx).AiAutomationRun
		var current *models.AiAutomationRun
		current, err = query.WithContext(txCtx).Where(query.ID.Eq(run.ID)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
		if err != nil {
			return err
		}
		if current.CancelRequested != 0 {
			status, summary, code = "canceled", "本次执行已停止", ""
		}
		if current.ToolStarted > 0 || unknown {
			status, summary, code = "unknown", "工具已发出但结果未确认，请核对业务结果", "TOOL_RESULT_UNKNOWN"
			unknown = true
		}
		unknownValue := int32(0)
		if unknown {
			unknownValue = 1
		}
		var result gen.ResultInfo
		conditions := []gen.Condition{query.ID.Eq(run.ID), query.LeaseVersion.Eq(run.LeaseVersion), query.Status.In("queued", "running", "waiting_approval")}
		if run.Status == "running" {
			conditions = append(conditions, query.LeaseUntil.Gt(now.UnixMilli()))
		}
		if status == "waiting_approval" {
			result, err = query.WithContext(txCtx).Where(conditions...).UpdateSimple(query.Status.Value(status), query.Summary.Value(summary), query.LeaseUntil.Value(0), query.Notified.Value(0), query.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
			if result.RowsAffected != 1 {
				return errorsx.Conflict("执行租约或状态已失效")
			}
			return nil
		}
		result, err = query.WithContext(txCtx).Where(conditions...).UpdateSimple(query.Status.Value(status), query.Summary.Value(string([]rune(summary)[:min(len([]rune(summary)), 1000)])), query.ErrorCode.Value(code), query.ResultUnknown.Value(unknownValue), query.FinishedAt.Value(now.UnixMilli()), query.LeaseUntil.Value(0), query.Notified.Value(0), query.UpdatedAt.Value(now))
		if err != nil {
			return err
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("执行租约或状态已失效")
		}
		if err = c.closeAutomationMessages(txCtx, run, summary); err != nil {
			return err
		}
		plan := c.Query(txCtx).AiAutomation
		_, err = plan.WithContext(txCtx).Where(plan.ID.Eq(run.AutomationID), plan.ActiveRunID.Eq(run.ID)).UpdateSimple(plan.ActiveRunID.Value(0), plan.UpdatedAt.Value(now))
		if err != nil {
			return err
		}
		if code == "DEPENDENCY_UNAVAILABLE" {
			_, err = plan.WithContext(txCtx).Where(plan.ID.Eq(run.AutomationID)).UpdateSimple(plan.Status.Value("paused"), plan.NextFireAt.Value(0), plan.Problem.Value("执行依赖不可用，请检查模型、知识库和插件账号后恢复"))
		}
		return err
	})
}

// invalidateAutomationRun 在计划锁内撤回未开始或等待确认的许可，运行中只请求停止。
func (c *AiAutomationCase) invalidateAutomationRun(ctx context.Context, row *models.AiAutomation, editing bool) error {
	if row.ActiveRunID == 0 {
		return nil
	}
	run, err := c.runRepo.FindByID(ctx, row.ActiveRunID)
	if err != nil {
		return err
	}
	if run.Status == "running" {
		if editing {
			return errorsx.Conflict("执行进行中，请先停止本次并等待结束")
		}
		query := c.runRepo.Query(ctx).AiAutomationRun
		_, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq("running")).UpdateSimple(query.CancelRequested.Value(1), query.UpdatedAt.Value(time.Now()))
		return err
	}
	return c.cancelIdleAutomationRun(ctx, row, run)
}

// cancelIdleAutomationRun 作废等待快照及消息并释放名额，不能用于正在调用工具的执行。
func (c *AiAutomationCase) cancelIdleAutomationRun(ctx context.Context, row *models.AiAutomation, run *models.AiAutomationRun) error {
	now := time.Now()
	query := c.runRepo.Query(ctx).AiAutomationRun
	result, err := query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.In("queued", "waiting_approval")).UpdateSimple(query.Status.Value("canceled"), query.CancelRequested.Value(1), query.FinishedAt.Value(now.UnixMilli()), query.LeaseUntil.Value(0), query.Notified.Value(0), query.UpdatedAt.Value(now))
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("执行状态已变化，请刷新后重试")
	}
	if run.SessionID > 0 {
		approvals := c.aiMessageCase.aiApprovalRepo.Query(ctx).AiToolApproval
		_, err = approvals.WithContext(ctx).Where(approvals.SessionID.Eq(run.SessionID), approvals.Status.In(aiRunRunning, aiRunPending)).UpdateSimple(approvals.Status.Value(aiRunRejected), approvals.CheckpointData.Value([]byte{}), approvals.UpdatedAt.Value(now))
		if err != nil {
			return err
		}
		messages := c.aiMessageCase.aiMessageRepo.Query(ctx).AiMessage
		_, err = messages.WithContext(ctx).Where(messages.SessionID.Eq(run.SessionID), messages.Status.In(1, 4)).UpdateColumnSimple(messages.Status.Value(3), messages.UpdatedAt.Value(now))
		if err != nil {
			return err
		}
	}
	plan := c.Query(ctx).AiAutomation
	_, err = plan.WithContext(ctx).Where(plan.ID.Eq(row.ID), plan.ActiveRunID.Eq(run.ID)).UpdateSimple(plan.ActiveRunID.Value(0), plan.UpdatedAt.Value(now))
	return err
}

// closeAutomationMessages 在终态事务中作废残余许可，保留已提交的成功结果及工具审计。
func (c *AiAutomationCase) closeAutomationMessages(ctx context.Context, run *models.AiAutomationRun, summary string) error {
	if run.SessionID <= 0 {
		return nil
	}
	now := time.Now()
	approvals := c.aiMessageCase.aiApprovalRepo.Query(ctx).AiToolApproval
	_, err := approvals.WithContext(ctx).Where(approvals.SessionID.Eq(run.SessionID), approvals.Status.In(aiRunRunning, aiRunPending)).UpdateSimple(approvals.Status.Value(aiRunExpired), approvals.CheckpointData.Value([]byte{}), approvals.UpdatedAt.Value(now))
	if err != nil {
		return err
	}
	messages := c.aiMessageCase.aiMessageRepo.Query(ctx).AiMessage
	_, err = messages.WithContext(ctx).Where(messages.SessionID.Eq(run.SessionID), messages.Status.In(1, 4)).UpdateColumnSimple(messages.Status.Value(3), messages.OutputContent.Value(ai.MarshalReplyContent(&ai.Response{Content: summary})), messages.UpdatedAt.Value(now))
	return err
}
