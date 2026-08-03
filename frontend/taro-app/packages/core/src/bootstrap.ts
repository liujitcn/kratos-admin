import {
  registerKratosTaroModules,
  type KratosTaroModule,
} from './module'
import { initializeAppNavigation } from './navigation'
import {
  registerUserStoreExtension,
  startUserStoreEventBridge,
  useUserStore,
 useSettingStore } from './stores'
import { applyLanguageConfig, initializeLocale, registerLocaleChangeHandler, registerLocaleMessages } from './locales'
import { defLanguageService } from './api/base/language'

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
  void defLanguageService
    .OptionLanguage({})
    .then((response) => {
      applyLanguageConfig(response)
    })
    .catch(() => {
      // 语言公共接口失败时继续使用静态语言包和系统语言。
    })
  if (bootstrapped) return
  bootstrapped = true
  registerLocaleChangeHandler(initializeAppNavigation)
  registerLocaleChangeHandler(() => useSettingStore.getState().loadData())
  startUserStoreEventBridge()
  registerUserStoreExtension({
    onLogin: initializeAppNavigation,
    onLogout: initializeAppNavigation,
    onSilentLogout: initializeAppNavigation,
  })
  useUserStore.getState().hydrate()
}
