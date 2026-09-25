import { t } from "@liujitcn/kratos-admin-core";

const PERSISTED_MESSAGE_PREFIX = "__I18N__:";

/** 将持久化的国际化消息标记按当前语言渲染，兼容历史纯文本记录。 */
export function resolvePersistedMessage(value?: string): string {
  if (value === "系统") return t("system.notification.sender.system");
  if (!value?.startsWith(PERSISTED_MESSAGE_PREFIX)) return value ?? "";
  const marker = value.split(/\r?\n/, 1)[0].slice(PERSISTED_MESSAGE_PREFIX.length);
  const separator = marker.indexOf("?");
  const key = separator < 0 ? marker : marker.slice(0, separator);
  if (!key) return value;
  const params = separator < 0 ? {} : Object.fromEntries(new URLSearchParams(marker.slice(separator + 1)));
  const localized = t(key, params);
  return localized === key ? value : localized;
}

/** 翻译异步任务输出中的消息标记，并保留多条输出的分行格式。 */
export function resolvePersistedTaskOutput(value?: string): string {
  return (value ?? "").split("<br/>").map(resolvePersistedMessage).join("\n");
}
