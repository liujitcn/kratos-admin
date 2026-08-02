CREATE TABLE IF NOT EXISTS `base_menu_translation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `menu_id` BIGINT NOT NULL COMMENT '菜单ID',
  `locale` VARCHAR(16) NOT NULL COMMENT '语言区域',
  `title` VARCHAR(255) NOT NULL COMMENT '菜单标题',
  `translation_status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '翻译状态',
  `source_hash` CHAR(64) NOT NULL COMMENT '中文源文SHA-256',
  `translation_provider` VARCHAR(32) NULL COMMENT '机器翻译提供方',
  `translated_at` DATETIME NULL COMMENT '最近机器翻译时间',
  `reviewed_by` BIGINT NULL COMMENT '审核人ID',
  `reviewed_at` DATETIME NULL COMMENT '审核时间',
  `created_by` BIGINT NOT NULL DEFAULT 1 COMMENT '创建人ID',
  `updated_by` BIGINT NOT NULL DEFAULT 1 COMMENT '更新人ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_base_menu_translation` (`menu_id`, `locale`, `deleted_at`),
  KEY `idx_base_menu_translation_menu_id` (`menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='菜单翻译';

CREATE TABLE IF NOT EXISTS `base_dict_translation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `dict_id` BIGINT NOT NULL COMMENT '字典ID',
  `locale` VARCHAR(16) NOT NULL COMMENT '语言区域',
  `name` VARCHAR(50) NOT NULL COMMENT '字典名称',
  `translation_status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '翻译状态',
  `source_hash` CHAR(64) NOT NULL COMMENT '中文源文SHA-256',
  `translation_provider` VARCHAR(32) NULL COMMENT '机器翻译提供方',
  `translated_at` DATETIME NULL COMMENT '最近机器翻译时间',
  `reviewed_by` BIGINT NULL COMMENT '审核人ID',
  `reviewed_at` DATETIME NULL COMMENT '审核时间',
  `created_by` BIGINT NOT NULL DEFAULT 1 COMMENT '创建人ID',
  `updated_by` BIGINT NOT NULL DEFAULT 1 COMMENT '更新人ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_base_dict_translation` (`dict_id`, `locale`, `deleted_at`),
  KEY `idx_base_dict_translation_dict_id` (`dict_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='字典翻译';

CREATE TABLE IF NOT EXISTS `base_dict_item_translation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `dict_item_id` BIGINT NOT NULL COMMENT '字典项ID',
  `locale` VARCHAR(16) NOT NULL COMMENT '语言区域',
  `label` VARCHAR(100) NOT NULL COMMENT '字典项标签',
  `translation_status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '翻译状态',
  `source_hash` CHAR(64) NOT NULL COMMENT '中文源文SHA-256',
  `translation_provider` VARCHAR(32) NULL COMMENT '机器翻译提供方',
  `translated_at` DATETIME NULL COMMENT '最近机器翻译时间',
  `reviewed_by` BIGINT NULL COMMENT '审核人ID',
  `reviewed_at` DATETIME NULL COMMENT '审核时间',
  `created_by` BIGINT NOT NULL DEFAULT 1 COMMENT '创建人ID',
  `updated_by` BIGINT NOT NULL DEFAULT 1 COMMENT '更新人ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_base_dict_item_translation` (`dict_item_id`, `locale`, `deleted_at`),
  KEY `idx_base_dict_item_translation_dict_item_id` (`dict_item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='字典项翻译';

SET @code_gen_table_i18n_config_sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `code_gen_table` ADD COLUMN `i18n_config` JSON NULL COMMENT ''表级国际化配置JSON'' AFTER `left_tree_config`',
    'SELECT 1'
  )
  FROM `information_schema`.`COLUMNS`
  WHERE `TABLE_SCHEMA` = DATABASE()
    AND `TABLE_NAME` = 'code_gen_table'
    AND `COLUMN_NAME` = 'i18n_config'
);
PREPARE code_gen_table_i18n_config_stmt FROM @code_gen_table_i18n_config_sql;
EXECUTE code_gen_table_i18n_config_stmt;
DEALLOCATE PREPARE code_gen_table_i18n_config_stmt;

SET @code_gen_column_i18n_config_sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `code_gen_column` ADD COLUMN `i18n_config` JSON NULL COMMENT ''字段国际化配置JSON'' AFTER `form_config`',
    'SELECT 1'
  )
  FROM `information_schema`.`COLUMNS`
  WHERE `TABLE_SCHEMA` = DATABASE()
    AND `TABLE_NAME` = 'code_gen_column'
    AND `COLUMN_NAME` = 'i18n_config'
);
PREPARE code_gen_column_i18n_config_stmt FROM @code_gen_column_i18n_config_sql;
EXECUTE code_gen_column_i18n_config_stmt;
DEALLOCATE PREPARE code_gen_column_i18n_config_stmt;

UPDATE `base_menu`
SET `api` = JSON_ARRAY_APPEND(`api`, '$', '/system.admin.v1.BaseTranslationService/GenerateTranslationDraft')
WHERE `id` IN (2000103, 2000203, 2000303)
  AND JSON_VALID(`api`)
  AND NOT JSON_CONTAINS(`api`, JSON_QUOTE('/system.admin.v1.BaseTranslationService/GenerateTranslationDraft'));

INSERT INTO `base_menu_translation`
  (`menu_id`, `locale`, `title`, `translation_status`, `source_hash`, `translation_provider`, `translated_at`, `reviewed_by`, `reviewed_at`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)
SELECT translations.menu_id,
       locales.locale,
       IF(locales.locale = 'en-US', translations.en_title, translations.ja_title),
       'reviewed',
       SHA2(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(menu.meta, '$.title')), ''), 256),
       NULL,
       NULL,
       1,
       CURRENT_TIMESTAMP,
       1,
       1,
       CURRENT_TIMESTAMP,
       CURRENT_TIMESTAMP,
       0
FROM (
  SELECT 100 AS menu_id, 'Home' AS en_title, 'ホーム' AS ja_title
  UNION ALL SELECT 10004, 'AI Assistant', 'AIアシスタント'
  UNION ALL SELECT 10005, 'Profile', 'プロフィール'
  UNION ALL SELECT 200, 'System Management', 'システム管理'
  UNION ALL SELECT 20001, 'Menu Management', 'メニュー管理'
  UNION ALL SELECT 2000101, 'Create Menu', 'メニューを新規作成'
  UNION ALL SELECT 2000102, 'Delete Menu', 'メニューを削除'
  UNION ALL SELECT 2000103, 'Edit Menu', 'メニューを編集'
  UNION ALL SELECT 2000104, 'Change Menu Status', 'メニューステータスを変更'
  UNION ALL SELECT 20002, 'Dictionary Management', '辞書管理'
  UNION ALL SELECT 2000201, 'Create Dictionary', '辞書を新規作成'
  UNION ALL SELECT 2000202, 'Delete Dictionary', '辞書を削除'
  UNION ALL SELECT 2000203, 'Edit Dictionary', '辞書を編集'
  UNION ALL SELECT 2000204, 'Change Dictionary Status', '辞書ステータスを変更'
  UNION ALL SELECT 2000205, 'Dictionary Items', '辞書項目'
  UNION ALL SELECT 20003, 'Dictionary Items', '辞書データ'
  UNION ALL SELECT 2000301, 'Create Dictionary Item', '辞書項目を新規作成'
  UNION ALL SELECT 2000302, 'Delete Dictionary Item', '辞書項目を削除'
  UNION ALL SELECT 2000303, 'Edit Dictionary Item', '辞書項目を編集'
  UNION ALL SELECT 2000304, 'Change Dictionary Item Status', '辞書項目ステータスを変更'
  UNION ALL SELECT 20004, 'System Configuration', 'システム設定'
  UNION ALL SELECT 2000401, 'Create Configuration', '設定を新規作成'
  UNION ALL SELECT 2000402, 'Delete Configuration', '設定を削除'
  UNION ALL SELECT 2000403, 'Edit Configuration', '設定を編集'
  UNION ALL SELECT 2000404, 'Change Configuration Status', '設定ステータスを変更'
  UNION ALL SELECT 2000405, 'Refresh Configuration Cache', '設定キャッシュを更新'
  UNION ALL SELECT 20005, 'Scheduled Jobs', '定期ジョブ'
  UNION ALL SELECT 2000501, 'Create Scheduled Job', '定期ジョブを新規作成'
  UNION ALL SELECT 2000502, 'Delete Scheduled Job', '定期ジョブを削除'
  UNION ALL SELECT 2000503, 'Edit Scheduled Job', '定期ジョブを編集'
  UNION ALL SELECT 2000504, 'Change Scheduled Job Status', '定期ジョブステータスを変更'
  UNION ALL SELECT 2000505, 'Start Scheduled Job', '定期ジョブを開始'
  UNION ALL SELECT 2000506, 'Stop Scheduled Job', '定期ジョブを停止'
  UNION ALL SELECT 2000507, 'Run Scheduled Job', '定期ジョブを実行'
  UNION ALL SELECT 2000508, 'Scheduled Job Logs', '定期ジョブログ'
  UNION ALL SELECT 20006, 'Scheduled Job Logs', '定期ジョブログ'
  UNION ALL SELECT 2000601, 'View Scheduled Job Log Details', '定期ジョブログ詳細を表示'
  UNION ALL SELECT 20007, 'API Management', 'API管理'
  UNION ALL SELECT 2000701, 'View API Details', 'API詳細を表示'
  UNION ALL SELECT 2000702, 'Set API MCP Tool Status', 'API MCPツールステータスを設定'
  UNION ALL SELECT 2000703, 'Set API Agent Tool Status', 'API Agentツールステータスを設定'
  UNION ALL SELECT 2000704, 'Edit API Configuration', 'API設定を編集'
  UNION ALL SELECT 20008, 'Region Management', '地域管理'
  UNION ALL SELECT 2000801, 'Create Region', '地域を新規作成'
  UNION ALL SELECT 2000802, 'Delete Region', '地域を削除'
  UNION ALL SELECT 2000803, 'Edit Region', '地域を編集'
  UNION ALL SELECT 20009, 'System Logs', 'システムログ'
  UNION ALL SELECT 2000901, 'View Log Details', 'ログ詳細を表示'
  UNION ALL SELECT 20010, 'Upgrade History', 'アップグレード履歴'
  UNION ALL SELECT 300, 'User Management', 'ユーザー管理'
  UNION ALL SELECT 30001, 'Tenant Management', 'テナント管理'
  UNION ALL SELECT 3000101, 'Create Tenant', 'テナントを新規作成'
  UNION ALL SELECT 3000102, 'Delete Tenant', 'テナントを削除'
  UNION ALL SELECT 3000103, 'Edit Tenant', 'テナントを編集'
  UNION ALL SELECT 3000104, 'Change Tenant Status', 'テナントステータスを変更'
  UNION ALL SELECT 30002, 'User Management', 'ユーザー管理'
  UNION ALL SELECT 3000201, 'Create User', 'ユーザーを新規作成'
  UNION ALL SELECT 3000202, 'Delete User', 'ユーザーを削除'
  UNION ALL SELECT 3000203, 'Edit User', 'ユーザーを編集'
  UNION ALL SELECT 3000204, 'Change User Status', 'ユーザーステータスを変更'
  UNION ALL SELECT 3000205, 'Reset User Password', 'ユーザーパスワードをリセット'
  UNION ALL SELECT 30003, 'Role Management', 'ロール管理'
  UNION ALL SELECT 3000301, 'Create Role', 'ロールを新規作成'
  UNION ALL SELECT 3000302, 'Delete Role', 'ロールを削除'
  UNION ALL SELECT 3000303, 'Edit Role', 'ロールを編集'
  UNION ALL SELECT 3000304, 'Change Role Status', 'ロールステータスを変更'
  UNION ALL SELECT 3000305, 'Assign Role Permissions', 'ロール権限を割り当て'
  UNION ALL SELECT 30004, 'Department Management', '部門管理'
  UNION ALL SELECT 3000401, 'Create Department', '部門を新規作成'
  UNION ALL SELECT 3000402, 'Delete Department', '部門を削除'
  UNION ALL SELECT 3000403, 'Edit Department', '部門を編集'
  UNION ALL SELECT 3000404, 'Change Department Status', '部門ステータスを変更'
  UNION ALL SELECT 30005, 'Position Management', '役職管理'
  UNION ALL SELECT 3000501, 'Create Position', '役職を新規作成'
  UNION ALL SELECT 3000502, 'Delete Position', '役職を削除'
  UNION ALL SELECT 3000503, 'Edit Position', '役職を編集'
  UNION ALL SELECT 3000504, 'Change Position Status', '役職ステータスを変更'
  UNION ALL SELECT 950, 'Development Tools', '開発ツール'
  UNION ALL SELECT 95001, 'Code Generation', 'コード生成'
  UNION ALL SELECT 9500101, 'Create Code Generation Table Configuration', 'コード生成テーブル設定を新規作成'
  UNION ALL SELECT 9500102, 'Edit Code Generation Table Configuration', 'コード生成テーブル設定を編集'
  UNION ALL SELECT 9500103, 'Delete Code Generation Table Configuration', 'コード生成テーブル設定を削除'
  UNION ALL SELECT 9500104, 'Maintain Code Generation Field Configuration', 'コード生成フィールド設定を管理'
  UNION ALL SELECT 9500105, 'Maintain Code Generation Proto Configuration', 'コード生成Proto設定を管理'
  UNION ALL SELECT 9500106, 'Preview Generated Page', '生成ページをプレビュー'
  UNION ALL SELECT 9500107, 'Preview Generated Files', '生成ファイルをプレビュー'
  UNION ALL SELECT 9500108, 'Run Code Generation', 'コード生成を実行'
  UNION ALL SELECT 9500109, 'Restore Code Generation Results', 'コード生成結果を復元'
  UNION ALL SELECT 95002, 'Code Generation Field Configuration', 'コード生成フィールド設定'
  UNION ALL SELECT 95003, 'Code Generation Proto Configuration', 'コード生成Proto設定'
  UNION ALL SELECT 95004, 'Generated Page Preview', '生成ページプレビュー'
  UNION ALL SELECT 95005, 'Generated Code Preview', '生成コードプレビュー'
  UNION ALL SELECT 95006, 'API Documentation', 'APIドキュメント'
  UNION ALL SELECT 95007, 'Project Documentation', 'プロジェクトドキュメント'
  UNION ALL SELECT 999, 'Mobile App', 'モバイルアプリ'
  UNION ALL SELECT 99901, 'Home', 'ホーム'
  UNION ALL SELECT 9990101, 'Login', 'ログイン'
  UNION ALL SELECT 999010101, 'Terms', '利用規約'
  UNION ALL SELECT 9990102, 'Web Page', 'Webページ'
  UNION ALL SELECT 99909, 'My Account', 'マイページ'
  UNION ALL SELECT 9990901, 'Profile', 'プロフィール'
  UNION ALL SELECT 9990902, 'Settings', '設定'
  UNION ALL SELECT 9990903, 'AI Assistant', 'AIアシスタント'
) AS translations
JOIN `base_menu` AS menu ON menu.id = translations.menu_id
CROSS JOIN (
  SELECT 'en-US' AS locale
  UNION ALL SELECT 'ja-JP'
) AS locales
WHERE NOT EXISTS (
  SELECT 1
  FROM `base_menu_translation` AS existing
  WHERE existing.menu_id = translations.menu_id
    AND existing.locale = locales.locale
    AND existing.deleted_at = 0
);

INSERT INTO `base_dict_translation`
  (`dict_id`, `locale`, `name`, `translation_status`, `source_hash`, `translation_provider`, `translated_at`, `reviewed_by`, `reviewed_at`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)
SELECT translations.dict_id,
       locales.locale,
       IF(locales.locale = 'en-US', translations.en_name, translations.ja_name),
       'reviewed',
       SHA2(dict.name, 256),
       NULL,
       NULL,
       1,
       CURRENT_TIMESTAMP,
       1,
       1,
       CURRENT_TIMESTAMP,
       CURRENT_TIMESTAMP,
       0
FROM (
  SELECT 1 AS dict_id, 'Status' AS en_name, 'ステータス' AS ja_name
  UNION ALL SELECT 1000, 'System Configuration Location', 'システム設定の配置'
  UNION ALL SELECT 1010, 'System Configuration Type', 'システム設定タイプ'
  UNION ALL SELECT 1020, 'CAPTCHA Type', 'CAPTCHAタイプ'
  UNION ALL SELECT 1100, 'Scheduled Job Log Status', '定期ジョブログステータス'
  UNION ALL SELECT 1200, 'Menu Type', 'メニュータイプ'
  UNION ALL SELECT 1300, 'User Role Data Scope', 'ユーザーロールのデータ範囲'
  UNION ALL SELECT 1400, 'Business Module', '業務モジュール'
  UNION ALL SELECT 1500, 'Code Generation Table Status', 'コード生成テーブルステータス'
  UNION ALL SELECT 2000, 'User Gender', 'ユーザーの性別'
) AS translations
JOIN `base_dict` AS dict ON dict.id = translations.dict_id
CROSS JOIN (
  SELECT 'en-US' AS locale
  UNION ALL SELECT 'ja-JP'
) AS locales
WHERE NOT EXISTS (
  SELECT 1
  FROM `base_dict_translation` AS existing
  WHERE existing.dict_id = translations.dict_id
    AND existing.locale = locales.locale
    AND existing.deleted_at = 0
);

INSERT INTO `base_dict_item_translation`
  (`dict_item_id`, `locale`, `label`, `translation_status`, `source_hash`, `translation_provider`, `translated_at`, `reviewed_by`, `reviewed_at`, `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`)
SELECT translations.dict_item_id,
       locales.locale,
       IF(locales.locale = 'en-US', translations.en_label, translations.ja_label),
       'reviewed',
       SHA2(item.label, 256),
       NULL,
       NULL,
       1,
       CURRENT_TIMESTAMP,
       1,
       1,
       CURRENT_TIMESTAMP,
       CURRENT_TIMESTAMP,
       0
FROM (
  SELECT 1 AS dict_item_id, 'Enabled' AS en_label, '有効' AS ja_label
  UNION ALL SELECT 2, 'Disabled', '無効'
  UNION ALL SELECT 10001, 'Built-in', 'システム組み込み'
  UNION ALL SELECT 10002, 'Admin Console', '管理画面'
  UNION ALL SELECT 10003, 'Application', 'アプリ'
  UNION ALL SELECT 10101, 'Text', 'テキスト'
  UNION ALL SELECT 10102, 'Image', '画像'
  UNION ALL SELECT 10103, 'Rich Text', 'リッチテキスト'
  UNION ALL SELECT 10104, 'Dictionary', '辞書'
  UNION ALL SELECT 10105, 'Boolean', '真偽値'
  UNION ALL SELECT 10201, 'Random CAPTCHA', 'ランダムCAPTCHA'
  UNION ALL SELECT 10202, 'Numeric CAPTCHA', '数字CAPTCHA'
  UNION ALL SELECT 10203, 'String CAPTCHA', '文字列CAPTCHA'
  UNION ALL SELECT 10204, 'Arithmetic CAPTCHA', '計算CAPTCHA'
  UNION ALL SELECT 10205, 'Chinese CAPTCHA', '中国語CAPTCHA'
  UNION ALL SELECT 10206, 'Slider Puzzle CAPTCHA', 'スライドパズルCAPTCHA'
  UNION ALL SELECT 10207, 'Click Text CAPTCHA', '文字クリックCAPTCHA'
  UNION ALL SELECT 10208, 'Rotation CAPTCHA', '回転CAPTCHA'
  UNION ALL SELECT 11001, 'Success', '成功'
  UNION ALL SELECT 11002, 'Failure', '失敗'
  UNION ALL SELECT 12001, 'Directory', 'ディレクトリ'
  UNION ALL SELECT 12002, 'Menu', 'メニュー'
  UNION ALL SELECT 12003, 'Button', 'ボタン'
  UNION ALL SELECT 12004, 'External Link', '外部リンク'
  UNION ALL SELECT 13001, 'All Data', 'すべてのデータ'
  UNION ALL SELECT 13002, 'Department and Subdepartment Data', '部門および下位部門のデータ'
  UNION ALL SELECT 13003, 'Current Department Data', '所属部門のデータ'
  UNION ALL SELECT 13004, 'Personal Data', '本人のデータ'
  UNION ALL SELECT 14001, 'System Management', 'システム管理'
  UNION ALL SELECT 15001, 'Draft', '下書き'
  UNION ALL SELECT 15002, 'Generated', '生成済み'
  UNION ALL SELECT 15003, 'Disabled', '無効'
  UNION ALL SELECT 20001, 'Confidential', '非公開'
  UNION ALL SELECT 20002, 'Male', '男性'
  UNION ALL SELECT 20003, 'Female', '女性'
) AS translations
JOIN `base_dict_item` AS item ON item.id = translations.dict_item_id
CROSS JOIN (
  SELECT 'en-US' AS locale
  UNION ALL SELECT 'ja-JP'
) AS locales
WHERE NOT EXISTS (
  SELECT 1
  FROM `base_dict_item_translation` AS existing
  WHERE existing.dict_item_id = translations.dict_item_id
    AND existing.locale = locales.locale
    AND existing.deleted_at = 0
);
