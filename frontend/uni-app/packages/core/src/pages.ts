import type { KratosAppPageConfig } from './module'

/** core 页面编译配置。 */
export const corePages: Record<string, KratosAppPageConfig> = {
  'pages/index/index': {
    style: {
      navigationStyle: 'custom',
      navigationBarTitleText: '',
    },
  },
  'pages/login/login': {
    style: { navigationStyle: 'custom', navigationBarTitleText: '' },
  },
  'pages/login/protocal': { style: { navigationStyle: 'custom', navigationBarTitleText: '' } },
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
