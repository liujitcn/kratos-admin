// Core 语言包由同步脚本生成并在本文件的 coreModule 中注册，供 Taro 公共页面、请求工具和业务模块复用。
import { corePages } from './pages'
import { defineKratosTaroModule } from './module'
import { LOCALE_MESSAGES } from './locales/generated'

/** core 内置模块。 */
export const coreModule = defineKratosTaroModule({
  name: '@liujitcn/kratos-taro-app-core',
  pages: corePages,
  views: {
    HOME: 'pages/index/index',
    LOGIN: 'pages/login/login',
    PROTOCOL: 'pages/login/protocal',
    WEBVIEW: 'pages/webview/webview',
    BOOTSTRAP_LOADING: 'pages/status/index',
    FORBIDDEN: 'pages/status/index',
    NOT_FOUND: 'pages/status/index',
    OFFLINE: 'pages/status/index',
    CONFIG_ERROR: 'pages/status/index',
    PAGE_UNAVAILABLE: 'pages/status/index',
  },
  icons: {
    HOME_DEFAULT: 'static/tabs/home_default.png',
    HOME_SELECTED: 'static/tabs/home_selected.png',
    USER_DEFAULT: 'static/tabs/user_default.png',
    USER_SELECTED: 'static/tabs/user_selected.png',
  },
  messages: LOCALE_MESSAGES,
})

export * from './module'
export * from './bootstrap'
export * from './navigation'
export * from './locales'
export * from './stores'
export * from './utils/auth'
export * from './utils/file'
export * from './utils/index'
export * from './utils/navigation'
export * from './utils/passwordCrypto'
