import { corePages } from './pages'

/** bootstrap 可见状态键。 */
export type BootstrapViewKey =
  | 'BOOTSTRAP_LOADING'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'OFFLINE'
  | 'CONFIG_ERROR'
  | 'PAGE_UNAVAILABLE'

/** 页面编译配置。 */
export interface KratosAppPageConfig {
  route?: string
  style?: Record<string, unknown>
}

/** app 模块定义。 */
export interface KratosAppModule {
  name: string
  pages: Record<string, KratosAppPageConfig>
  views: Record<string, string>
  icons?: Record<string, string>
}

/** 声明 app 模块。 */
export function defineKratosAppModule<T extends KratosAppModule>(module: T): T {
  return module
}

let registeredModules: KratosAppModule[] = []

/** 原子替换宿主当前注册的模块。 */
export function registerKratosAppModules(modules: KratosAppModule[]): void {
  registeredModules = [...modules]
}

/** 获取当前模块列表。 */
export function getRegisteredKratosAppModules(): KratosAppModule[] {
  return [...registeredModules]
}

/** 按后注册优先级解析静态视图。 */
export function resolveStaticView(viewKey: string): string | undefined {
  for (let index = registeredModules.length - 1; index >= 0; index -= 1) {
    const route = registeredModules[index].views[viewKey]
    if (route) return route
  }
}

/** 按后注册优先级解析模块图标。 */
export function resolveModuleIcon(icon: string | undefined): string | undefined {
  if (!icon || /^https:\/\//.test(icon)) return icon
  for (let index = registeredModules.length - 1; index >= 0; index -= 1) {
    const source = registeredModules[index].icons?.[icon]
    if (source) {
      const base = import.meta.env.BASE_URL || '/'
      return `${base.replace(/\/?$/, '/')}${source.replace(/^\/+/, '')}`
    }
  }
}

/** core 内置模块。 */
export const coreModule = defineKratosAppModule({
  name: '@liujitcn/kratos-app-core',
  pages: corePages,
  views: {
    HOME: 'pages/index/index',
    LOGIN: 'pages/login/login',
    PROTOCOL: 'pages/login/protocal',
    WEBVIEW: 'pages/webview/webview',
    BOOTSTRAP_LOADING: 'pages/status/index',
    FORBIDDEN: 'pages/status/index',
    NOT_FOUND: 'pages/status/index',
    OFFLINE: 'pages/status/index',
    CONFIG_ERROR: 'pages/status/index',
    PAGE_UNAVAILABLE: 'pages/status/index',
  },
  icons: {
    HOME_DEFAULT: 'static/tabs/home_default.png',
    HOME_SELECTED: 'static/tabs/home_selected.png',
    USER_DEFAULT: 'static/tabs/user_default.png',
    USER_SELECTED: 'static/tabs/user_selected.png',
  },
})
