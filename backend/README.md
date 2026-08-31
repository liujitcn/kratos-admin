# backend

Backend 同时提供消息分类、站内信管理、用户收件箱、Redis 投递恢复、后台工作台统计、文件资产元数据、登录来源策略、会话撤销、审计事件异步落库、日志归档和受控数据库备份任务。安全、消息和开放授权默认数据统一由 `v0.0.1` 初始化迁移提供。

`backend` 保留 API 契约、Go 生成接口、Service 实现、Biz 业务层，以及任务调度、HTTP/gRPC/MCP/AI 注册和必要的数据访问闭包；进程入口位于 `internal/cmd/server`。根包通过 `ProviderSet`、`NewModuleResources`、`NewModules`、`NewTasks`、`NewStreams` 和 `NewQueueConsumers` 提供可被外部 Core 宿主复用的公共边界，`internal/module` 仅承载内部实现。AI Runtime 实现在 `internal/biz`，对外复用入口为 `pkg/agent`；业务模块通过 `pkg/notification.Publish` 发布站内信，由内部事务和 Dispatch 恢复链路负责最终投递。开放授权客户端使用单表 JSON operation 白名单并绑定租户，公开端点签发客户端 Bearer Token；HTTP middleware 分别校验租户、状态、IP 白名单和 API 范围，HTTP 加解密 Filter 在请求绑定前解密客户端数据并在成功响应后加密。登录认证支持 TOTP 多因素认证、一次性恢复码，以及全局和租户/用户定向登录来源策略。

## 目录

```text
backend
├── internal/cmd/server             # Admin 独立启动入口与 Wire 组合根
├── api
│   ├── proto                         # Proto 契约
│   └── gen/go                        # Buf 生成的 Go 接口、HTTP、gRPC 和工具代码
├── internal/biz                      # 业务 Case、DTO、代码生成和辅助领域代码
├── bootstrap.go                      # 对外 ProviderSet 和模块/任务/SSE/队列/资源入口
├── internal/module                   # Admin 到 kratos-core 的内部模块适配和资源实现
│   ├── module.go                     # Core Module 协议注册
│   ├── resources.go                  # module.Module 静态资源
│   ├── init.go                        # Admin 模块 ProviderSet
│   └── wire.go / wire_gen.go          # 公共入口使用的内部依赖装配
├── pkg/agent                         # 对外复用的 AI Runtime、模型和工具 API
├── pkg/notification                  # 对外复用的站内信发布 API
├── internal/task                     # 异步任务与定时任务执行器
├── internal/server                   # 服务拦截器和 API 模块注册适配
│   └── middleware/{oauth,logstream}  # 按业务分组的 HTTP/gRPC 服务拦截器
├── internal/data/gen                 # GORM 生成的模型、查询和仓储
├── internal/service                  # Proto Service 实现
├── internal/const                    # 业务常量
├── internal/i18n/assets              # 业务语言资源
└── migration                         # 代码生成业务使用的迁移资源
```

## 常用流程

进入 `backend` 目录后，先通过帮助查看全部目标、参数默认值和覆盖示例：

```bash
make help
```

首次开发安装 Buf、protoc、Wire、gorm-gen、goimports 和检查工具：

```bash
make init
```

日常启动使用：

```bash
make run
```

`make run` 会按“protobuf Go -> OpenAPI -> 独立入口 Wire -> 启动服务”的顺序刷新必要产物。确认生成产物没有变化时，可跳过生成直接启动：

```bash
make run-only
```

默认配置目录为 `./configs`，默认运行环境为 `dev`。基础配置使用 `<name>.yaml`，环境差异使用 `<name>.<env>.yaml`；环境文件存在时在基础配置之后加载，不存在时回退基础配置。可以覆盖配置目录、运行环境或追加启动参数：

多因素认证方式由系统配置 `securityMfaMethod` 选择，当前支持 `totp` 和 `webauthn`。运行时 MFA 参数通过 `mfa.yaml` 或环境覆盖文件 `mfa.dev.yaml` 的 `mfa` 节点加载；启用 TOTP 绑定前，需要配置 `mfa.encryption_key`（base64 编码的 32 字节密钥）。管理端和应用端禁用 TOTP 需要当前密码和动态口令或恢复码，禁用 WebAuthn 需要当前密码和一次 Passkey 或恢复码验证。生产环境应从 KMS、Vault 或 Secret Manager 注入配置文件或其挂载内容，不要把真实密钥写入仓库或数据库。完整字段以 `kratos-kit/api/proto/config/v1/mfa.proto` 为准。

```bash
make run-only CONF=/path/to/configs
make run-only APP_ENV=prod
make run-only RUN_ARGS='--help'
```

例如 `APP_ENV=dev` 会加载 `data.yaml` 后再加载 `data.dev.yaml`，同时忽略 `data.prod.yaml`。本地开发配置统一保存在 `*.dev.yaml`，这类文件默认不纳入 Git。

独立入口注入 `kratoscore.ProviderSet` 与内部模块 ProviderSet；Core 负责统一创建和管理 HTTP、gRPC、MCP、SSE、队列与定时任务运行时。Admin 注册六张完整审计日志模型并负责自动迁移；Core 异步写入 API/策略日志，Admin 异步写入登录、操作、数据访问和权限日志。

## 修改后执行

