import type { Component } from "vue";
import type { RouteLocationNormalizedLoaded } from "vue-router";
import type { LocaleMessages, SupportedLocale } from "@/locales";

/**
 * 管理端视图加载器。
 */
export type AdminViewLoader = () => Promise<{
  default: Component;
}>;

/**
 * 管理端视图模块表。
 */
export type AdminViewModules = Record<string, AdminViewLoader>;

/** ADMIN_STATIC_VIEWS 管理端可替换静态页面的固定视图键。 */
export const ADMIN_STATIC_VIEWS = {
  LOGIN: "login/index",
  FORBIDDEN: "error/403",
  NOT_FOUND: "error/404",
  SERVER_ERROR: "error/500",
  PENDING: "error/pending"
} as const;

/** 管理端静态页面固定视图键。 */
export type AdminStaticViewKey = (typeof ADMIN_STATIC_VIEWS)[keyof typeof ADMIN_STATIC_VIEWS];

/** 管理端静态页面实现表。 */
export type AdminStaticViewModules = Partial<Record<AdminStaticViewKey, AdminViewLoader>>;

/**
 * 管理端顶部工具。
 */
export interface AdminHeaderTool {
  /** 工具唯一名称，同时作为渲染节点 ID。 */
  name: string;
  /** 工具对应的 Vue 组件。 */
  component: Component;
}

/** 管理端用户菜单操作。 */
export interface AdminUserMenuAction {
  /** 操作唯一名称。 */
  name: string;
	/** 操作显示名称对应的稳定语言键。 */
	labelKey: string;
  /** 对应后端动态菜单名称，菜单不存在时不显示操作。 */
  menuName: string;
  /** 操作图标。 */
  icon?: Component;
}

/** 管理端动态路由行为配置。 */
export interface AdminRouteOptions {
  /** 查询参数变化时是否复用同一个页签。 */
  reuseTabAcrossQuery?: boolean;
}

/**
 * 管理端模块定义。
 */
export interface AdminModule {
  /** 模块名称。 */
  name: string;
  /** 模块提供的业务页面加载器，页面统一通过模块前缀解析。 */
  views?: AdminViewModules;
  /** 模块提供的静态页面实现，后注册模块可替换已有实现。 */
  staticViews?: AdminStaticViewModules;
  /** 模块提供的顶部工具。 */
  headerTools?: AdminHeaderTool[];
  /** 模块提供的用户菜单操作。 */
  userMenuActions?: AdminUserMenuAction[];
  /** 模块提供的动态路由行为配置，以后端菜单名称为键。 */
  routeOptions?: Record<string, AdminRouteOptions>;
  /** 模块提供的命名扩展，由扩展所属业务包定义具体类型。 */
  extensions?: Record<string, unknown>;
  /** 模块按语言区域贡献的扁平语言包。 */
  messages?: Partial<Record<SupportedLocale, LocaleMessages>>;
}

const registeredModules = new Map<string, AdminModule>();

/**
 * 声明一个可被管理端宿主注册的模块。
 */
export function defineAdminModule(module: AdminModule): AdminModule {
  return module;
}

/**
 * 注册一个管理端模块，名称或模块级扩展点重复时抛出错误。
 */
export function registerAdminModule(module: AdminModule): void {
  registerAdminModules([module]);
}

/**
 * 批量注册管理端模块，全部通过唯一性校验后再写入注册表。
 */
export function registerAdminModules(modules: AdminModule[]): void {
  const nextModules = [...registeredModules.values(), ...modules];
  assertUniqueAdminModuleKeys(nextModules, "模块", module => [module.name]);
  assertUniqueAdminModuleKeys(nextModules, "顶部工具", module => module.headerTools?.map(tool => tool.name) ?? []);
  assertUniqueAdminModuleKeys(nextModules, "用户菜单", module => module.userMenuActions?.map(action => action.name) ?? []);
  assertUniqueAdminModuleKeys(nextModules, "路由配置", module => Object.keys(module.routeOptions ?? {}));
  modules.forEach(module => registeredModules.set(module.name, module));
}

/**
 * 获取当前已注册的管理端模块。
 */
export function getRegisteredAdminModules(): AdminModule[] {
  return [...registeredModules.values()];
}

/**
 * 管理端视图注册表。
 */
