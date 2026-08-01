# core

`backend/core` 是独立 Go module，提供不依赖业务 Proto、数据库模型和具体服务实现的 Kratos 宿主运行时。业务模块负责接口和实现，Core 负责统一装配生命周期与可选运行能力。

模块路径：

```text
github.com/liujitcn/kratos-admin/backend/core
```

本仓库 Backend 通过 `replace => ./core` 使用它。Core 与 Backend 主模块独立维护版本；Core 使用 `backend/core/vX.Y.Z` 形式的模块标签，Backend 主模块使用 `backend/vX.Y.Z` 标签。

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
| `pkg/errorsx` | 六类结构化业务错误、冲突元数据、数据库错误兜底。 |
| `pkg/event` | 类型安全的进程内发布订阅。 |
| `pkg/health` | `/healthz` 和可扩展 readiness 检查。 |
| `pkg/localgrpc` | 将已注册服务暴露为 `grpc.ClientConnInterface`。仅支持 unary RPC。 |
| `pkg/openapi` | 多文档注册、冲突检查、原始 OpenAPI 和 Swagger UI。 |
| `pkg/queue` | 队列消费者注册、运行生命周期、JSON 发布和解码。 |
| `pkg/script` | 有依赖顺序的启动脚本。 |
| `pkg/sse` | SSE 流注册和 JSON 发布。 |
| `pkg/startup` | 启动钩子、失败回滚和反向清理。 |
| `pkg/static` | 静态目录和 SPA fallback。 |
| `pkg/task` | 具名任务、Cron 调度、立即执行、panic 恢复和观察器。 |

## Contributor

模块只实现需要的可选接口：

| 接口 | 贡献内容 |
| --- | --- |
| `HTTPMiddlewareContributor`、`GRPCMiddlewareContributor` | 服务端中间件。 |
| `ServerContributor` | 纳入 Kratos 生命周期的后台 Server。 |
| `OpenAPIContributor` | 具名 OpenAPI 文档。 |
| `TaskContributor` | 静态任务和 Cron 表达式。 |
| `QueueConsumerContributor` | 队列消费者。 |
| `SSEContributor` | SSE 流定义。 |
| `ScriptContributor` | 启动脚本。 |
| `StartupContributor` | 启动和清理钩子。 |
| `HealthContributor` | 外部依赖就绪检查。 |
| `StaticContributor` | 静态资源或 SPA 挂载。 |

组合根也可以使用对应的 `WithHTTPMiddlewares`、`WithGRPCMiddlewares`、`WithServers`、`WithOpenAPIDocuments`、`WithTasks`、`WithQueueConsumers`、`WithSSEStreams`、`WithScripts`、`WithStartupHooks`、`WithHealthChecks` 和 `WithStaticMounts` 直接注入。

## 运行顺序

`NewApp` 会先收集和校验模块贡献，然后创建传输与后台服务，按依赖执行启动脚本，再执行启动钩子并启动 Kratos。退出时启动钩子按相反顺序清理。

OpenAPI 默认路径为 `/api/docs/openapi/{key}`，Swagger UI 默认路径为 `/api/docs/swagger/{key}/`，可通过 `WithOpenAPIPaths` 和 `WithOpenAPIAuthorizer` 调整。

队列适配器通过 `WithQueue` 注入；任务注册表、SSE Server 和 SSE 注册表均可由宿主注入，以便与业务运行时共享。

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
