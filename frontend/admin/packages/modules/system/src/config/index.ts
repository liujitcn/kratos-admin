import { shallowReactive } from "vue";
import type { RuntimeConfigDefinition } from "./types";

import { auditLogSpoolConfig } from "./audit-log-spool";

export { auditLogSpoolConfig } from "./audit-log-spool";
export type {
  RuntimeConfigDefinition,
  RuntimeConfigField,
  RuntimeConfigMessageArg,
  RuntimeConfigModel,
  RuntimeConfigRule
} from "./types";

/** Admin 内置运行配置定义集合。 */
export const runtimeConfigDefinitions = shallowReactive<RuntimeConfigDefinition[]>([
  auditLogSpoolConfig
]);

/** 注册外部业务模块提供的运行配置定义。 */
export function registerRuntimeConfig(definition: RuntimeConfigDefinition) {
  if (runtimeConfigDefinitions.some(item => item.key === definition.key)) {
    throw new Error(`Runtime config key already registered: ${definition.key}`);
  }
  runtimeConfigDefinitions.push(definition);
}
