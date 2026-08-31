package task

import (
	"github.com/google/wire"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/task/system/admin"
	"github.com/liujitcn/kratos-core/job"
)

// ProviderSet 统一注册 Backend 全部任务执行器。
var ProviderSet = wire.NewSet(
	biz.ProviderSet,
	admin.NewBaseI18nTask,
	admin.NewMessageDispatchTask,
	admin.NewAuditRetentionTask,
	admin.NewBackupTask,
	NewTask,
)

// NewTask 收集 Backend 全部任务执行器。
func NewTask(baseI18nTask *admin.BaseI18nTask, messageDispatchTask *admin.MessageDispatchTask, auditRetentionTask *admin.AuditRetentionTask, backupTask *admin.BackupTask) job.Tasks {
	return job.Tasks{
		{
			Name: admin.BaseI18nTaskName,
			Exec: baseI18nTask,
		},
		{
			Name: admin.MessageDispatchTaskName,
			Exec: messageDispatchTask,
		},
		{
			Name: admin.AuditRetentionTaskName,
			Exec: auditRetentionTask,
		},
		{
			Name: admin.BackupTaskName,
			Exec: backupTask,
		},
	}
}
