import type { CSSProperties } from 'react'
import { Image, ScrollView, View } from '@tarojs/components'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import {
  formatSrc,
  getMpCapsuleRightGap,
  navigateAppRoute,
  navigateToLogin,
  useI18n,
  useSettingStore,
  useThemeStore,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './my.scss'

/** 个人中心首页。 */
export default function MyPage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const profile = useUserStore((state) => state.userInfo)
  const isLoggedIn = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const aiEnabled = useSettingStore((state) => state.aiEnabled)
  const avatarText = (profile?.nick_name || 'U').slice(0, 1)
  /** MP-WEIXIN 原生胶囊按钮右侧避让宽度，写入 --mp-capsule-gap 供 .k-top 预留空间。 */
  const capsuleGap = getMpCapsuleRightGap()
  const topNavStyle = { '--mp-capsule-gap': `${capsuleGap}px` } as CSSProperties
  /** 设计稿齿轮线稿，随主题取色的 data URI，替代 CSS 手绘图标。 */
  const gearIcon = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${
      themeStore.theme === 'light' ? '#102232' : '#e9f4f6'
    }" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8 2 2 0 1 1-2.8 2.8 1.6 1.6 0 0 0-2.7 1.1 2 2 0 1 1-4 0 1.6 1.6 0 0 0-2.6-1.1 2 2 0 1 1-2.8-2.8A1.6 1.6 0 0 0 3 13.9a2 2 0 1 1 0-4 1.6 1.6 0 0 0 1.5-2.7 2 2 0 1 1 2.8-2.8A1.6 1.6 0 0 0 10 4.6a2 2 0 1 1 4 0 1.6 1.6 0 0 0 2.7.6 2 2 0 1 1 2.8 2.8 1.6 1.6 0 0 0-.1 2.6a2.2 2.2 0 1 1 0 4.4Z"/></svg>`,
  )}`
  /** 个人卡账号行：登录展示账号，未登录提示去登录。 */
  const accountText =
    isLoggedIn && profile ? profile.user_name || profile.nick_name || '' : t('system.profile.login_prompt')

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
      className={`my-viewport${process.env.TARO_ENV === 'weapp' ? ' my-viewport--weapp' : ''} ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <View className='my-content'>
        <View className='k-top' style={topNavStyle}>
          <View className='k-top__title'>{t('core.navigation.my')}</View>
          <View className='k-ic' onClick={() => openAuthenticatedPage('app/settings')}>
            <Image className='my-gear' src={gearIcon} mode='aspectFit' />
          </View>
        </View>

        <View className='my-stack'>
          <View
            className='k-glass k-glass--lit k-pfx my-profile'
            onClick={() => navigateAppRoute('app/profile')}
          >
            <View className='my-pfx-main'>
              {isLoggedIn && profile ? (
                <Image
                  className='k-pfx__av'
                  src={profile.avatar ? formatSrc(profile.avatar) : defaultAvatar}
                  mode='aspectFill'
                />
              ) : (
                <View className='k-pfx__av'>{avatarText}</View>
              )}
              <View className='my-pfx-meta'>
                <View className='k-pfx__nm'>
                  {isLoggedIn && profile ? profile.nick_name : t('system.profile.not_logged_in')}
                </View>
                <View className='k-pfx__ac'>{accountText}</View>
              </View>
            </View>
            <View className='k-pfx__ent'>{t('system.profile.title')} ›</View>
          </View>

          {aiEnabled ? (
            <View className='k-aix' onClick={() => openAuthenticatedPage('app/ai')}>
              <View className='k-aix__ic'>AI</View>
              <View className='my-aix-copy'>
                <View className='k-aix__t'>{t('system.settings.ai_title')}</View>
                <View className='k-aix__d'>{t('system.settings.ai_description')}</View>
              </View>
              <View className='k-aix__b'>{t('system.settings.go_ask')}</View>
            </View>
          ) : null}
        </View>
      </View>
    </ScrollView>
  )
}
