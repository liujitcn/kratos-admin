# kratos-admin 与 go-wind-admin 当前快照比较

> 比较日期：2026-08-29（Asia/Shanghai）
> 结论类型：**事实**表示源码、配置或 Git 元数据可直接证明；**推断**表示基于多个事实作出的工程判断；**建议**表示面向 `kratos-admin` 的取舍。
> 路径约定：`KA:` 后的路径相对 `kratos-admin` 仓库根目录，`GW:` 后的路径相对 `go-wind-admin` 仓库根目录。

## 1. 范围与证据边界

### 1.1 快照

| 项目 | 当前提交 | 最近提交时间 | 工作树 | 历史元数据 |
| --- | --- | --- | --- | --- |
| `kratos-admin` | `4c54ddc6124123da19ea9aaab5c28cafdee29231` | 2026-08-24 23:07:21 +08:00 | **大量已暂存、未暂存、未跟踪改动** | 146 个提交；最近标签 `v0.0.29`，当前描述为 `backend/v0.0.29-3-g4c54ddc-dirty` |
| `go-wind-admin` | `8c716f58fd3c019b0248a2d3c45c66b4589149df` | 2026-08-28 09:24:00 +08:00 | **干净** | 1,256 个提交；当前未发现标签 |

**事实：**本文比较的是上述工作树快照，不是两个项目最近发布版本之间的比较。`kratos-admin` 的 `README.md`、依赖、MFA、WebAuthn、OAuth Client、三个前端等均有开发中改动，因此报告会把“当前能从工作树看到”与“已进入 HEAD”分开。

证据：Git `HEAD`、`git status --short --branch`、`git describe --tags --always --dirty`；`KA:README.md`；`GW:README.md`。

### 1.2 方法

- 只使用两个仓库的一手源码、README、配置、Makefile、Proto、生成配置和 Git 元数据。
- 对能力按三层核验：**项目宣称**、**接口/目录契约存在**、**已接入 Wire/Server/前端的可运行实现**。
- 本轮未启动服务、未连接数据库；已执行后端全量测试、`go vet`、前端类型检查和国际化校验。“已接线”仍表示静态调用链完整，不等于真实数据库/Redis 故障演练通过。
- 不把 `AGENTS.md` 中的愿景或开发规范单独当作能力证据；它只用于理解项目约束，并由源码再次核验。

## 2. 先说结论

### 2.1 核心定位

**事实：**两个项目都源自 Go + Kratos 的管理系统思路，但已经走向不同产品形态：

| 维度 | `kratos-admin` | `go-wind-admin` |
| --- | --- | --- |
| 核心定位 | 可嵌入外部 Go 宿主的模块化管理底座，覆盖管理端和两个跨端 App 技术栈 | 单个 Admin 服务配三套可替换桌面管理前端的企业后台脚手架 |
| 后端边界 | Core 宿主 + Admin `module.Module`，注册 HTTP、gRPC、MCP、SSE、Cron | 单体 `app/admin/service`，注册 REST、SSE、Asynq |
| 数据建模 | MySQL 数据库优先，GORM/Gen 反向生成强类型 Model/Query/Repo | Ent Schema 优先，生成 Entity/Query；GORM 仅备用脚手架 |
| API 组织 | `base.v1`、`system.admin.v1`、`system.app.v1` 按终端与模块划分 | 源领域 Proto + Admin BFF Proto 两层划分 |
| 前端策略 | 1 套 Vue 管理端 + uni-app + Taro，均拆为 core/module/CLI 可发布包 | Vue Vben、Vue Element、React 三套平行管理端，无应用端 |
| 产品特色 | AI/MCP、动态国际化、文档聚合、代码生成、运行日志、跨端模块化 | 审计、登录策略、套餐配额、站内信、持久化异步任务、三套 Admin UI |

**推断：**`kratos-admin` 更像“可组合、可发布的应用平台底座”；`go-wind-admin` 更像“功能面较宽、后端治理较重的传统企业后台”。因此适合吸收的是后者成熟的业务治理能力，不是整套技术路线。

### 2.2 最值得关注的差异

1. **测试方向仍有差距。**`go-wind-admin` 有 42 个 Go 测试文件、32 个前端测试源文件和 1 个快照文件；`kratos-admin` 当前工作树已有 5 个后端规则测试文件，仍缺少数据库、事务和端到端故障回归，仅三个前端 workspace 有 21 个测试文件。
2. **异步任务成熟度仍有差距。**`go-wind-admin` 的 Asynq 已接入 Wire，负责备份、租户到期扫描和站内信广播；`kratos-admin` 当前工作树已接入 Redis Streams 站内信消费者、租约、失败退避和积压摘要，仍需 Redis 故障演练。
3. **安全治理覆盖面不同。**两者都有 JWT、租户和角色权限；`go-wind-admin` 额外落地登录策略、令牌撤销/黑名单、多类审计和租户套餐到期拦截；`kratos-admin` 在当前 dirty 工作树中则领先于 WebAuthn、恢复码、MFA 全局策略和 OAuth Client 协议。
4. **“存在”不等于“可用”。**`go-wind-admin` 宣称的双 ORM、多鉴权引擎、OAuth、MFA 多方式、脚本引擎中，有多项只是契约、适配层或独立库；默认运行链并未全部启用。
5. **前端扩张方向不同。**`go-wind-admin` 为同一后台维护三套实现；`kratos-admin` 把投入放在管理端、H5、微信小程序以及可发布模块。两边的重复成本不能简单叠加。

