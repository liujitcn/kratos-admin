package task

import (
	"github.com/google/wire"
	adminTask "github.com/liujitcn/kratos-admin/backend/internal/task/system/admin"
)

// BaseTranslationTask 是系统管理翻译任务的类型别名。
type BaseTranslationTask = adminTask.BaseTranslationTask

// ProviderSet 统一注册 Backend 全部任务执行器。
var ProviderSet = wire.NewSet(adminTask.NewBaseTranslationTask)
