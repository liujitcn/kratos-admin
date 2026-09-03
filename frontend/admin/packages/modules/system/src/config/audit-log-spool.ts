import { Document } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 日志入库回退配置模型。 */
export interface AuditLogSpoolConfig {
  file_path: string;
}

/** 日志入库回退配置定义。 */
export const auditLogSpoolConfig: RuntimeConfigDefinition<AuditLogSpoolConfig> = {
  key: "auditLogSpool",
  titleKey: "system.base.runtime_config.audit_log_spool.title",
  descriptionKey: "system.base.runtime_config.audit_log_spool.description",
  icon: Document,
  createModel: () => ({
    file_path: "./data/audit-log-spool"
  }),
  fields: [
    {
      prop: "file_path",
      labelKey: "system.base.runtime_config.audit_log_spool.field.file_path",
      component: "input",
      rules: [{ required: true, messageKey: "system.base.runtime_config.audit_log_spool.validation.file_path", trigger: "blur" }]
    }
  ]
};
