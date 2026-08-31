# go-wind-admin 对 kratos-admin 的补充差异核对

> 比较日期：2026-08-29（Asia/Shanghai）
> 范围：补充现有 `research/go-wind-admin-comparison.md` 未单独展开的认证/会话、权限、审计、任务、存储、监控、前端和测试差异。
> 口径：`KA` 为 `kratos-admin`，`GW` 为 `go-wind-admin`；“当前”指工作树快照，KA 工作树含大量未提交改动。本文只给迁移建议，不修改业务代码。

## 结论摘要

| 优先级 | 建议 | 当前差异 | 迁移边界 |
| --- | --- | --- | --- |
| P0 | 将 Wind 的关键规则测试方法继续吸收到 KA | GW 对 token cache、登录策略、权限、SQLite/网络安全有独立回归测试；KA 当前已补密码、登录来源、文件、OAuth IP、备份等测试，但会话撤销、审计队列和多租户失败路径仍缺少端到端覆盖 | 迁移测试思路和夹具，不迁 Ent/Asynq 实现 |
| P1 | Dashboard 只读概览 | GW 有四张统计卡、登录趋势和两类分布；KA 没有 Dashboard Proto、Service 或页面 | 复用 KA 审计表和现有 OpsMonitoring，避免引入另一套仪表盘框架 |
| P1（按产品需要） | 文件元数据目录与流式传输 | GW 有文件表 CRUD、内容 hash、桶/目录元数据和下载传输；KA 目前只有 base File 上传/下载字节接口 | 先做受租户隔离的文件元数据和分页查询，再评估 range/presign；不要直接复制已禁用的 presign 分支 |
| P1（按产品需要） | 细粒度登录策略 | GW 策略按租户/用户保存，支持 IP、MAC、REGION、TIME、DEVICE；KA 当前是平台级配置，支持 IP/CIDR、时间窗和设备黑白名单 | 仅在确有“按租户/用户限制”需求时扩展数据模型；MAC/REGION 需要可信采集和地理库 |
| P1/P2（模块接入频率） | API 目录从 OpenAPI/路由同步 | GW 有 `SyncApis`（实际主路径读 OpenAPI）和 `GetWalkRouteData` 调试接口，可把运行契约补入权限 API 目录；KA BaseApi 当前没有同步 RPC | 优先复用 KA OpenAPI/Module 注册边界实现幂等同步，不迁源领域/BFF 结构 |
| P0（若尚未等价接入） | 登录失败限流 | GW 有 Redis IP+账号双维度失败计数，5 次后锁 15 分钟；来源策略（IP/设备/时间）不能替代暴力破解限流 | 仅迁移原子计数/锁定语义和测试，按 KA Cache/错误协议接入 |
| P2 | 单设备会话清单和单令牌撤销 | GW 领域契约有令牌查询、按 JTI 撤销、拉黑/解除；KA 当前会话接口是“查看当前 + 撤销当前用户全部” | 先确认管理端/个人中心产品需求；不把 GW 未暴露到 Admin BFF 的领域 RPC 当作现成功能 |
| P2 | 细粒度权限组、时间生效角色和选定组织单元 | GW 有 PermissionGroup、角色权限点、MembershipRole 生效区间和 SELECTED_UNITS；KA 已有基础角色数据范围（全部/部门及子部门/本部门/本人） | 复杂组织场景再建模型，不能只搬 Proto 字段而不改授权查询 |
| P2 | 任务选项增强 | GW TaskOption 支持 max retry、timeout、deadline、delay、unique、retention、group、task ID；KA BaseJob 主要是 cron、参数、启停和手动执行 | 若 Core cron 没有对应语义，只在队列边界增加经过验证的能力；不整套迁 Asynq |
| P2（按合规需要） | 审计防篡改增强 | GW 登录/操作日志计算风险分、SHA256、ECDSA 签名并记录设备/地理信息；KA 已有六类审计、SQL 脱敏/分类、异步落库、JSONL+SHA/HMAC 归档 | 只吸收字段和验证策略；先解决密钥托管、隐私、签名验真和失败处理 |

## 1. 认证与会话

### 1.1 Wind 的单令牌管理契约尚未完整对应

