import { ADMIN_STATIC_VIEWS, defineAdminModule, type AdminStaticViewModules, type AdminViewModules } from "./index";

// 语言包由同步脚本生成并注册，供管理端公共组件、登录页和错误处理使用。
import { LOCALE_MESSAGES } from "../locales/generated";

const staticViewModules: AdminStaticViewModules = {
  [ADMIN_STATIC_VIEWS.LOGIN]: () => import("../views/login/index.vue"),
  [ADMIN_STATIC_VIEWS.FORBIDDEN]: () => import("../views/error/403.vue"),
  [ADMIN_STATIC_VIEWS.NOT_FOUND]: () => import("../views/error/404.vue"),
  [ADMIN_STATIC_VIEWS.SERVER_ERROR]: () => import("../views/error/500.vue"),
  [ADMIN_STATIC_VIEWS.PENDING]: () => import("../views/error/pending.vue")
};

// Demo 演示页面：集中展示 core 的 ProTable / ProForm / TreeFilter 等组件能力。
// 路由完全由后端菜单控制，菜单 component 前缀为 "kratos-admin/demo"；
// 页面数量固定，显式注册代替 glob，避免依赖 vite 的 glob 解析行为。
const demoViewModules: AdminViewModules = {
  "demo/tree-table/index": () => import("../views/demo/tree-table/index.vue"),
  "demo/table/index": () => import("../views/demo/table/index.vue"),
  "demo/tree-grid/index": () => import("../views/demo/tree-grid/index.vue")
};

/**
 * kratos-admin 内置基础模块。
 */
export const kratosAdminModule = defineAdminModule({
  name: "kratos-admin",
  views: demoViewModules,
  staticViews: staticViewModules,
  messages: LOCALE_MESSAGES
});
