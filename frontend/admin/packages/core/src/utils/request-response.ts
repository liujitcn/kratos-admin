/** 判断响应是否包含结构化错误码。字符串 code 属于业务数据字段，不作为错误码。 */
export function hasStructuredErrorCode<T extends object>(value: T): value is T & { code: number } {
  return typeof value === "object" && !Array.isArray(value) && "code" in value && typeof value.code === "number";
}

/** 返回结构化错误提示，并为缺少 message 的限流响应提供本地化回退。 */
export function resolveStructuredErrorMessage(
  value: { message?: string; reason?: string },
  translate: (key: string) => string
): string {
  if (value.message) return value.message;
  if (value.reason === "RATE_LIMITED") return translate("common.error.rate_limited");
  return "";
}

/** 过滤 Axios 和浏览器网络层的技术错误，避免原始英文文案直接展示给用户。 */
export function resolveRequestErrorMessage(value: unknown, fallback: string): string {
  const message = typeof value === "string" ? value.trim() : "";
  if (!message) return fallback;
  if (/^(?:axioserror:\s*)?request failed with status code \d+(?:\s|:|$|[（(])/i.test(message)) return fallback;
  if (/^network error$/i.test(message) || /^axioserror(?::|$)/i.test(message) || /^err_[a-z_]+$/i.test(message)) return fallback;
  return message;
}
