import { defineStore } from "pinia";
import { defAuthService } from "@/api/system/auth";
import { AuthState } from "@/stores/interface";
import { getFlatMenuList, getShowMenuList, getAllBreadcrumbList, isExternalPath } from "@/utils";
import type { RouteItem } from "@/rpc/system/admin/v1/auth";

const GLOBAL_AUTH_BUTTON_KEY = "__global__";

/** 规范化路由路径，统一转换为绝对路径。 */
function normalizeRoutePath(path?: string, parentPath = "") {
  if (!path) return "";
  if (path === "/") return "/";
  if (isExternalPath(path)) return path;
  if (path.startsWith("/")) return path;

  const normalizedParentPath = parentPath && parentPath !== "/" ? parentPath.replace(/\/+$/, "") : "";
  const normalizedCurrentPath = path.replace(/^\/+/, "");
  const pathSegments = [normalizedParentPath.replace(/^\/+/, ""), normalizedCurrentPath].filter(Boolean);
  return `/${pathSegments.join("/")}`;
}

/** 递归规范化菜单树，避免菜单点击时发生相对路径拼接。 */
function normalizeRouteTree(menuList: RouteItem[], parentPath = ""): RouteItem[] {
  return menuList.map(item => {
    const currentPath = normalizeRoutePath(item.path, parentPath);
    return {
      ...item,
      path: currentPath,
      children: normalizeRouteTree(item.children ?? [], currentPath)
    };
  });
}

export const useAuthStore = defineStore("admin-auth", {
  state: (): AuthState => ({
    // 按钮权限列表
    authButtonList: {},
    // 菜单权限列表
    authMenuList: [],
    // 当前页面的 router name，用来做按钮权限筛选
    routeName: ""
  }),
  getters: {
    // 按钮权限列表
    authButtonListGet: state => state.authButtonList,
    // 菜单权限列表 ==> 这里的菜单没有经过任何处理
    authMenuListGet: state => state.authMenuList,
    // 菜单权限列表 ==> 左侧菜单栏渲染，需要剔除 hide == true
    showMenuListGet: state => getShowMenuList(state.authMenuList),
    // 菜单权限列表 ==> 扁平化之后的一维数组菜单，主要用来添加动态路由
    flatMenuListGet: state => getFlatMenuList(state.authMenuList),
    // 递归处理后的所有面包屑导航列表
    breadcrumbListGet: state => getAllBreadcrumbList(state.authMenuList)
  },
  actions: {
    /** 获取按钮权限列表 */
    async getAuthButtonList() {
      const data = await defAuthService.ListUserButton({});
      this.authButtonList = {
        [GLOBAL_AUTH_BUTTON_KEY]: data.value ?? []
      };
    },
    /** 获取菜单权限列表 */
    async getAuthMenuList() {
      const data = await defAuthService.TreeUserMenu({});
      this.authMenuList = normalizeRouteTree(data.routes ?? []);
    },
    /** 设置当前路由名称 */
    async setRouteName(name: string) {
      this.routeName = name;
    }
  }
});
