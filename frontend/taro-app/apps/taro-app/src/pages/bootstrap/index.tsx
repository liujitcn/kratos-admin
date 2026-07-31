import Taro, { useLoad } from '@tarojs/taro'
import { View } from '@tarojs/components'
import { resolveStaticView } from '@liujitcn/kratos-taro-app-core'

/** 固定启动页，模块注册完成后进入启动状态页。 */
export default function BootstrapPage() {
  useLoad((options) => {
    const route = resolveStaticView('BOOTSTRAP_LOADING') ?? 'pages/status/index'
    const target = options?.route ? decodeURIComponent(options.route) : 'app/home'
    void Taro.reLaunch({
      url: `/${route}?state=BOOTSTRAP_LOADING&bootstrap=1&route=${encodeURIComponent(target)}`,
    })
  })
  return <View />
}
