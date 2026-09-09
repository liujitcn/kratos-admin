package core

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/data"
)

// ProviderSet 提供 Core 所需的 Admin 数据适配器。
var ProviderSet = wire.NewSet(
	NewAPIStoreAdapter,
	wire.Bind(new(data.APIStore), new(*APIStoreAdapter)),
	NewJobStoreAdapter,
	wire.Bind(new(data.JobStore), new(*JobStoreAdapter)),
	NewLogStoreAdapter,
	wire.Bind(new(data.LogStore), new(*LogStoreAdapter)),
	NewPermissionStoreAdapter,
	wire.Bind(new(data.PermissionStore), new(*PermissionStoreAdapter)),
	NewTransaction,
)