## 3. 技术栈与版本

### 3.1 后端

| 项目 | `kratos-admin` | `go-wind-admin` | 证据 |
| --- | --- | --- | --- |
| Go | `1.27.0` | `1.26.4` | `KA:backend/go.mod`；`GW:backend/go.mod` |
| Kratos | `github.com/go-kratos/kratos/v3 v3.0.0` | `github.com/go-kratos/kratos/v2 v2.9.2` | 同上 |
| 主 ORM | GORM `v1.31.2` + GORM Gen `v0.3.29` | Ent `v0.14.6` + `go-crud/entgo` | 同上 |
| 数据库 | 当前入口只启用 MySQL 驱动；配置注释预留其他驱动 | Ent Client 直接导入 MySQL、PGX/libpq；默认配置 PostgreSQL；有 SQLite 兼容测试 | `KA:backend/internal/cmd/server/main.go`；`GW:backend/app/admin/service/internal/data/ent_client.go` |
| Redis | `go-redis/v9 v9.22.0` | `go-redis/v9 v9.22.0` | 两边 `backend/go.mod` |
| 授权 | Casbin 引擎及租户/角色/Operation 策略 | authz 适配 Casbin/OPA/Zanzibar/Noop，默认配置 Noop | `KA:backend/internal/biz/system/admin/casbin_rule.go`；`GW:backend/app/admin/service/configs/auth.yaml` |
| 异步/调度 | Core Cron Job + Redis Streams；站内信 Dispatch consumer 和恢复任务已接线 | Asynq `v0.26.0`，周期与一次性任务已接线 | `KA:backend/internal/module/module.go`、`KA:backend/internal/biz/system/admin/base_message.go`；`GW:backend/app/admin/service/internal/server/asynq_server.go` |
| AI/Agent | Eino、OpenAI Go SDK、Agent Tool、MCP SDK | 无 AI/Agent 业务依赖 | `KA:backend/go.mod`；`KA:backend/api/buf.gen.yaml` |
| 可观测性 | tracing、pprof/Pyroscope、运行日志 SSE | OpenTelemetry、pprof、API/登录/操作/数据/权限/策略评估审计 | 两边 `backend/go.mod` 和 Server/Service 源码 |

**事实：**`kratos-admin` 配置注释虽列出 SQLite/PostgreSQL，但当前启动入口仅空白导入 MySQL 驱动，版本化 SQL 也只有 `mysql`。不能按注释宣称多数据库已支持。

**事实：**`go-wind-admin` 的 Ent 主链支持 PostgreSQL/MySQL；SQLite 主要有依赖和兼容测试证据。报告不把 SQL Server、MongoDB 等间接依赖算作 Admin 数据库能力。

### 3.2 前端

| 应用 | 主要版本/框架 | 形态 | 证据 |
| --- | --- | --- | --- |
| KA Admin | Vue `3.5.35`、TypeScript `6.0.3`、Vite `8.0.14`、Element Plus、Pinia 3 | core + System module + CLI + 私有薄宿主 | `KA:frontend/admin/package.json`；`KA:frontend/admin/packages/core/package.json` |
| KA uni-app | Vue `3.4.21`、Vite `5.2.8`、Pinia 2 | H5 + 微信小程序；core/System/CLI 可发布 | `KA:frontend/uni-app/packages/core/package.json` |
| KA Taro | Taro `4.2.1`、React `18.3.1`、Webpack 5、Zustand 5、NutUI | H5 + 微信小程序；core/UI/System/CLI 可发布 | `KA:frontend/taro-app/package.json`；`KA:frontend/taro-app/packages/core/package.json` |
| GW Vben | Vue 3.5、Vben、Ant Design Vue、TanStack Vue Query、VxeTable | 完整桌面 Admin monorepo | `GW:frontend/admin/vue-vben/package.json`；其 `AGENTS.md` |
| GW Element | Vue `3.5.30`、TS `5.9.3`、Vite 8、Element Plus、Pinia 3、Vue Query | 独立桌面 Admin | `GW:frontend/admin/vue-element/package.json` |
| GW React | React `19.2.6`、TS 6、Vite 8、Ant Design 6、Zustand 5、React Query | 独立桌面 Admin | `GW:frontend/admin/react/package.json` |

