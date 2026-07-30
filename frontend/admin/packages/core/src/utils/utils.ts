/**
 * JSON 格式化显示。
 */
export function formatJson(str: string) {
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

/**
 * 将后端分单位金额转换为元字符串。
 */
export function formatPrice(price?: number) {
  if (!price) return "0.00";
  return (price / 100).toFixed(2);
}

/**
 * 按静态资源域名补齐图片地址。
 */
export function formatSrc(src: string) {
  const value = String(src ?? "").trim();
  if (!value) return value;
  if (/^(https?:)?\/\//.test(value) || value.startsWith("data:") || value.startsWith("blob:")) {
    return value;
  }

  const configuredBase = String(import.meta.env.VITE_APP_STATIC_URL ?? "").trim();
  if (value.startsWith("/admin/")) {
    return new URL(value, `${window.location.origin}/`).toString();
  }

  const staticBase = configuredBase
    ? new URL(`${configuredBase.replace(/\/$/, "")}/`, `${window.location.origin}/`).toString()
    : `${window.location.origin}/admin/`;
  const normalizedPath = value.replace(/^\/+/, "").replace(/^admin\/+/, "");
  return new URL(normalizedPath, staticBase).toString();
}
