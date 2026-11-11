import { t } from "@liujitcn/kratos-admin-core";
import { defAiPluginService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_plugin";
import { readAiPluginOAuthCallback } from "../../views/ai/chat/workspace/state";

/** 完成专用插件回调；状态归属、一次性消费和PKCE兑换全部由服务端负责。 */
export async function completeAiPluginOAuthCallback(): Promise<void> {
  if (typeof window === "undefined") return;
  const request = readAiPluginOAuthCallback(window.location.search);
  if (!request) return;
  const address = new URL(window.location.href);
  for (const key of ["ai_plugin_oauth", "state", "code", "error", "error_description", "iss"]) address.searchParams.delete(key);
  window.history.replaceState(window.history.state, "", address.pathname + address.search + address.hash);
  let result: string;
  try {
    const connection = await defAiPluginService.CompleteAiPluginOAuth(request);
    if (connection.state === "connected") {
      if (window.sessionStorage.getItem("ai-plugin-oauth-popup") === "1") {
        window.sessionStorage.removeItem("ai-plugin-oauth-popup");
        window.close();
        return;
      }
      result = t("system.ai.workspace.oauth_completed");
    } else {
      result = connection.last_error || t("system.ai.workspace.failed");
    }
  } catch (error) {
    result = error instanceof Error ? error.message : t("system.ai.workspace.failed");
  }
  await ElMessageBox.alert(result, t("system.ai.workspace.authorize"), {
    confirmButtonText: t("system.ai.workspace.oauth_return"),
    showClose: false,
    closeOnClickModal: false,
    closeOnPressEscape: false
  });
  window.location.replace(address.pathname + address.search + "#/ai/chat");
}
