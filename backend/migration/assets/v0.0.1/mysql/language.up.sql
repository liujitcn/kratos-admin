-- 语言包同步生成的语言初始化数据。
-- 可重复执行：只补充不存在的语言，不覆盖数据库中的启用状态、名称和主语言配置。
-- 与 default_data.up.sql 的 base_language 完全对齐：id 使用 10/20/30/40，主语言为 zh-CN。

SET NAMES utf8mb4;

INSERT IGNORE INTO `base_language` (`id`, `language_code`, `language_name`, `native_name`, `sort`, `is_primary`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`) VALUES (10, 'zh-CN', '中文（简体）', '简体中文', 10, 1, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);
INSERT IGNORE INTO `base_language` (`id`, `language_code`, `language_name`, `native_name`, `sort`, `is_primary`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`) VALUES (20, 'zh-TW', '中文（繁体）', '繁體中文', 20, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);
INSERT IGNORE INTO `base_language` (`id`, `language_code`, `language_name`, `native_name`, `sort`, `is_primary`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`) VALUES (30, 'en-US', '英语', 'English', 30, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);
INSERT IGNORE INTO `base_language` (`id`, `language_code`, `language_name`, `native_name`, `sort`, `is_primary`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`) VALUES (40, 'ja-JP', '日语', '日本語', 40, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);