| 修改场景 | 命令 | 说明 |
| --- | --- | --- |
| Proto 契约 | `make api openapi` | 生成 Backend protobuf Go 和 OpenAPI 源文档；前端 TypeScript RPC 使用仓库根目录的 `make -C ../frontend ts`，也可按端执行 `ts-admin`、`ts-uni-app` 或 `ts-taro-app`。 |
| 数据库表结构 | `make gorm-gen` | 先更新开发库，再按 `GORM_GEN_CONFIG`、`GORM_GEN_DATABASE` 和 `GORM_TABLE` 生成。 |
| ProviderSet 或构造参数 | `make public-wire wire` | 分别刷新公共入口内部装配和独立服务入口。 |
| README 或 docs | `make -C .. project-docs` | 从仓库根目录收集 Markdown，并生成各语言文档目录。 |
| 语言包或国际化资源 | `make -C .. i18n` | 国际化属于仓库根目录公共命令。 |
| Go import 别名 | `make cli fmt` | `cli` 安装 `kratos-kit/cmd/normalize-go-imports`，`fmt` 运行它并使用 `goimports` 格式化。 |
| 多类后端生成源同时变化 | `make gen` | 依次执行 GORM、接口、OpenAPI、Wire 和格式化；需要可访问开发数据库。 |

所有生成产物都必须通过上述命令刷新，不能手工修改。

Go import 别名规范化命令由 `kratos-kit/cmd/normalize-go-imports` 提供，Admin 不保留本地副本；先安装命令，再执行格式化：

```bash
make cli
make fmt
```


## 检查

提交前执行完整 Backend 检查：

```bash
make check
```

`make check` 依次运行 `make lint` 和 `make test`，不会自动格式化代码。全仓检查（含国际化）请在仓库根目录执行 `make check`。需要格式化时先执行：

```bash
make fmt
```

也可以单独运行某一项：

```bash
make lint
make test
```

## 构建

默认构建 `linux/amd64`、`CGO_ENABLED=0` 的可执行文件：

```bash
make build
```

产物为 `bin/server`。其他平台可以覆盖参数：

```bash
make build GOOS=darwin GOARCH=arm64 BINARY=bin/server-darwin-arm64
```

发布压缩包、Docker 镜像和三端静态资源属于仓库级流程，请在根目录执行 `make package` 或 `make docker-build`；对应参数和运行示例见根目录 README。

## 常用参数

| 参数 | 默认值 | 用途 |
| --- | --- | --- |
| `CONF` | `./configs` | 服务运行配置目录。 |
| `APP_ENV` | `dev` | 选择 `<name>.<env>.yaml` 环境覆盖配置。 |
| `RUN_ARGS` | 空 | 追加到服务命令后的参数。 |
| `CGO_ENABLED` | `0` | Go 构建时是否启用 CGO。 |
| `GOOS` / `GOARCH` | `linux` / `amd64` | 构建目标平台。 |
| `BINARY` | `bin/server` | 可执行文件输出路径。 |
| `ARCHIVE` | `dist/backend-<os>-<arch>.tar.gz` | 后端二进制压缩包输出路径（由根目录 `make package` 调用）。 |
| `PUBLIC_WIRE_DIR` | `internal/module` | 公共入口使用的内部 `wire.go` 所在目录。 |
| `WIRE_DIR` | `internal/cmd/server` | 独立入口 `wire.go` 所在目录。 |
| `GORM_GEN_CONFIG` | `configs/data.dev.yaml` | GORM 生成使用的数据源配置。 |
| `GORM_GEN_DATABASE` | 空 | 可选数据库名，默认读取配置文件。 |
| `GORM_TABLE` | 内置表清单 | 逗号分隔的 GORM 生成表。 |

## 外部宿主复用

外部 Go 项目复用 Backend 时，在自己的 Wire 组合根中加入 `kratoscore.ProviderSet`、`backend.ProviderSet` 和宿主的合并 ProviderSet：

```go
func NewApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		kratoscore.ProviderSet,
		backend.ProviderSet,
		mergeProviderSet,
	))
}
```

根包通过 `AdminResources`、`AdminModules`、`AdminTasks`、`AdminStreams` 和 `AdminConsumers` 输出具名贡献，宿主的合并 ProviderSet 将它们与其他业务模块的贡献显式追加为 Core 最终集合。公开构造器只使用 Core 公共类型，外部生成的 `wire_gen.go` 不会依赖 `backend/internal`。

`backend.NewModules` 初始化的是宿主进程级运行日志采集器；外部项目按上述方式接入 Backend 后，其自身以及其他已注册模块写入 stdout/stderr 的日志也会进入运行日志实时控制台，历史日志文件则按宿主的日志配置读取。

`configs/auth.yaml` 中的 JWT 密钥仅用于本地开发示例；生产环境必须通过对应的 `auth.<env>.yaml` 覆盖为密钥管理服务提供的随机密钥。

外部模块接入 AI 时使用 `pkg/agent.NewRuntime` 创建运行时，通过 `RuntimeConfig.AdminTools/AppTools` 或 `Runtime.RegisterTool` 注册 Eino `InvokableTool`；简单结构化工具优先使用 `pkg/agent.InferTool` 自动生成参数 schema。评论审核、内容提取等固定流程可以组合 `NewChatClient`、`NewStructuredRunner`、`SchemaFor` 和多模态 Part 构造函数，不需要引用 `internal` 包。需要权限控制时实现 `ToolAccessChecker`，不接入权限系统则保持 `Checker` 为 `nil`。
