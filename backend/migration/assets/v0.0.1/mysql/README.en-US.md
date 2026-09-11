# MySQL Default Initialization Resources

This directory contains the MySQL initialization resources for `v0.0.1`. The scripts insert default data only; they do not create tables. At startup, GORM `AutoMigrate` creates the enabled models, and the `.up.sql` files in this directory are then applied in filename order.

## File Responsibilities

| File | Contents |
| --- | --- |
| `default_data.up.sql` | Default languages, configuration, departments, dictionaries, dictionary items, jobs, tenant, message categories, menus, roles, redaction rules, redaction fields, redaction policies, and development accounts. |
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

`default_data.up.sql` currently writes the following default-data tables:

`base_language`, `base_config`, `base_dept`, `base_dict`, `base_dict_item`, `base_job`, `base_tenant`, `base_message_category`, `base_menu`, `base_role`, `base_user`, `base_redact_rule`, `base_redact_storage_policy`, and `base_redact_output_policy`.

- Languages: `zh-CN`, `zh-TW`, `en-US`, and `ja-JP` are provided; `zh-CN` is the primary language.
- Login tenant code: Admin (`site=2`) and app (`site=3`) both initialize the boolean setting `showTenantCode=true`, displaying the tenant-code input by default. The setting name includes English, Traditional Chinese, and Japanese translations.
- Identity data: a default tenant, system departments, five role templates, and the local development accounts `super` and `admin` are provided. The initial password is `112233` and is for local use only.
- Configuration and dictionaries: the data covers Admin and app settings, CAPTCHA, OAuth auto-registration, tenant-code display, MFA policy and methods; log ingestion fallback uses form configuration `baseLogFallback`, retention policies use `base_table_archive`, and backup policies use `base_table_backup`, while session lifetime and upload scanning use `authn.session` and `oss.upload_security`. Persistent dictionaries cover menu, audit, login-policy, message, permission, and code-generation enums.
- Jobs and messages: job IDs `1000-1004` are resource translation, message-delivery recovery, table archiving, table backup, and log ingestion fallback. Message-delivery recovery and log ingestion fallback are enabled by default; archiving and backup are disabled. Archive settings are stored in `base_table_archive`; backup settings are stored in `base_table_backup`. Four message categories are provided: system, security, task, and business.
- Menus and permissions: Admin has four roots: Home, Tenant Management, User Management, and System Management. The mobile root remains `99000000`. System Management has 11 second-level groups in priority order: Basic Management, Permission Management, Login Management, Notifications, Data Redaction, Data Backup, Data Archiving, Audit Logs, Scheduled Jobs, System Operations, and Development Tools. Their IDs run from `91010000` to `91110000`, with sort values from `10` to `110` in steps of 10. Third-level pages use `AA BB CC 00`; buttons use their page prefix with `DD=01-99`. Role references and all three menu translation scripts follow the new IDs; page paths, route names, components, and service permissions are preserved.
- Basic Management pages use consecutive IDs from `91010100` to `91010600`. System Configuration (`91010500`) maintains keys, types and values in one place. Type 6 loads registered form fields in the edit dialog; the separate System Parameters page is removed. Data Backup and Data Archiving each directly contain configuration, records, and restore records. System Operations contains Runtime Monitoring, Runtime Logs, Cache Query, and Database Migration Records (formerly Upgrade History). Development Tools displays only API Documentation and Code Generation. Dictionary Data remains hidden under Basic Management at `91010600`; hidden code-generation configuration and preview pages use `91110300` through `91110600` under Development Tools, and code-generation buttons use `91110201` through `91110209`.
- Security capabilities: the login-management menu and its service/button permissions are granted only to the platform super administrator, and no login-policy record is created by default. Data Masking menus and the default phone-response mask are granted to the system administrator. OAuth client management menus, MFA configuration, and authentication-method dictionaries are initialized directly; the related business tables are created by the data layer.
- Redaction rules: `base_redact_rule.up.sql` initializes every fixed template supported by code, inbound policies keyed by data source for user phone, email, and ID number, and outbound policies keyed by service name for user list, page, and detail operations. The Admin console manages them through the Data Masking pages for Redaction Rules, Storage Redaction Policies, and Response Redaction Policies.

`base_area.up.sql` writes administrative-division data in a separate script and is applied with `default_data.up.sql` in filename order. Message, MFA, OAuth-client, language, translation, and redaction tables are created by the data layer; these SQL files do not create tables.

## Localized Data

The three `i18n.{locale}.up.sql` files contain the `base_i18n` records for their locale. They cover system-configuration values and names, dictionary names, dictionary-item labels, menu titles, and scheduled-job names. The `target_type` convention is `1` configuration value, `2` configuration name, `3` dictionary name, `4` dictionary item, `5` menu, and `6` scheduled job. Every translation is inserted separately, and the primary-data filename sorts before the locale files so referenced records exist first.

When adding or changing configuration, dictionaries, menus, or jobs, update all three locale scripts and the matching `README.<locale>.md` files so `target_id`, locale codes, and primary data remain aligned.

The profile menu grants access to the current user’s login history; the server fixes both user and tenant from the authenticated identity. Online session listing and revocation are available only to the platform super administrator.

Tenant Management uses root `20000000` (sort 20), page `20010000`, and actions `20010100`–`20010400`. User Management remains root `30000000` (sort 30).

Code generation merges per-table menu permission blocks into `default_data.up.sql` without creating a new version. Restore recovers the script and previous menus, translations and permissions; business table data is unaffected.

The /dashboard menu uses Home and HomeFilled. Admin menu icons come from the Element Plus icon selector and distinguish configuration, backup/archive, audit logs, and monitoring; mobile icons follow their own convention.
