import {
  registerKratosTaroModules,
  type KratosTaroModule,
} from './module'
import { initializeAppNavigation } from './navigation'
import {
  registerUserStoreExtension,
  startUserStoreEventBridge,
  useSettingStore,
  useUserStore,
} from './stores'
import {
  applyLanguageConfig,
  initializeLocale,
  registerLocaleChangeHandler,
  registerLocaleMessages,
  t,
} from './locales'
import { defLanguageService } from './api/base/v1/language'

/** Taro 应用启动参数。 */
export interface KratosTaroBootstrapOptions {
  /** 按覆盖优先级排列的模块。 */
  modules: KratosTaroModule[]
}

let bootstrapped = false
let titleObserver: MutationObserver | undefined

/** 使用宿主声明的应用名称同步 H5 浏览器页签标题。 */
function syncH5DocumentTitle(): void {
  if (process.env.TARO_ENV !== 'h5' || typeof document === 'undefined') return
  const titleElement = document.querySelector('title')
  const appTitle = titleElement?.dataset.appTitle || t('core.home.main_title')
  const applyTitle = () => {
    if (document.title !== appTitle) document.title = appTitle
  }
  applyTitle()
  if (!titleElement || titleObserver) return
  titleObserver = new MutationObserver(applyTitle)
  titleObserver.observe(titleElement, { childList: true })
}

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
  syncH5DocumentTitle()
  registerLocaleChangeHandler(initializeAppNavigation)
  registerLocaleChangeHandler(() => useSettingStore.getState().loadData())
  registerLocaleChangeHandler(syncH5DocumentTitle)
  startUserStoreEventBridge()
  registerUserStoreExtension({
    onLogin: initializeAppNavigation,
    onLogout: initializeAppNavigation,
    onSilentLogout: initializeAppNavigation,
  })
  useUserStore.getState().hydrate()
}
