-- 语言包同步生成的语言初始化数据。
-- 可重复执行：只补充不存在的语言，不覆盖数据库中的启用状态、名称和主语言配置。

SET NAMES utf8mb4;

INSERT IGNORE INTO `base_language` (`language_code`, `language_name`, `native_name`, `sort`, `is_primary`, `status`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`) VALUES
  ('zh-CN', 'zh-CN', 'zh-CN', 10, 1, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0),
  ('en-US', 'en-US', 'en-US', 20, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0),
  ('ja-JP', 'ja-JP', 'ja-JP', 30, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0),
  ('zh-TW', 'zh-TW', 'zh-TW', 40, 0, 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0);
