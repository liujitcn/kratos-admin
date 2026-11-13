import type { Component } from "vue";
import { getRegisteredAdminModules } from "@liujitcn/kratos-admin-core";

/** ADMIN_AI_EXTENSION AI 助手扩展名称。 */
export const ADMIN_AI_EXTENSION = "admin.ai";

/** AdminAiExtension AI 助手可扩展能力。 */
export interface AdminAiExtension {
  /** 结构化流程卡片组件。 */
  flowBlocks?: Component;
  /** BI 模块按需提供的问数建卡弹窗。 */
  biCardDialog?: Component;
}

/** getAdminAiExtension 合并各模块的 AI 助手能力，同一字段由后注册模块覆盖。 */
export function getAdminAiExtension(): AdminAiExtension {
  return getRegisteredAdminModules().reduce<AdminAiExtension>(
    (extension, module) => Object.assign(extension, module.extensions?.[ADMIN_AI_EXTENSION]),
    {}
  );
}
