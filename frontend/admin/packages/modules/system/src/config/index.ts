import { shallowReactive } from "vue";
import type { RuntimeConfigDefinition } from "./types";

import { baseLogFallbackConfig } from "./base-log-fallback";
import { securityIdentityConfig } from "./security-identity";
import { securityMigrationConfig } from "./security-migration";
import { securityOutboundConfig } from "./security-outbound";
import { securityProxyConfig } from "./security-proxy";
import { securityScriptConfig } from "./security-script";

export { baseLogFallbackConfig } from "./base-log-fallback";
export { securityIdentityConfig } from "./security-identity";
export { securityMigrationConfig } from "./security-migration";
export { securityOutboundConfig } from "./security-outbound";
export { securityProxyConfig } from "./security-proxy";
export { securityScriptConfig } from "./security-script";
export type {
  RuntimeConfigDefinition,
  RuntimeConfigField,
  RuntimeConfigMessageArg,
  RuntimeConfigModel,
  RuntimeConfigRule
} from "./types";

/** Admin 内置运行配置定义集合。 */
export const runtimeConfigDefinitions = shallowReactive<RuntimeConfigDefinition[]>([
  baseLogFallbackConfig,
  securityProxyConfig,
  securityScriptConfig,
  securityOutboundConfig,
  securityIdentityConfig,
  securityMigrationConfig
]);

/** 注册外部业务模块提供的运行配置定义。 */
export function registerRuntimeConfig(definition: RuntimeConfigDefinition) {
  if (runtimeConfigDefinitions.some(item => item.key === definition.key)) {
    throw new Error(`Runtime config key already registered: ${definition.key}`);
  }
  runtimeConfigDefinitions.push(definition);
}
