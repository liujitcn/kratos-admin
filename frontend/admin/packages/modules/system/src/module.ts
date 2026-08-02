import type { Component } from "vue";
import { User } from "@element-plus/icons-vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";
import Ai from "./components/Ai.vue";

// 语言包随 System 模块注册，供 System 页面、组件和代码生成页面使用。
import enUS from "./locales/en-US.json";
import jaJP from "./locales/ja-JP.json";
import zhCN from "./locales/zh-CN.json";
import zhTW from "./locales/zh-TW.json";
import koKR from "./locales/ko-KR.json";
import frFR from "./locales/fr-FR.json";
import esES from "./locales/es-ES.json";

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
    "zh-TW": zhTW,
    "ko-KR": koKR,
    "fr-FR": frFR,
    "es-ES": esES,
    "en-US": enUS,
    "ja-JP": jaJP
  }
});