GW 的认证领域服务除登录、登出、刷新外，还声明 `ValidateToken`、`GetAccessTokens`、`RevokeTokenById`、`BlockToken`、`UnblockToken`（`GW:backend/api/protos/authentication/service/v1/authentication.proto:37-50`）。`TokenChecker` 通过认证器校验 token（`GW:backend/app/admin/service/internal/data/token_checker.go:34-65`），缓存层还按 JTI 提供单令牌撤销和黑名单键（`GW:backend/app/admin/service/internal/data/user_token_cache.go:120-193`）。

KA 当前工作树已经有 `BaseSessionService`，可以查询当前会话并撤销当前用户全部令牌（`KA:backend/api/proto/system/admin/v1/base_session.proto:9-23`；`KA:backend/internal/biz/system/admin/base_session.go:29-66`），用户禁用、改密等入口也会撤销用户令牌。因此真正尚缺的是“列出我的多个设备/令牌并撤销单个 JTI”，不是基础 token 校验。

**建议：P2。**只有产品需要设备会话管理、管理员踢出单设备或安全中心展示时再增加持久化/Redis 索引和审计；先补 token cache 的跨实例、过期、并发撤销测试。GW 这些 RPC 只在领域 Proto 中出现，Admin BFF `i_authentication.proto` 目前主要暴露登录、登出、注册、刷新和验证码（`GW:backend/api/protos/admin/service/v1/i_authentication.proto:11-73`），不能把它们当作现成的管理页面能力直接搬运。

### 1.2 登录策略是“全局配置”与“按对象策略”的产品差异

GW 的 `LoginPolicy` 有 `target_id`、`tenant_id`、黑/白名单类型，以及 IP、MAC、REGION、TIME、DEVICE 方法（`GW:backend/api/protos/authentication/service/v1/login_policy.proto:16-56`、`:66-106`）；匹配器明确合并全局和定向用户条目，并支持 CIDR、跨午夜时间窗、设备匹配（`GW:backend/app/admin/service/internal/data/login_policy_checker.go:10-73`）。

KA 当前工作树已有 `BaseLoginPolicyService`，策略保存在系统配置并刷新缓存（`KA:backend/api/proto/system/admin/v1/base_login_policy.proto:9-40`；`KA:backend/internal/biz/system/admin/base_login_policy.go:22-106`），字段为平台级 IP 黑/白名单、时间窗和设备黑/白名单。它没有 Wind 的 per-tenant/per-user 目标，也没有 MAC/REGION 判定。

**建议：P1（按产品需要）。**SaaS 或高安全场景需要不同租户/用户登录边界时再迁移“目标维度 + 策略表”；MAC 在普通 HTTP 请求中并不可信，REGION 还依赖 IP 地理库，不能只照抄枚举。

### 1.3 口令策略无需迁移

GW 的口令包是环境变量控制的复杂度、90 天有效期和最近 3 条历史口令（`GW:backend/pkg/password/policy.go:1-77`，历史写入 `credential.extra_info` 见 `GW:backend/app/admin/service/internal/data/user_credential_password_policy.go:18-92`）。KA 当前工作树已有等价且更细的实现：复杂度、历史口令、有效期、缓存故障区分和数据库修改时间检查（`KA:backend/internal/biz/base/password/policy.go:21-57`、`:90-244`）。不建议再复制 Wind 包或改变 KA 的错误/缓存语义。

## 2. 权限和多租户

### 2.1 细粒度权限组与角色生效范围

GW 有独立 `PermissionGroupService` CRUD 和树形分组模型（`GW:backend/api/protos/permission/service/v1/permission_group.proto:13-84`），角色直接携带权限点 ID 列表（`GW:backend/api/protos/permission/service/v1/role.proto:45-106`）。成员-角色关联还带主角色、状态、`start_at/end_at` 和 `data_scope`（`GW:backend/api/protos/permission/service/v1/membership_role.proto:10-73`），`DataScope` 可选本人、组织单元、递归子组织或选定组织单元（`GW:backend/api/protos/identity/service/v1/types.proto:5-16`）。

KA 当前已有角色数据范围枚举“全部/部门及子部门/本部门/本人”（`KA:backend/api/proto/system/admin/v1/base_role.proto:75-87`），并将角色菜单写入 Casbin/用户令牌上下文；没有 Wind 的权限组实体、选定组织单元列表和角色生效起止时间。

