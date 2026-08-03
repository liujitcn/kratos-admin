import { defineKratosAppModule } from '@liujitcn/kratos-uni-app-core/module'
import { systemPages } from './pages'

// System 语言包由同步脚本生成并注册，供 uni-app System 页面、个人中心和 AI 功能使用。
import { LOCALE_MESSAGES } from './locales/generated'

/** system 内置模块，AI 能力也归属 system。 */
export const systemModule = defineKratosAppModule({
  name: '@liujitcn/kratos-uni-app-system',
  pages: systemPages,
  views: {
    PROFILE_HOME: 'pages/my/my',
    PROFILE: 'pagesMember/profile/profile',
    SETTINGS: 'pagesMember/settings/settings',
    AI: 'pagesMember/ai/index',
  },
  messages: LOCALE_MESSAGES,
})

export default systemModule
