# kratos-admin

`kratos-admin` 是一个前后端分离的管理系统仓库，包含 Go + Kratos 后端、Vue 管理后台、uni-app 应用底座、React/Taro 应用底座、版本化数据库迁移和模块脚手架。

## 已实现能力

- 账号密码、验证码、OAuth、JWT 刷新、租户和 Casbin 权限。
- 用户、角色、部门、岗位、菜单、字典、配置、任务、日志、地区、API 和迁移记录管理。
- Proto 驱动的 HTTP、gRPC、OpenAPI、Agent Tool、MCP Tool 和 TypeScript RPC 生成。
- AI 会话、流式消息、附件、工具调用、重试、再生成和分支会话。
- 管理端代码生成配置、预览、生成进度和还原。
- 构建期收集当前项目、宿主项目和外部模块的 README/docs，并在管理端统一查看。
- 可挂载的 Go Core 模块，以及管理端、应用端的独立 workspace、模块协议和脚手架。
- 管理端、uni-app、Taro 和后端错误目录的语言集合由语言包自动发现；动态菜单、字典和代码生成同步支持所有已注册语言。

仓库不包含商城、订单、支付或推荐等业务模块。

## 目录

| 目录 | 说明 | 文档 |
| --- | --- | --- |
| `backend` | Kratos 服务、Proto、GORM、迁移和 Core 运行时。 | [backend/README.md](backend/README.md) |
| `frontend/admin` | 管理端 workspace，包含默认宿主、core、System 和 CLI。 | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/uni-app` | uni-app workspace，包含默认宿主、core、system 和 CLI。 | [frontend/uni-app/README.md](frontend/uni-app/README.md) |
| `frontend/taro-app` | React/Taro workspace，包含默认宿主、core、UI、system 和 CLI。 | [frontend/taro-app/README.md](frontend/taro-app/README.md) |
| `docs` | 当前架构、操作流程和专题说明。 | [docs/README.md](docs/README.md) |

## 环境

- Go `1.26.3`。
- Node.js `^20.19.0` 或 `>=22.12.0`。
- pnpm 版本以各 workspace 的 `packageManager` 为准：管理端 `10.33.4`，uni-app 与 Taro 应用端 `10.13.1`。
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

如果依赖目录需要完全重建：

```bash
make -C frontend reinstall
```

启动后端：

```bash
make -C backend run
```

启动管理后台、uni-app 或 Taro H5（每个命令都应在独立终端运行）：

```bash
make -C frontend run-admin
cd frontend/uni-app && pnpm dev:h5
cd ../taro-app && pnpm dev:h5
```

| 服务 | 默认地址 |
| --- | --- |
| 后端 HTTP | `http://localhost:7001` |
| 后端 gRPC | `localhost:6001` |
| 管理后台 | `http://localhost:8848` |
| uni-app H5 | `http://localhost:5004` |
| Taro H5 | `http://localhost:5002` |

uni-app 和 Taro H5 默认分别使用 `5004` 与 `5002`，可以同时启动。局域网设备访问 uni-app 时，将 `localhost` 替换为开发机局域网 IP。

默认迁移提供开发账号 `super / 112233` 和 `admin / 112233`。部署前必须修改默认密码、JWT 密钥、数据库和 Redis 凭据。

## 生成与检查

```bash
make -C backend project-docs
make -C backend gen
cd backend && go test ./...
make -C backend i18n-check

cd frontend/admin
pnpm check:exports
pnpm test
pnpm type:check
pnpm lint:oxlint

cd ../uni-app
pnpm test
pnpm check:exports
pnpm tsc
pnpm lint
pnpm build:packages

cd ../taro-app
pnpm test
pnpm check:exports
pnpm tsc
pnpm lint
pnpm build:packages
pnpm build:h5
pnpm build:mp-weixin
```

`backend/api/gen`、`backend/internal/data/gen`、`backend/internal/docs/assets/docs.json`、`backend/internal/docs/docs.go`、各前端包的 `src/rpc`、OpenAPI 及 `wire_gen.go` 都是生成产物，不得手工修改。所有前端 RPC 的 Buf 配置统一位于 `backend/api`，分别通过 `make -C backend ts`、`make -C backend ts-uni-app` 和 `make -C backend ts-taro-app` 生成。

`make -C backend gen` 在当前工作区没有 `wire.go` 时会跳过 Wire 依赖注入生成；配置 `WIRE_DIR` 指向包含 `wire.go` 的目录后即可恢复该阶段。

