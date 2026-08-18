import type { CodeGenLocaleConfig } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

/** 代码生成国际化配置的 JSON 兼容形态。 */
type CodeGenLocaleConfigRecord = Record<string, CodeGenLocaleConfig>;

/** 将 REST 响应中的 JSON 对象规范为生成 RPC 声明使用的 Map。 */
export function normalizeCodeGenLocaleConfigMap(value: unknown): Map<string, CodeGenLocaleConfig> {
  if (value instanceof Map) return new Map(value);
  if (!value || typeof value !== "object") return new Map();
  return new Map(Object.entries(value as CodeGenLocaleConfigRecord));
}

/** 将代码生成国际化 Map 转为后端可反序列化的 JSON 对象。 */
export function serializeCodeGenLocaleConfigMap(value: unknown): CodeGenLocaleConfigRecord {
  if (value instanceof Map) return Object.fromEntries(value);
  if (!value || typeof value !== "object") return {};
  return value as CodeGenLocaleConfigRecord;
}
