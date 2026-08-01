# core

`backend/core` 是独立 Go module，提供不依赖业务 Proto、数据库模型和具体服务实现的 Kratos 宿主运行时、公共协议和可复用错误包。业务模块负责接口和实现，Core 负责统一装配生命周期、可选运行能力和项目文档收集。

Core 不创建数据库客户端，也不负责数据库迁移；`pkg/errorsx` 仅依赖 GORM 和 MySQL 的错误类型，用于在不同上层项目中统一分类底层数据库错误。

模块路径：

```text
github.com/liujitcn/kratos-admin/backend/core
```

本仓库 Backend 通过 `replace => ./core` 使用它。Core 与 Backend 主模块独立维护版本；Core 使用 `backend/core/vX.Y.Z` 形式的模块标签，Backend 主模块使用 `backend/vX.Y.Z` 标签。

Core 同时维护通用 `common/v1` Proto，源码位于 `api/proto/common/v1`，发布模块为 `buf.build/liujitcn/kratos-admin-core`。修改通用协议后，在 Core 目录执行 `make api` 生成 Go 代码，再发布新的 Buf 提交供 Admin、Shop 和其他消费者锁定使用。

## 模块契约

业务模块实现 `core.Module`：

```go
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*http.Server)
	RegisterMCP(*mcpserver.Server)
}
```

只使用部分传输协议时可嵌入 `core.ModuleAdapter`。Core 不扫描包、不依赖 `init()` 注册；模块必须通过 `core.WithModules(...)` 显式加入组合根。

```go
app, cleanup, err := core.NewApp(ctx, core.WithModules(orderModule))
```

## 公共包

| 包 | 已实现能力 |
| --- | --- |
| `pkg/errorsx` | Kratos 结构化业务错误、冲突 metadata、GORM `ErrRecordNotFound` 和 MySQL 唯一键错误分类。 |
| `pkg/projectdoc` | 项目文档目录解析、稳定文档标识和文档贡献类型。 |
| `pkg/event` | 类型安全的进程内发布订阅。 |
| `pkg/health` | `/healthz` 和可扩展 readiness 检查。 |
| `pkg/localgrpc` | 将已注册服务暴露为 `grpc.ClientConnInterface`，支持 unary 和 streaming RPC。 |
| `pkg/openapi` | 多文档注册、冲突检查、原始 OpenAPI 和 Swagger UI。 |
| `pkg/queue` | 队列消费者注册、运行生命周期、JSON 发布和解码。 |
| `pkg/script` | 有依赖顺序的启动脚本。 |
| `pkg/sse` | SSE 流注册和 JSON 发布。 |
| `pkg/startup` | 启动钩子、失败回滚和反向清理。 |
| `pkg/static` | 静态目录和 SPA fallback。 |
| `pkg/task` | 具名任务、Cron 调度、立即执行、panic 恢复和观察器。 |

## 模块可选接口

模块只实现需要的可选接口：

| 接口 | 贡献内容 |
| --- | --- |
| `HTTPMiddlewareContributor`、`GRPCMiddlewareContributor` | 服务端中间件。 |
| `ServerContributor` | 纳入 Kratos 生命周期的后台 Server。 |
| `OpenAPIContributor` | 具名 OpenAPI 文档。 |
| `ProjectDocumentContributor` | 项目文档。 |
| `TaskContributor` | 静态任务和 Cron 表达式。 |
| `QueueConsumerContributor` | 队列消费者。 |
| `SSEContributor` | SSE 流定义。 |
| `SSEPublisherAware` | 接收宿主创建的 SSE 发布器。 |
| `ScriptContributor` | 启动脚本。 |
| `StartupContributor` | 启动和清理钩子。 |
| `HealthContributor` | 外部依赖就绪检查。 |
| `StaticContributor` | 静态资源或 SPA 挂载。 |

Core 不提供数据库客户端或迁移 Runner；数据库错误分类和业务错误包装由 `pkg/errorsx` 统一提供，数据库连接与迁移仍由 Backend 或其他上层宿主负责。

组合根的 `With...` 选项按 Core 装配生命周期排列如下，便于查阅和组织配置；这不是强制的调用顺序：

1. `WithModules`
2. `WithOpenAPIRegistry`、`WithOpenAPIDocuments`、`WithOpenAPIPaths`、`WithOpenAPIAuthorizer`
3. `WithTaskRegistry`、`WithTasks`、`WithTaskObserver`
4. `WithSSERegistry`、`WithSSEStreams`
5. `WithHealthChecks`
6. `WithScripts`、`WithStartupHooks`
7. `WithSSEServer`、`WithSSEServerListener`
8. `WithGRPCMiddlewares`、`WithHTTPMiddlewares`
9. `WithStaticMounts`
10. `WithServers`
11. `WithQueue`、`WithQueueConsumers`

## 运行顺序

`NewApp` 的装配和启动顺序如下：

1. 应用先应用全部 `With...` 选项，收集外部模块和组合根直接提供的能力。
2. 按顺序准备 OpenAPI、任务、SSE、健康检查和启动管理器注册表。
3. 创建 MCP 和 SSE 服务，并把 SSE 发布器注入需要它的模块。
4. 依次创建 gRPC 和 HTTP 服务。HTTP 服务按健康检查、OpenAPI、SSE、进程内 MCP、模块传输服务、静态资源的顺序注册路由，避免静态前缀覆盖固定入口。
5. 组装后台 Server：组合根和模块提供的 Server、队列运行时、任务调度器、SSE Server、外置 MCP Server，以及 gRPC/HTTP Server。
6. 在创建 `bootstrap.NewApp` 之前启动启动管理器：启动脚本先按依赖顺序执行，再按注册顺序执行启动钩子。任一启动钩子失败时，已经启动的钩子按相反顺序清理。
7. `NewApp` 返回 Kratos App；传输服务的实际运行由调用方启动。应用退出时，启动钩子按启动顺序的相反方向清理。

OpenAPI 默认路径为 `/api/docs/openapi/{key}`，Swagger UI 默认路径为 `/api/docs/swagger/{key}/`，可通过 `WithOpenAPIPaths` 和 `WithOpenAPIAuthorizer` 调整。

队列适配器通过 `WithQueue` 注入；任务注册表、SSE Server 和 SSE 注册表均可由宿主注入，以便与业务运行时共享。注入独立 HTTP SSE Server 时，必须同时通过 `WithSSEServerListener` 提供对应监听器，由 Core 统一负责关闭；进程内 SSE Handler 不需要监听器。模块提供队列消费者时必须同时注入队列适配器，否则 `NewApp` 会返回错误。

## 最小配置

Core 使用 `kratos-kit/bootstrap` 配置。至少需要声明启用的 HTTP 或 gRPC Server：

```yaml
server:
  http:
    addr: :7001
    middleware:
      enable_recovery: true
      enable_validate: true
  grpc:
    addr: 0.0.0.0:6001
    middleware:
      enable_recovery: true
```

认证、鉴权和业务日志由宿主或模块中间件贡献，不是 Core 的固定策略。

## 校验

```bash
make fmt
make test
make vet
```
