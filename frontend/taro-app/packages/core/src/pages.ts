import type { KratosTaroPageConfig } from './module'

/** core 页面编译配置。 */
export const corePages: Record<string, KratosTaroPageConfig> = {
  'pages/index/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '',
    },
  },
  'pages/login/login': { style: { navigationBarTitleText: '' } },
  'pages/login/protocal': { style: { navigationBarTitleText: '' } },
  'pages/webview/webview': { style: { navigationBarTitleText: '' } },
  'pages/status/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: '',
    },
  },
}
