package core

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	coredata "github.com/liujitcn/kratos-core/data"
)

// ProviderSet 提供 Core 所需的 Admin 数据适配器。
var ProviderSet = wire.NewSet(
	data.ProviderSet,
	wire.Bind(new(data.QueryProvider), new(*data.Data)),
	NewAPIStoreAdapter,
	wire.Bind(new(coredata.APIStore), new(*APIStoreAdapter)),
	NewJobStoreAdapter,
	wire.Bind(new(coredata.JobStore), new(*JobStoreAdapter)),
	NewLogStoreAdapter,
	wire.Bind(new(coredata.LogStore), new(*LogStoreAdapter)),
	NewPermissionStoreAdapter,
	wire.Bind(new(coredata.PermissionStore), new(*PermissionStoreAdapter)),
	wire.Bind(new(coredata.Transaction), new(*data.Data)),
)
