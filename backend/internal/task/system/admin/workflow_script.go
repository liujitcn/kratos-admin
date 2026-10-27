package admin

import (
	"context"
	"strconv"
	"time"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

// WorkflowScriptRetryTaskName 是工作流脚本恢复任务的稳定调用目标。
const WorkflowScriptRetryTaskName = "system.admin.WorkflowScriptRetry"

var _ cron.TaskExec = (*WorkflowScriptRetryTask)(nil)

// WorkflowScriptRetryTask 恢复待执行、失败退避到期和租约过期的工作流脚本。
type WorkflowScriptRetryTask struct {
	instanceCase *biz.WorkflowInstanceCase
}

// NewWorkflowScriptRetryTask 创建工作流脚本恢复任务。
func NewWorkflowScriptRetryTask(instanceCase *biz.WorkflowInstanceCase) *WorkflowScriptRetryTask {
	return &WorkflowScriptRetryTask{instanceCase: instanceCase}
}

// Exec 执行一次工作流脚本恢复扫描。
func (t *WorkflowScriptRetryTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	count, err := t.instanceCase.ScanWorkflowScripts(ctx, time.Now())
	if err != nil {
		return nil, i18n.WrapMessageError("system.workflow.job.error.script_scan_failed", err)
	}
	return []string{i18n.EncodeMessage("system.workflow.job.result.scripts_scanned", map[string]string{
		"Count": strconv.Itoa(count),
	})}, nil
}
