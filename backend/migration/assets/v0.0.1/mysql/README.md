由 `scripts/sync_locales.py` 根据语言包生成。

安全初始化：平台管理员登录名为 `mgr`，新建租户管理员登录名为 `tmgr`，初始账号必须首登改密；系统配置按代理、脚本、出站、账号和迁移功能拆分为安全表单；OAuth 客户端示例不携带可用密钥。六类安全日志默认在线与归档各保留 365 天并启用归档任务，租户联系人姓名、电话新增入库和出库保护策略。存量数据库须备份并补齐这些策略与留存配置，再执行受控更名和幂等加密回填；本版本脚本不会自动重放，具体步骤见仓库安全部署文档。

该迁移只补充语言目录中不存在的 `base_language` 记录；数据库中的启用状态、排序和主语言标记由部署配置维护。

语言集合：zh-CN, en-US, ja-JP, zh-TW。

管理端一级菜单按 AI 助手 `90000000`、BI 分析 `91000000`、系统管理 `97000000`、Demo 演示 `98000000` 排列；BI 子页面为 `91010000` 至 `91070000`，数据源按钮为 `91050100` 至 `91050400`，角色与译文同步使用这些编号。

BI 表结构由开发库通过 GORM 生成并在启动时自动建表，`default_data.up.sql` 只维护默认数据、菜单和权限。可见菜单依次为工作台、智能问数、卡片库、仪表盘管理、数据源管理；设计器和通用展示页为隐藏路由，问数记录并入历史栏。新工作台保存仪表盘入口，卡片保存来源查询记录及展示配置，不兼容旧数据；应使用全新数据库初始化验证。

BI 查询统一使用 `bi_query`，保存问题、语义查询计划、项目与仪表盘上下文、执行指标和结果快照，结果快照包含列编码、名称和类型，字段采用 MEDIUMTEXT；卡片通过查询记录 ID 追溯来源。

数据源页面仅授予列表与详情读取权限；新增、编辑、删除和启停由对应按钮单独授权，连接测试、数据库选项、表结构发现和模型保存随新增或编辑按钮授权。

AI 工具人工确认表 `ai_tool_approval` 由已注册的 `AiToolApproval` Model 在启用 `enable_migrate` 时自动创建，不重复维护建表 SQL。`default_data.up.sql` 维护消息状态 `4` 和管理端 AI 对话菜单审批权限。表内同时保存 Eino checkpoint 和审批状态；终止运行后清空 checkpoint，待确认有效期为 10 分钟。`base_api` 路径由 Proto 路由在启动时同步，确认路由为 `POST /api/v1/base/ai/session/{session_id}/message/{message_id}/approval`。

`checkpoint_data` 使用 LONGBLOB 保存 zstd 压缩快照；MySQL `max_allowed_packet` 至少为 64 MiB。运行时将编码后的上下文输入限制为 28 MiB，并限制未压缩/压缩快照为 192/48 MiB，保存失败不会生成可执行审批。

存量库启用 `enable_migrate` 后会自动补齐注册模型的表结构，但不会自动重放本版本的初始化数据；菜单权限、字典数据仍需按开发环境流程重建或补齐。脚本和执行型工具由运行时硬性封闭，不受 `agent_status` 开关绕过。

AI 对话沿用管理菜单 `90990000` 与移动菜单 `99101000`，不增加独立插件或自动化菜单。管理端菜单授予 `AiPluginService`、`AiAutomationService` 和 `SetAiSessionPlugins`；移动端仅授予目录/账号/工具查询、本人任务与记录查询、运行一次、停止本次、结果继续讨论及本人审批，不授予插件或计划管理写操作。

Models 自动创建 AI 插件、自动化和 AI 文件表；`default_data.up.sql` 维护内置、Xberg、Gotenberg、MarkItDown 插件种子及默认租户安装。AI 文件对象存储目录继续复用 `ai/`，不与 `base_file` 双写；当前没有存量 AI 文件，不包含历史数据迁移。三个 MCP 端点使用 `192.168.60.52` 私网白名单和 `allow_private` 配置，Gotenberg Bearer Token 需在连接页面录入。`default_data.up.sql` 登记唯一 `base_job`：ID `1008`、`base.AiAutomationDispatch`、秒级 cron `0 * * * * *`；用户计划不逐条登记系统任务。调度读取持久化计划、租约和状态，浏览器无需保持在线。

消息分类 ID `6`、code `ai_automation` 仅开启站内 inbox，`provider_id=[]`，不创建或启用任何外部 Provider。运行结果提醒由业务发布为 `VIEW_KEY` 到既有 `AI` 页面，携带 `automation_id`、`run_id`、`session_id`；不在初始化 SQL 写入真实运行或消息。分类只控制投递，结果归属与审批权限仍由服务端校验。

管理端与应用端共用公开登录 `POST /api/v1/base/session` 和跳转授权 `POST /api/v1/base/oauth/authorization`，对应 operation 统一列入最小/完整认证白名单。`base_api` 仍统一由 Proto/OpenAPI 启动同步，公开登录不额外创建菜单权限或静态 API 种子。登录认证和本人 AI 对话跨端共享，并发登录由原策略控制；角色/API 权限、租户、资源归属和审批继续生效，本次不调整表结构或种子。
