package queue

import (
	"context"
	"fmt"

	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-kit/sdk"
)

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queueName _const.Queue, data any) bool {
	queueID := string(queueName)
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过异步投递。
	if q == nil {
		return false
	}

	publisher, err := coreQueue.NewPublisher(q)
	if err != nil {
		log.Error(fmt.Sprintf("create queue publisher error, %s", err.Error()))
		return false
	}
	err = publisher.Publish(context.Background(), queueID, data)
	// 队列追加失败时，只记录日志，不影响主流程。
	if err != nil {
		log.Error(fmt.Sprintf("Append message error, %s", err.Error()))
		return false
	}
	return true
}
