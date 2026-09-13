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

/** 将时间值格式化为统一的本地日期时间文本。 */
export function formatDateTime(value?: string | number | Date) {
  if (value === undefined || value === null || value === "") return "-";
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
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
  const staticBase = configuredBase
    ? new URL(`${configuredBase.replace(/\/$/, "")}/`, `${window.location.origin}/`).toString()
    : `${window.location.origin}/`;
  const normalizedPath = normalizeStaticAssetPath(value).replace(/^\/+/, "");
  return new URL(normalizedPath, staticBase).toString();
}

/** 将对象路径转换为统一的数据访问路径。 */
export function normalizeStaticAssetPath(src: string) {
  const value = String(src ?? "");
  if (value.startsWith("http://") || value.startsWith("https://")) {
    return value;
  }
  if (value.startsWith("/data/") || value.startsWith("/admin/")) return value;
  if (value.startsWith("data/")) return `/${value}`;
  return value;
}

/** 将富文本中的媒体路径转换为统一的数据访问路径。 */
export function normalizeRichTextMediaPaths(html: string) {
  if (typeof DOMParser === "undefined") return html;

  const document = new DOMParser().parseFromString(html, "text/html");
  document.body.querySelectorAll<HTMLElement>("img[src], video[src]").forEach(element => {
    const src = element.getAttribute("src");
    if (src) element.setAttribute("src", normalizeStaticAssetPath(src));
  });
  return document.body.innerHTML;
}
