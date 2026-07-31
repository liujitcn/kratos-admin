# backend

`backend` 是 Go + Kratos 服务端，既可作为独立 HTTP/gRPC 服务运行，也可作为进程内模块挂到其他 Core 宿主。当前实现包含认证授权、系统管理、文件、任务、AI、SSE、MCP、OpenAPI、静态资源和版本化数据库迁移。

## 目录

```text
backend
├── api
│   ├── proto                         # Backend 自有 Proto
│   └── gen/go                        # 生成的 Go 协议代码
├── configs                           # 运行配置
├── core                              # 无业务 Proto 的 Kratos 宿主运行时
├── internal
│   ├── agent                         # Eino Agent 适配
│   ├── biz/{admin,app,base,event,job}
│   ├── cmd/server                    # 独立服务入口和内嵌 OpenAPI
│   ├── config                        # 配置和数据库客户端装配
│   ├── data                          # GORM 生成代码和队列适配
│   ├── server                        # HTTP、gRPC、MCP、中间件和模块注册
│   └── service/{admin,app,base}      # Proto 服务实现
├── migration/assets                  # 内嵌版本化 SQL
├── app.go                            # 对外模块门面
├── wire.go                           # Wire 声明
└── wire_gen.go                       # Wire 生成结果
```

`common.v1` 通过 `buf.build/liujitcn/kratos-common` 引入，其余协议位于 `api/proto`。

## 接口域

| Proto package | 用途 | 消费端 |
| --- | --- | --- |
| `base.v1` | 登录、OAuth、配置、文件、AI、SSE、MCP。 | admin 与 app 共用 |
| `system.admin.v1` | 系统管理、个人中心、代码生成和迁移历史。 | 管理后台 |
| `system.app.v1` | 应用端资料、地区和字典。 | 应用端 |

HTTP 路径由 Proto 的 `google.api.http` 生成；请求校验由 `buf.validate` 和全局 protovalidate 中间件执行。服务端业务错误统一使用 `core/pkg/errorsx`。

## 运行形态

独立运行入口为 `internal/cmd/server`：

```bash
go run ./internal/cmd/server --conf ./configs
```

进程内使用 `NewModule`，调用方通过 `ClientConn()` 创建与远程形态相同的生成客户端：

```go
module, cleanup, err := kratosadmin.NewModule(ctx)
if err != nil {
	return err
}
defer cleanup()

client := systemadminv1.NewBaseUserServiceClient(module.ClientConn())
```

`NewApp` 复用同一业务装配，再创建独立 HTTP 和 gRPC Server。`WithAdditionalModules` 在两种形态下使用相同的 Core Contributor 契约：独立形态由 Backend 消费中间件、OpenAPI、任务、队列、SSE、启动钩子、健康检查和静态资源；模块形态由返回的 `Runtime` 继续贡献给外层 Core 宿主。

调用方不得导入 `internal` 下的 Case、Repository、Model、Query 或 Service；跨模块调用只使用 `api/gen/go` 中的生成客户端。

## 配置

配置位于 `configs`：

| 文件 | 说明 |
| --- | --- |
| `server.yaml` | HTTP、gRPC、MCP、SSE 和中间件。 |
| `data.yaml`、`data_local.yaml` | MySQL、Redis 和队列。 |
| `auth.yaml` | JWT、白名单和认证配置。 |
| `oauth.yaml`、`oauth_local.yaml` | OAuth provider。 |
| `ai.yaml`、`ai_local.yaml` | 模型连接。未配置时 AI 客户端保持关闭。 |
| `oss.yaml` | 文件存储。 |
| `logger.yaml`、`trace.yaml`、`pprof.yaml`、`registry.yaml` | 日志、追踪、性能分析和注册中心。 |

默认数据库：

```text
root:112233@tcp(127.0.0.1:3306)/kratos_admin?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
```

`data.database` 配置单一默认数据源；`data.databases` 可按名称配置多个数据源。默认数据源保存系统数据和所有模块的 `base_migration` 记录。

## 数据库迁移

当前资源结构为“版本 → 数据库类型 → 数据源”：

```text
migration/assets/
└── v0.0.1/
    └── mysql/
        ├── default_data.up.sql        # 默认数据源
        ├── default_data.description.md
        ├── base_area.up.sql
        └── analytics/                 # 可选的命名数据源
            └── analytics.up.sql
```

`mysql` 下的直系文件属于 `default` 数据源，一级子目录名必须与 `data.databases` 中的数据源名称一致。迁移版本支持 `vX.Y.Z` 和项目生成器兼容的纯数字格式；同一目标内按文件名排序执行 `.up.sql`，`.md` 会写入迁移描述。

启动时先创建数据库客户端，再执行 `kratos-admin` 迁移模块及其依赖。每个版本、模块和数据源只记录一次；当前版本任一脚本失败会回滚并阻止启动。

`enable_migrate` 只控制 GORM `AutoMigrate` 和表注释回填，不会关闭版本化 SQL。全新数据库通常需要默认数据源设置为 `true`，以便先创建 `base_migration` 和业务表。

## 生成

| 命令 | 产物 |
| --- | --- |
| `make api` | `api/gen/go` 中的 Go、HTTP、gRPC、错误、Agent Tool 和 MCP Tool 代码。 |
| `make openapi` | `internal/cmd/server/assets/openapi.yaml`。 |
| `make ts` | 管理端 core 与 System 包的 TypeScript RPC。 |
| `make ts-app` | `frontend/app/src/rpc`。 |
| `make gorm-gen` | `internal/data/gen`。可用 `GORM_GEN_SOURCE`、`GORM_TABLE` 覆盖数据源和表。 |
| `make wire` | `wire_gen.go`。 |
| `make gen` | 依次执行以上生成和 Go 格式化。 |

生成产物不得手工修改。Proto 改动后至少重新执行 `make api openapi`，再按消费端执行 `make ts` 或 `make ts-app`。

## HTTP 入口

| 路径 | 用途 |
| --- | --- |
| `/api/v1/...` | 业务 HTTP API。 |
| `/events/{stream}` | SSE。 |
| `/mcp/{terminal}` | MCP Streamable HTTP。 |
| `/healthz` | 存活和就绪检查。 |
| `/api/docs/openapi/{key}` | 具名 OpenAPI 文档，如 `admin`。 |
| `/admin/`、`/app/` | `data/admin`、`data/app` 中存在 `index.html` 时自动挂载的 SPA。 |

Swagger 是否启用由 `server.http.enable_swagger` 控制。管理后台的 API 文档页通过 `BaseApiService` 获取已注册文档选项。

## 构建与校验

```bash
go test ./...
go vet ./...
go build -o bin/server ./internal/cmd/server
```

新增业务的顺序和迁移要求见 [docs/new-feature.md](docs/new-feature.md)。
