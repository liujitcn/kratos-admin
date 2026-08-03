# backend

`backend` 是 Go + Kratos 服务端，既可作为独立 HTTP/gRPC 服务运行，也可作为进程内模块挂到其他 Core 宿主。当前实现包含认证授权、系统管理、文件、任务、AI、SSE、MCP、OpenAPI、静态资源和版本化数据库迁移。

## 目录

```text
backend
├── api
│   ├── proto                         # Backend 自有 Proto
│   ├── gen/go                        # 生成的 Go 协议代码
│   ├── buf.admin*.typescript.gen.yaml
│   ├── buf.app*.typescript.gen.yaml
│   └── buf.taro-app*.typescript.gen.yaml # admin/uni-app/Taro 的 RPC 规则
├── configs                           # 运行配置
├── core                              # 通用运行时、公共 Proto 和公共包
│   ├── api/proto/common/v1            # 唯一通用 Proto 源码
│   └── pkg/{errorsx,projectdoc}      # 公共错误与文档能力
├── internal
│   ├── agent                         # Eino Agent 适配
│   ├── biz/{admin,app,base,event,job}
│   ├── cmd/server                    # 独立服务入口和内嵌 OpenAPI
│   ├── config                        # 配置和数据库客户端装配
│   ├── data                          # GORM 生成代码和队列适配
│   ├── docs                          # Backend 内嵌项目文档目录
│   ├── server                        # HTTP、gRPC、MCP、中间件和模块注册
│   └── service/{admin,app,base}      # Proto 服务实现
├── migration/assets                  # 内嵌版本化 SQL
├── module                            # Backend 宿主契约（迁移、用户、AI、运行时）
├── bootstrap.go                      # 独立应用生命周期
├── module.go                         # Backend Runtime 与模块装配
├── option.go                         # 数据库、迁移和文档装配选项
├── wire.go                           # Wire 声明
└── wire_gen.go                       # Wire 生成结果
```

`common.v1` 通过 `buf.build/liujitcn/kratos-admin-core` 引入，其余协议位于 `api/proto`。

## 接口域

| Proto package | 用途 | 消费端 |
| --- | --- | --- |
| `base.v1` | 登录、OAuth、配置、文件、AI、SSE、MCP。 | admin、uni-app 与 Taro 共用 |
| `system.admin.v1` | 系统管理、个人中心、代码生成和迁移历史。 | 管理后台 |
| `system.app.v1` | 应用端资料、地区、字典和移动菜单。 | 应用端 |

HTTP 路径由 Proto 的 `google.api.http` 生成；请求校验由 `buf.validate` 和全局 protovalidate 中间件执行。服务端及其他上层模块统一使用 `backend/core/pkg/errorsx`，其中包含结构化业务错误和常见 GORM/MySQL 错误分类。

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

`NewApp` 复用同一业务装配，再创建独立 HTTP 和 gRPC Server。`WithAdditionalModules` 在两种形态下使用相同的 Core Contributor 契约：独立形态由 Backend 消费中间件、OpenAPI、任务、队列、SSE、迁移、启动钩子、健康检查和静态资源；模块形态由返回的 `Runtime` 继续贡献给外层 Core 宿主。

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

启动时先创建全部数据库客户端，再以每个已注册 Contributor 的模块名称作为根模块，统一执行 `kratos-admin` 和外部 `backend/module.MigrationContributor` 提供的迁移；Runner 会递归处理模块依赖。外部模块只贡献资源和依赖声明，不自行创建数据库客户端或调用迁移 Runner。每个版本、模块和数据源只记录一次；当前版本任一脚本失败会回滚并阻止启动。

`enable_migrate` 只控制 GORM `AutoMigrate` 和表注释回填，不会关闭版本化 SQL。全新数据库通常需要默认数据源设置为 `true`，以便先创建 `base_migration` 和业务表。

## 生成

| 命令 | 产物 |
| --- | --- |
| `make api` | `api/gen/go` 中的 Go、HTTP、gRPC、错误、Agent Tool 和 MCP Tool 代码。 |
| `make -C core api` | Core 的 `common/v1` Go 协议代码。 |
| `make openapi` | `internal/cmd/server/assets/openapi.yaml`。 |
| `make ts` | 管理端 core 与 System 包的 TypeScript RPC。 |
| `make ts-app` | uni-app core 与 system 包的 TypeScript RPC。 |
| `make ts-taro-app` | Taro React core 与 system 包的 TypeScript RPC。 |
| `make project-docs` | `internal/docs/assets/docs.json` 和 `internal/docs/docs.go`，收集三层范围内的 `README.md` 和 `docs` Markdown。 |
| `make gorm-gen` | `internal/data/gen`。默认读取 `configs/data_local.yaml`，可用 `GORM_GEN_CONFIG`、`GORM_GEN_DATABASE`、`GORM_TABLE` 覆盖配置、数据源和表。 |
| `make wire` | `wire_gen.go`。 |
| `make gen` | 依次执行以上生成和 Go 格式化。 |