**推断：**`go-wind-admin` 的三套 Admin 有利于展示和团队选型，但用户、权限、租户、审计等页面要维护三份；`kratos-admin` 的三端也有成本，但管理端与应用端解决的是不同终端需求，复用目标更明确。

## 4. 目录、分层与运行边界

### 4.1 `kratos-admin`

**事实：**主调用链为：

```text
Proto -> 生成 HTTP/gRPC/MCP/Agent Tool 契约
      -> service 传输适配
      -> biz Case 业务规则
      -> GORM/Gen 生成 Repository/Query/Model
      -> MySQL
```

- `KA:backend/api/proto`：Proto 源文件；当前有 `base`、`system/admin`、`system/app` 三类接口面。
- `KA:backend/internal/service`：生成接口的服务适配层，通常将错误记录/包装后交给 Biz。
- `KA:backend/internal/biz`：登录、租户、角色、AI、代码生成、运行日志等业务规则。
- `KA:backend/internal/data/gen`：由数据库反向生成的 Model、Query、Repository，禁止手改。
- `KA:backend/internal/module/module.go`：一个 Module 同时注册 HTTP、gRPC、MCP。
- `KA:backend/internal/module/resources.go`：向 Core 宿主贡献 Models、Migrations、Docs、I18n、OpenAPI。
- `KA:backend/internal/cmd/server/main.go`：默认独立宿主；外部项目也可组合公共 Module/Resources。

**推断：**这种边界适合“一个仓库提供通用模块，多个宿主项目组合”的目标；代价是依赖 `kratos-core`、`kratos-kit`、`gorm-kit` 等自有模块，理解完整运行链需要跨仓库。

### 4.2 `go-wind-admin`

**事实：**主调用链为：

```text
源领域 Proto -> Admin BFF Proto/REST
             -> Service（业务编排）
             -> Repo
             -> Ent Query/Entity -> PostgreSQL/MySQL
```

- `GW:backend/api/protos/{identity,permission,authentication,audit,...}/service/v1`：不带 HTTP 注解的源领域服务和消息。
- `GW:backend/api/protos/admin/service/v1/i_*.proto`：Admin BFF，复用源领域消息并添加 HTTP 映射。
- `GW:backend/app/admin/service/internal/service`：直接承载业务逻辑，没有单独 Biz 层。
- `GW:backend/app/admin/service/internal/data`：Ent Repository、认证器、租户检查器等。
- `GW:backend/app/admin/service/internal/server`：REST、SSE、Asynq 三种 Server。
- `GW:backend/pkg`：授权、脚本、事件总线、OSS、JWT、网络安全和任务等可复用库。

**事实：**仓库当前只有 `app/admin/service` 一个实际应用。README 所称“微服务 + 单体切换”更准确地说是：按微服务框架和领域 Proto 预留边界，并通过 `gow extract` 提供未来拆分工具；当前不是多服务部署。

**推断：**源领域/BFF 两层 Proto 对真实微服务拆分和多 BFF 有价值；只有一个 Admin 消费者时，会带来类型跳转、重复 Service 声明和生成配置成本。

## 5. 数据、接口与生成链路

| 维度 | `kratos-admin` | `go-wind-admin` |
| --- | --- | --- |
| 数据模型来源 | 先在开发库建表，再运行 `make gorm-gen` | 先写 Ent Schema，再运行 `make ent` |
| 查询方式 | GORM Gen 强类型字段 + `gorm-kit/repository` | Ent Builder + `go-crud/entgo` 泛型 Repository |
| 事务 | Biz Case 注入 transaction，跨 Repo 编排 | Repo 暴露 Ent Tx，Service 跨 Repo 编排 |
| 结构迁移 | GORM AutoMigrate + 按版本/数据库/数据源嵌入 `.up.sql` | 启动时 Ent `Schema.Create`；另有 MySQL/PostgreSQL 默认/演示数据 SQL |
| Proto | 终端/模块划分，消息和 HTTP 契约同文件 | 源领域完整 RPC + Admin BFF 裁剪 REST 面 |
| Go 生成 | Go、gRPC、HTTP、errors、Agent Tool、MCP Tool | Go、HTTP、OpenAPI；领域和 BFF 分模板 |
| TS 生成 | Buf/ts-proto 分发到 Admin、uni-app、Taro 的 core/module | Buf 生成到 Vben、Element、React 三套 Admin |
| DI | 两套 Wire：公共 Module 与独立宿主 | Admin 服务一套 Wire |
| CRUD 工具 | 管理端有配置、预览、进度、补丁/还原的代码生成产品功能 | `gow generate --dsn ... --orm ent` CLI 脚手架 |

