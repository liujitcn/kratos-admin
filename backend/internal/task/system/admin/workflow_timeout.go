package admin

import (
	"context"
	"strconv"
	"time"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// WorkflowTimeoutTaskName 是工作流超时催办任务的稳定调用目标。
	WorkflowTimeoutTaskName = "system.admin.WorkflowTimeout"
)

var _ cron.TaskExec = (*WorkflowTimeoutTask)(nil)

// WorkflowTimeoutTask 扫描超时未处理的审批待办，催办提醒办理人并升级提醒申请人。
type WorkflowTimeoutTask struct {
	instanceCase *biz.WorkflowInstanceCase
}

// NewWorkflowTimeoutTask 创建工作流超时催办任务。
func NewWorkflowTimeoutTask(instanceCase *biz.WorkflowInstanceCase) *WorkflowTimeoutTask {
	return &WorkflowTimeoutTask{instanceCase: instanceCase}
}

// Exec 执行一次工作流超时催办扫描。
func (t *WorkflowTimeoutTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	reminded, escalated, err := t.instanceCase.ScanWorkflowTimeouts(ctx, time.Now())
	if err != nil {
		return nil, i18n.WrapMessageError("system.workflow.job.error.timeout_scan_failed", err)
	}
	return []string{i18n.EncodeMessage("system.workflow.job.result.timeout_scanned", map[string]string{
		"Reminded": strconv.Itoa(reminded), "Escalated": strconv.Itoa(escalated),
	})}, nil
}