## 国际化

语言包定义系统能够渲染的语言集合，`base_language` 表只负责运行时启用状态、名称、排序和主语言配置。管理端语言偏好保存为 `kratos-admin:locale`，uni-app 和 Taro 保存为 `kratos-app:locale`；所有 HTTP、刷新令牌、fetch、SSE、uni.request 和 Taro.request 请求都会发送规范化的 `Accept-Language`。固定文案由各 workspace 的 core/System JSON 语言包维护，动态菜单和字典由后端翻译表按请求语言解析，缺少当前语言译文时回退主语言。

新增语言不需要修改 Go、TypeScript 或模块注册代码：在后端错误目录和三个 workspace 的六个前端语言包目录中增加同名 JSON，并在代码生成 `catalog.json` 中增加同名数据，然后执行 `make i18n-sync`。脚本会校验语言集合、语言键和占位符，并生成后端 manifest、六个前端注册文件、Element Plus 和 Day.js 映射。语言名称、排序、启用状态和主语言由 `base_language` 数据库记录提供；`common.language.*` 用于编译期离线显示和生成语言迁移的初始名称。新增语言的完整文件清单和迁移流程见 [国际化语言扩展指南](docs/国际化语言扩展指南.md)。需要把语言加入新部署数据库时，再执行 `make i18n-sync I18N_MIGRATION_VERSION=vX.Y.Z`，提交脚本生成的版本化 `base_language` 迁移；已有数据库的启用状态不会被迁移覆盖。

动态资源的主语言由 `base_language.is_primary` 配置。创建或更新菜单、字典、字典项和系统配置时，后端按请求 `Accept-Language` 将输入文本转换为主语言写入主表；请求语言不是主语言时，原文写入对应翻译表，其他已启用非主语言也只保存在翻译表。系统配置名称、菜单标题、字典名称和字典项标签支持在管理端点击名称打开翻译弹窗，文本/富文本配置值支持运行时翻译回退。

后端错误目录检查与草稿命令：

```bash
make i18n-sync
make -C backend i18n-check
make -C backend i18n-draft
I18N_WRITE=1 make -C backend i18n-draft
```

`make -C backend i18n-locales` 是可选的批量语言包/动态翻译生成器；新增语言可通过 `I18N_LOCALE=de-DE I18N_MIGRATION_VERSION=vX.Y.Z make -C backend i18n-locales` 指定，并在提交前审核生成文件，避免修改已发布迁移。该脚本在线模式使用独立的 Google V1 请求，离线环境可加 `I18N_OFFLINE=1` 使用内置术语表；服务运行时和 `i18n-draft` 则读取 `backend/configs/translator.yaml` 选择 Provider。管理端翻译表单可对有内容的主语言文本执行单个或批量即时翻译，Draft 接口只返回译文、不写入数据库；运行时创建或更新资源后会异步补齐空译文，提交空文本时仍按原有保存逻辑补齐；已有非空译文不覆盖。Provider 不可用时不影响已有译文和主语言回退。

## 发布

统一发布会更新并发布 10 个 npm 包：

- `@liujitcn/kratos-admin-core`
- `@liujitcn/kratos-admin-system`
- `@liujitcn/kratos-admin-cli`
- `@liujitcn/kratos-uni-app-core`
- `@liujitcn/kratos-uni-app-system`
- `@liujitcn/kratos-uni-app-cli`
- `@liujitcn/kratos-taro-app-core`
- `@liujitcn/kratos-taro-app-ui`
- `@liujitcn/kratos-taro-app-system`
- `@liujitcn/kratos-taro-app-cli`

```bash
make tag VERSION=0.0.18
```

`make tag` 会先执行 `make -C backend project-docs` 刷新内嵌项目文档，再由发布脚本将文档和版本更新一起提交。脚本要求当前分支为远程默认分支且与 `origin` 同步，执行后端测试和前端打包，然后推送 `vX.Y.Z`、`backend/vX.Y.Z`、`npm/vX.Y.Z`。`npm/vX.Y.Z` 触发 `.github/workflows/publish-npm.yml`，通过 npm Trusted Publishing 发布以上 10 个包；三个默认宿主均为私有包，不参与发布。本机需要可用的 `git`、`gh` 和 GitHub 登录态。

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
| 国际化设计 | [docs/国际化最终方案.md](docs/国际化最终方案.md) |
| 新增语言 | [docs/国际化语言扩展指南.md](docs/国际化语言扩展指南.md) |
