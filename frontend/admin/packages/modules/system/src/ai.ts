import type { Component } from "vue";
import { getRegisteredAdminModules } from "@liujitcn/kratos-admin-core";

/** ADMIN_AI_EXTENSION AI 助手扩展名称。 */
export const ADMIN_AI_EXTENSION = "admin.ai";

/** AdminAiFlowBlockRenderer 描述一个可组合的结构化流程块渲染器。 */
export interface AdminAiFlowBlockRenderer {
  /** 模块内稳定的渲染器名称。 */
  key: string;
  /** 负责渲染匹配流程块的 Vue 组件。 */
  component: Component;
  /** 仅接管指定类型；为空时接管当前消息中的全部流程块。 */
  types?: readonly string[];
}

/** AdminAiExtension AI 助手可扩展能力。 */
export interface AdminAiExtension {
  /** 结构化流程卡片组件。 */
  flowBlocks?: Component;
  /** 可与其他业务模块并列组合的流程块渲染器。 */
  flowBlockRenderers?: AdminAiFlowBlockRenderer[];
  /** BI 模块按需提供的问数建卡弹窗。 */
  biCardDialog?: Component;
}

/** getAdminAiExtension 合并各模块的 AI 助手能力，流程渲染器按模块顺序组合。 */
export function getAdminAiExtension(): AdminAiExtension {
  const extension = getRegisteredAdminModules().reduce<AdminAiExtension>(
    (extension, module) => Object.assign(extension, module.extensions?.[ADMIN_AI_EXTENSION]),
    {}
  );
  const flowBlockRenderers = getAdminAiFlowBlockRenderers();
  if (flowBlockRenderers.length) extension.flowBlockRenderers = flowBlockRenderers;
  return extension;
}

/** getAdminAiFlowBlockRenderers 收集所有模块的结构化流程块渲染器，并兼容旧版单组件扩展。 */
export function getAdminAiFlowBlockRenderers(): AdminAiFlowBlockRenderer[] {
  return getRegisteredAdminModules().flatMap(module => {
    const extension = module.extensions?.[ADMIN_AI_EXTENSION] as AdminAiExtension | undefined;
    if (!extension) return [];
    const legacy = extension.flowBlocks
      ? [{ key: `${module.name}:flowBlocks`, component: extension.flowBlocks }]
      : [];
    const renderers = (extension.flowBlockRenderers ?? []).filter(item => item?.key && item.component);
    return [
      ...legacy,
      ...renderers.map(item => ({ ...item, key: `${module.name}:${item.key}` }))
    ];
  });
}
