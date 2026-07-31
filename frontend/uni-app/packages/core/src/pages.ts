import type { KratosAppPageConfig } from './module'

/** core 页面编译配置。 */
export const corePages: Record<string, KratosAppPageConfig> = {
  'pages/index/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTextStyle: 'white',
      navigationBarTitleText: '首页',
    },
  },
  'pages/login/login': { style: { navigationBarTitleText: '登录' } },
  'pages/login/protocal': { style: { navigationBarTitleText: '协议详情' } },
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
