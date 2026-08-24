# backend

`backend` 保留 API 契约、Go 生成接口、Service 实现、Biz 业务层，以及任务调度、HTTP/gRPC/MCP/AI 注册和必要的数据访问闭包；进程入口位于 `internal/cmd/server`。根包通过 `ProviderSet`、`NewModuleResources`、`NewModules`、`NewTasks`、`NewStreams` 和 `NewQueueConsumers` 提供可被外部 Core 宿主复用的公共边界，`internal/module` 仅承载内部实现。AI Runtime 实现在 `internal/biz`，对外复用入口为 `pkg/agent`。

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
├── internal/task                     # 异步任务与定时任务执行器
├── internal/server                   # 服务中间件和 API 模块注册适配
├── internal/task                     # 异步业务任务
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

`make run` 会按“项目文档 -> protobuf Go -> OpenAPI -> 独立入口 Wire -> 启动服务”的顺序刷新必要产物。确认生成产物没有变化时，可跳过生成直接启动：

```bash
make run-only
```

默认配置目录为 `./configs`，默认运行环境为 `dev`。基础配置使用 `<name>.yaml`，环境差异使用 `<name>.<env>.yaml`；环境文件存在时在基础配置之后加载，不存在时回退基础配置。可以覆盖配置目录、运行环境或追加启动参数：

```bash
make run-only CONF=/path/to/configs
make run-only APP_ENV=prod
make run-only RUN_ARGS='--help'
```

例如 `APP_ENV=dev` 会加载 `data.yaml` 后再加载 `data.dev.yaml`，同时忽略 `data.prod.yaml`。本地开发配置统一保存在 `*.dev.yaml`，这类文件默认不纳入 Git。

独立入口注入 `kratoscore.ProviderSet` 与内部模块 ProviderSet；Core 负责统一创建和管理 HTTP、gRPC、MCP、SSE、队列与定时任务运行时。

## 修改后执行

| 修改场景 | 命令 | 说明 |
| --- | --- | --- |
| Proto 契约 | `make api openapi ts` | 生成 Go、OpenAPI 和三个前端的 TypeScript RPC。只影响一个前端时可改用 `ts-admin`、`ts-uni-app` 或 `ts-taro-app`。 |
| 数据库表结构 | `make gorm-gen` | 先更新开发库，再按 `GORM_GEN_CONFIG`、`GORM_GEN_DATABASE` 和 `GORM_TABLE` 生成。 |
| ProviderSet 或构造参数 | `make public-wire wire` | 分别刷新公共入口内部装配和独立服务入口。 |
| README 或 docs | `make project-docs` | 收集 Markdown，并生成各语言文档目录。 |
| 语言包或国际化资源 | `make i18n` | 依次同步语言包、项目文档和 OpenAPI 多语言产物。 |
| Go import 别名 | `make normalize-go-imports` | 默认预览别名规范化结果；设置 `NORMALIZE_GO_IMPORTS_WRITE=1` 时写回文件。 |
| 多类生成源同时变化 | `make gen` | 依次执行 GORM、接口、前端、文档、Wire 和格式化；需要可访问开发数据库。 |

所有生成产物都必须通过上述命令刷新，不能手工修改。

## 检查

提交前执行完整 Backend 检查：

```bash
make check
```

`make check` 依次运行 `make lint`、`make test` 和 `make i18n-check`，不会自动格式化代码。需要格式化时先执行：

```bash
make fmt
```

也可以单独运行某一项：

```bash
make lint
make test
make i18n-check
```

## 构建与打包

默认构建 `linux/amd64`、`CGO_ENABLED=0` 的可执行文件：

```bash
make build
```

产物为 `bin/server`。发布压缩包同时包含可执行文件和 `configs` 目录：

```bash
make package
```

默认输出 `dist/backend-linux-amd64.tar.gz`。其他平台可以覆盖参数：

```bash
make package GOOS=linux GOARCH=arm64
make build GOOS=darwin GOARCH=arm64 BINARY=bin/server-darwin-arm64
```

解压发布包后，可通过 `./bin/server --conf ./configs` 启动。打包前应检查 `configs` 内的数据库、Redis、JWT 和其他部署配置，不要直接发布本地凭据。

仓库内置的 `docker-build` 会先验证 Docker CLI 和 Docker daemon，然后依次构建管理后台、uni-app H5、Taro H5、Linux 后端程序和最终镜像：

```bash
make docker-build \
  IMAGE=kratos-admin \
  TAG=v1.0.0
```

默认生成 `linux/amd64` 镜像。构建 ARM64 镜像时同时指定 Go 和 Docker 平台：

```bash
make docker-build GOARCH=arm64 DOCKER_PLATFORM=linux/arm64
```

镜像把三端静态站点保存在只读种子目录，容器启动时将其合并到 `/app/data`。本地 OSS 上传的图片、附件及其他运行期文件也位于 `/app/data`，因此部署时应将整个目录绑定到宿主机；启动脚本只覆盖镜像提供的站点文件，不会清空已有上传目录。

本地构建完成后使用 `docker-run` 启动容器，运行环境通过 `APP_ENV` 选择：

