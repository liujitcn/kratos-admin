import { Image, ScrollView, Text, View } from '@tarojs/components'
import centerBackground from '@liujitcn/kratos-taro-app-core/static/images/center_bg.png'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import {
  formatSrc,
  navigateAppRoute,
  navigateToLogin,
  useI18n,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import Taro from '@tarojs/taro'
import { useEffect } from 'react'
import './my.scss'

/** 个人中心首页。 */
export default function MyPage() {
  const { locale, t } = useI18n()
  const profile = useUserStore((state) => state.userInfo)
  const isLoggedIn = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('core.navigation.my') })
  }, [locale, t])

  const openAuthenticatedPage = (route: string) => {
    if (!ensureAuthenticated()) {
      navigateToLogin()
      return
    }
    navigateAppRoute(route)
  }

  return (
    <ScrollView
      enableBackToTop
      scrollY
      className='my-viewport'
      style={{ backgroundImage: `url(${centerBackground})` }}
    >
      <View className={`my-profile${process.env.TARO_ENV === 'weapp' ? ' my-profile--weapp' : ''}`}>
        {isLoggedIn && profile ? (
          <View className='my-overview'>
            <View onClick={() => navigateAppRoute('app/profile')}>
              <Image
                className='my-avatar'
                src={profile.avatar ? formatSrc(profile.avatar) : defaultAvatar}
                mode='aspectFill'
              />
            </View>
            <View className='my-meta'>
              <View className='my-nickname'>{profile.nick_name}</View>
              <View className='my-extra' onClick={() => navigateAppRoute('app/profile')}>
                <Text className='my-update'>{t('system.profile.avatar_update')}</Text>
              </View>
            </View>
          </View>
        ) : (
          <View className='my-overview'>
            <View onClick={() => navigateToLogin()}>
              <Image className='my-avatar my-avatar--gray' src={defaultAvatar} mode='aspectFill' />
            </View>
            <View className='my-meta'>
              <View className='my-nickname' onClick={() => navigateToLogin()}>
                {t('system.profile.not_logged_in')}
              </View>
              <View className='my-extra'>
                <Text className='my-tips'>{t('system.profile.login_prompt')}</Text>
              </View>
            </View>
          </View>
        )}
        <View className='my-settings' onClick={() => openAuthenticatedPage('app/settings')}>
          {t('system.settings.title')}
        </View>
      </View>

      <View className='my-ai-entry' onClick={() => openAuthenticatedPage('app/ai')}>
        <View className='my-ai-entry__icon'>AI</View>
        <View className='my-ai-entry__content'>
          <View className='my-ai-entry__title'>{t('system.settings.ai_title')}</View>
          <View className='my-ai-entry__desc'>{t('system.settings.ai_description')}</View>
        </View>
        <View className='my-ai-entry__action'>{t('system.settings.go_ask')}</View>
      </View>
    </ScrollView>
  )
}
