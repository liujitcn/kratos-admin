import router from "@/routers/index";
import { LOGIN_URL } from "@/config";
import { t } from "@/locales";
import { RouteRecordRaw } from "vue-router";
import { ElNotification } from "element-plus";
import { useUserStore } from "@/stores/modules/user";
import { useAuthStore } from "@/stores/modules/auth";
import type { RouteItem } from "@/rpc/system/admin/v1/auth";
import { BaseMenuType } from "@/rpc/system/common/v1/enum";
import { getRouteMetaFull } from "@/utils";
import { ADMIN_STATIC_VIEWS, getAdminViewRegistry } from "../../modules";

/** 解析后的后端路由项，补齐完整路径和重定向。 */
type ResolvedRouteItem = {
  item: RouteItem;
  path: string;
  redirect?: string;
};

/**
 * 判断是否为外链路径。
 */
function isExternalPath(path: string) {
  return /^(https?:|mailto:|tel:)/.test(path);
}

/**
 * 规范化绝对路由路径，统一前导斜杠并移除重复/尾部斜杠。
 */
function normalizeAbsolutePath(path: string) {
  if (!path) return "";
  const pathWithSlash = path.startsWith("/") ? path : `/${path}`;
  const normalizedPath = pathWithSlash.replace(/\/{2,}/g, "/");
  if (normalizedPath === "/") return normalizedPath;
  return normalizedPath.replace(/\/+$/, "");
}

/**
 * 结合父级路径解析当前路由路径。
 */
function resolveRoutePath(rawPath: string | undefined, parentPath = "") {
  if (!rawPath) return "";
  const trimmedPath = rawPath.trim();
  if (!trimmedPath) return "";
  if (isExternalPath(trimmedPath)) return trimmedPath;
  if (trimmedPath.startsWith("/")) return normalizeAbsolutePath(trimmedPath);

  const normalizedParentPath = normalizeAbsolutePath(parentPath);
  if (!normalizedParentPath || normalizedParentPath === "/") {
    return normalizeAbsolutePath(trimmedPath);
  }
  return normalizeAbsolutePath(`${normalizedParentPath}/${trimmedPath}`);
}

/**
 * 递归解析菜单的完整路径，兼容子菜单相对路径。
 */
function buildResolvedRouteItems(menuList: RouteItem[], parentPath = ""): ResolvedRouteItem[] {
  const resolvedRouteItems: ResolvedRouteItem[] = [];

  menuList.forEach(item => {
    const currentPath = resolveRoutePath(item.path, parentPath);
    const redirectBasePath = currentPath || parentPath;
    const currentRedirect = resolveRoutePath(item.redirect, redirectBasePath);

    resolvedRouteItems.push({
      item,
      path: currentPath,
      redirect: currentRedirect || undefined
    });

    if (!item.children?.length) return;

    // 子路由优先基于当前节点路径进行拼接，保证目录 + 菜单路径可还原为完整地址。
    const childParentPath = currentPath || parentPath;
    resolvedRouteItems.push(...buildResolvedRouteItems(item.children, childParentPath));
  });

  return resolvedRouteItems;
}

/**
 * 解析菜单对应的页面组件。
 */
function resolveRouteComponent(component?: string) {
  const viewRegistry = getAdminViewRegistry();
  const viewLoader = viewRegistry.resolve(component);
  if (viewLoader) return viewLoader;

  const pendingViewLoader = viewRegistry.resolve(ADMIN_STATIC_VIEWS.PENDING);
  if (!pendingViewLoader) throw new Error(`未注册管理端静态页面：${ADMIN_STATIC_VIEWS.PENDING}`);
  return pendingViewLoader;
}

/**
 * 将后端路由项转换为前端路由记录。
 */
function createRouteRecord(item: RouteItem, path: string, redirect?: string) {
  return {
    path,
    redirect,
    name: item.name ?? undefined,
    component: item.component ?? undefined,
    meta: item.meta ?? undefined
  } as RouteRecordRaw & {
    component?: string | (() => Promise<unknown>);
  };
}

/**
 * @description 初始化动态路由
 */
export const initDynamicRouter = async () => {
  const userStore = useUserStore();
  const authStore = useAuthStore();

  try {
    // 1.获取菜单列表 && 按钮权限列表
    await authStore.getAuthMenuList();
    await authStore.getAuthButtonList();

    // 2.判断当前用户有没有菜单权限
    if (!authStore.authMenuListGet.length) {
      ElNotification({
        title: t("core.auth.noPermission"),
        message: t("core.auth.noMenuPermission"),
        type: "warning",
        duration: 3000
      });
      userStore.clearAuthData();
      router.replace(LOGIN_URL);
      return Promise.reject("No permission");
    }

    // 3.添加动态路由
    const resolvedRouteItems = buildResolvedRouteItems(authStore.authMenuListGet);
    resolvedRouteItems.forEach(({ item, path, redirect }) => {
      if (item.type === BaseMenuType.EXT_LINK) return;
      const routeRecord = createRouteRecord(item, path, redirect);

      if (typeof routeRecord.component === "string") {
        const resolvedComponent = resolveRouteComponent(routeRecord.component);
        if (!resolvedComponent) return;
        routeRecord.component = resolvedComponent;
      }
      if (!routeRecord.component) return;
      if (!routeRecord.path) return;
      if (routeRecord.name && router.hasRoute(routeRecord.name)) return;
      if (getRouteMetaFull(item.meta)) {
        router.addRoute(routeRecord);
      } else {
        router.addRoute("Layout", routeRecord);
      }
    });
  } catch (error) {
    // 当按钮 || 菜单请求出错时，重定向到登陆页
    userStore.clearAuthData();
    router.replace(LOGIN_URL);
    return Promise.reject(error);
  }
};
