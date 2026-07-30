package job

import (
	"github.com/google/wire"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
)

// ProviderSet 注册定时任务模块依赖。
var ProviderSet = wire.NewSet(
	coreTask.NewRegistry,
	NewCronServer,
)
