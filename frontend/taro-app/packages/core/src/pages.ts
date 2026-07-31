import type { KratosTaroPageConfig } from './module'

/** core 页面编译配置。 */
export const corePages: Record<string, KratosTaroPageConfig> = {
  'pages/index/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '首页',
    },
  },
  'pages/login/login': { style: { navigationBarTitleText: '登录' } },
  'pages/login/protocal': { style: { navigationBarTitleText: '协议详情' } },
  'pages/webview/webview': { style: { navigationBarTitleText: '' } },
  'pages/status/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: '',
    },
  },
}
