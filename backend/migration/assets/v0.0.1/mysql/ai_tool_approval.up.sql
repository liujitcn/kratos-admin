-- Agent 运行快照与人工确认审计，checkpoint 只供服务端恢复执行。
CREATE TABLE IF NOT EXISTS `ai_tool_approval` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '运行ID',
  `approval_id` varchar(36) NOT NULL COMMENT '一次性审批编号',
  `tenant_id` bigint NOT NULL COMMENT '租户ID',
  `session_id` bigint NOT NULL COMMENT '会话ID',
  `message_id` bigint NOT NULL COMMENT '消息ID',
  `user_id` bigint NOT NULL COMMENT '所属用户ID',
  `checkpoint_data` longblob NOT NULL COMMENT 'Eino运行快照',
  `runtime_state` json NOT NULL COMMENT '模型和工具配置JSON',
  `pending_calls` json NOT NULL COMMENT '冻结的待确认工具JSON',
  `status` smallint NOT NULL DEFAULT '0' COMMENT '审批状态：0运行中1待确认2已完成3已拒绝4已过期5失败',
  `expires_at` datetime(3) NOT NULL COMMENT '确认有效期',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_ai_tool_approval` (`approval_id`),
  KEY `idx_ai_tool_approval_message_id` (`message_id`),
  KEY `idx_ai_tool_approval_tenant_id_user_id` (`tenant_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI工具人工确认';
