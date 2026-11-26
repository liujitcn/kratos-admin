import type { AiAttachment } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";

export const AI_ATTACHMENT_MAX_COUNT = 6;
export const AI_ATTACHMENT_MAX_SIZE_MB = 20;
const MAX_SIZE = AI_ATTACHMENT_MAX_SIZE_MB * 1024 * 1024;
const SUPPORTED_EXTENSIONS = ["txt", "md", "markdown", "log", "json", "xml", "csv", "xlsx", "pdf", "png", "jpg", "jpeg", "gif", "webp"];
export const AI_ATTACHMENT_ACCEPT = SUPPORTED_EXTENSIONS.map(extension => `.${extension}`).join(",");
const IMAGE_MIME_TYPES = new Map([
  ["png", "image/png"],
  ["jpg", "image/jpeg"],
  ["jpeg", "image/jpeg"],
  ["gif", "image/gif"],
  ["webp", "image/webp"],
  ["bmp", "image/bmp"],
  ["svg", "image/svg+xml"]
]);

/** 阻止本地附件上传的文件校验原因。 */
export type AIAttachmentUploadIssue = "unsupported" | "too_large" | "total_too_large";

/** 根据草稿中已占用的大小，为新文件分配剩余的附件数量与容量。 */
export function planAIAttachmentUploads(files: readonly File[], selectedSizes: readonly number[]) {
  const accepted: File[] = [];
  const rejected: { file: File; reason: AIAttachmentUploadIssue }[] = [];
  const remaining = Math.max(0, AI_ATTACHMENT_MAX_COUNT - selectedSizes.length);
  let total = selectedSizes.reduce((sum, size) => sum + size, 0);
  let truncated = false;
  for (const file of files) {
    if (accepted.length >= remaining) {
      truncated = true;
      break;
    }
    if (file.size > MAX_SIZE) {
      rejected.push({ file, reason: "too_large" });
    } else if (!SUPPORTED_EXTENSIONS.includes(getAIAttachmentExtension(file.name))) {
      rejected.push({ file, reason: "unsupported" });
    } else if (total + file.size > MAX_SIZE) {
      rejected.push({ file, reason: "total_too_large" });
    } else {
      accepted.push(file);
      total += file.size;
    }
  }
  return { accepted, rejected, remaining, truncated };
}

/** 获取小写的文件扩展名，忽略无扩展名的文件。 */
export function getAIAttachmentExtension(name?: string) {
  if (!name) return "";
  const dot = name.lastIndexOf(".");
  return dot < 0 ? "" : name.slice(dot + 1).toLowerCase();
}

/** 根据附件元信息补齐鉴权下载响应中的图片 MIME 类型。 */
export function getAIImageMimeType({ name, mime_type }: Pick<AiAttachment, "name" | "mime_type">) {
  const [mimeType = ""] = (mime_type ?? "").toLowerCase().split(";", 1);
  return mimeType.startsWith("image/") ? mimeType : IMAGE_MIME_TYPES.get(getAIAttachmentExtension(name));
}

/** 判断附件是否可以按图片预览。 */
export function isAIImageAttachment(attachment: Pick<AiAttachment, "name" | "mime_type">) {
  return Boolean(getAIImageMimeType(attachment));
}

/** 持久化的生成文件与用户上传文件共享消息附件字段，通过存储目录区分。 */
export function isAIGeneratedAttachment({ url }: Pick<AiAttachment, "url">) {
  const [path = ""] = (url ?? "").split(/[?#]/, 1);
  return path.includes("/data/ai/generated/") || /^\/?(?:data\/)?ai\/generated\//.test(path);
}
