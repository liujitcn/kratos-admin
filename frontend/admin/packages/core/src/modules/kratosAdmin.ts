import { ADMIN_STATIC_VIEWS, defineAdminModule, type AdminStaticViewModules } from "./index";

// 语言包由同步脚本生成并注册，供管理端公共组件、登录页和错误处理使用。
import { LOCALE_MESSAGES } from "../locales/generated";

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
  messages: LOCALE_MESSAGES
});