证据：`KA:backend/Makefile`、`KA:frontend/Makefile`、`KA:backend/api/buf.gen.yaml`、`KA:backend/migration/migration.go`、`KA:docs/数据库与初始化数据设计.md`；`GW:backend/Makefile`、`GW:backend/app.mk`、`GW:backend/app/admin/service/internal/data/ent_client.go`、`GW:backend/api` 下各 Buf 模板。

**事实：**KA 当前 42 个 Proto 文件中有 33 个引用 `buf.validate`；GW 当前 97 个 Proto 文件中只有 5 个包含 `validate.rules` 或 `buf.validate`。GW 虽在 REST 链安装了 `validate.Validator()`，其 Server 注释也明确说明多数业务 Proto 尚未补校验规则。证据：两边 Proto 目录的规则引用；`GW:backend/app/admin/service/internal/server/rest_server.go`。

**推断：**

- KA 的数据库优先链路适合已有表和 DBA 主导项目，版本化数据/菜单/API 权限迁移也更明确。
- GW 的 Ent Schema 优先链路让关系、索引、生成查询和测试数据库更自洽，但生产演进仍需要比 AutoMigrate 更明确的版本迁移治理。
- 不建议为了“ORM 可选”同时维护两套完整 Repo；两边代码已经证明 ORM 语义并不等价。

## 6. 产品功能对照

### 6.1 共同且已接线的主干

两边都有用户、角色、组织/部门、岗位/职位、租户、菜单、API、字典、文件、任务、日志、语言、个人中心、JWT 登录、验证码、权限中间件和 Swagger/OpenAPI。对应入口可从以下目录交叉验证：

- `KA:backend/api/proto/system/admin/v1`、`KA:frontend/admin/packages/modules/system/src/views/base`
- `GW:backend/api/protos/admin/service/v1`、`GW:frontend/admin/vue-element/src/pages/app`

### 6.2 `kratos-admin` 当前独有或明显更深

| 能力 | 状态 | 证据 |
| --- | --- | --- |
| AI 会话、流式消息、附件、工具调用、重试/再生成/分支 | 已有完整后端和三端页面；本轮也有修改 | `KA:backend/internal/biz/base/ai`；`KA:frontend/admin/packages/modules/system/src/views/ai/chat` |
| MCP + Agent Tool | 已注册/生成 | `KA:backend/api/buf.gen.yaml`；`KA:backend/internal/module/module.go` |
| 社交 OAuth 登录/绑定 | HEAD 已有，当前继续修改 | `KA:backend/api/proto/base/v1/oauth.proto`；`KA:backend/internal/biz/base/oauth.go` |
| OAuth Client 开放授权、IP/API 白名单、请求响应加密 | **dirty 工作树新增，HEAD 不存在** | `KA:backend/api/proto/base/v1/oauth_client.proto`；`KA:backend/internal/server/middleware/oauth` |
| TOTP + WebAuthn + 恢复码 + MFA 全局策略 | **dirty 工作树新增，HEAD 不存在** | `KA:backend/api/proto/base/v1/mfa.proto`；`KA:backend/internal/biz/base/mfa.go` |
| 管理端代码生成、预览、进度、还原 | 已接线 | `KA:backend/internal/biz/system/admin/codegen`；`KA:frontend/admin/packages/modules/system/src/views/tool/code-gen` |
| 运行日志实时控制台/历史文件 | 已接线，当前有重构 | `KA:backend/internal/biz/system/admin/logstream`；`KA:frontend/admin/packages/modules/system/src/views/tool/runtime-log` |
| 项目 README/docs 构建期聚合 | 已接线 | `KA:backend/internal/module/resources.go`；`KA:scripts/project_docs.py` |
| 动态国际化和多语言 OpenAPI/docs | 已接线且体系化 | `KA:backend/internal/biz/system/admin/base_i18n.go`；`KA:scripts/verify_i18n.py` |
| 站内信、分类、收件箱、已读状态、SSE 推送 | **dirty 工作树已接线，需生产化验收** | `KA:backend/internal/biz/system/admin/base_message.go`；`KA:backend/internal/biz/base/notification.go`；三端消息页面 |
| uni-app 与 Taro 应用底座 | 已接线；H5/微信小程序 | `KA:frontend/uni-app`；`KA:frontend/taro-app` |
| 可发布前后端模块 | 已接线；前端共 10 个 npm 包 | `KA:README.md`；三个前端 workspace 的 `packages` |

### 6.3 `go-wind-admin` 当前独有或明显更深

