package admin

import "github.com/google/wire"

// ProviderSet 汇总 bi.admin.v1 服务依赖。
var ProviderSet = wire.NewSet(NewBiCardService, NewBiSourceService, NewBiQueryService, NewBiQueryRecordService, NewBiDashboardService)
