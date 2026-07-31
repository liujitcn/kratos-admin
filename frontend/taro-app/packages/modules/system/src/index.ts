import { defineKratosTaroModule } from '@liujitcn/kratos-taro-app-core'
import { systemPages } from './pages'

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
})
