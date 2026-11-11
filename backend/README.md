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

- 账号密码、验证码、OAuth、TOTP/WebAuthn MFA、恢复码、JWT 刷新和会话撤销。管理端与移动端统一使用 `/api/v1/base/session` 登录与 `/api/v1/base/oauth/authorization` 授权入口，共用认证和权限；并发登录遵循原策略，允许时可同时使用，禁止时后登录撤销已有会话。
- 多租户、用户、角色、部门、岗位、菜单、字典、配置、API、Casbin 权限和项目授权。
- 文件上传、公开文件、对象存储、敏感字段加密和按权限查看密钥。
- AI 会话、附件、SSE 流式响应、工具调用、智能问数、供应商/模型和知识库检索。
- 站内信分类、发布、投递、收件箱、Provider、失败恢复和三端通知。
- 定时任务、运行日志、六类审计日志、表归档、数据库备份恢复、迁移记录和 MQTT 宿主订阅装配。
- 开放授权客户端、Bearer Token、API 白名单、来源限制、请求加解密和限流。

经典工作流配置样例见 [testdata/workflow](testdata/workflow/README.md)，包含串行审批、金额分支、会签、或签及并行汇聚，可作为流程配置参考。

## 配置

最小启动配置在 `configs`，完整字段模板在 `configs/full`。常用配置文件如下：

| 文件 | 作用 |
| --- | --- |
| `data.yaml` | MySQL/TiDB、Redis 和可选 AI 知识库数据源 |
| `oss.yaml` | 本地或 S3 兼容对象存储 |
| `auth.yaml` | JWT、会话、验证码和认证策略 |
| `mfa.yaml` | TOTP/WebAuthn 参数 |
| `key.yaml` | 运行时根密钥和密钥服务 |
| `server.yaml` | HTTP、gRPC、SSE、MCP、MQTT 和超时 |
| `registry.yaml` | 服务注册配置 |

配置敏感值使用 `ENC[...]` 保存。生产环境必须使用有效的密钥服务、TLS 和最小权限凭据；密钥服务不可用时服务启动失败，不回退明文文件。

Redis 未配置时，单实例缓存、任务锁和队列使用进程内实现；多实例必须连接同一个 Redis。AI 知识库默认使用 `data.databases.rag`，未配置时回落默认数据源。

### TiDB 配置

TiDB 使用 MySQL 兼容协议，主数据源将 `driver` 设置为 `tidb`，连接端口通常为 `4000`：

```yaml
data:
  database:
    driver: tidb
    source: root:password@tcp(127.0.0.1:4000)/kratos_admin?charset=utf8mb4&parseTime=True&loc=Local
    enable_migrate: true
```

服务会自动注册 TiDB 驱动并补充多语句连接参数；初始化迁移和版本化迁移继续读取 `migration/assets/v0.0.1/mysql`。使用 TiDB 生成业务表模型时，将 `gorm-gen` 数据源的 `driver` 同样设为 `tidb`，生成的 Model 可直接用于 GORM 自动迁移和表结构往返验证。

### MQTT 宿主订阅

业务宿主在自己的 Wire Provider 中接收 `*github.com/liujitcn/kratos-kit/transport/mqtt.Server`，用 `backend/pkg/mqtt` 注册订阅并发布业务消息。配置存在 `server.mqtt` 节点时创建并连接 MQTT 服务；删除整个节点即可不创建服务。`server.mqtt.record_all` 开启后会在现有连接上追加 `#` 审计订阅，并使用 No Local 避免把当前客户端发布的消息重复当作订阅消息处理；该订阅不会替换宿主已有主题订阅，但会增加 Broker 下发和消息入库流量。运维监控会展示连接、订阅、发布等待、接收处理中和错误计数；设备凭据、租户 ACL、主题中的租户/项目归属和消息入库逻辑由宿主业务实现。

```go
func NewCloudModule(mqttServer *mqtttransport.Server) (*Module, error) {
	if err := mqttpkg.Subscribe("$share/cloud-scp/device/+/status", handleStatus, bindStatus); err != nil {
		return nil, err
	}
	if err := mqttpkg.Publish(context.Background(), "cloud/platform/ready", map[string]any{"ready": true}); err != nil {
		return nil, err
	}
	return &Module{}, nil
}
```

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

GORM 生成器依据列元数据统一保留大文本容量，并将可空时间列生成为 `*time.Time`，业务项目无需维护按表修正的补丁脚本。更新生成器源码后需重新安装 `gorm-gen` 再执行生成。

