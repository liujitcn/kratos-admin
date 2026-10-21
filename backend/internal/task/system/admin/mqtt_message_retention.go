package admin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// MqttMessageRetentionTaskName 是 MQTT 消息保留清理任务的稳定调用目标。
	MqttMessageRetentionTaskName = "system.admin.MqttMessageRetention"
	mqttMessageRetentionDays     = 30
	mqttMessageRetentionBatch    = 1000
)

var _ cron.TaskExec = (*MqttMessageRetentionTask)(nil)

// MqttMessageRetentionTask 清理超过默认保留周期的 MQTT 消息记录。
type MqttMessageRetentionTask struct {
	mqttMessageRepository *data.MqttMessageRepository
}

// NewMqttMessageRetentionTask 创建 MQTT 消息保留清理任务。
func NewMqttMessageRetentionTask(mqttMessageRepository *data.MqttMessageRepository) *MqttMessageRetentionTask {
	return &MqttMessageRetentionTask{mqttMessageRepository: mqttMessageRepository}
}

// Exec 批量删除超过 30 天的 MQTT 消息记录。
func (t *MqttMessageRetentionTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	cutoff := time.Now().AddDate(0, 0, -mqttMessageRetentionDays)
	query := t.mqttMessageRepository.Query(ctx).MqttMessage
	deleted := 0
	for {
		items, err := t.mqttMessageRepository.List(ctx,
			repository.Where(query.CreatedAt.Lt(cutoff)),
			repository.Order(query.ID.Asc()),
			repository.Limit(mqttMessageRetentionBatch),
		)
		if err != nil {
			return nil, fmt.Errorf("查询过期 MQTT 消息失败: %w", err)
		}
		if len(items) == 0 {
			break
		}
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		if err = t.mqttMessageRepository.DeleteByIDs(ctx, ids); err != nil {
			return nil, fmt.Errorf("删除过期 MQTT 消息失败: %w", err)
		}
		deleted += len(ids)
		if len(ids) < mqttMessageRetentionBatch {
			break
		}
	}
	return []string{fmt.Sprintf("MQTT 消息保留清理完成，删除 %s 条记录", strconv.Itoa(deleted))}, nil
}

// Task 返回交由 base_job 调度的任务定义。
func (t *MqttMessageRetentionTask) Task() cron.Task {
	return cron.Task{Name: MqttMessageRetentionTaskName, Exec: t}
}
