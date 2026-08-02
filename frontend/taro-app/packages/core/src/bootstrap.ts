import {
  registerKratosTaroModules,
  type KratosTaroModule,
} from './module'
import { initializeAppNavigation } from './navigation'
import {
  registerUserStoreExtension,
  startUserStoreEventBridge,
  useUserStore,
} from './stores'
import { initializeLocale, registerLocaleChangeHandler, registerLocaleMessages } from './locales'

/** Taro 应用启动参数。 */
export interface KratosTaroBootstrapOptions {
  /** 按覆盖优先级排列的模块。 */
  modules: KratosTaroModule[]
}

let bootstrapped = false

/** 注册模块、认证生命周期和导航运行时。 */
export function bootstrapKratosTaroApp(options: KratosTaroBootstrapOptions): void {
  registerKratosTaroModules(options.modules)
  registerLocaleMessages(options.modules)
  initializeLocale()
  if (bootstrapped) return
  bootstrapped = true
  registerLocaleChangeHandler(initializeAppNavigation)
  startUserStoreEventBridge()
  registerUserStoreExtension({
    onLogin: initializeAppNavigation,
    onLogout: initializeAppNavigation,
    onSilentLogout: initializeAppNavigation,
  })
  useUserStore.getState().hydrate()
}