## 数据与迁移

初始化资源统一位于 `migration/assets/v0.0.1/mysql`；TiDB 复用该目录，不需要复制 TiDB 专用 SQL：

- `default_data.up.sql`：语言、租户、账号、菜单、按钮、API 权限和默认配置；
- GORM Models 自动创建 AI 插件、自动化和 AI 文件表；`default_data.up.sql` 维护内置、Xberg、Gotenberg 插件初始化；
- `i18n.*.up.sql`：菜单、字典、配置和任务的非主语言译文；
- `README*.md`：迁移说明，由服务保存文件引用和校验摘要。

本仓库不新增 `v0.0.2` 等迁移目录。新表由 GORM 自动迁移创建，菜单、权限和多语言在唯一的 `v0.0.1` 初始化资源中维护。

## 运行检查

```bash
go test ./... # 单元测试与回归检查
make -C backend fmt
make -C backend menu-sort
```

涉及 Proto、Wire、前端调用或初始化权限时，完成后在仓库根目录执行 `make gen`、`make check` 和 `make menu-sort`。

脚本工作流闭环示例见 [script-expense](testdata/workflow/script-expense/README.md)，包含提交校验、SCRIPT 变量计算、动态审批人及终态动作重试；宿主动作注册与幂等约束见 [工作流接入指南](../docs/工作流接入指南.md)。

## AI 插件与自动化接入

插件能力通过 `pkg/aiplugin` 的 Manifest 显式装配到 `AdminModules.RegisterAIPlugin`，再启动服务；数据库中的定义、安装和工具策略不能动态加载 Go 组件。未装配能力显示不可用。

宿主未装配 AI 运行时、插件或 BI 模块时，对应的注册方法正常跳过，不阻止主服务启动。已装配模块返回的实际错误仍向上传递。

AI 会话列表按当前租户与用户共享，两端均可查看和继续。会话通过 `SetAiSessionPlugins` 保存插件与具体账号，保存选择不执行工具；空列表清除选择并可普通聊天。会话及自动化的 `terminal` 只表示工具能力配置与执行快照，跨端继续使用已保存配置；身份与权限始终由现有认证、角色/API 权限、租户和资源归属校验，写操作保留审批。

唯一内部扫描任务是初始化 ID `1008`、调用目标 `base.AiAutomationDispatch`、秒级 cron `0 * * * * *`。每条计划保存在 `ai_automation`，每次执行保存冻结配置、幂等键、租约版本和独立结果到 `ai_automation_run`，不为每个用户计划创建 `base_job`。计划使用 IANA 时区，预览与调度共用服务端时间算法；仍有执行或待确认事项时拒绝恢复，结束后才可恢复；恢复只计算未来时隙，不补跑错过的周期。多实例领取、失效审批与结果未知均以数据库状态为准，扫描任务成功不代表某条业务执行成功。

通知分类 `ai_automation` 仅开启 inbox，`provider_id=[]`，不发邮件、钉钉或其他外部通知。结果提醒使用 `VIEW_KEY`，目标为既有 `AI`，参数仅含 `automation_id`、`run_id`、`session_id` 和安全摘要；目标页重新校验本人访问权限。提醒失败可重试，不能因此重跑工具。

初始化直接维护 `migration/assets/v0.0.1/mysql`，对应默认数据、权限和三语言任务名称同步到 `postgres`：现有 AI 对话菜单承载插件与自动化权限，移动菜单仅承载查询、选用、运行、停止本次及本人审批。插件、自动化和 AI 文件表均已注册 Models，PostgreSQL 启用 `enable_migrate` 后先自动建表，再执行内置、Xberg、Gotenberg 插件和缺失安装种子；已有安装策略不会被覆盖。内置插件固定 ID `1`，Xberg 和 Gotenberg 固定 ID `2`、`3`，运行期同步要求内置种子已存在；关闭迁移时须提前完成初始化。统一登录、OAuth operation 及插件 OAuth 回调 `CompleteAiPluginOAuth` 已列入最小/完整认证白名单，插件回调仍校验一次性 state、发行方与 PKCE；HTTP 路由与其他 `base_api` 一样从 Proto/OpenAPI 启动同步，不创建第二份静态登录 API 种子。存量库需按开发环境流程重建或显式补齐初始化记录及角色授权；修改本版本不会自动重放。
