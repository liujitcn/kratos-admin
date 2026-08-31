export default defineAppConfig({
  "pages": [
    "pages/bootstrap/index",
    "pages/index/index",
    "pages/login/login",
    "pages/login/protocal",
    "pages/status/index",
    "pages/webview/webview",
    "pages/my/my"
  ],
  "window": {
    "backgroundTextStyle": "light",
    "navigationBarBackgroundColor": "#f8f8f8",
    "navigationBarTitleText": "",
    "navigationBarTextStyle": "black",
    "backgroundColor": "#f8f8f8"
  },
  "tabBar": {
    "custom": true,
    "list": [
      {
        "pagePath": "pages/index/index",
        "text": ""
      },
      {
        "pagePath": "pages/my/my",
        "text": ""
      }
    ]
  },
  "subpackages": [
    {
      "root": "pagesMember",
      "pages": [
        "ai/index",
        "message/detail",
        "message/index",
        "profile/profile",
        "settings/settings"
      ]
    }
  ]
})
