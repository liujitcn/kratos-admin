import { createRouter, createWebHashHistory, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/modules/auth";
import { HOME_URL, LOGIN_URL, ROUTER_WHITE_LIST } from "@/config";
import { initDynamicRouter } from "@/routers/modules/dynamicRouter";
import { staticRouter, errorRouter } from "@/routers/modules/staticRouter";
import NProgress from "@/config/nprogress";
import { isUnmatchedRoute } from "@/utils/router";
import { ensureAccessToken } from "@/utils/request";

const mode = import.meta.env.VITE_ROUTER_MODE;

const routerMode = {
  hash: () => createWebHashHistory(),
  history: () => createWebHistory()
};

/**
 * @description 📚 路由参数配置简介
 * @param path ==> 路由菜单访问路径
 * @param name ==> 路由 name (对应页面组件 name, 可用作 KeepAlive 缓存标识 && 按钮权限筛选)
 * @param redirect ==> 路由重定向地址
 * @param component ==> 视图文件路径
 * @param meta ==> 路由菜单元信息
 * @param meta.icon ==> 菜单和面包屑对应的图标
 * @param meta.title ==> 路由标题 (用作 document.title || 菜单的名称)
 * @param meta.hide ==> 是否在菜单中隐藏 (通常列表详情页需要隐藏)
 * @param meta.alwaysShow ==> 目录只有一个子路由时是否始终显示
 * @param meta.full ==> 菜单是否全屏 (示例：数据大屏页面)
 * @param meta.affix ==> 菜单是否固定在标签页中 (首页通常是固定项)
 * @param meta.keepAlive ==> 当前路由是否缓存
 * */
const router = createRouter({
  history: routerMode[mode](),
  routes: [...staticRouter, ...errorRouter],
  strict: false,
  scrollBehavior: () => ({ left: 0, top: 0 })
});

/**
 * @description 路由拦截 beforeEach
 * */
router.beforeEach(async to => {
  const authStore = useAuthStore();
  const redirectQuery = typeof to.query.redirect === "string" ? to.query.redirect : "";

  // 1.NProgress 开始
  NProgress.start();

  // 2.动态设置标题
  const title = import.meta.env.VITE_GLOB_APP_TITLE;
  document.title = to.meta.title ? `${to.meta.title} - ${title}` : title;

  const hasAccessToken = await ensureAccessToken();

  // 3.判断是访问登陆页，有 Token 就在当前页面，没有 Token 重置路由到登陆页
  if (to.path.toLocaleLowerCase() === LOGIN_URL) {
    const hasOauthCallback = typeof to.query.oauth_ticket === "string" || typeof to.query.oauth_error === "string";
    if (hasOauthCallback) {
      resetRouter();
      return true;
    }
    if (hasAccessToken) {
      // 登录态访问登录页时优先回到显式 redirect，避免 from 为根路径时触发重复重定向。
      const targetPath = redirectQuery && redirectQuery !== LOGIN_URL ? redirectQuery : HOME_URL;
      return targetPath;
    }
    resetRouter();
    return true;
  }

  // 4.判断访问页面是否在路由白名单地址(静态路由)中，如果存在直接放行
  if (ROUTER_WHITE_LIST.includes(to.path)) return true;

  // 5.判断是否有可用 Token，没有重定向到 login 页面
  if (!hasAccessToken) {
    const redirect = to.path === LOGIN_URL ? undefined : to.fullPath;
    return {
      path: LOGIN_URL,
      query: redirect ? { redirect } : undefined,
      replace: true
    };
  }

  // 6.如果没有菜单列表，就重新请求菜单列表并添加动态路由
  if (!authStore.authMenuListGet.length) {
    await initDynamicRouter();
    if (isUnmatchedRoute(router, to.path)) return getFirstAccessibleRoutePath();
    return { ...to, replace: true };
  }

  // 6.1 菜单已恢复但路由实例尚未重新挂载时，补跑一次动态路由注册，避免刷新或登录首跳直接命中 404。
  if (isUnmatchedRoute(router, to.path)) {
    await initDynamicRouter();
    if (isUnmatchedRoute(router, to.path)) return getFirstAccessibleRoutePath();
    return { ...to, replace: true };
  }

  // 7.存储 routerName 做按钮权限筛选
  authStore.setRouteName(to.name as string);

  // 8.正常访问页面
  return true;
});

/**
 * @description 重置路由
 * */
export const resetRouter = () => {
  const authStore = useAuthStore();
  authStore.flatMenuListGet.forEach(route => {
    const { name } = route;
    if (name && router.hasRoute(name)) router.removeRoute(name);
  });
};

/** 获取当前已注册动态菜单中的第一个可访问页面，作为首页不可用时的兜底落点。 */
function getFirstAccessibleRoutePath() {
  const authStore = useAuthStore();
  const systemRouteSet = new Set(["/", "/layout", LOGIN_URL, "/403", "/404", "/500"]);
  const firstRoute = authStore.flatMenuListGet.find(item => {
    if (!item.path || systemRouteSet.has(item.path) || item.meta?.hidden) return false;
    if (item.children?.length || !item.component || item.component === "Layout") return false;
    return !isUnmatchedRoute(router, item.path);
  });
  return firstRoute?.path ?? "/404";
}

/**
 * @description 路由跳转错误
 * */
router.onError(error => {
  NProgress.done();
  console.warn("路由错误", error.message);
});

/**
 * @description 路由跳转结束
 * */
router.afterEach(() => {
  NProgress.done();
});

export default router;
