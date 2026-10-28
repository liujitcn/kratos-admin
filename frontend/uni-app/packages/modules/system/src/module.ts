import { defineKratosAppModule } from '@liujitcn/kratos-uni-app-core/module'
import { systemPages } from './pages'
import { LOCALE_MESSAGES } from './locales/generated'

/** system 内置模块，声明页面、视图和语言资源。 */
export const systemModule = defineKratosAppModule({
  name: '@liujitcn/kratos-uni-app-system',
  pages: systemPages,
  views: {
    PROFILE_HOME: 'pages/my/my',
    PROFILE: 'pagesMember/profile/profile',
    PASSWORD: 'pagesMember/password/password',
    SETTINGS: 'pagesMember/settings/settings',
    AI: 'pagesMember/ai/index',
    MESSAGE_INBOX: 'pagesMember/message/index',
    MESSAGE_DETAIL: 'pagesMember/message/detail',
  },
  messages: LOCALE_MESSAGES,
})

export default systemModule
