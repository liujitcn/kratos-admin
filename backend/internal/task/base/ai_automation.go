package base

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/pkg/aiautomation"
)

// AiAutomationTaskName 是唯一 AI 自动化内部扫描器的调用目标。
const AiAutomationTaskName = "base.AiAutomationDispatch"

// AiAutomationTask 扫描本人计划，由持久化状态控制每条任务的执行。
type AiAutomationTask struct{ dispatcher *aiautomation.Dispatcher }

// NewAiAutomationTask 创建共享运行时的定时唤醒入口。
func NewAiAutomationTask(dispatcher *aiautomation.Dispatcher) *AiAutomationTask {
	return &AiAutomationTask{dispatcher: dispatcher}
}

// Exec 唤醒派发、恢复及站内提醒，不将扫描成功当作业务执行成功。
func (t *AiAutomationTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	err := t.dispatcher.Dispatch(ctx)
	if err != nil {
		return nil, err
	}
	return []string{"AI自动化扫描完成，结果请查看执行记录"}, nil
}