| 能力 | 状态 | 证据 |
| --- | --- | --- |
| 套餐、模块白名单、配额、租户用量、到期策略 | 已接线 | `GW:backend/api/protos/identity/service/v1/plan*.proto`；`GW:backend/app/admin/service/internal/data/tenant_access_checker.go` |
| 登录策略 | 已接线；支持 IP、设备、时间窗等匹配并有测试 | `GW:backend/app/admin/service/internal/data/login_policy_checker.go` 及其测试 |
| 登录/API/操作/数据/权限/策略评估六类审计 | 已有 Proto、Repo、Service 和页面 | `GW:backend/api/protos/audit/service/v1`；三套前端的 `views/pages/app/log` |
| 令牌查询、撤销、拉黑/解除 | 源领域契约及认证实现可见 | `GW:backend/api/protos/authentication/service/v1/authentication.proto`；`GW:backend/app/admin/service/internal/data/token_checker.go` |
| 站内信、分类、收件箱、已读状态、SSE 推送 | 已接线 | `GW:backend/app/admin/service/internal/service/internal_message_service.go`；三套前端的 `internal_message` 页面 |
| Asynq 持久化任务 | 已接线；备份、租户到期扫描、广播 | `GW:backend/app/admin/service/internal/server/asynq_server.go` |
| 数据备份任务 | 已接线到 Asynq，当前覆盖核心身份/权限/组织表 | `GW:backend/app/admin/service/internal/service/task_service.go`；`GW:backend/app/admin/service/internal/data/backup_repo.go` |
| MinIO 文件管理 | 已接线 | `GW:backend/app/admin/service/configs/oss.yaml`；`GW:backend/app/admin/service/internal/service/file_transfer_service.go` |
| Redis 只读监控 | 已接线 | `GW:backend/app/admin/service/internal/data/redis_cache_monitor_repo.go`；三套前端对应页面 |
| 三套完整 Admin UI | 已接线，功能目录基本对应 | `GW:frontend/admin/{vue-vben,vue-element,react}` |

## 7. 认证、权限与多租户

### 7.1 `kratos-admin`

**事实：**

- 登录按 `tenant_code + username` 查找用户，租户禁用会拒绝登录；访问/刷新令牌由统一 `UserToken` 管理。证据：`KA:backend/internal/biz/base/login.go`。
- 登录前有验证码和一次性验证码票据，密码使用临时公钥加密后再校验哈希。证据：同上及 `KA:backend/internal/biz/base/utils/password_crypto.go`。
- Casbin 策略域使用租户编码，角色菜单映射到 Proto Operation/HTTP Method；角色或菜单变化后重建内存策略。证据：`KA:backend/internal/biz/system/admin/casbin_rule.go`。
- 普通租户用户/角色操作在 Biz 层显式校验租户 ID；默认租户维护租户管理员角色模板。证据：`KA:backend/internal/biz/system/admin/base_user.go`、`base_role.go`。
- 当前工作树的 MFA 支持 TOTP、WebAuthn、恢复码及 `disabled/optional/all_required`；但这组文件未进入 HEAD。证据：`KA:backend/internal/biz/base/mfa.go` 与 `git cat-file HEAD:<path>`。

### 7.2 `go-wind-admin`

**事实：**

- REST 链已安装认证、租户访问检查和 authz 中间件；白名单覆盖登录、注册、验证码、刷新令牌和 MFA 登录挑战。证据：`GW:backend/app/admin/service/internal/server/rest_server.go`。
- 默认 JWT 使用 RS256，刷新令牌通过 HttpOnly Cookie 恢复会话。证据：`GW:backend/app/admin/service/configs/auth.yaml` 及上述 Server 注释。
- Ent Schema 大量使用 `go-crud/entgo/mixin.TenantID`，生成 Runtime 为这些实体装配 Privacy Policy；HTTP Auth 中间件注入 Viewer。证据：`GW:backend/app/admin/service/internal/data/ent/schema/user.go`、`ent/runtime/runtime.go`、`GW:backend/pkg/entgo/viewer`。
- 租户访问检查会校验状态、套餐到期只读策略、API 所属模块和套餐模块白名单，并采用 fail-closed。证据：`GW:backend/app/admin/service/internal/data/tenant_access_checker.go`。
- 授权策略以租户 ID 为 Domain，将角色映射到 API Path/Method。证据：`GW:backend/app/admin/service/internal/data/authorizer_provider.go`。

**关键限制：**默认 `authz.type` 是 `noop`。因此仓库“支持 Casbin/OPA/Zanzibar”表示适配代码存在，不表示克隆后默认部署已经启用 RBAC。证据：`GW:backend/app/admin/service/configs/auth.yaml`。

### 7.3 比较判断

**推断：**

- KA 当前权限模型较直接：租户 + 角色 + 菜单 + Operation；适合现有模块化系统，但租户隔离依赖 Biz 明确过滤与底层库行为，审查时要逐业务确认。
- GW 将 Viewer/Ent Privacy、租户套餐检查、角色 API 策略分为多道闸，治理维度更丰富；但默认 Noop 授权削弱了开箱安全性，GORM 备用实现也不具备同等租户隔离。
- GW 的套餐白名单适合 SaaS 商业化；如果 KA 没有套餐售卖场景，只吸收到期状态、只读策略和配额接口即可，不必照搬全部 Plan 模型。