所有前端 RPC 的 Buf 模板都归属 `api` 契约目录。`make ts` 生成管理端 RPC，`make ts-app` 生成 uni-app RPC，`make ts-taro-app` 生成 React/Taro RPC；每条命令分别输出到对应 workspace 的 core 与 system 包。生成产物不得手工修改。Proto 改动后至少重新执行 `make api openapi`，再按实际消费端运行对应 TypeScript 生成命令。

## 国际化

后端支持的语言由 `internal/i18n/locales` 自动发现。`core/pkg/locale` 负责规范化 `Accept-Language`，locale 中间件将语言区域写入请求上下文，并在响应边界本地化结构化错误；动态菜单、字典和字典项的审核译文由版本化翻译表按请求语言解析，缺少当前语言译文时回退主语言。

新增语言时，在 `internal/i18n/locales` 增加错误目录，并同步三个前端 workspace 的语言包和代码生成语言目录，然后从仓库根目录执行 `make i18n-sync`。脚本生成 `core/pkg/locale/manifest.json` 和前端注册产物，不需要修改 Go/TypeScript 源码。若新部署需要默认插入语言记录，执行 `make i18n-sync I18N_MIGRATION_VERSION=vX.Y.Z` 生成版本化迁移；`base_language` 的启用状态、排序和主语言标记仍由数据库维护。

```bash
make i18n-sync
make i18n-check
make i18n-draft
I18N_WRITE=1 make i18n-draft
```

草稿命令的 Google V1 仅用于显式生成可审核的非主语言草稿，不进入普通业务请求链路；`make i18n-locales` 支持在线和 `I18N_OFFLINE=1` 离线生成；关闭 Provider 不影响主语言回退和已审核译文。

动态资源的主语言由 `base_language.is_primary` 配置。创建或更新菜单、字典、字典项和系统配置时，后端按请求 `Accept-Language` 将输入文本转换为主语言写入主表；请求语言不是主语言时，原文写入对应翻译表，其他已启用非主语言也只保存在翻译表。

## 项目文档

Admin 内置项目文档与 OpenAPI/Swagger 固定使用项目标识 `admin` 和展示名称
“系统管理”，避免 Backend 被其他宿主组合时继承宿主身份并与外部模块冲突。
独立启动入口复用同一组项目身份；构建期生成物不保存项目身份，服务加载后才生成稳定文档 ID。
正常执行 `make run`、`make build` 或 `make gen` 时会自动刷新内嵌目录，也可以
单独执行：

```bash
go install github.com/liujitcn/kratos-kit/cmd/project-docs@latest
make project-docs
```

`make project-docs` 在仓库根目录零参数调用 PATH 中已安装的 `project-docs`
二进制，从项目根目录扫描相对路径不超过三段的文件，只收集精确命名的
`README.md`，以及任意 `docs` 目录中的 Markdown。命令输出递归目录树
`internal/docs/assets/docs.json`，并自动生成
`internal/docs/docs.go`。文档节点只保存路径、正文和源文件的
RFC3339 更新时间；服务装配时使用 Admin 内置项目身份补齐项目标识、展示名称和稳定
编号。`make wire` 会先执行该命令，因此生成目录被删除后也能自动恢复。

引用 Backend 的宿主可通过 `WithProjectDocuments` 注入自己的生成文档。通过 `WithAdditionalModules` 注册的外部模块实现 `core.ProjectDocumentContributor` 后会被自动汇总；`Runtime` 本身也实现该接口，因此 Backend 被更外层宿主引用时会继续贡献已经聚合的文档。文档类型和目录解析位于 `core/pkg/projectdoc`，数据库迁移仍由 Backend 宿主负责。

## HTTP 入口

| 路径 | 用途 |
| --- | --- |
| `/api/v1/...` | 业务 HTTP API。 |
| `/events/{stream}` | SSE。 |
| `/mcp/{terminal}` | MCP Streamable HTTP。 |
| `/healthz` | 存活和就绪检查。 |
| `/api/docs/openapi/{key}` | 具名 OpenAPI 文档，如 `admin`。 |
| `/api/v1/admin/project-document/tree` | 按项目和目录递归组织的项目文档树。 |
| `/api/v1/admin/project-document/{id}` | 按稳定 ID 查询 Markdown 文档详情。 |
| `/admin/`、`/app/`、`/taro-app/`、`/uni-app/` | `data/<目录名>` 中存在 `index.html` 时按目录名自动挂载的 SPA。 |

Swagger 是否启用由 `server.http.enable_swagger` 控制。管理后台的 API 文档页通过 `BaseApiService` 获取已注册文档选项。

## 构建与校验

```bash
make project-docs
go test ./...
go vet ./...
make build
```

新增业务的顺序和迁移要求见 [docs/new-feature.md](docs/new-feature.md)。
