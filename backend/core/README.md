# core

`core` 是不包含 Proto、业务实现和数据库访问的 Kratos 基础运行时。它负责配置加载、应用生命周期、
HTTP/gRPC/MCP/SSE、框架拦截器、OpenAPI/Swagger、定时任务、队列、启动脚本、健康检查、
静态资源和外部模块装配。数据库模型、权限策略、业务队列主题和实际迁移资源由接入项目维护。

接口协议、生成代码及其具体实现全部由接入模块维护，Core 只负责挂载模块提供的服务。

模块路径为 `github.com/liujitcn/kratos-admin/backend/core`。仓库内的 Backend 通过本地 `replace`
引用；发布独立版本时使用 `backend/core/vX.Y.Z` tag，使用方改为对应版本后不再需要 `replace`。

```text
core
├── app.go、module.go、options.go  # 宿主入口与模块装配
└── pkg                           # 可供接入项目引用的公共能力包（含 errorsx）
```

## 快速接入

### 1. 添加依赖

发布 `backend/core/vX.Y.Z` tag 后，接入项目使用对应模块版本：

```bash
go get github.com/liujitcn/kratos-admin/backend/core@v0.1.0
```

本地联调可先使用 `replace`，路径必须指向实际的 Core 目录：

```go
require github.com/liujitcn/kratos-admin/backend/core v0.0.0

replace github.com/liujitcn/kratos-admin/backend/core v0.0.0 => ../kratos-admin/backend/core
```

本仓库 Backend 的 `go.mod` 使用 `=> ./core`，并直接复用 Core 的模块契约、OpenAPI、
任务、队列、SSE 和事件能力。发布 Core 后删除 `replace`，将 `require` 替换为实际版本。

### 2. 实现并注册模块

接入项目自己维护 Proto、生成代码和服务实现。模块嵌入 `core.ModuleAdapter`，
只实现需要的传输层注册方法：

```go
package order

import (
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-admin/backend/core"
	orderv1 "github.com/example/order/api/gen/go/order/v1"
)

type Module struct {
	core.ModuleAdapter
	service orderv1.OrderServiceHTTPServer
}

func NewModule(service orderv1.OrderServiceHTTPServer) *Module {
	return &Module{service: service}
}

func (module *Module) RegisterHTTP(server *http.Server) {
	orderv1.RegisterOrderServiceHTTPServer(server, module.service)
}
```

不需要数据库、生成接口或具体业务的能力可直接通过 `core.Option` 注入。
完整宿主入口如下：

```go
package main

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/core"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	_ "github.com/liujitcn/kratos-kit/logger/zap"
)

func main() {
	module := newOrderModule()
	err := core.Run(
		context.Background(),
		&bootstrapConfigv1.AppInfo{
			Project: "order",
			AppId:   "server",
			Version: "dev",
		},
		core.WithModules(module),
	)
	if err != nil {
		panic(err)
	}
}
```

`newOrderModule` 由接入项目的手工组合根或 Wire 提供。Core 不扫描模块，所有服务和扩展都必须在入口显式注册。

### 3. 准备配置并启动

Core 通过 `kratos-kit/bootstrap` 读取配置。最小 `configs/server.yaml` 示例：

```yaml
server:
  http:
    addr: :7001
    middleware:
      enable_recovery: true
      enable_tracing: true
      enable_validate: true
      enable_metadata: true
  grpc:
    addr: 0.0.0.0:6001
    middleware:
      enable_recovery: true
      enable_tracing: true
      enable_validate: true
      enable_metadata: true
```

```bash
go run ./cmd/server --conf ./configs
```

需要认证、鉴权或业务日志时，通过 `HTTPMiddlewareContributor` /
`GRPCMiddlewareContributor` 或 `core.WithHTTPMiddlewares` / `core.WithGRPCMiddlewares` 注入；
Core 只负责框架拦截器和扩展中间件的统一装配。

## 基础能力

| 包 | 职责 |
| --- | --- |
| `pkg/errorsx` | 稳定的 Kratos 业务错误构造、兜底包装和数据库错误类型识别。 |
| `pkg/openapi` | 文档注册、HTTP 操作冲突检查、原始文档和 Swagger UI。 |
| `pkg/localgrpc` | 将已注册服务暴露为生成客户端可用的进程内 gRPC 连接。 |
| `pkg/task` | 任务注册、静态 Cron 调度、立即执行、panic 恢复和执行观察器。 |
| `pkg/queue` | 队列生命周期、消费者注册、JSON 发布与载荷解码。 |
| `pkg/sse` | SSE 流注册、解析和 JSON 事件发布。 |
| `pkg/script` | 启动脚本注册、依赖排序和执行，数据库迁移通过适配器接入。 |
| `pkg/startup` | 服务启动前的初始化钩子、失败回滚和退出清理。 |
| `pkg/health` | 存活检查和外部依赖就绪检查聚合。 |
| `pkg/static` | 静态目录和 SPA fallback 挂载。 |
| `pkg/event` | 类型安全的进程内发布订阅。 |

## 扩展能力

模块按需实现可选 Contributor，不需要为未使用的能力增加空方法：

| Contributor | 贡献内容 |
| --- | --- |
| `HTTPMiddlewareContributor` / `GRPCMiddlewareContributor` | 认证、鉴权等外部拦截器。 |
| `OpenAPIContributor` | 内嵌的 OpenAPI 文档。 |
| `TaskContributor` | 具名任务和可选 Cron 表达式。 |
| `QueueConsumerContributor` | 队列主题与消费者。 |
| `SSEContributor` | SSE 流定义。 |
| `ScriptContributor` | 数据库迁移适配器等启动脚本。 |
| `StartupContributor` | 服务启动前的初始化和退出清理钩子。 |
| `HealthContributor` | 数据库、缓存等 readiness 检查。 |
| `StaticContributor` | 静态资源或 SPA。 |
| `ServerContributor` | 其他 Kratos 后台 Server。 |

OpenAPI 文档启用后分别暴露为 `/api/docs/openapi/{key}`，Swagger UI 位于
`/api/docs/swagger/{key}/`。业务模块负责生成协议和文档，并通过 `go:embed` 向 Core 宿主提供文档内容。

队列适配器通过 `WithQueue` 注入；动态任务可通过 `WithTaskRegistry` 与外部任务仓储共享注册表；
SSE 订阅身份解析和业务发布器由外部模块持有注入的 `SSE Registry` 与 `SSE Server` 完成。
文档鉴权通过 `WithOpenAPIAuthorizer` 注入。组合根也可以使用 `WithTasks`、
`WithQueueConsumers`、`WithSSEStreams`、`WithScripts`、`WithStartupHooks`、
`WithHealthChecks`、`WithStaticMounts` 和 `WithServers` 直接追加能力。

启动顺序固定为：收集并校验贡献 → 构造后台和传输服务 → 按依赖执行启动脚本 → 执行启动
钩子 → Kratos 启动全部 Server。退出时启动钩子按相反顺序清理。

## 验证

```bash
make lint
make fmt
make test
make vet
```
