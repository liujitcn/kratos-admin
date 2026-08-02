import type { KratosAppPageConfig } from './module'

/** core 页面编译配置。 */
export const corePages: Record<string, KratosAppPageConfig> = {
  'pages/index/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '',
    },
  },
  'pages/login/login': { style: { navigationBarTitleText: '' } },
  'pages/login/protocal': { style: { navigationBarTitleText: '' } },
  'pages/webview/webview': {
    style: {
      navigationBarTitleText: '',
      'app-plus': { titleNView: false },
    },
  },
  'pages/status/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: '',
    },
  },
}
