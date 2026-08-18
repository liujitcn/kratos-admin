# backend

`backend` 保留 API 契约、Go 生成接口、Service 实现、Biz 业务层，以及任务调度、HTTP/gRPC/MCP/AI 注册和必要的数据访问闭包；进程入口位于 `internal/cmd/server`，`internal/module` 负责提供唯一的 Admin `module.Module`。AI Runtime 实现在 `internal/biz`，对外复用入口为 `pkg/agent`。

## 目录

```text
backend
├── internal/cmd/server             # Admin 独立启动入口与 Wire 组合根
├── api
│   ├── proto                         # Proto 契约
│   └── gen/go                        # Buf 生成的 Go 接口、HTTP、gRPC 和工具代码
├── internal/biz                      # 业务 Case、DTO、代码生成和辅助领域代码
├── internal/module                   # Admin 到 kratos-core 的模块适配、服务和资源
│   ├── module.go                     # Core Module 协议注册
│   ├── resources.go                  # module.Module 静态资源
│   └── init.go                        # Admin 模块 ProviderSet
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

## 启动

```bash
make run
```

`make run` 会先生成项目文档、OpenAPI 和 `internal/cmd/server` 下的 Wire 装配产物。`internal/cmd/server/wire.go` 直接注入 `kratoscore.ProviderSet`，并将 Admin 提供的唯一 `module.Module` 交给 Core；Core 负责统一创建和管理 HTTP、gRPC、MCP、SSE、队列与定时任务运行时。

外部模块接入 AI 时使用 `pkg/agent.NewRuntime` 创建运行时，通过 `RuntimeConfig.AdminTools/AppTools` 或 `Runtime.RegisterTool` 注册 Eino `InvokableTool`；简单结构化工具优先使用 `pkg/agent.InferTool` 自动生成参数 schema。需要权限控制时实现 `ToolAccessChecker`，不接入权限系统则保持 `Checker` 为 `nil`。

## 检查

```bash
go test ./...
```

## API 生成

需要预先安装 Buf、`protoc` Go 插件和 `goimports`。

```bash
make api
```

生成配置为 `api/buf.yaml` 和 `api/buf.gen.yaml`，输入为 `api/proto`，产物为 `api/gen/go`。
