package task

import (
	"github.com/google/wire"
	adminTask "github.com/liujitcn/kratos-admin/backend/internal/task/system/admin"
	corejob "github.com/liujitcn/kratos-core/job"
)

// ProviderSet 统一注册 Backend 全部任务执行器。
var ProviderSet = wire.NewSet(adminTask.NewBaseTranslationTask, NewTask)

// NewTask 收集 Backend 全部任务执行器。
func NewTask(baseTranslationTask *adminTask.BaseTranslationTask) corejob.Tasks {
	return corejob.Tasks{
		{
			Name: adminTask.BaseTranslationTaskName,
			Exec: baseTranslationTask,
		},
	}
}
