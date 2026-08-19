package task

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/task/system/admin"
	"github.com/liujitcn/kratos-core/job"
)

// ProviderSet 统一注册 Backend 全部任务执行器。
var ProviderSet = wire.NewSet(
	biz.ProviderSet,
	admin.NewBaseTranslationTask,
	NewTask,
)

// NewTask 收集 Backend 全部任务执行器。
func NewTask(baseTranslationTask *admin.BaseTranslationTask) job.Tasks {
	return job.Tasks{
		{
			Name: admin.BaseTranslationTaskName,
			Exec: baseTranslationTask,
		},
	}
}