**建议：P2。**除非需要复杂组织授权或临时角色，否则不引入第二套 Permission/Role 关系。若实施，必须同步查询过滤器、Casbin/授权上下文、迁移、菜单/API 权限和审计，不可只增加 Proto 字段。

### 2.2 AdminPortal 初始上下文是调用优化，不是新的权限模型

GW 的 `AdminPortalService` 契约同时声明菜单路由、权限码和一次性 `GetInitialContext`（`GW:backend/api/protos/admin/service/v1/i_admin_portal.proto:10-31`）；菜单查询还按多角色合并菜单并可按套餐模块白名单过滤（`GW:backend/app/admin/service/internal/service/admin_portal_service.go:83-96`、`:165-241`）。但当前 `admin_portal_service.go` 只看到 `GetNavigation`/`GetMyPermissionCode` 的具体实现，`GetInitialContext` 由嵌入的生成接口兜底，调用时可能返回未实现或触发 nil 接口问题；这项能力应先补齐实现再作为参考。

KA `AuthService` 当前把菜单树、按钮列表、用户信息拆成多个 RPC（`KA:backend/api/proto/system/admin/v1/auth.proto:12-72`），前端模块依靠现有动态路由和权限请求。一次性聚合可减少冷启动往返，但会放大缓存失效和权限变更后的陈旧窗口。

**建议：P2。**只在真实测量显示初始化请求过多时增加聚合接口；复用 KA 的 `AuthCase` 和模块路由模型，不复制 GW 的源领域/BFF 层。

### 2.3 租户用量与清理是套餐治理的配套缺口

GW Tenant BFF 除 CRUD 外还声明 `GetUsage` 和 `CleanupData`（`GW:backend/api/protos/admin/service/v1/i_tenant.proto:51-73`）；`TenantUsageRepo` 聚合用户、文件、API 审计用量，并实现跨表租户清理和到期策略（`GW:backend/app/admin/service/internal/data/tenant_usage_repo.go:54-143`、`:165-302`）。KA 当前 `BaseTenantService` 只有租户基础信息和管理员初始化，没有用量/清理 RPC。该能力与 Wind Plan/Quota、到期只读/阻断策略强耦合。

**建议：P1（SaaS/合规场景）；当前可继续不实施。**若产品不售卖套餐，不建议为了“功能对齐”引入四表聚合和跨表删除；若未来引入套餐，应先定义用量口径、清理可恢复性、异步任务和租户管理员权限，再迁移。

### 2.4 API 目录同步是模块化部署中的治理缺口

GW API 领域和 Admin BFF 都声明 `SyncApis`、`GetWalkRouteData`（`GW:backend/api/protos/permission/service/v1/api.proto:16-39`；`GW:backend/api/protos/admin/service/v1/i_api.proto:45-60`）。实现的 `SyncApis` 当前会清空并按内嵌 OpenAPI 文档同步（`GW:backend/app/admin/service/internal/service/api_service.go:145-231`）；`syncWithWalkRoute` 代码路径被注释，`GetWalkRouteData` 仅用于通过 `RouteWalker` 调试查看路由（`:148-153`、`:233-260`）。Vben 页面有显式同步入口（`GW:frontend/admin/vue-vben/apps/admin/src/views/app/permission/api/index.vue:17-25`、`:181-215`）。

KA `BaseApiService` 当前只提供 Option/Page/Get/GetApiDoc/Update/Agent 状态/MCP 状态/OpenAPI 服务选项（`KA:backend/api/proto/system/admin/v1/base_api.proto:12-58`），前端 API 页面也只有查询、详情和状态编辑（`KA:frontend/admin/packages/modules/system/src/views/base/api/index.vue:307-407`）。在 KA 的可挂载外部 Module 场景中，新 HTTP/gRPC/MCP 操作如果没有同步流程，可能出现“路由已存在但无权限点”或“旧权限点残留”。

**建议：P1（外部模块经常动态接入）/P2（接口主要由迁移固定维护）。**建议实现基于 Core 已注册路由/OpenAPI 的幂等同步，保留手工描述和启用状态，记录删除/失效 API 的审计；不要复制 GW 的源领域/BFF 两层 Proto。GW 当前先 `Truncate` 再写入，且忽略截断错误（`:145-153`），OpenAPI 解析失败可能清空现有 API；KA 若实现应采用临时表/事务或 upsert+失效标记，不能照搬该顺序。

