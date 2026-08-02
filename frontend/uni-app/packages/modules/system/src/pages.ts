import type { KratosAppPageConfig } from '@liujitcn/kratos-uni-app-core/module'

/** system 页面编译配置。 */
export const systemPages: Record<string, KratosAppPageConfig> = {
  'pages/my/my': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '',
    },
  },
  'pagesMember/settings/settings': { style: { navigationBarTitleText: '' } },
  'pagesMember/ai/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: '',
    },
  },
  'pagesMember/profile/profile': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '',
    },
  },
}
