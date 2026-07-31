/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** H5 开发服务端口 */
  readonly VITE_APP_PORT: string
  /** 接口基础路径 */
  readonly VITE_APP_BASE_API: string
  /** 页面实际请求使用的接口地址 */
  readonly VITE_APP_API_URL: string
  /** 静态资源基础路径 */
  readonly VITE_APP_STATIC_API: string
  /** 页面实际请求使用的静态资源地址 */
  readonly VITE_APP_STATIC_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
