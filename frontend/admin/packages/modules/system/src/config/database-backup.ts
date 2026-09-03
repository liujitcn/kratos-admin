import { Coin } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 数据库备份配置模型。 */
export interface DatabaseBackupConfig {
  enabled: boolean;
  integrity_key: string;
  encryption_key: string;
  directory: string;
  gzip: boolean;
  retention_count: number;
}

/** 数据库备份配置定义。 */
export const databaseBackupConfig: RuntimeConfigDefinition<DatabaseBackupConfig> = {
  key: "databaseBackup",
  titleKey: "system.base.runtime_config.backup.title",
  descriptionKey: "system.base.runtime_config.backup.description",
  icon: Coin,
  createModel: () => ({
    enabled: false,
    integrity_key: "",
    encryption_key: "",
    directory: "./backups",
    gzip: true,
    retention_count: 7
  }),
  fields: [
    { prop: "enabled", labelKey: "system.base.runtime_config.field.enabled", component: "switch" },
    {
      prop: "integrity_key",
      labelKey: "system.base.runtime_config.field.integrity_key",
      component: "password",
      props: { showPassword: true, autocomplete: "new-password" }
    },
    {
      prop: "encryption_key",
      labelKey: "system.base.runtime_config.field.encryption_key",
      component: "password",
      props: { showPassword: true, autocomplete: "new-password" }
    },
    {
      prop: "directory",
      labelKey: "system.base.runtime_config.field.directory",
      component: "input",
      rules: [{ required: true, messageKey: "system.base.runtime_config.validation.directory", trigger: "blur" }]
    },
    { prop: "gzip", labelKey: "system.base.runtime_config.field.gzip", component: "switch" },
    {
      prop: "retention_count",
      labelKey: "system.base.runtime_config.field.retention_count",
      component: "input-number",
      props: { min: 1, max: 1000, controls: true },
      rules: [{ type: "number", min: 1, messageKey: "system.base.runtime_config.validation.retention_count", trigger: "blur" }]
    }
  ]
};
