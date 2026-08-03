import type { Component } from "vue";
import { User } from "@element-plus/icons-vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";
import Ai from "./components/Ai.vue";

// 语言包由同步脚本生成并注册，供 System 页面、组件和代码生成页面使用。
import { LOCALE_MESSAGES } from "./locales/generated";

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
  messages: LOCALE_MESSAGES
});
