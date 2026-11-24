-- 在所有种子脚本之后校准显式写入固定 ID 的自增表。
-- 取当前序列值和表中最大 ID 的较大者，重放时只向前推进。
SELECT setval(pg_get_serial_sequence('ai_plugin', 'id'), GREATEST((SELECT MAX(id) FROM "ai_plugin"), nextval(pg_get_serial_sequence('ai_plugin', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_area', 'id'), GREATEST((SELECT MAX(id) FROM "base_area"), nextval(pg_get_serial_sequence('base_area', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_config', 'id'), GREATEST((SELECT MAX(id) FROM "base_config"), nextval(pg_get_serial_sequence('base_config', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_dept', 'id'), GREATEST((SELECT MAX(id) FROM "base_dept"), nextval(pg_get_serial_sequence('base_dept', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_dict', 'id'), GREATEST((SELECT MAX(id) FROM "base_dict"), nextval(pg_get_serial_sequence('base_dict', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_dict_item', 'id'), GREATEST((SELECT MAX(id) FROM "base_dict_item"), nextval(pg_get_serial_sequence('base_dict_item', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_job', 'id'), GREATEST((SELECT MAX(id) FROM "base_job"), nextval(pg_get_serial_sequence('base_job', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_language', 'id'), GREATEST((SELECT MAX(id) FROM "base_language"), nextval(pg_get_serial_sequence('base_language', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_login_policy', 'id'), GREATEST((SELECT MAX(id) FROM "base_login_policy"), nextval(pg_get_serial_sequence('base_login_policy', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_message_category', 'id'), GREATEST((SELECT MAX(id) FROM "base_message_category"), nextval(pg_get_serial_sequence('base_message_category', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_message_provider', 'id'), GREATEST((SELECT MAX(id) FROM "base_message_provider"), nextval(pg_get_serial_sequence('base_message_provider', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_oauth_provider', 'id'), GREATEST((SELECT MAX(id) FROM "base_oauth_provider"), nextval(pg_get_serial_sequence('base_oauth_provider', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_post', 'id'), GREATEST((SELECT MAX(id) FROM "base_post"), nextval(pg_get_serial_sequence('base_post', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_redact_output_policy', 'id'), GREATEST((SELECT MAX(id) FROM "base_redact_output_policy"), nextval(pg_get_serial_sequence('base_redact_output_policy', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_redact_rule', 'id'), GREATEST((SELECT MAX(id) FROM "base_redact_rule"), nextval(pg_get_serial_sequence('base_redact_rule', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_redact_storage_policy', 'id'), GREATEST((SELECT MAX(id) FROM "base_redact_storage_policy"), nextval(pg_get_serial_sequence('base_redact_storage_policy', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_role', 'id'), GREATEST((SELECT MAX(id) FROM "base_role"), nextval(pg_get_serial_sequence('base_role', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_script', 'id'), GREATEST((SELECT MAX(id) FROM "base_script"), nextval(pg_get_serial_sequence('base_script', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_table_archive', 'id'), GREATEST((SELECT MAX(id) FROM "base_table_archive"), nextval(pg_get_serial_sequence('base_table_archive', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_table_backup', 'id'), GREATEST((SELECT MAX(id) FROM "base_table_backup"), nextval(pg_get_serial_sequence('base_table_backup', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_tenant', 'id'), GREATEST((SELECT MAX(id) FROM "base_tenant"), nextval(pg_get_serial_sequence('base_tenant', 'id'))), true);
SELECT setval(pg_get_serial_sequence('base_user', 'id'), GREATEST((SELECT MAX(id) FROM "base_user"), nextval(pg_get_serial_sequence('base_user', 'id'))), true);
SELECT setval(pg_get_serial_sequence('workflow_definition', 'id'), GREATEST((SELECT MAX(id) FROM "workflow_definition"), nextval(pg_get_serial_sequence('workflow_definition', 'id'))), true);
