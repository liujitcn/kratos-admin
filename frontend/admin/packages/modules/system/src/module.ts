import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";
import Ai from "./components/ai/Ai.vue";
import ForcedPasswordDialog from "./components/account/ForcedPasswordDialog.vue";
import Notification from "./components/notification/Notification.vue";

// 语言包由同步脚本生成并注册，供 System 页面、组件和代码生成页面使用。
import { LOCALE_MESSAGES } from "./locales/generated";
import { registerSecretRevealHandler } from "@liujitcn/kratos-admin-core/security";
import { defSecretCryptoService } from "@liujitcn/kratos-admin-core/api/base/v1/secret_crypto";

// 注册密钥明文查看实现，供 ProForm secret 字段的眼睛按需拉取。
registerSecretRevealHandler(async (resource, id, field) => {
  const res = await defSecretCryptoService.RevealSecretField({ resource, id, field });
  return res.text;
});

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** System 管理端业务模块。 */
export const systemAdminModule = defineAdminModule({
  name: "system",
  views: viewModules,
  headerTools: [
    { name: "notification", component: Notification },
    { name: "ai", component: Ai },
    { name: "forced-password-dialog", component: ForcedPasswordDialog }
  ],
  userMenuActions: [{ name: "profile", labelKey: "system.profile.title", menuName: "Profile", icon: User }],
  routeOptions: {
    Profile: { reuseTabAcrossQuery: true }
  },
  messages: LOCALE_MESSAGES
});
