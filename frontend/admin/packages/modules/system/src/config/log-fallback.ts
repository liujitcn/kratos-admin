import { Document } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 日志 fallback 配置模型。 */
export interface LogFallbackConfig {
  fallback_file: string;
  fallback_integrity_key: string;
}

/** 日志 fallback 配置定义。 */
export const logFallbackConfig: RuntimeConfigDefinition<LogFallbackConfig> = {
  key: "logFallback",
  titleKey: "system.base.runtime_config.log_fallback.title",
  descriptionKey: "system.base.runtime_config.log_fallback.description",
  icon: Document,
  createModel: () => ({
    fallback_file: "./data/log-fallback/admin-log.jsonl",
    fallback_integrity_key: ""
  }),
  fields: [
    {
      prop: "fallback_file",
      labelKey: "system.base.runtime_config.field.fallback_file",
      component: "input",
      rules: [{ required: true, messageKey: "system.base.runtime_config.validation.fallback_file", trigger: "blur" }]
    },
    {
      prop: "fallback_integrity_key",
      labelKey: "system.base.runtime_config.field.fallback_integrity_key",
      component: "password",
      props: { showPassword: true, autocomplete: "new-password" }
    }
  ]
};