## 8. “宣称 / 契约 / 已接线”核验

### 8.1 `go-wind-admin` 易被高估的能力

| 项目宣称或目录 | 契约/代码情况 | 实际运行状态 | 结论 |
| --- | --- | --- | --- |
| Ent + GORM 双 ORM | `internal/data/gorm` 有大量平行 Repo 和 Model | 文件带 `gorm_backend` build tag；ProviderSet 只注册 `NewEntClient` 和 Ent Repo；多处注释明确“死代码、不接 Wire”，且部分方法直接返回 not implemented；还明确警告没有租户隔离 | **生产主链只有 Ent；GORM 是不完整脚手架，不能热切换** |
| MFA 多方式 | Proto 定义 TOTP、SMS、Email、WebAuthn、备份码 | `mfa_service.go` 顶部明确“本轮仅落地 TOTP”，其他方法/方式返回未实现 | **当前可用能力是 TOTP，不应宣称完整多因素矩阵** |
| OAuth Service | `authentication/service/v1/oauth.proto` 定义完整领域服务 | 没有 `admin/service/v1/i_oauth.proto`，没有 OAuth Service 实现，也未在 `NewRestServer` 注册 | **只有领域契约，没有 Admin REST 能力** |
| Casbin/OPA/Zanzibar | 配置和 Authorizer 适配存在 | 默认 `authz.type: noop` | **可选引擎框架存在，默认授权未启用** |
| Lua/JavaScript Hook 插件 | `backend/pkg/scripting` 实现丰富且有大量测试/示例 | 仓库外部没有生产代码导入该包；`app/admin/service/scripts` 主要是示例和接入文档 | **是可复用库/原型，不是当前 Admin 生命周期能力** |
| 微服务/单体切换 | 领域 Proto 和 `gow extract` 支持未来拆分 | 当前只有一个 Admin App、一个 Wire 组合 | **具备演进工具，不是现成多服务部署** |
| 全局 Proto 校验 | REST 链安装 `validate.Validator()` | Server 注释明确多数业务 Proto 尚未补验证规则 | **校验基础设施存在，规则覆盖尚不完整** |

主要证据：`GW:backend/app/admin/service/internal/data/providers/wire_set.go`、`GW:backend/app/admin/service/internal/data/gorm/operation_audit_log_repo.go`、`GW:backend/app/admin/service/internal/data/gorm/plan_quota_repo.go`、`GW:backend/app/admin/service/internal/service/mfa_service.go`、`GW:backend/app/admin/service/internal/server/rest_server.go`。

### 8.2 `kratos-admin` 当前快照需标注的能力

| 能力 | 当前工作树 | HEAD | 报告口径 |
| --- | --- | --- | --- |
| TOTP/WebAuthn/恢复码/MFA 策略 | Proto、Biz、Repo、三端 UI、配置均存在且已接线 | 关键源文件在 HEAD 中不存在 | **未提交快照能力，不能当作 v0.0.29 已发布能力** |
| OAuth Client 开放授权 | Proto、Biz、Repo、管理页面、Operation/加密 Filter 存在 | 关键源文件在 HEAD 中不存在 | **未提交快照能力** |
| 社交 OAuth | 原能力存在，本轮继续修改 | HEAD 已有 | **已有能力，但当前快照行为可能尚未稳定** |
| Admin Queue Consumer | Core 提供 Queue Server 抽象；当前工作树已注册站内信 Dispatch consumer | `base.message.dispatch`、`MessageDispatchTask`、Dispatch 租约、失败退避和积压摘要已接线 | **站内信消费者已接入；Redis 故障演练仍待补齐** |
| README“已实现能力” | 已更新为当前工作树状态 | README 本身 dirty | **反映开发中目标，不代表已发布标签** |

主要证据：`KA:git status` 对应路径、`KA:backend/internal/module/module.go`、`KA:backend/api/proto/base/v1/mfa.proto`、`KA:backend/api/proto/base/v1/oauth_client.proto`。

## 9. 工程化、部署与测试

### 9.1 命令与生成

**事实：**

- KA 根 Makefile 统一编排后端、三个前端、i18n、OpenAPI、构建、打包、Docker、标签与 npm 发布。证据：`KA:Makefile`、`KA:frontend/Makefile`、`KA:backend/Makefile`。
- GW 后端 Makefile 覆盖 Ent/Wire/API/OpenAPI/测试/覆盖率/lint/Docker/环境安装；三套前端各自维护脚本。证据：`GW:backend/Makefile`、三个前端 `package.json`。
- KA 有版本化、多语言、可嵌入的迁移资源；GW 使用 Ent AutoMigration 和两种数据库的默认/演示 SQL。证据：`KA:backend/migration/assets`；`GW:backend/sql`。
- KA 有 npm Trusted Publishing workflow；GW 仓库当前没有 CI workflow，但有 Issue/PR/Security/Contributing 模板。证据：两边 `.github`。

