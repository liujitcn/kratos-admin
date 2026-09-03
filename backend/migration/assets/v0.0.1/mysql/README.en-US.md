# MySQL Default Initialization Resources

This directory contains the MySQL initialization resources for `v0.0.1`. The scripts insert default data only; they do not create tables. At startup, GORM `AutoMigrate` creates the enabled models, and the `.up.sql` files in this directory are then applied in filename order.

## File Responsibilities

| File | Contents |
| --- | --- |
| `default_data.up.sql` | Default languages, configuration, departments, dictionaries, dictionary items, jobs, tenant, message categories, menus, roles, and development accounts. |
| `base_area.up.sql` | Administrative-division data in a separate script that participates in the default data-source migration. |
| `i18n.en-US.up.sql` | `en-US` `base_i18n` translations. |
| `i18n.ja-JP.up.sql` | `ja-JP` `base_i18n` translations. |
| `i18n.zh-TW.up.sql` | `zh-TW` `base_i18n` translations. |
| `README.<locale>.md` | The localized migration description synchronized to the shared translation table by the migration framework. |

## Execution and Idempotency

Startup performs automatic table creation, SQL migrations, migration-description synchronization, and then OpenAPI, `base_api`, tenant-menu, and Casbin-policy synchronization. `default_data.up.sql` and `base_area.up.sql` temporarily disable foreign-key checks and restore the original value when they finish.

Every initialization record is written with an individual `INSERT IGNORE`: an existing unique-key record is skipped and business data is not overwritten. The scripts contain no batch `INSERT`, `UPDATE`, `DELETE`, or `TRUNCATE`. Records for each table in `default_data.up.sql` are maintained in ascending `id` order; new default rows should be placed beside the matching ID range.

A database that has already recorded `v0.0.1` will not replay the migration because an initialization file changed. Validate changes on a fresh database or by rebuilding the development database. New features must complete the `v0.0.1` initialization state; do not add a later version or an incremental script for existing databases.

## Default Data

`default_data.up.sql` currently writes 11 tables:

`base_language`, `base_config`, `base_dept`, `base_dict`, `base_dict_item`, `base_job`, `base_tenant`, `base_message_category`, `base_menu`, `base_role`, and `base_user`.

- Languages: `zh-CN`, `zh-TW`, `en-US`, and `ja-JP` are provided; `zh-CN` is the primary language.
- Identity data: a default tenant, system departments, five role templates, and the local development accounts `super` and `admin` are provided. The initial password is `112233` and is for local use only.
- Configuration and dictionaries: the data covers Admin and app settings, CAPTCHA, OAuth auto-registration, tenant-code display, MFA policy and methods; audit-log ingestion fallback uses hidden configuration `auditLogSpool`, retention policies use `base_table_archive`, and backup policies use `base_table_backup`, while session lifetime and upload scanning use `authn.session` and `oss.upload_security`. Persistent dictionaries cover menu, audit, login-policy, message, permission, and code-generation enums.
- Jobs and messages: job IDs `1000-1004` are resource translation, message-delivery recovery, table archiving, table backup, and audit-log ingestion fallback. Message-delivery recovery and audit-log ingestion fallback are enabled by default; archiving and backup are disabled. Archive settings are stored in `base_table_archive`; backup settings are stored in `base_table_backup`. Four message categories are provided: system, security, task, and business.
- Menus and permissions: the Admin roots are Home, User Management, System Management, and Development Tools. The mobile root is `99000000`, with Home at `99010000` and My Account at `99090000`. System Management contains Backup Management (data archiving, data backup, and their configuration/record/restore pages) and Configuration Management. Development Tools includes operations monitoring, runtime logs, cache query, project documents, and code generation. Each menu's `api` field lists the protected service methods it actually calls, and roles receive permissions through menu IDs.
- Security capabilities: the login-policy menu and its service/button permissions are granted only to the platform super administrator, and no login-policy record is created by default. OAuth client management menus, MFA configuration, and authentication-method dictionaries are initialized directly; the related business tables are created by the data layer.

`base_area.up.sql` writes administrative-division data in a separate script and is applied with `default_data.up.sql` in filename order. Message, MFA, OAuth-client, language, and translation tables are created by the data layer; these SQL files do not create tables.

## Localized Data

The three `i18n.{locale}.up.sql` files contain the `base_i18n` records for their locale. They cover system-configuration values and names, dictionary names, dictionary-item labels, menu titles, and scheduled-job names. The `target_type` convention is `1` configuration value, `2` configuration name, `3` dictionary name, `4` dictionary item, `5` menu, and `6` scheduled job. Every translation is inserted separately, and the primary-data filename sorts before the locale files so referenced records exist first.

When adding or changing configuration, dictionaries, menus, or jobs, update all three locale scripts and the matching `README.<locale>.md` files so `target_id`, locale codes, and primary data remain aligned.