### 2.5 登录来源策略不等于失败限流

GW 的 `LoginRateLimiter` 用 Redis Lua 脚本按 IP 和用户名分别原子计数，达到 5 次失败后在 15 分钟窗口内锁定，并提供 `IsLocked`/`Reset`（`GW:backend/app/admin/service/internal/data/login_rate_limiter.go:13-50`、`:53-150`）。这层防护独立于 `LoginPolicy` 的 IP/设备/时间黑白名单。

KA 当前工作树已有登录来源策略（`KA:backend/internal/biz/base/loginpolicy/policy.go`），它解决的是“来源是否允许”，不是失败次数/暴力破解锁定。若当前 `LoginCase` 尚未接入等价的失败计数器，应按 KA 的 Cache 接口和统一错误原因补上，并覆盖 IP、账号、Redis 故障降级、成功清零和并发阈值测试；若并行改动已接入，则只需保留这些回归测试。

**建议：P0（安全基线）。**不要把限流键直接照搬 `gowind:*`，应纳入 KA 命名空间并确认多租户账号键是否需要 tenant_code，避免同名账号跨租户互相锁定或绕过。

## 3. 审计与安全证据

### 3.1 Wind 有日志风险评分和签名字段，但实现成本/风险较高

GW 登录审计中间件填充设备、地理位置、风险分和风险因素，并计算日志 hash 与 ECDSA 签名（`GW:backend/pkg/middleware/logging/login_audit_log.go:87-137`、`:146-219`、`:222-380`）。操作审计同样生成 SHA256 和 ECDSA 签名，并只对写方法落库（`GW:backend/pkg/middleware/logging/operation_audit_log.go:78-139`、`:142-212`）；数据访问日志从脱敏 SQL 提取表名和数据分类（`GW:backend/pkg/middleware/logging/data_access_audit_log.go:51-99`）。

KA 当前工作树的审计中间件已覆盖 API、登录、操作、数据访问、权限和策略评估六类，并通过 Core emitter/Redis queue 异步落库（`KA:backend/internal/server/middleware/auditlog/auditlog.go:44-93`、`:114-177`、`:179-225`）；数据访问模型已经有脱敏 SQL、SQL 指纹、表名、数据范围等字段（`KA:backend/internal/data/gen/models/base_data_access_log.gen.go:13-42`）。KA 的审计留存任务还生成 JSONL 和 SHA/HMAC sidecar 后再删除在线行（`KA:backend/internal/task/system/admin/audit_retention.go:31-110`、`:215-245`）。

**建议：P2（按合规需要）。**可参考 Wind 的风险因素字段和签名验证，但不要直接迁移 ECDSA 私钥配置、`fmt.Printf` 错误路径或未定义的验真协议；先明确密钥轮换、签名失败是否阻断落库、PII 最小化和离线验真工具。当前 KA 的 HMAC 归档完整性已比 Wind 的归档仓库更适合作为基础。

### 3.2 不建议迁移 Wind 的审计归档实现

GW 的归档仓库按批导出 JSONL 后删除在线行，但删除失败时会留下已落盘文件并可能在重跑时重复（`GW:backend/app/admin/service/internal/data/audit_log_archive_repo.go:185-234`），该实现没有 KA 当前任务中的完整性 sidecar 和“校验成功后删除”约束。迁移价值仅限于表级批处理思路，不能整段替换 KA 任务。

## 4. 任务和异步执行

### 4.1 Wind 的 TaskOption 比 KA BaseJob 丰富

GW `TaskOption` 支持最大重试、超时、截止时间、延迟/指定时间、唯一锁定、结果保留、分组和自定义任务 ID（`GW:backend/api/protos/task/service/v1/task.proto:52-125`），Asynq Server 注册备份、租户到期、审计归档和站内信广播处理器（`GW:backend/app/admin/service/internal/server/asynq_server.go:17-63`）。

