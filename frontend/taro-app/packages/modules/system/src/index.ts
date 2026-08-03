// System 语言包由同步脚本生成并注册，供 Taro System 页面、个人中心和 AI 功能使用。
import { defineKratosTaroModule } from '@liujitcn/kratos-taro-app-core'
import { systemPages } from './pages'
import { LOCALE_MESSAGES } from './locales/generated'

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
  messages: LOCALE_MESSAGES,
})
