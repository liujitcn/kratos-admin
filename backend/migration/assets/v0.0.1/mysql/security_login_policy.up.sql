-- 平台管理员登录来源策略菜单；普通租户角色不授予该菜单。
INSERT IGNORE INTO `base_menu` (`id`, `parent_id`, `type`, `path`, `name`, `component`, `redirect`, `meta`, `api`, `sort`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)
VALUES (2000105, 20001, 2, 'base/login-policy', 'BaseLoginPolicy', 'system/base/login-policy/index', '', '{"icon":"Lock","title":"登录策略","hidden":false,"keep_alive":true,"always_show":false}', '["/system.admin.v1.BaseLoginPolicyService/GetBaseLoginPolicy","/system.admin.v1.BaseLoginPolicyService/UpdateBaseLoginPolicy"]', 105, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);

UPDATE `base_role`
SET `menus` = JSON_ARRAY_APPEND(`menus`, '$', 2000105)
WHERE `id` = 1 AND JSON_CONTAINS(`menus`, '2000105', '$') = 0;
