package admin

import (
	"context"
	"strconv"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// MessageDispatchTaskName 是消息投递恢复任务的稳定调用目标。
	MessageDispatchTaskName = "system.admin.BaseMessageDispatch"
)

var _ cron.TaskExec = (*MessageDispatchTask)(nil)

// MessageDispatchTask 恢复到期定时消息和遗漏的 Redis Streams 投递任务。
type MessageDispatchTask struct {
	baseMessageCase *biz.BaseMessageCase
}

// NewMessageDispatchTask 创建消息投递恢复任务。
func NewMessageDispatchTask(baseMessageCase *biz.BaseMessageCase) *MessageDispatchTask {
	return &MessageDispatchTask{baseMessageCase: baseMessageCase}
}

// Exec 扫描并重新入队需要继续处理的消息投递任务。
func (t *MessageDispatchTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	count, err := t.baseMessageCase.RecoverPendingDispatches(ctx)
	if err != nil {
		return nil, i18n.WrapMessageError("system.base.job.error.message_dispatch_failed", err)
	}
	var cleaned int
	cleaned, err = t.baseMessageCase.CleanupExpiredDeliveries(ctx)
	if err != nil {
		return nil, i18n.WrapMessageError("system.base.job.error.message_dispatch_failed", err)
	}
	return []string{i18n.EncodeMessage("system.base.job.result.message_dispatch_recovered", map[string]string{
		"Count": strconv.Itoa(count), "Cleaned": strconv.Itoa(cleaned),
	})}, nil
}
