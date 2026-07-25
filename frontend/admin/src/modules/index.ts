import type { Component } from "vue";

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

/**
 * AI 助手可扩展能力。
 */
export interface AdminAiExtension {
  /** 结构化流程卡片组件。 */
  flowBlocks?: Component;
}

/**
 * 管理端模块定义。
 */
export interface AdminModule {
  /** 模块名称。 */
  name: string;
  /** 模块提供的页面加载器。 */
  views?: AdminViewModules;
  /** 模块提供的 AI 助手扩展。 */
  ai?: AdminAiExtension;
}

const registeredModules = new Map<string, AdminModule>();

/**
 * 声明一个可被管理端宿主注册的模块。
 */
export function defineAdminModule(module: AdminModule): AdminModule {
  return module;
}

/**
 * 注册一个管理端模块，重复名称会以最新注册的模块为准。
 */
export function registerAdminModule(module: AdminModule): void {
  registeredModules.set(module.name, module);
}

/**
 * 批量注册管理端模块。
 */
export function registerAdminModules(modules: AdminModule[]): void {
  modules.forEach(registerAdminModule);
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
  /** 根据后端菜单组件路径查找页面加载器。 */
  resolve(component?: string): AdminViewLoader | undefined;
}

/**
 * 创建管理端视图注册表，后注册的模块可以覆盖同名页面。
 */
export function createAdminViewRegistry(modules: AdminModule[]): AdminViewRegistry {
  const viewMap = new Map<string, AdminViewLoader>();

  modules.forEach(module => {
    Object.entries(module.views ?? {}).forEach(([path, loader]) => {
      viewMap.set(normalizeAdminViewPath(path), loader);
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
 * 获取当前已注册的 AI 助手扩展。
 */
export function getAdminAiExtension(): AdminAiExtension {
  return getRegisteredAdminModules().reduce<AdminAiExtension>((extension, module) => {
    return Object.assign(extension, module.ai);
  }, {});
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