export interface AdminViewRegistry {
  /** 根据固定视图键或后端菜单组件路径查找页面加载器。 */
  resolve(component?: string): AdminViewLoader | undefined;
}

/**
 * 创建管理端视图注册表，业务页面按模块隔离，静态页面允许后注册模块替换。
 */
export function createAdminViewRegistry(modules: AdminModule[]): AdminViewRegistry {
  const viewMap = new Map<string, AdminViewLoader>();

  modules.forEach(module => {
    Object.entries(module.views ?? {}).forEach(([path, loader]) => {
      const normalizedPath = normalizeAdminViewPath(path);
      if (!normalizedPath) return;
      viewMap.set(`${module.name}/${normalizedPath}`, loader);
    });
    Object.entries(module.staticViews ?? {}).forEach(([path, loader]) => {
      const normalizedPath = normalizeAdminViewPath(path);
      if (!normalizedPath || !loader) return;
      viewMap.set(normalizedPath, loader);
    });
  });

  return {
    resolve(component) {
      const normalizedPath = normalizeAdminViewPath(component);
      if (!normalizedPath || normalizedPath === "Layout") return undefined;

      const candidates = [
        normalizedPath,
        `${normalizedPath}/index`,
        normalizedPath.replace(/\/index$/, ""),
        `${normalizedPath.replace(/\/index$/, "")}/index`
      ];

      return candidates.map(path => viewMap.get(path)).find(Boolean);
    }
  };
}

/**
 * 根据当前已注册模块创建视图注册表。
 */
export function getAdminViewRegistry(): AdminViewRegistry {
  return createAdminViewRegistry(getRegisteredAdminModules());
}

/**
 * 获取当前已注册模块提供的顶部工具。
 */
export function getAdminHeaderTools(): AdminHeaderTool[] {
  return getRegisteredAdminModules().flatMap(module => module.headerTools ?? []);
}

/**
 * 获取当前已注册模块提供的用户菜单操作。
 */
export function getAdminUserMenuActions(): AdminUserMenuAction[] {
  return getRegisteredAdminModules().flatMap(module => module.userMenuActions ?? []);
}

/**
 * 获取指定动态路由的行为配置。
 */
export function getAdminRouteOptions(name?: string): AdminRouteOptions | undefined {
  if (!name) return undefined;
  return getRegisteredAdminModules().reduce<AdminRouteOptions | undefined>((options, module) => {
    return module.routeOptions?.[name] ?? options;
  }, undefined);
}

/** getAdminTabPath 根据路由扩展配置返回当前页签唯一标识。 */
export function getAdminTabPath(route: Pick<RouteLocationNormalizedLoaded, "name" | "path" | "fullPath">): string {
  const routeName = typeof route.name === "string" ? route.name : undefined;
  return getAdminRouteOptions(routeName)?.reuseTabAcrossQuery ? route.path : route.fullPath;
}

/**
 * 获取最后注册模块提供的指定命名扩展。
 */
export function getAdminModuleExtension<T>(name: string): T | undefined {
  return getRegisteredAdminModules().reduce<T | undefined>((extension, module) => {
    return (module.extensions?.[name] as T | undefined) ?? extension;
  }, undefined);
}

/**
 * 统一模块路径和后端菜单组件路径的表示形式。
 */
function normalizeAdminViewPath(path?: string): string {
  if (!path) return "";

  let normalizedPath = path.trim().replace(/\\/g, "/");
  normalizedPath = normalizedPath.replace(/^.*\/src\/views\//, "");
  normalizedPath = normalizedPath.replace(/^.*\/views\//, "");
  normalizedPath = normalizedPath.replace(/^\.\/views\//, "");
  normalizedPath = normalizedPath.replace(/^\/+/, "");
  normalizedPath = normalizedPath.replace(/\.vue$/, "");

  return normalizedPath;
}

/** 校验模块提供的命名入口在整个宿主内唯一。 */
function assertUniqueAdminModuleKeys(modules: AdminModule[], label: string, getKeys: (module: AdminModule) => string[]): void {
  const owners = new Map<string, string>();

  modules.forEach(module => {
    getKeys(module).forEach(key => {
      const owner = owners.get(key);
      if (owner) {
        throw new Error(`管理端${label}名称重复: ${key}（模块 ${owner} 与 ${module.name}）`);
      }
      owners.set(key, module.name);
    });
  });
}
