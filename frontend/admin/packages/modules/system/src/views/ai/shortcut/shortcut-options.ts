import { t } from "@liujitcn/kratos-admin-core";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import { AiShortcutSourceType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_shortcut";

/** 返回快捷入口来源选项。 */
export function sources() {
  return [
    { label: t("system.ai.shortcut.source.api"), value: AiShortcutSourceType.AI_SHORTCUT_SOURCE_TYPE_API },
    { label: t("system.ai.shortcut.source.custom"), value: AiShortcutSourceType.AI_SHORTCUT_SOURCE_TYPE_CUSTOM }
  ];
}

/** 返回快捷入口终端选项。 */
export function terminals() {
  return [
    { label: t("system.ai.shortcut.terminal.admin"), value: Terminal.TERMINAL_ADMIN },
    { label: t("system.ai.shortcut.terminal.app"), value: Terminal.TERMINAL_APP }
  ];
}
