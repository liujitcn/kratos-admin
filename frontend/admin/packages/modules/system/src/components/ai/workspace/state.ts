import type { AiPluginSelection, CompleteAiPluginOAuthRequest } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";

/** 助手内部功能，不切换宿主路由或销毁聊天状态。 */
export type AssistantView = "chat" | "files" | "automations" | "plugins";

/** 判断下拉菜单值是否为助手工作区视图。 */
export function isAssistantView(value: string): value is AssistantView {
  return value === "chat" || value === "files" || value === "automations" || value === "plugins";
}

/** 仅桌面且获得配置权限时展示管理操作。 */
export function canConfigureAssistant(mobile: boolean, permitted: boolean): boolean {
  return !mobile && permitted;
}

/** 判断插件是否已有已连接账号，避免重复发起连接。 */
export function hasConnectedPluginConnection(
  pluginID: string,
  connections: readonly { plugin_id: string; state: string; can_manage: boolean }[]
): boolean {
  return connections.some(item => item.plugin_id === pluginID && item.can_manage && item.state === "connected");
}

/** 用字符串编码确定插件及账号，不把雪花ID转为数字。 */
export function pluginSelectionKey(selection: AiPluginSelection): string {
  // protojson 会省略空连接字段，归一化 undefined 避免出现 "1:undefined" 标签。
  return `${selection.plugin_id}:${selection.connection_id ?? ""}`;
}

/** 还原组件选择值，插件ID与连接ID均保留原字符串。 */
export function selectionFromKey(key: string): AiPluginSelection {
  const [plugin_id, connection_id = ""] = key.split(":");
  return { plugin_id, connection_id };
}

/** 仅解析专用插件回调，授权码、拒绝原因与发行方交给服务端校验。 */
export function readAiPluginOAuthCallback(search: string): CompleteAiPluginOAuthRequest | undefined {
  const query = new URLSearchParams(search);
  if (query.get("ai_plugin_oauth") !== "1") return;
  return {
    state: query.get("state") ?? "",
    code: query.get("code") ?? "",
    error: query.get("error") ?? "",
    iss: query.get("iss") ?? ""
  };
}
