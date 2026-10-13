import { Image, Text, View } from '@tarojs/components'
import { useLoad } from '@tarojs/taro'
import { useEffect, useState } from 'react'
import defaultLogo from '@liujitcn/kratos-taro-app-core/static/images/logo_icon.png'
import { useSettingStore, useThemeStore } from '../../../stores'
import { formatSrc } from '../../../utils'
import './index.scss'
import KratosNavBar from '../../../components/KratosNavBar'
import { useI18n } from '../../../locales'

/** HomePage 展示应用首页，并复用固定高度的 Tab 标题栏。 */
export default function HomePage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const settings = useSettingStore((state) => state.data)
  const loadData = useSettingStore((state) => state.loadData)
  const mainTitle = t('core.navigation.home')
  const subTitle = settings?.get('subTitle') || t('core.home.sub_title')
  const configuredAppLogo = settings?.get('appLogo')
  const configuredLogoUrl = configuredAppLogo ? formatSrc(configuredAppLogo) : defaultLogo
  const [appLogo, setAppLogo] = useState(configuredLogoUrl)

  useEffect(() => {
    setAppLogo(configuredLogoUrl)
  }, [configuredLogoUrl])

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
    [t('core.home.message_center'), t('core.home.message_center_description')],
    [t('core.home.dynamic_config'), t('core.home.dynamic_config_description')],
  ]
  return (
    <View
      className={`home-page ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <View className='k-grid-bg' aria-hidden />
      <View className='home-content'>
        <KratosNavBar title={mainTitle} variant='tab' />

        <View className='k-glass k-glass--lit home-hero'>
          <View className='home-hero__main'>
            <Image
              className='home-logo'
              src={appLogo}
              mode='aspectFit'
              onError={() => setAppLogo(defaultLogo)}
            />
            <View className='home-hero__copy'>
              <Text className='home-subtitle'>{subTitle}</Text>
            </View>
          </View>
        </View>

        <View className='home-section'>
          <Text className='home-section__title'>{t('core.home.tech_stack')}</Text>
          <View className='k-glass home-info-list'>
            {stack.map(([label, value]) => (
              <View className='k-field' key={label}>
                <Text className='k-field__k'>{label}</Text>
                <Text className='k-field__v'>{value}</Text>
              </View>
            ))}
          </View>
        </View>

        <View className='home-section'>
          <Text className='home-section__title'>{t('core.home.demo_scope')}</Text>
          <View className='k-glass home-demo-list'>
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
    </View>
  )
}
