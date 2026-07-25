import App from './App.vue'
import pinia from './stores'
import { bootstrapKratosApp } from './bootstrap'

/**
 * 创建通用 app 实例。
 */
export function createApp() {
  return bootstrapKratosApp({
    app: App,
    pinia,
  }).app
}

// #ifdef H5
const h5App = createApp()
h5App.mount('#app')
// #endif