### 9.2 部署

| 项目 | 已提供 |
| --- | --- |
| KA | 单镜像同时打包后端和 Admin/uni-app/Taro 三套静态站点；映射运行配置和数据；暴露 HTTP + gRPC |
| GW | Docker Compose 启动 PostgreSQL、Redis、MinIO、Admin；支持 libs-only/full-deploy；Unix/Windows 环境脚本；PM2；三套前端各自 Dockerfile |

证据：`KA:backend/Dockerfile`、`KA:Makefile`；`GW:backend/docker-compose.yaml`、`GW:backend/scripts`、三套前端 `scripts/deploy`。

**推断：**GW 的本地依赖环境更完整，KA 的单镜像交付更紧凑。KA 可优先补一个“仅基础设施”的 Compose，而不必改掉现有应用镜像策略。

### 9.3 测试

| 项目 | Go 测试文件 | 前端测试文件 | 特点 |
| --- | ---: | ---: | --- |
| KA | 5 | 21 | 后端新增登录失败策略、OAuth IP 白名单、站内信正文清洗、Dispatch 退避和 Publisher 适配纯规则回归测试；前端覆盖模块协议、导航、构建 runner、CLI scaffold、密码加密和 AI stream |
| GW | 42 | 32 + 1 个快照 | Go 覆盖认证、登录策略、租户 Repo、MFA、权限、网络安全、脚本、任务；前端主要是 Vben 上游/框架基础测试 |

证据：两个仓库 `rg --files -g '*_test.go' -g '*.test.*' -g '*.spec.*'` 的当前结果；`KA:backend/AGENTS.md` 已允许关键规则使用 `_test.go`。

**推断：**GW 的真正优势是后端关键规则有可执行回归样例，而不是前端测试数量。KA 当前最复杂、风险最高的 MFA、OAuth Client、租户角色同步、代码生成补丁/还原、迁移与国际化流程恰好缺少 Go 自动化回归保护。

## 10. 建议吸收清单

### P0：优先吸收

#### 1. 建立后端关键规则测试层

**已落地：**已调整 `KA:backend/AGENTS.md`，允许为纯规则和安全边界编写小而稳定的 `_test.go` 测试；当前先覆盖登录失败策略、OAuth IP 白名单、站内信正文清洗、Dispatch 退避和 Publisher 适配，不追求仓库级覆盖率数字。

优先对象：

- MFA 登录挑战次数、恢复码一次性消费、WebAuthn challenge/tenant/user 绑定。
- OAuth Client IP/API 白名单、密文协议、客户端禁用和租户禁用。
- 租户角色模板同步、Casbin 策略重建、跨租户拒绝。
- 代码生成 patch/revert、迁移版本幂等、i18n 占位符/回退。

参考但不照抄：`GW:backend/app/admin/service/internal/data/login_policy_checker_test.go`、`mfa_service_test.go`、`permission_service_test.go`、`GW:backend/pkg/netutil/*_test.go`。

#### 2. 登录策略与分层审计

**已落地：**登录接口已增加按租户、账号和对端地址隔离的失败次数锁定；用户禁用、删除、重置密码或角色关系变更会撤销已有令牌；日志审计已拆分为登录、API、操作、数据访问、权限、策略评估六张 `base_*_log` 表，并由统一审计中间件写入脱敏请求事实。日志菜单只授予平台管理角色，普通租户没有查询权限；旧 `base_log` 仅保留为 Core 队列兼容模型，不再作为系统日志入口。

设计参考：

- `GW:backend/api/protos/authentication/service/v1/login_policy.proto`
- `GW:backend/app/admin/service/internal/data/login_policy_checker.go`
- `GW:backend/api/protos/audit/service/v1`
- `GW:backend/app/admin/service/internal/server/rest_server.go` 的日志中间件接线

KA 保留 runtime log、SSE 和模块资源机制，同时按审计事实拆分六张独立表和管理端页面；策略评估日志不再通过 `base_log.audit_type` 伪装，避免不同事实在一张表中互相污染。

#### 3. 完成关键异步流程的持久化可靠性

**已落地：**复用 KA 已接入的 Queue 抽象和 Redis Streams consumer；Dispatch 租约、重复消费保护、到期恢复、指数退避和积压摘要已落地，失败最多自动尝试 5 次后进入 FAILED，下一步完成 Redis 故障演练，再扩展到备份、代码生成长任务等可重试流程。

证据差距：KA 当前已有 `base.message.dispatch` consumer、`MessageDispatchTask`、Dispatch 租约并发控制、失败退避和积压摘要；Redis 故障演练尚未完成。GW 在 `backend/app/admin/service/internal/server/asynq_server.go` 已处理注册、周期恢复、失败重试和广播幂等。

