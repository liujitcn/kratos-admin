# Backend

`backend` 是本仓库的 Go 服务，负责 HTTP/gRPC 接口、认证授权、系统管理、AI、消息、任务、迁移和静态资源托管。后端与管理端、uni-app、Taro 共用 `backend/api/proto` 中的接口契约。

## 目录

```text
backend
├── api/proto                         # Proto、HTTP 路由和校验
├── api/gen                           # 生成的 Go 协议代码
├── configs                           # 最小启动配置
├── configs/full                      # 完整配置模板
├── internal/cmd/server               # 独立服务入口
├── internal/service                  # HTTP/gRPC 服务适配
├── internal/biz                      # 认证、系统、AI、消息和任务规则
├── internal/data                     # 数据库、缓存、对象存储和仓储
├── internal/module                   # 模块、SSE、队列和任务装配
├── internal/openapi                  # OpenAPI 生成资源
├── migration/assets/v0.0.1/mysql     # 初始化数据、菜单、权限和多语言
└── pkg/agent、pkg/notification       # AI 与消息的稳定业务入口
```

## 已实现能力

- 账号密码、验证码、OAuth、TOTP/WebAuthn MFA、恢复码、JWT 刷新和会话撤销。
- 多租户、用户、角色、部门、岗位、菜单、字典、配置、API、Casbin 权限和项目授权。
- 文件上传、公开文件、对象存储、敏感字段加密和按权限查看密钥。
- AI 会话、附件、SSE 流式响应、工具调用、智能问数、供应商/模型和知识库检索。
- 站内信分类、发布、投递、收件箱、Provider、失败恢复和三端通知。
- 定时任务、运行日志、六类审计日志、表归档、数据库备份恢复和迁移记录。
- 开放授权客户端、Bearer Token、API 白名单、来源限制、请求加解密和限流。

## 配置

最小启动配置在 `configs`，完整字段模板在 `configs/full`。常用配置文件如下：

| 文件 | 作用 |
| --- | --- |
| `data.yaml` | MySQL、Redis 和可选 AI 知识库数据源 |
| `oss.yaml` | 本地或 S3 兼容对象存储 |
| `auth.yaml` | JWT、会话、验证码和认证策略 |
| `mfa.yaml` | TOTP/WebAuthn 参数 |
| `key.yaml` | 运行时根密钥和密钥服务 |
| `server.yaml` | HTTP、gRPC、SSE、MCP 和超时 |
| `registry.yaml` | 服务注册配置 |

配置敏感值使用 `ENC[...]` 保存。生产环境必须使用有效的密钥服务、TLS 和最小权限凭据；密钥服务不可用时服务启动失败，不回退明文文件。

Redis 未配置时，单实例缓存、任务锁和队列使用进程内实现；多实例必须连接同一个 Redis。AI 知识库默认使用 `data.databases.rag`，未配置时回落默认数据源。

## 常用流程

```bash
make -C backend help
make -C backend init
make -C backend run-minimal
make -C backend run-full
make -C backend build
make -C backend test
```

局域网 HTTPS 联调：

```bash
bash scripts/generate-dev-cert.sh 192.168.1.100
make -C backend run-only APP_ENV=https
```

独立服务默认监听 HTTP `7001`、gRPC `6001`。迁移、默认账号和三端静态资源均由服务启动流程读取；开发默认账号为 `super / 112233`、`admin / 112233`，部署前必须修改密码和密钥。

## 生成链路

| 命令 | 产物 |
| --- | --- |
| `make -C backend api` | Go 协议代码和 OpenAPI 源文件 |
| `make -C backend gorm-gen` | GORM Model、Query 和 Data 代码 |
| `make -C backend public-wire` | 模块装配和公共 Wire 代码 |
| `make -C backend wire` | 独立服务入口 Wire 代码 |
| `make -C backend fmt` | 格式化后端 Go 文件 |
| `make -C backend openapi` | OpenAPI YAML |

生成文件位于 `api/gen`、`internal/data/gen`、`internal/openapi` 和 Wire 文件，不得手工修改。Proto 改动后按“生成 → Service/Biz → 前端 RPC → 初始化权限”的顺序完成。

## 数据与迁移

初始化资源统一位于 `migration/assets/v0.0.1/mysql`：

- `default_data.up.sql`：语言、租户、账号、菜单、按钮、API 权限和默认配置；
- `i18n.*.up.sql`：菜单、字典、配置和任务的非主语言译文；
- `README*.md`：迁移说明，由服务保存文件引用和校验摘要。

本仓库不新增 `v0.0.2` 等迁移目录。新表由 GORM 自动迁移创建，菜单、权限和多语言在唯一的 `v0.0.1` 初始化资源中维护。

## 运行检查

```bash
go test ./...
make -C backend fmt
make -C backend menu-sort
```

涉及 Proto、Wire、前端调用或初始化权限时，完成后在仓库根目录执行 `make gen`、`make check` 和 `make menu-sort`。
