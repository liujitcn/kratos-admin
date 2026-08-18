package task

import (
	"github.com/google/wire"
	adminTask "github.com/liujitcn/kratos-admin/backend/internal/task/system/admin"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

// ProviderSet 统一注册 Backend 全部任务执行器。
var ProviderSet = wire.NewSet(adminTask.NewBaseTranslationTask, NewTask)

func NewTask(baseTranslationTask *adminTask.BaseTranslationTask) []*cron.Task {
	return []*cron.Task{
		{
			Name: adminTask.BaseTranslationTaskName,
			Exec: baseTranslationTask,
		},
	}
}
