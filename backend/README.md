# backend

`backend` 保留 API 契约、Go 生成接口、Service 实现、Biz 业务层，以及任务调度、HTTP/gRPC/MCP 注册和必要的数据访问闭包；不提供独立启动入口或前端资源。项目文档注册、稳定 ID 和目录查询由 `kratos-core/pkg/docs` 统一提供。

## 目录

```text
backend
├── api
│   ├── proto                         # Proto 契约
│   └── gen/go                        # Buf 生成的 Go 接口、HTTP、gRPC 和工具代码
├── internal/biz                      # 业务 Case、DTO、代码生成和辅助领域代码
├── internal/biz/job                  # 定时任务调度与任务日志
├── internal/server                   # 服务中间件和 API 模块注册适配
├── internal/task                     # 异步业务任务
├── internal/data/gen                 # GORM 生成的模型、查询和仓储
├── internal/service                  # Proto Service 实现
├── internal/const                    # 业务常量
├── internal/i18n/locales             # 业务语言资源
└── migration                         # 代码生成业务使用的迁移资源
```

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