```bash
make docker-run IMAGE=kratos-admin TAG=v1.0.0 APP_ENV=dev
```

该命令默认使用桥接网络并发布宿主机 `7001`、`6001` 端口。首次运行会把 `configs/*.yaml` 初始化到宿主机 `runtime/configs`，将 MySQL、Redis、Consul 等容器主动连接的本机地址改为 `host.docker.internal`，再只读映射到 `/app/configs`；OAuth 浏览器回调地址保持不变。宿主机修改 `runtime/configs` 中的 YAML 后重启容器即可生效，不需要重建镜像。

宿主机 `data` 映射到 `/app/data`。数据、配置、网络和发布端口可以通过 `DOCKER_DATA_DIR`、`DOCKER_CONFIG_DIR`、`DOCKER_NETWORK`、`DOCKER_HTTP_PORT`、`DOCKER_GRPC_PORT` 覆盖。删除 `runtime/configs` 后再次执行 `make docker-run` 可以按当前 `configs` 重新初始化运行配置。

停止本地服务但保留容器和宿主机数据：

```bash
make docker-stop IMAGE=kratos-admin TAG=v1.0.0
```

之后再次执行 `make docker-run` 会自动清理已停止的同名容器并重新创建。

镜像本身不会包含 `configs/*.dev.yaml`、历史上传文件、代码生成恢复文件或运行日志。`docker-run` 挂载宿主机配置目录后，开发环境配置文件只在运行期对容器可见。

## 常用参数

| 参数 | 默认值 | 用途 |
| --- | --- | --- |
| `CONF` | `./configs` | 服务运行配置目录。 |
| `APP_ENV` | `dev` | 选择 `<name>.<env>.yaml` 环境覆盖配置。 |
| `RUN_ARGS` | 空 | 追加到服务命令后的参数。 |
| `CGO_ENABLED` | `0` | Go 构建时是否启用 CGO。 |
| `GOOS` / `GOARCH` | `linux` / `amd64` | 构建目标平台。 |
| `BINARY` | `bin/server` | 可执行文件输出路径。 |
| `ARCHIVE` | `dist/backend-<os>-<arch>.tar.gz` | 发布压缩包输出路径。 |
| `DOCKER` | `docker` | Docker 命令路径或名称。 |
| `DOCKERFILE` | `Dockerfile` | Dockerfile 路径。 |
| `DOCKER_PLATFORM` | `linux/<GOARCH>` | 镜像目标平台。 |
| `IMAGE` / `TAG` | `backend` / `latest` | Docker 镜像名称和标签。 |
| `CONTAINER_NAME` | `kratos-admin` | 本地运行的容器名称。 |
| `DOCKER_NETWORK` | `bridge` | 容器网络。 |
| `DOCKER_HTTP_PORT` | `7001` | 映射到容器 `7001` 的宿主机 HTTP 端口。 |
| `DOCKER_GRPC_PORT` | `6001` | 映射到容器 `6001` 的宿主机 gRPC 端口。 |
| `DOCKER_DATA_DIR` | `backend/data` | 映射到 `/app/data` 的宿主机目录。 |
| `DOCKER_CONFIG_SOURCE_DIR` | `backend/configs` | 首次初始化 Docker 运行配置的源目录。 |
| `DOCKER_CONFIG_DIR` | `backend/runtime/configs` | 只读映射到 `/app/configs` 的宿主机运行配置目录。 |
| `DOCKER_RUN_ARGS` | 空 | 传给 `docker run` 的其他参数。 |
| `PUBLIC_WIRE_DIR` | `internal/module` | 公共入口使用的内部 `wire.go` 所在目录。 |
| `WIRE_DIR` | `internal/cmd/server` | 独立入口 `wire.go` 所在目录。 |
| `GORM_GEN_CONFIG` | `configs/data.dev.yaml` | GORM 生成使用的数据源配置。 |
| `GORM_GEN_DATABASE` | 空 | 可选数据库名，默认读取配置文件。 |
| `GORM_TABLE` | 内置表清单 | 逗号分隔的 GORM 生成表。 |
| `I18N_LOCALES` | `en-US,zh-TW,ja-JP` | 文档和 OpenAPI 的目标语言。 |
| `I18N_OFFLINE` | `0` | 设为 `1` 时禁用在线翻译。 |
| `I18N_BATCH_CHARS` | `400` | 单次项目文档翻译请求的最大字符数；Google 限流时自动切换 MyMemory。 |
| `PROJECT_DOCS_SCRIPT` | `../scripts/project_docs.py` | 项目文档收集与本地化脚本。 |

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

外部模块接入 AI 时使用 `pkg/agent.NewRuntime` 创建运行时，通过 `RuntimeConfig.AdminTools/AppTools` 或 `Runtime.RegisterTool` 注册 Eino `InvokableTool`；简单结构化工具优先使用 `pkg/agent.InferTool` 自动生成参数 schema。评论审核、内容提取等固定流程可以组合 `NewChatClient`、`NewStructuredRunner`、`SchemaFor` 和多模态 Part 构造函数，不需要引用 `internal` 包。需要权限控制时实现 `ToolAccessChecker`，不接入权限系统则保持 `Checker` 为 `nil`。
