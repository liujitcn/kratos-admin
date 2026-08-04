import { Image, Text, View } from '@tarojs/components'
import { useLoad } from '@tarojs/taro'
import defaultLogo from '@liujitcn/kratos-taro-app-core/static/images/logo_icon.png'
import { useSettingStore } from '../../../stores'
import './index.scss'
import { useI18n } from '../../../locales'

/** 应用首页。 */
export default function HomePage() {
  const { t } = useI18n()
  const settings = useSettingStore((state) => state.data)
  const loadData = useSettingStore((state) => state.loadData)
  const mainTitle = settings?.get('mainTitle') || t('core.home.main_title')
  const subTitle = settings?.get('subTitle') || t('core.home.sub_title')
  const appLogo = settings?.get('appLogo') || defaultLogo
  useLoad(() => {
    void loadData().catch(() => undefined)
  })

  const stack = [
    [t('core.home.framework'), 'Taro + React 18'],
    [t('core.home.language'), 'TypeScript'],
    [t('core.home.state_management'), 'Zustand'],
    [t('core.home.style_solution'), 'Sass + px'],
  ]
  const demos = [
    [t('core.home.cross_platform'), t('core.home.cross_platform_description')],
    [t('core.home.account_capability'), t('core.home.account_capability_description')],
    [t('core.home.static_home'), t('core.home.static_home_description')],
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
        <Text className='home-section__title'>{t('core.home.tech_stack')}</Text>
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
        <Text className='home-section__title'>{t('core.home.demo_scope')}</Text>
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
