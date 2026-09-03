import { Timer } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 日志在线保留配置模型。 */
export interface LogRetentionConfig {
  retention_days: number;
}

/** 日志在线保留配置定义。 */
export const logRetentionConfig: RuntimeConfigDefinition<LogRetentionConfig> = {
  key: "logRetention",
  titleKey: "system.base.runtime_config.log_retention.title",
  descriptionKey: "system.base.runtime_config.log_retention.description",
  icon: Timer,
  createModel: () => ({
    retention_days: 180
  }),
  fields: [
    {
      prop: "retention_days",
      labelKey: "system.base.runtime_config.field.online_retention_days",
      component: "input-number",
      props: { min: 0, max: 36500, controls: true },
      rules: [{ type: "number", min: 0, messageKey: "system.base.runtime_config.validation.retention_days", trigger: "blur" }]
    }
  ]
};
