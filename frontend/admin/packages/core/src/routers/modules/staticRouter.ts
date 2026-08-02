import { RouteRecordRaw } from "vue-router";
import { HOME_URL, LOGIN_URL } from "@/config";
import { ADMIN_STATIC_VIEWS, getAdminViewRegistry } from "@/modules";

/** 创建通过视图注册表解析的静态页面加载器。 */
function createAdminStaticViewLoader(viewPath: string) {
  return () => {
    const viewLoader = getAdminViewRegistry().resolve(viewPath);
    if (!viewLoader) throw new Error(`未注册管理端静态页面：${viewPath}`);
    return viewLoader();
  };
}

/**
 * staticRouter (静态路由)
 */
export const staticRouter: RouteRecordRaw[] = [
  {
    path: "/",
    redirect: HOME_URL
  },
  {
    path: LOGIN_URL,
    name: "Login",
    component: createAdminStaticViewLoader(ADMIN_STATIC_VIEWS.LOGIN),
    meta: {
      title: "",
      titleKey: "core.route.login"
    }
  },
  {
    path: "/layout",
    name: "Layout",
    component: () => import("@/layouts/index.vue"),
    // component: () => import("@/layouts/indexAsync.vue"),
    redirect: HOME_URL,
    children: []
  }
];

/**
 * errorRouter (错误页面路由)
 */
export const errorRouter = [
  {
    path: "/403",
    name: "403",
    component: createAdminStaticViewLoader(ADMIN_STATIC_VIEWS.FORBIDDEN),
    meta: {
      title: "",
      titleKey: "core.route.forbidden"
    }
  },
  {
    path: "/404",
    name: "404",
    component: createAdminStaticViewLoader(ADMIN_STATIC_VIEWS.NOT_FOUND),
    meta: {
      title: "",
      titleKey: "core.route.notFound"
    }
  },
  {
    path: "/500",
    name: "500",
    component: createAdminStaticViewLoader(ADMIN_STATIC_VIEWS.SERVER_ERROR),
    meta: {
      title: "",
      titleKey: "core.route.serverError"
    }
  },
  // Resolve refresh page, route warnings
  {
    path: "/:pathMatch(.*)*",
    component: createAdminStaticViewLoader(ADMIN_STATIC_VIEWS.NOT_FOUND)
  }
];
