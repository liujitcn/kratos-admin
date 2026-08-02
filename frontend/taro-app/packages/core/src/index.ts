// Core 语言包在本文件的 coreModule 中注册，供 Taro 公共页面、请求工具和业务模块复用。
import { corePages } from './pages'
import { defineKratosTaroModule } from './module'
import coreEnUS from './locales/en-US.json'
import coreJaJP from './locales/ja-JP.json'
import coreZhCN from './locales/zh-CN.json'
import coreZhTW from './locales/zh-TW.json'
import coreKoKR from './locales/ko-KR.json'
import coreFrFR from './locales/fr-FR.json'
import coreEsES from './locales/es-ES.json'

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
  messages: {
    'zh-CN': coreZhCN,
    'zh-TW': coreZhTW,
    'ko-KR': coreKoKR,
    'fr-FR': coreFrFR,
    'es-ES': coreEsES,
    'en-US': coreEnUS,
    'ja-JP': coreJaJP,
  },
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
