package admin

import (
	"context"
	"fmt"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"

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
		return nil, err
	}
	var cleaned int
	cleaned, err = t.baseMessageCase.CleanupExpiredDeliveries(ctx)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("恢复消息投递任务 %d 条，清理过期收件 %d 条", count, cleaned)}, nil
}
