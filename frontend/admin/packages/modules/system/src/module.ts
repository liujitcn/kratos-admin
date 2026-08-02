import type { Component } from "vue";
import { User } from "@element-plus/icons-vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";
import Ai from "./components/Ai.vue";
import enUS from "./locales/en-US.json";
import jaJP from "./locales/ja-JP.json";
import zhCN from "./locales/zh-CN.json";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** System 管理端业务模块。 */
export const systemAdminModule = defineAdminModule({
  name: "system",
  views: viewModules,
  headerTools: [{ name: "ai", component: Ai }],
  userMenuActions: [{ name: "profile", labelKey: "system.profile.title", menuName: "Profile", icon: User }],
  routeOptions: {
    Profile: { reuseTabAcrossQuery: true }
  },
  messages: {
    "zh-CN": zhCN,
    "en-US": enUS,
    "ja-JP": jaJP
  }
});
