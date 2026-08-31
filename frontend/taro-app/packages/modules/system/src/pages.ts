import type { KratosTaroPageConfig } from '@liujitcn/kratos-taro-app-core'

/** system 页面编译配置。 */
export const systemPages: Record<string, KratosTaroPageConfig> = {
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
  'pagesMember/message/index': { style: { navigationBarTitleText: '' } },
  'pagesMember/message/detail': { style: { navigationBarTitleText: '' } },
}
