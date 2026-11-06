package bi

import "github.com/google/wire"

// ProviderSet 汇总 bi.admin.v1 服务注册依赖。
var ProviderSet = wire.NewSet(wire.Struct(new(Services), "*"))
