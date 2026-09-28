/** 判断响应是否包含结构化错误码。 */
export function hasStructuredErrorCode(value: unknown): value is { code: string | number } {
  return typeof value === "object" && value !== null && "code" in value;
}
