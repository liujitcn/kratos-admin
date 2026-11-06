import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";
import { LOCALE_MESSAGES } from "./locales/generated";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** BI 分析管理端业务模块。 */
export const biAdminModule = defineAdminModule({
  name: "bi",
  views: viewModules,
  messages: LOCALE_MESSAGES
});
