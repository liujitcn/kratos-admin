import type { LocaleMessages, SupportedLocale } from './locales'

/** bootstrap 可见状态键。 */
export type BootstrapViewKey =
  | 'BOOTSTRAP_LOADING'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'OFFLINE'
  | 'CONFIG_ERROR'
  | 'PAGE_UNAVAILABLE'

/** 页面编译配置。 */
export interface KratosTaroPageConfig {
  route?: string
  style?: Record<string, unknown>
}

/** Taro 模块定义。 */
export interface KratosTaroModule {
  name: string
  pages: Record<string, KratosTaroPageConfig>
  views: Record<string, string>
  icons?: Record<string, string>
  /** 模块按语言区域贡献的扁平语言包。 */
  messages?: Partial<Record<SupportedLocale, LocaleMessages>>
}

/** 声明 Taro 模块。 */
export function defineKratosTaroModule<T extends KratosTaroModule>(module: T): T {
  return module
}

let registeredModules: KratosTaroModule[] = []

/** 原子替换宿主当前注册的模块。 */
export function registerKratosTaroModules(modules: KratosTaroModule[]): void {
  registeredModules = [...modules]
}

/** 获取当前模块列表。 */
export function getRegisteredKratosTaroModules(): KratosTaroModule[] {
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
    if (source) return resolveBundledAsset(source)
  }
}

/** 解析随应用打包的静态资源路径。 */
export function resolveBundledAsset(source: string): string {
  if (/^(?:https?:|data:)/.test(source)) return source
  const base = process.env.KRATOS_TARO_PUBLIC_PATH || '/'
  return `${base.replace(/\/?$/, '/')}${source.replace(/^\/+/, '')}`
}
