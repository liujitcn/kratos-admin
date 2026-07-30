import type { RouteLocationRaw, Router } from "vue-router";
import { initDynamicRouter } from "@/routers/modules/dynamicRouter";

/**
 * 判断当前解析结果是否只命中了全局 404 占位路由。
 */
export function isUnmatchedRoute(router: Router, path: string) {
  const resolved = router.resolve(path);
  if (!resolved.matched.length) return true;
  return resolved.matched.some(item => item.path === "/:pathMatch(.*)*");
}

/**
 * 将前端路由地址转换为当前站点的绝对 URL，供后端 OAuth 回调重定向使用。
 */
export function resolveFrontendRouteURL(router: Router, location: RouteLocationRaw) {
  const href = router.resolve(location).href;
  if (href.startsWith("#")) {
    return `${window.location.origin}${window.location.pathname}${window.location.search}${href}`;
  }
  const frontendHref = href.startsWith("/") ? href : `${window.location.pathname}${window.location.search}${href}`;
  return new URL(frontendHref, window.location.origin).toString();
}

/**
 * 统一处理站内页面跳转；若目标路由尚未注册，则补跑动态路由初始化后重试。
 * 若路由实例已可解析但 push 失败，则降级为浏览器地址跳转。
 */
export async function navigateTo(router: Router, path: string, query?: Record<string, string | number>) {
  const target = { path, query };

  // 隐藏业务页依赖动态菜单注册，首次进入或权限刚调整后可能尚未挂载到路由实例。
  if (isUnmatchedRoute(router, path)) {
    await initDynamicRouter().catch(() => undefined);
  }

  if (isUnmatchedRoute(router, path)) return;

  return router.push(target).catch(() => {
    const fallbackResolved = router.resolve(target);
    window.location.href = fallbackResolved.href;
  });
}
