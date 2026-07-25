import { createSSRApp, type Component } from 'vue'
import type { Pinia } from 'pinia'
import { kratosAppModule } from './modules/kratosApp'
import { registerKratosAppModules, type KratosAppModule } from './modules'

/**
 * app 宿主启动参数。
 */
export interface KratosAppBootstrapOptions {
  /** 宿主根组件。 */
  app: Component
  /** 宿主 Pinia 实例。 */
  pinia: Pinia
  /** 业务模块列表。 */
  modules?: KratosAppModule[]
}

/**
 * 创建 uni-app 应用实例并注册公共与业务模块。
 */
export function bootstrapKratosApp(options: KratosAppBootstrapOptions) {
  registerKratosAppModules([kratosAppModule, ...(options.modules ?? [])])

  const app = createSSRApp(options.app)
  app.use(options.pinia)

  return {
    app,
  }
}