### P1：业务需要明确时吸收

#### 4. 租户套餐、配额与到期策略（暂不实施）

**按用户要求跳过：**当前不引入 SaaS 套餐、模块白名单、配额或租户到期策略，保留现有租户与权限模型。

参考：`GW:backend/app/admin/service/internal/data/tenant_access_checker.go`、`GW:backend/api/protos/identity/service/v1/plan*.proto`。

#### 5. 站内信与通知中心生产化

**已落地部分：**在 KA 已有四表、SSE 和三端收件箱基础上，已补公开 Publisher、幂等发布、后端富文本白名单清洗、Dispatch 失败退避、积压摘要和过期收件分批清理；Redis/SSE 故障演练仍待补齐，不只移植 Admin 页面。

参考：`GW:backend/api/protos/internal_message/service/v1`、`GW:backend/app/admin/service/internal/service/internal_message_service.go`。

#### 6. 备份、Redis 监控和依赖 Compose

**建议：**

- 将核心表逻辑备份作为受控任务，明确加密、保留期、对象存储和恢复演练。
- Redis 仅开放 INFO/DBSIZE/慢日志等只读能力。
- 增加 MySQL/Redis/可选对象存储的 libs-only Compose，保持现有单镜像部署不变。

参考：`GW:backend/app/admin/service/internal/data/backup_repo.go`、`redis_cache_monitor_repo.go`、`GW:backend/docker-compose.libs.yaml`。

### P2：条件成熟再评估

#### 7. 源领域 Proto + BFF 分层

只有出现第二个独立服务、第二个真正不同的 BFF，或需要限制外部 API 面时再引入。KA 已按 Admin/App 终端拆 Proto，当前直接复制 GW 两层结构会增加生成和类型维护成本。

#### 8. 脚本 Hook 平台

先确认真实扩展场景、租户隔离、资源配额、超时、沙箱和审计要求。GW 的脚本包值得参考 API/沙箱测试，但它自己尚未接入 Admin 运行链，不能作为成熟产品直接移植。

参考：`GW:backend/pkg/scripting`、`GW:backend/app/admin/service/scripts/README.md`。

### 明确不建议直接照搬

1. **双 ORM。**GW 已明确展示了平行 Repo 的语义漂移、未实现方法和租户隔离风险；KA 应继续收敛 GORM/Gen 主链。
2. **三套 Admin UI。**KA 已承担 Admin + uni-app + Taro 三端，新增 Vben/React Admin 会把业务页面维护面进一步放大。
3. **默认 Noop 授权。**可插拔引擎不能以不安全默认值为代价；新环境应 fail-closed，并在启动时验证策略引擎已启用。
4. **先写大而全 Proto、后补实现。**GW 的 OAuth 和 MFA 契约说明这种做法容易让 README、生成代码和运行能力失真；KA 应延续“契约、实现、前端、迁移一起闭环”。
5. **未接线的脚本/事件总线。**只有落到实际生命周期和业务 Hook，才值得作为产品能力维护。

## 11. 推荐实施顺序

```text
P0-1 关键后端测试
  -> P0-2 登录策略 + 登录/API/操作审计
  -> P0-3 队列租约、幂等、重试与监控
  -> P1-5 站内信生产化收尾（复用 SSE，覆盖三端）
  -> P1-6 备份/监控/依赖 Compose
  -> P2 Proto BFF 或脚本 Hook（有真实边界后）
```

**建议：**第一阶段不要同时调整 ORM、Proto 分层和前端框架。先以当前 KA 模块边界落地测试、审计和持久化任务，能用最小架构扰动获得最大收益。

## 12. 最终判断

**事实总结：**

- KA 在协议种类、AI/MCP、跨端应用、动态国际化、文档/代码生成、模块发布和三端站内信接入上明显领先。
- GW 在审计、登录策略、租户商业化治理、站内信、持久化任务和 Go 测试上明显领先。
- GW 的 Ent、REST/SSE/Asynq 是实际主链；GORM、OAuth、多方式 MFA、脚本插件和多 authz 引擎不能全部按 README 字面理解。
- KA 当前最亮眼的 WebAuthn、恢复码、MFA 策略、OAuth Client 和站内信仍属于 dirty 工作树；虽然生成与自动化校验已通过，仍需整理提交边界并完成真实数据库/Redis 端到端验证后才能视为稳定能力。

**推断总结：**两个项目不是简单的“谁功能更多”。KA 的强项是平台化与多端组合，GW 的强项是传统企业后台的治理深度。把 GW 的测试、审计、登录策略、持久化异步任务和可选的 SaaS 配额吸收到 KA 现有架构中，比迁移到 Ent、维护三套 Admin 或复制未接线插件更合理。
