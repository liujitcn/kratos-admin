import { Refresh } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 一次性安全迁移配置表单。 */
export interface SecurityMigrationConfig {
  backfill_sensitive_data: boolean;
  legacy_plaintext_manifest: string;
}

/** 一次性安全迁移配置定义。 */
export const securityMigrationConfig: RuntimeConfigDefinition<SecurityMigrationConfig> = {
  key: "securityMigration",
  titleKey: "system.base.runtime_config.security_migration.title",
  descriptionKey: "system.base.runtime_config.security_migration.description",
  icon: Refresh,
  createModel: () => ({ backfill_sensitive_data: false, legacy_plaintext_manifest: "" }),
  fields: [
    {
      prop: "backfill_sensitive_data",
      labelKey: "system.base.runtime_config.security_migration.field.backfill_sensitive_data",
      component: "switch"
    },
    {
      prop: "legacy_plaintext_manifest",
      labelKey: "system.base.runtime_config.security_migration.field.legacy_plaintext_manifest",
      component: "textarea",
      colSpan: 24,
      props: { rows: 8, placeholder: '{"ai_provider":{"api_key":[3]}}' }
    }
  ]
};
