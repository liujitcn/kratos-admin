package admin

import "github.com/google/wire"

// ProviderSet 汇总 BI 管理端业务依赖。
var ProviderSet = wire.NewSet(NewBiCardCase, NewBiSourceCase, NewBiQueryCase, NewBiQueryRecordCase, NewBiDashboardCase)
