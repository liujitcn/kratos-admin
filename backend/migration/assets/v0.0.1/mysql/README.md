由 `scripts/sync_locales.py` 根据语言包生成。

该迁移只补充语言目录中不存在的 `base_language` 记录；数据库中的启用状态、排序和主语言标记由部署配置维护。

语言集合：zh-CN, en-US, ja-JP, zh-TW。

AI 工具人工确认的完整初始化包括 `ai_tool_approval.up.sql`、`default_data.up.sql` 中的消息状态 `4` 和管理端 AI 对话菜单审批权限。表内同时保存 Eino checkpoint 和审批状态；终止运行后清空 checkpoint，待确认有效期为 10 分钟。`base_api` 路径由 Proto 路由在启动时同步，确认路由为 `POST /api/v1/base/ai/session/{session_id}/message/{message_id}/approval`。

`checkpoint_data` 使用 LONGBLOB 保存 zstd 压缩快照；MySQL `max_allowed_packet` 至少为 64 MiB。运行时将编码后的上下文输入限制为 28 MiB，并限制未压缩/压缩快照为 192/48 MiB，保存失败不会生成可执行审批。

存量库不会自动重放本版本；需按开发环境流程重建，或单独建表并更新菜单权限、字典数据。脚本和执行型工具由运行时硬性封闭，不受 `agent_status` 开关绕过。
