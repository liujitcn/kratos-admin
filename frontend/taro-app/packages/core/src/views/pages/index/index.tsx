import { Image, Text, View } from '@tarojs/components'
import { useLoad } from '@tarojs/taro'
import { resolveBundledAsset } from '../../../module'
import { useSettingStore } from '../../../stores'
import './index.scss'

const defaultLogo = resolveBundledAsset('static/images/logo_icon.png')

/** 应用首页。 */
export default function HomePage() {
  const settings = useSettingStore((state) => state.data)
  const loadData = useSettingStore((state) => state.loadData)
  const mainTitle = settings?.get('mainTitle') || '应用框架示例'
  const subTitle = settings?.get('subTitle') || '保留通用导航与个人中心体验'
  const appLogo = settings?.get('appLogo') || defaultLogo
  useLoad(() => {
    void loadData().catch(() => undefined)
  })

  const stack = [
    ['应用框架', 'Taro + React 18'],
    ['开发语言', 'TypeScript'],
    ['状态管理', 'Zustand'],
    ['样式方案', 'Sass + px'],
  ]
  const demos = [
    ['跨端页面', '同一套页面代码适配 H5 和微信小程序'],
    ['账户能力', '个人资料、应用设置与智能助手入口已保留'],
    ['静态首页', '当前首页用于展示框架信息，不依赖业务接口'],
  ]

  return (
    <View className={`home-page${process.env.TARO_ENV === 'weapp' ? ' home-page--weapp' : ''}`}>
      <View className='home-hero'>
        <Image
          className='home-logo'
          src={appLogo}
          mode='aspectFit'
        />
        <View className='home-hero__copy'>
          <Text className='home-title'>{mainTitle}</Text>
          <Text className='home-subtitle'>{subTitle}</Text>
        </View>
      </View>

      <View className='home-section'>
        <Text className='home-section__title'>当前技术栈</Text>
        <View className='home-info-list'>
          {stack.map(([label, value]) => (
            <View className='home-info-row' key={label}>
              <Text className='home-info-label'>{label}</Text>
              <Text className='home-info-value'>{value}</Text>
            </View>
          ))}
        </View>
      </View>

      <View className='home-section'>
        <Text className='home-section__title'>演示范围</Text>
        <View className='home-demo-list'>
          {demos.map(([title, description], index) => (
            <View className='home-demo-item' key={title}>
              <View className='home-demo-dot'>{index + 1}</View>
              <View className='home-demo-copy'>
                <Text className='home-demo-title'>{title}</Text>
                <Text className='home-demo-desc'>{description}</Text>
              </View>
            </View>
          ))}
        </View>
      </View>
    </View>
  )
}
