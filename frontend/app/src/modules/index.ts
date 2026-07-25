/**
 * app 宿主模块定义。
 */
export interface KratosAppModule {
  /** 模块名称。 */
  name: string
}

const registeredModules = new Map<string, KratosAppModule>()

/**
 * 声明一个可被 app 宿主注册的业务模块。
 */
export function defineKratosAppModule<T extends KratosAppModule>(module: T): T {
  return module
}

/**
 * 注册一个 app 业务模块，重复名称以最新注册的模块为准。
 */
export function registerKratosAppModule(module: KratosAppModule): void {
  registeredModules.set(module.name, module)
}

/**
 * 批量注册 app 业务模块。
 */
export function registerKratosAppModules(modules: KratosAppModule[]): void {
  modules.forEach(registerKratosAppModule)
}

/**
 * 获取当前已注册的 app 业务模块。
 */
export function getRegisteredKratosAppModules(): KratosAppModule[] {
  return [...registeredModules.values()]
}
