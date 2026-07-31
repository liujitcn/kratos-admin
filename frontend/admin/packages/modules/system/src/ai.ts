import type { Component } from "vue";
import { getAdminModuleExtension } from "@liujitcn/kratos-admin-core";

/** ADMIN_AI_EXTENSION AI 助手扩展名称。 */
export const ADMIN_AI_EXTENSION = "admin.ai";

/** AdminAiExtension AI 助手可扩展能力。 */
export interface AdminAiExtension {
  /** 结构化流程卡片组件。 */
  flowBlocks?: Component;
}

/** getAdminAiExtension 获取当前宿主注册的 AI 助手扩展。 */
export function getAdminAiExtension(): AdminAiExtension {
  return getAdminModuleExtension<AdminAiExtension>(ADMIN_AI_EXTENSION) ?? {};
}
