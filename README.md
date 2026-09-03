# kratos-admin

`kratos-admin` 是一个前后端分离的管理系统仓库，包含 Go + Kratos 后端、Vue 管理后台、uni-app 应用底座、React/Taro 应用底座、版本化数据库迁移和模块脚手架。

## 已实现能力

- 账号密码、验证码、OAuth、TOTP/WebAuthn 多因素认证、JWT 刷新、租户和 Casbin 权限。
- TOTP 多因素认证、一次性恢复码和 `disabled`、`optional`、`all_required` 全局策略。
- 开放授权客户端管理：客户端按租户绑定，凭据换取 Bearer Token，并由 operation 拦截器和 HTTP 加解密 Filter 校验租户、状态、IP 白名单、JSON API 白名单及协议密文。
- 用户、角色、部门、岗位、菜单、字典、配置、任务、文件资产、日志、地区、API 和迁移记录管理。
- 后台工作台提供用户/角色概览、登录趋势、登录结果和操作动作分布统计。
- 消息分类、租户内定向/全员站内信、收件箱已读/归档、Redis 投递恢复和 Admin/uni-app/Taro 消息中心。
- Proto 驱动的 HTTP、gRPC、OpenAPI、Agent Tool、MCP Tool 和 TypeScript RPC 生成。
- AI 会话、流式消息、附件、工具调用、重试、再生成和分支会话。
- 管理端代码生成配置、预览、生成进度和还原。
- 运行日志浏览：实时控制台 SSE、历史日志查询、级别和关键字筛选及历史原文件下载。
- 登录来源策略（全局及租户/用户定向规则）、密码复杂度策略、当前会话撤销、审计日志异步落库与保留清理、受控 MySQL 备份恢复任务。
- 构建期收集当前项目、宿主项目和外部模块的 README/docs，并在管理端统一查看。
- 可挂载的 Go Core 模块；后端实现 `module.Module`，通过 `Resources` 提供静态资源，并由启动入口交给 Core 统一注册协议服务。
- 管理端、uni-app、Taro 和后端错误目录的语言集合由语言包自动发现；动态菜单、字典和代码生成同步支持所有已注册语言。

仓库不包含商城、订单、支付或推荐等业务模块。

## 目录

| 目录 | 说明 | 文档 |
| --- | --- | --- |
| `backend` | Kratos 服务、Proto、GORM、迁移和 Core 宿主组合。 | [backend/README.md](backend/README.md) |
| `frontend/admin` | 管理端 workspace，包含默认宿主、core、System 和 CLI。 | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/uni-app` | uni-app workspace，包含默认宿主、core、system 和 CLI。 | [frontend/uni-app/README.md](frontend/uni-app/README.md) |
| `frontend/taro-app` | React/Taro workspace，包含默认宿主、core、UI、system 和 CLI。 | [frontend/taro-app/README.md](frontend/taro-app/README.md) |
| `docs` | 当前架构、操作流程和专题说明。 | [docs/README.md](docs/README.md) |

开放授权协议的接口范围、拦截器边界和加密扩展点见 [docs/开放授权协议设计.md](docs/开放授权协议设计.md)。

## 环境

- Go `1.27.0`。
- Node.js `^20.19.0` 或 `>=22.12.0`。
- pnpm 版本以各 workspace 的 `packageManager` 为准：管理端 `10.33.4`，uni-app 与 Taro 应用端 `10.13.1`。
- MySQL 和 Redis；默认连接见 `backend/configs/data.yaml`。
- Docker 部署需要可用的 Docker CLI 与 Docker daemon。
- 启用 TOTP 绑定时，`mfa.encryption_key` 有显式值则使用该值，留空时在实际保护 TOTP 密钥时按 `kratos-kit:mfa/encryption` 从运行时密钥服务派生；启用 WebAuthn 时还要配置 `mfa.webauthn.rp_id` 与 `mfa.webauthn.rp_origins`。配置文件中的敏感值应使用 `ENC[...]` 保存。
- Buf、protoc 插件、Wire 和 gorm-gen 只在重新生成代码时需要，可通过 `make -C backend init` 安装。

直接执行 `make` 或 `make help` 查看仓库级命令；Backend 和 Frontend 的完整目标分别使用 `make -C backend help`、`make -C frontend help` 查看。

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
make -C backend run APP_ENV=dev
```

