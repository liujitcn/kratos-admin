import { createSSRApp } from 'vue'
import App from './App.vue'
import pinia from './stores'

/**
 * 创建通用 app 实例。
 */
export function createApp() {
  const app = createSSRApp(App)
  app.use(pinia)
  return {
    app,
  }
}
