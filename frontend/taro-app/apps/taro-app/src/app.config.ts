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
  "subpackages": [
    {
      "root": "pagesMember",
      "pages": [
        "ai/index",
        "profile/profile",
        "settings/settings"
      ]
    }
  ]
})
