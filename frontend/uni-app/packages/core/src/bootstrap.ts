import type { Pinia } from 'pinia'
import type { App, Component } from 'vue'
import { registerKratosAppModules, type KratosAppModule } from './module'
import {
  applyLanguageConfig,
  initializeLocale,
  registerLocaleChangeHandler,
  registerLocaleMessages,
} from './locales'
import { initializeAppNavigation } from './navigation'
import { useSettingStore } from './stores'
import { defLanguageService } from './api/base/v1/language'

/** uni-app 启动参数。 */
export interface KratosAppBootstrapOptions {
  /** 宿主根组件。 */
  app: Component
  /** uni-app 宿主入口导入的 Vue SSR app 工厂。 */
  createSSRApp: (rootComponent: Component) => App
  /** 宿主 Pinia。 */
  pinia: Pinia
  /** 按覆盖优先级排列的模块。 */
  modules: KratosAppModule[]
}

/** 创建 uni-app 实例并注册模块。 */
export function bootstrapKratosApp(options: KratosAppBootstrapOptions) {
  registerKratosAppModules(options.modules)
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
  registerLocaleChangeHandler(initializeAppNavigation)
  registerLocaleChangeHandler(() => useSettingStore(options.pinia).loadData())
  const app = options.createSSRApp(options.app)
  app.use(options.pinia)
  return { app }
}
