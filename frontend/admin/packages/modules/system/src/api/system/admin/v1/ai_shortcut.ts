import service from "@liujitcn/kratos-admin-core/request";
import type { AiShortcutService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_shortcut";

// 保持与 system.admin.v1.AiShortcutService 的 HTTP 契约一致。
const URL = "/v1/admin/ai/shortcut";

/** 快捷入口管理请求。 */
export const defAiShortcutService: AiShortcutService = {
  PageAiShortcut: request => service({ url: URL, method: "get", params: request }),
  GetAiShortcut: ({ id }) => service({ url: `${URL}/${id}`, method: "get" }),
  OptionAiShortcutTool: request => service({ url: `${URL}/tool/option`, method: "get", params: request }),
  CreateAiShortcut: ({ shortcut }) => service({ url: URL, method: "post", data: shortcut }),
  UpdateAiShortcut: ({ shortcut }) => service({ url: `${URL}/${shortcut?.id}`, method: "put", data: shortcut }),
  DeleteAiShortcut: ({ id }) => service({ url: `${URL}/${id}`, method: "delete" }),
  SetAiShortcutStatus: request => service({ url: `${URL}/${request.id}/status`, method: "put", data: request })
};
