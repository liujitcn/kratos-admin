import { ADMIN_STATIC_VIEWS, defineAdminModule, type AdminStaticViewModules } from "./index";

// 语言包由 core 模块注册，供管理端公共组件、登录页和错误处理使用。
import enUS from "../locales/en-US.json";
import jaJP from "../locales/ja-JP.json";
import zhCN from "../locales/zh-CN.json";
import zhTW from "../locales/zh-TW.json";
import koKR from "../locales/ko-KR.json";
import frFR from "../locales/fr-FR.json";
import esES from "../locales/es-ES.json";

const staticViewModules: AdminStaticViewModules = {
  [ADMIN_STATIC_VIEWS.LOGIN]: () => import("../views/login/index.vue"),
  [ADMIN_STATIC_VIEWS.FORBIDDEN]: () => import("../views/error/403.vue"),
  [ADMIN_STATIC_VIEWS.NOT_FOUND]: () => import("../views/error/404.vue"),
  [ADMIN_STATIC_VIEWS.SERVER_ERROR]: () => import("../views/error/500.vue"),
  [ADMIN_STATIC_VIEWS.PENDING]: () => import("../views/error/pending.vue")
};

/**
 * kratos-admin 内置基础模块。
 */
export const kratosAdminModule = defineAdminModule({
  name: "kratos-admin",
  staticViews: staticViewModules,
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