KA `BaseJobService` 当前契约是 cron、参数、启停、立即执行和执行日志（`KA:backend/api/proto/system/admin/v1/base_job.proto:12-94`），Core 任务集合已经接入消息投递、审计留存、备份和 i18n 任务（`KA:backend/internal/task/init.go:20-38`；`KA:backend/internal/module/wire_gen.go:352-390`）。

**建议：P2。**如需延迟任务、独立重试策略或幂等任务 ID，先在 KA 的 Core cron/queue 边界确认语义；不要因为 Wind 使用 Asynq 就替换 KA 的 Redis Streams 和 Core 生命周期。Wind 的任务选项还涉及用户可控 payload，迁移前需补权限、资源配额和审计。

### 4.2 备份实现是数据库取舍而非直接功能缺口

KA 当前备份任务使用 `mysqldump`，要求显式开启、加密密钥和完整性密钥，默认 7 份轮换（`KA:backend/internal/task/system/admin/backup.go:22-112`、`:190-225`）；GW 提供 PostgreSQL `pg_dump -Fc` Docker/本地脚本，默认保留 30 份（`GW:backend/scripts/backup/pg_backup.sh:1-53`）。

**建议：P2（仅 PostgreSQL/30 日恢复点需求）。**可吸收“容器内/本地 dump + 恢复演练”流程，但不要复制 GW 脚本中的默认数据库密码（`:19-26`），也不能把未加密物理备份当成 KA 现有加密备份的替代品。KA 当前代码仍需独立补恢复演练/校验命令，但这不是 Wind Asynq 代码的直接移植。

## 5. 存储和文件传输

GW 的文件领域服务提供文件列表、详情、创建、更新、删除和数量统计，实体记录供应商、桶、目录、原/存储文件名、大小、内容 hash、链接和租户（`GW:backend/api/protos/storage/service/v1/file.proto:15-49`、`:51-145`）。文件传输服务另有流式下载、PUT/POST 上传、range、Content-Disposition、MIME 和 presign 选项（`GW:backend/api/protos/storage/service/v1/file_transfer.proto:11-70`）。实际直传路径会嗅探 MIME、限制大小、拒绝目录穿越并记录元数据（`GW:backend/app/admin/service/internal/service/file_transfer_service.go:141-226`）；预签名上传明确返回未实现，因为无法可靠记录租户/用户/hash（`:229-237`）。

KA 的 FileService 只定义多文件上传、单文件上传和下载字节（`KA:backend/api/proto/base/v1/file.proto:11-30`），FileCase 已有租户路径约束、内容/扩展名校验和 OSS 上传下载（`KA:backend/internal/biz/base/file.go:53-127`），但没有可分页的文件元数据目录或独立文件管理页面。

**建议：P1（文件治理需求）/P2（大文件性能）。**优先迁移“元数据表 + 租户隔离 + hash/大小/MIME + 删除孤儿对象清理”；range/presign 需结合 KA Core HTTP handler 和三端上传器设计。Wind 的 presign 分支本身未实现，不能作为现成功能迁移。

补充安全边界：GW 在按外部 URL 下载时做 scheme/host 校验、解析后逐 IP 阻断内网、固定拨号 IP、重定向复检和响应体上限（`GW:backend/app/admin/service/internal/service/file_transfer_service.go:254-339`）。KA 当前下载只接受租户路径/OSS 对象，因此无需迁移；若未来允许代理外链，必须先吸收这类 SSRF 防护和测试，不能直接使用 `http.Get`。

## 6. 监控、前端和测试

### 6.1 Dashboard 与 Redis 监控要区分

GW Dashboard 提供四张概览卡、近 N 天登录趋势、操作 action 分布和登录 status 分布（`GW:backend/api/protos/admin/service/v1/i_dashboard.proto:9-81`；`GW:backend/app/admin/service/internal/service/dashboard_service.go:35-112`），React 页面消费四个查询并展示图表（`GW:frontend/admin/react/src/pages/app/dashboard/index.tsx:18-85`）。KA 当前没有对应 Dashboard Proto/Service/页面，但已有 OpsMonitoring 的运行、流量、服务、存储、端点、节点和告警视图，且其中已采集 Redis INFO/DBSIZE/slowlog（`KA:backend/api/proto/system/admin/v1/ops_monitoring.proto:10-43`；`KA:backend/internal/biz/system/admin/ops_monitoring.go:178-372`）。

