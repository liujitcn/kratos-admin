import { Document } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 日志入库回退配置模型。 */
export interface BaseLogFallbackConfig {
  file_path: string;
}

/** 日志入库回退配置定义。 */
export const baseLogFallbackConfig: RuntimeConfigDefinition<BaseLogFallbackConfig> = {
  key: "baseLogFallback",
  titleKey: "system.base.runtime_config.base_log_fallback.title",
  descriptionKey: "system.base.runtime_config.base_log_fallback.description",
  icon: Document,
  createModel: () => ({
    file_path: "./logs/base-log-fallback"
  }),
  fields: [
    {
      prop: "file_path",
      labelKey: "system.base.runtime_config.base_log_fallback.field.file_path",
      component: "input",
      rules: [{ required: true, messageKey: "system.base.runtime_config.base_log_fallback.validation.file_path", trigger: "blur" }]
    }
  ]
};
