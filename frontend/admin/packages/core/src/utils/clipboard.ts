/** 将文本复制到系统剪贴板，并兼容非安全上下文的管理端开发地址。 */
export async function copyText(content: string): Promise<void> {
  if (typeof navigator !== "undefined" && navigator.clipboard && typeof window !== "undefined" && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(content);
      return;
    } catch {
      // Clipboard API 受权限限制时继续尝试传统 DOM 方案。
    }
  }
  if (typeof document === "undefined") throw new Error("copy failed");
  const textarea = document.createElement("textarea");
  textarea.value = content;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) throw new Error("copy failed");
}
