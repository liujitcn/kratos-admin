# kratos-admin

`kratos-admin` 是一个前后端分离的管理系统仓库，包含 Go + Kratos 后端、Vue 管理后台、uni-app 应用底座、版本化数据库迁移和模块脚手架。

## 已实现能力

- 账号密码、验证码、OAuth、JWT 刷新、租户和 Casbin 权限。
- 用户、角色、部门、岗位、菜单、字典、配置、任务、日志、地区、API 和迁移记录管理。
- Proto 驱动的 HTTP、gRPC、OpenAPI、Agent Tool、MCP Tool 和 TypeScript RPC 生成。
- AI 会话、流式消息、附件、工具调用、重试、再生成和分支会话。
- 管理端代码生成配置、预览、生成进度和还原。
- 可挂载的 Go Core 模块、管理端业务模块和 uni-app 具名模块注册边界。

仓库不包含商城、订单、支付或推荐等业务模块。

## 目录

| 目录 | 说明 | 文档 |
| --- | --- | --- |
| `backend` | Kratos 服务、Proto、GORM、迁移和 Core 运行时。 | [backend/README.md](backend/README.md) |
| `frontend/admin` | 管理端 workspace，包含 core、System、CLI 和默认宿主。 | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/app` | 可直接运行和复用的 uni-app 应用底座。 | [frontend/app/README.md](frontend/app/README.md) |
| `docs` | 当前架构和专题说明。 | [docs/系统总体设计.md](docs/系统总体设计.md) |

## 环境

- Go `1.26.3`。
- Node.js `^20.19.0` 或 `>=22.12.0`。
- pnpm `10.33.4`。
- MySQL 和 Redis；默认连接见 `backend/configs/data.yaml`。
- Buf、protoc 插件、Wire 和 gorm-gen 只在重新生成代码时需要，可通过 `make -C backend init` 安装。

## 本地启动

先创建数据库：

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

安装前端依赖：

```bash
make -C frontend init
```

启动后端：

```bash
cd backend
go run ./internal/cmd/server --conf ./configs
```

启动管理后台或应用端 H5：

```bash
make -C frontend run-admin
cd frontend/app && pnpm dev:h5
```

| 服务 | 默认地址 |
| --- | --- |
| 后端 HTTP | `http://localhost:7001` |
| 后端 gRPC | `localhost:6001` |
| 管理后台 | `http://localhost:8848` |
| 应用端 H5 | `http://localhost:5002` |

默认迁移提供开发账号 `super / 112233` 和 `admin / 112233`。部署前必须修改默认密码、JWT 密钥、数据库和 Redis 凭据。

## 生成与检查

```bash
make -C backend gen
cd backend && go test ./...

cd frontend/admin
pnpm check:exports
pnpm test
pnpm type:check
pnpm lint:oxlint

cd ../app
pnpm test:vite
pnpm tsc
pnpm lint
```

`backend/api/gen`、`backend/internal/data/gen`、管理端和应用端 `rpc`、OpenAPI 及 `wire_gen.go` 都是生成产物，不得手工修改。

## 发布

统一发布会更新四个 npm 包的版本：

- `@liujitcn/kratos-admin`
- `@liujitcn/kratos-admin-system`
- `@liujitcn/kratos-admin-cli`
- `@liujitcn/kratos-app`

```bash
make tag VERSION=0.0.16
```

脚本要求当前分支为远程默认分支且与 `origin` 同步，会纳入当前工作区改动，执行后端测试和前端打包，然后推送 `vX.Y.Z`、`backend/vX.Y.Z`、`npm/vX.Y.Z`。`npm/vX.Y.Z` 触发 `.github/workflows/publish-npm.yml`，通过 npm Trusted Publishing 发布四个包；本机需要可用的 `git`、`gh` 和 GitHub 登录态。

只做本地 npm 发布时：

```bash
pnpm login
make -C frontend publish
```

发布脚本会跳过 registry 中已存在的同版本包，支持失败后重试。私有 registry 可通过 `NPM_REGISTRY`、`NPM_ACCESS` 和 `NPM_TAG` 覆盖。

## 文档

| 主题 | 文档 |
| --- | --- |
| 总体架构 | [docs/系统总体设计.md](docs/系统总体设计.md) |
| 新能力接入 | [docs/服务接入指南.md](docs/服务接入指南.md) |
| 数据库迁移 | [docs/数据库与初始化数据设计.md](docs/数据库与初始化数据设计.md) |
| 参数校验 | [docs/接口参数校验设计.md](docs/接口参数校验设计.md) |
| 登录和密码 | [docs/登录与密码加密流程.md](docs/登录与密码加密流程.md) |
| AI 助手 | [docs/AI助手设计.md](docs/AI助手设计.md) |
| 管理端组件 | [docs/前端组件清单.md](docs/前端组件清单.md) |