`run` 会先刷新启动所需的接口、OpenAPI 和 Wire 产物；确认生成产物未变化时可使用 `make -C backend run-only APP_ENV=dev` 直接启动。基础配置使用 `<name>.yaml`，环境差异使用 `<name>.<env>.yaml`，缺少当前环境文件时自动回退基础配置。完整目标、执行顺序和参数见 [Backend 常用流程](backend/README.md#常用流程)。

启动全部前端开发环境（管理后台、uni-app/Taro H5 和微信小程序）：

```bash
make -C frontend run
```

也可以按端启动（每个常驻命令都应在独立终端运行）：

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
make gen
make check
make build
make -C backend fmt
```

`make gen` 按 Backend、Frontend、文档和 OpenAPI 的顺序生成全仓产物；`make check` 按 Backend、三个前端 workspace 和国际化的顺序执行检查。根目录 `make build` 构建后端二进制及三个前端 H5 宿主；只构建全部前端（H5 + 微信小程序）可使用 `make -C frontend build`，仅构建 H5 使用 `make -C frontend build-h5`，生成全部 npm 发布包使用 `make -C frontend package`。

`make -C backend cli` 会安装 `kratos-kit/cmd/normalize-go-imports`，`make -C backend fmt` 再运行该命令并使用 `goimports` 格式化 Backend 全部 Go 文件。

该工具实现统一位于 `kratos-kit/cmd/normalize-go-imports`，不再在 Admin 仓库保留独立 Make 目标。

后端默认通过 `make -C backend build` 构建 `linux/amd64` 二进制；仓库根目录的 `make package` 会生成包含 `bin/server` 和 `configs` 的后端发布压缩包，并同时生成全部前端 npm 包。目标平台可使用 `GOOS`、`GOARCH` 覆盖。

Docker 镜像通过仓库根目录命令构建：

```bash
make docker-build IMAGE=kratos-admin TAG=latest
make docker-run IMAGE=kratos-admin TAG=latest APP_ENV=dev
make docker-stop IMAGE=kratos-admin TAG=latest
```

构建命令先检查 Docker，再构建管理后台、uni-app H5、Taro H5 和 Linux 后端程序。运行命令发布宿主机 `7001/6001` 端口，将 `backend/data`、`backend/logs`、`backend/backups` 分别映射到容器的 `/app/data`、`/app/logs`、`/app/backups`，并首次初始化可在宿主机修改的 `backend/runtime/configs` 后映射到 `/app/configs`。三端静态站点随镜像发布，启动时合并到 `/app/data`，已有上传文件不会被清空；Core 根据 `oss.root_directory` 将本地对象统一映射到 `/data/`。完整构建参数和运行示例见本节。

本地只启动依赖服务可使用：

```bash
docker compose -f docker-compose.libs.yml up -d
docker compose -f docker-compose.libs.yml --profile object-storage up -d
```

该 Compose 只负责 MySQL、Redis 和可选 MinIO，不改变现有单镜像应用部署方式。

`I18N_LOCALES` 使用逗号分隔的 BCP 47 语言代码列表（默认 `en-US,zh-TW,ja-JP`），统一控制项目文档和 OpenAPI 的目标语言。`make i18n-docs` 由仓库内 `scripts/project_docs.py` 按三段路径范围收集 README 与 docs Markdown，再将正文按 `I18N_BATCH_CHARS`（默认 400）分片翻译并按原顺序合并，生成 `docs.json` 和 `docs.<locale>.json`；对应的 `README.en-US.md`、`guide.ja-JP.md` 等语言源文件存在时直接使用，否则才执行机器翻译。Google V1 会按多个 client 顺序切换，全部不可用时再使用 MyMemory，并保留 Markdown 代码、链接和占位符。语言目录只本地化文档正文和显示文件名，`README.md` 显示名、目录名称及稳定路径保持不变。如需使用外部实现，可通过 `PROJECT_DOCS_SCRIPT` 覆盖脚本路径。`make i18n-openapi` 生成 OpenAPI 多语言 YAML。离线生成使用 `I18N_OFFLINE=1 make i18n`。

`backend/api/gen`、`backend/internal/data/gen`、`backend/internal/docs/assets/docs*.json`、`backend/internal/docs/docs.go`、各前端包的 `src/rpc`、OpenAPI 及 `wire_gen.go` 都是生成产物，不得手工修改。所有前端 RPC 的 Buf 配置统一位于 `backend/api`，管理端通过 `make -C frontend ts-admin` 生成，应用端分别通过 `make -C frontend ts-uni-app` 和 `make -C frontend ts-taro-app` 生成；需要一次生成三端时执行 `make -C frontend ts`，全仓生成使用根目录 `make gen`。

`make -C backend gen` 会同时生成 `backend/internal/module` 的公共入口内部 Wire 产物和 `backend/internal/cmd/server` 的独立启动 Wire 产物；单独刷新前者使用 `make -C backend public-wire`，单独刷新自定义组合根可通过 `WIRE_DIR` 指定包含 `wire.go` 的目录。

外部 Go 项目将 `github.com/liujitcn/kratos-admin/backend` 的具名模块、资源、定时任务、SSE 流和队列消费者贡献与自身贡献合并后，再交给 `github.com/liujitcn/kratos-core` 创建协议服务及应用生命周期；外部生成代码不会引用 `backend/internal`。

`make i18n-openapi` 会在生成 `openapi.yaml` 后同步生成 `openapi.en-US.yaml`、`openapi.zh-TW.yaml` 和 `openapi.ja-JP.yaml`。默认资源来自后端错误目录和管理端 Core 语言包；外部资源可通过 `OPENAPI_I18N_CONTENT="语言=路径"` 传入。未命中的文案默认自动翻译：英文和日语使用 Google V1，繁体中文使用 OpenCC；设置 `I18N_AUTO_LOCALIZE=0` 可关闭自动翻译。无网络环境使用 `I18N_OFFLINE=1 make i18n-openapi`。

## 国际化

语言包定义系统能够渲染的语言集合，`base_language` 表只负责运行时启用状态、名称、排序和主语言配置。管理端语言偏好保存为 `kratos-admin:locale`，uni-app 和 Taro 保存为 `kratos-app:locale`；所有 HTTP、刷新令牌、fetch、SSE、uni.request 和 Taro.request 请求都会发送规范化的 `Accept-Language`。固定文案由各 workspace 的 core/System JSON 语言包维护，动态菜单和字典由后端翻译表按请求语言解析，缺少当前语言译文时回退主语言。

新增语言不需要修改 Go、TypeScript 或模块注册代码：在 `backend/internal/i18n/assets` 和三个 workspace 的六个前端语言包目录中增加同名 JSON，然后执行 `make i18n-sync`。脚本会校验语言集合、语言键和占位符，并生成六个前端注册文件、Element Plus 和 Day.js 映射。语言名称、排序、启用状态和主语言由 `base_language` 数据库记录提供；`common.language.*` 用于编译期离线显示和生成语言迁移的初始名称。新增语言的完整文件清单和迁移流程见 [国际化语言扩展指南](docs/国际化语言扩展指南.md)。需要把语言加入新部署数据库时，直接更新唯一的 `v0.0.1` 初始化迁移；已有数据库的启用状态不会被迁移覆盖。

动态资源的主语言由 `base_language.is_primary` 配置。创建或更新菜单、字典、字典项和系统配置时，后端按请求 `Accept-Language` 将输入文本转换为主语言写入主表；请求语言不是主语言时，原文写入对应翻译表，其他已启用非主语言也只保存在翻译表。系统配置名称、菜单标题、字典名称和字典项标签支持在管理端点击名称打开翻译弹窗，文本/富文本配置值支持运行时翻译回退。

国际化常用命令：

```bash
make i18n-check
make i18n-verify
make i18n-sync
make i18n-docs
make i18n-openapi
make i18n
```

新增语言属于初始化数据，统一写入唯一的 `v0.0.1` 迁移：

```bash
I18N_LOCALE=ja-JP make i18n-locale
I18N_LOCALE=de-DE I18N_OFFLINE=1 make i18n-locale
```

在线模式优先使用 Google V1 的多个 client，全部不可用时使用 MyMemory；繁体中文优先使用 OpenCC，离线模式使用内置术语表和 OpenCC。生成后应审核 JSON 与 SQL；已有数据库的语言启用状态不会被覆盖。运行时翻译表单仍可对动态资源执行即时翻译，已有非空译文不会覆盖。

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
make tag VERSION=0.0.30
```

`make tag` 会先执行只读的 `make i18n-verify`，检查语言包、SQL 翻译脚本、OpenAPI 多语言文档和项目文档生成物；生成物未同步时会直接拒绝发布，不会在发布过程中自动翻译或改写文件。随后发布脚本要求当前分支为远程默认分支且与 `origin` 同步，执行后端测试和前端打包，然后推送 `vX.Y.Z`、`backend/vX.Y.Z`、`npm/vX.Y.Z`。`npm/vX.Y.Z` 触发 `.github/workflows/publish-npm.yml`，通过 npm Trusted Publishing 发布以上 10 个包；三个默认宿主均为私有包，不参与发布。本机需要可用的 `git`、`gh` 和 GitHub 登录态。

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
| 站内信 | [docs/站内信设计.md](docs/站内信设计.md) |
| 管理端组件 | [docs/前端组件清单.md](docs/前端组件清单.md) |
| 国际化设计 | [docs/国际化最终方案.md](docs/国际化最终方案.md) |
| 安全策略与运维任务 | [docs/安全策略与运维任务.md](docs/安全策略与运维任务.md) |
| 新增语言 | [docs/国际化语言扩展指南.md](docs/国际化语言扩展指南.md) |
