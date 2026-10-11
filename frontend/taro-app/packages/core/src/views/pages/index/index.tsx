import { Image, Text, View } from '@tarojs/components'
import { useLoad } from '@tarojs/taro'
import { useEffect, useState } from 'react'
import defaultLogo from '@liujitcn/kratos-taro-app-core/static/images/logo_icon.png'
import { useSettingStore, useThemeStore } from '../../../stores'
import { formatSrc } from '../../../utils'
import './index.scss'
import { useI18n } from '../../../locales'

/** 应用首页。 */
export default function HomePage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const settings = useSettingStore((state) => state.data)
  const loadData = useSettingStore((state) => state.loadData)
  const mainTitle = settings?.get('mainTitle') || t('core.home.main_title')
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
    [t('core.home.static_home'), t('core.home.static_home_description')],
  ]
  return (
    <View
      className={`home-page${process.env.TARO_ENV === 'weapp' ? ' home-page--weapp' : ''} ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <View className='k-grid-bg' aria-hidden />
      <View className='home-content'>
        <View className='home-top'>
          <View className='home-top__title'>{mainTitle}</View>
        </View>

        <View className='k-glass k-glass--lit home-hero'>
          <View className='k-cnr k-cnr--tl' aria-hidden />
          <View className='k-cnr k-cnr--tr' aria-hidden />
          <View className='k-cnr k-cnr--bl' aria-hidden />
          <View className='k-cnr k-cnr--br' aria-hidden />
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
