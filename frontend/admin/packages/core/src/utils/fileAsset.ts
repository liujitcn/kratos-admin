import type { AxiosResponse } from "axios";
import service from "@/utils/request";
import { normalizeStaticAssetPath } from "@/utils/utils";

/** 将业务文件路径转换为浏览器可展示的地址，并为 /data/ 文件携带当前访问令牌。 */
export async function loadFileAssetSource(source: string, fallback = ""): Promise<string> {
  const value = String(source ?? "");
  if (!value) return fallback;
  if (/^(https?:)?\/\//.test(value) || value.startsWith("blob:") || value.startsWith("data:")) {
    return value;
  }

  const normalizedPath = normalizeStaticAssetPath(value);
  if (normalizedPath.startsWith("/") && !normalizedPath.startsWith("/data/")) return normalizedPath;

  const filePath = normalizedPath.startsWith("/data/") ? normalizedPath : `/data/${normalizedPath.replace(/^\/+/, "")}`;
  const staticBase = String(import.meta.env.VITE_APP_STATIC_URL ?? "").trim() || window.location.origin;
  const fileURL = new URL(staticBase, window.location.origin);
  fileURL.pathname = filePath;

  const response = await service<unknown, AxiosResponse<Blob>>({
    url: fileURL.toString(),
    method: "get",
    responseType: "blob"
  });
  const blob = response.data instanceof Blob ? response.data : new Blob([response.data]);
  return URL.createObjectURL(blob);
}
