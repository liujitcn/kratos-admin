import {
  bootstrapKratosApp,
  initializeAppNavigation,
  pinia,
  registerKratosAppModules,
  registerUserStoreExtension,
} from '@liujitcn/kratos-app-core'
import { createSSRApp } from 'vue'
import App from './App.vue'
import { moduleManifest } from './module-manifest'

registerKratosAppModules(moduleManifest)
registerUserStoreExtension({
  onLogin: initializeAppNavigation,
  onLogout: initializeAppNavigation,
  onSilentLogout: initializeAppNavigation,
})

/** 创建宿主 app 实例。 */
export function createApp() {
  return bootstrapKratosApp({
    app: App,
    createSSRApp,
    pinia,
    modules: moduleManifest,
  })
}
