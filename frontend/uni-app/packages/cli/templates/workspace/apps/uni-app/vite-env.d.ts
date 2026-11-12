/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_APP_PORT: string
  readonly VITE_APP_BASE_API: string
  readonly VITE_APP_API_URL: string
  readonly VITE_APP_STATIC_API: string
  readonly VITE_APP_STATIC_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// 上游发布包直接导出 uni-app x uvue 源文件，为生成的页面代理提供组件类型。
declare module '@liujitcn/*.uvue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent
  export default component
}
