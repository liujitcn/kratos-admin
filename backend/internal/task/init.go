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
	admin.NewTableArchiveTask,
	admin.NewBaseLogFallbackTask,
	admin.NewTableBackupTask,
	NewTask,
)

// NewTask 收集 Backend 全部任务执行器。
func NewTask(baseI18nTask *admin.BaseI18nTask, messageDispatchTask *admin.MessageDispatchTask, tableArchiveTask *admin.TableArchiveTask, baseLogFallbackTask *admin.BaseLogFallbackTask, tableBackupTask *admin.TableBackupTask) job.Tasks {
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
			Name: admin.TableArchiveTaskName,
			Exec: tableArchiveTask,
		},
		{
			Name: admin.BaseLogFallbackTaskName,
			Exec: baseLogFallbackTask,
		},
		{
			Name: admin.TableBackupTaskName,
			Exec: tableBackupTask,
		},
	}
}