**建议：P1 只迁 Dashboard 的业务统计查询；不迁 GW 的 RedisCacheMonitor 页面或重复运维监控。**查询必须沿用 KA 的租户/平台权限和审计表，避免把跨租户总数泄露给普通租户。

### 6.2 三套桌面 Admin 不值得整体迁移

GW 同时维护 Vue Vben、Vue Element 和 React 三套页面，Dashboard/文件/权限组/日志等页面在各套实现中重复；KA 管理端采用 core + System 模块，同时维护 uni-app 和 Taro 应用端。迁移整套前端会复制业务页面、路由、语言包和 RPC 生成维护面。建议只按需参考 GW 的 Dashboard 图表、文件目录交互，不引入第二套桌面框架。

### 6.3 测试应迁移“边界样例”，不迁测试数量

GW 测试目录覆盖 token cache、登录策略、认证器、权限、SQLite 兼容、SSRF/客户端 IP、密码、脚本沙箱、备份轮换等（例如 `GW:backend/app/admin/service/internal/data/user_token_cache_test.go`、`login_policy_checker_test.go`、`sqlite_compat_test.go`、`GW:backend/pkg/netutil/ssrf_test.go`、`password/policy_test.go`、`task/backup_test.go`）。KA 当前工作树已经有登录、登录来源策略、文件、OAuth IP、消息和备份测试（例如 `KA:backend/internal/biz/base/login_test.go`、`backend/internal/biz/base/loginpolicy/policy_test.go`、`backend/internal/biz/base/file_test.go`、`backend/internal/server/middleware/oauth/ip_test.go`、`backend/internal/task/system/admin/backup_test.go`），但仍应补：多会话单 JTI 撤销、Redis/队列故障恢复、跨租户 Dashboard/文件查询、审计归档删除失败和备份恢复演练。

## 7. 明确不建议迁移项

1. **双 ORM 和 Ent Privacy 全套。**GW 的 GORM 目录多处是脚手架或未实现桩；KA 应继续使用 GORM Gen/版本化迁移。
2. **Asynq 替换 Core cron/Redis Streams。**两边生命周期、队列语义和可嵌入边界不同；只吸收任务幂等、重试和故障测试。
3. **Wind 的默认密码备份脚本。**脚本把默认连接密码写在环境默认值中，且输出未加密；只参考 Docker/local dump 流程。
4. **未接线的 Lua/JavaScript scripting 与 eventbus。**GW `backend/pkg/scripting` 和 `eventbus` 有实现/测试，但当前 Admin 生命周期未形成必须依赖；先确认租户隔离、沙箱、超时和审计边界。
5. **三套 Admin UI。**会显著增加页面、语言、路由、RPC 和回归维护成本，不能作为通用功能迁移。
6. **完整 PermissionGroup/RoleOverride 模型。**只有复杂组织或临时授权需求成立时才值得增加；否则 KA 现有角色菜单 + 数据范围更小、更易验证。
7. **字典导入/导出“宣称”。**GW 两套 Dict Proto 没有 import/export RPC；Vben 仅在 VxeGrid toolbar 打开 `import/export`（`GW:backend/api/protos/dict/service/v1/dict_entry.proto:14-31`；`GW:frontend/admin/vue-vben/apps/admin/src/views/app/system/dict/dict-entry-list.vue:45-56`），未见后端持久化处理。不能把通用表格按钮当作可迁移业务能力。

## 8. 建议实施顺序

```text
P0  补 KA 会话、审计、队列、文件/备份的故障与跨租户测试
  -> P1 按产品需要增加 Dashboard 只读统计
  -> P1 按文件治理需求增加元数据目录和清理闭环
  -> P1 若需要 per-tenant/per-user 登录限制，再扩展策略表
  -> P2 单设备会话、细粒度权限组/临时角色、任务高级选项、审计签名增强
```

总判断：Wind 新增能力中，真正未被 KA 当前工作树覆盖且通用价值最高的是 Dashboard、文件元数据目录和按对象登录策略；单令牌管理、丰富权限范围、任务选项和日志签名属于有明确产品/合规场景才应吸收的增量。KA 已有的密码策略、审计留存、受控备份、Redis 运维监控和 Core 任务边界不应被 Wind 的较粗实现覆盖。
