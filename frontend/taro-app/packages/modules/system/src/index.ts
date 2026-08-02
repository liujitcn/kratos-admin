// System 语言包随 systemModule 注册，供 Taro System 页面、个人中心和 AI 功能使用。
import { defineKratosTaroModule } from '@liujitcn/kratos-taro-app-core'
import { systemPages } from './pages'
import enUS from './locales/en-US.json'
import jaJP from './locales/ja-JP.json'
import zhCN from './locales/zh-CN.json'
import zhTW from './locales/zh-TW.json'
import koKR from './locales/ko-KR.json'
import frFR from './locales/fr-FR.json'
import esES from './locales/es-ES.json'

/** system 业务模块。 */
export const systemModule = defineKratosTaroModule({
  name: '@liujitcn/kratos-taro-app-system',
  pages: systemPages,
  views: {
    PROFILE_HOME: 'pages/my/my',
    PROFILE: 'pagesMember/profile/profile',
    SETTINGS: 'pagesMember/settings/settings',
    AI: 'pagesMember/ai/index',
  },
  messages: {
    'zh-CN': zhCN,
    'zh-TW': zhTW,
    'ko-KR': koKR,
    'fr-FR': frFR,
    'es-ES': esES,
    'en-US': enUS,
    'ja-JP': jaJP,
  },
})
