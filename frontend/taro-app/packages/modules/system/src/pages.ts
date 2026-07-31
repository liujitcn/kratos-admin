import type { KratosTaroPageConfig } from '@liujitcn/kratos-taro-app-core'

/** system 页面编译配置。 */
export const systemPages: Record<string, KratosTaroPageConfig> = {
  'pages/my/my': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '我的',
    },
  },
  'pagesMember/settings/settings': { style: { navigationBarTitleText: '设置' } },
  'pagesMember/ai/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: 'AI 助手',
    },
  },
  'pagesMember/profile/profile': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '个人信息',
    },
  },
}
