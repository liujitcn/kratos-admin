import type { ReactNode } from 'react'
import { Image, ScrollView, View } from '@tarojs/components'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import {
  formatSrc,
  KratosNavBar,
  navigateAppRoute,
  navigateToLogin,
  useI18n,
  useSettingStore,
  useThemeStore,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './KratosMyPage.scss'

/** KratosMyPageProps 个人中心的项目业务扩展属性。 */
export interface KratosMyPageProps {
  /** 在个人信息和 AI 助手之后渲染的项目业务内容。 */
  children?: ReactNode
}

/** KratosMyPage 保留公共个人中心入口，并在下方渲染项目业务内容。 */
export default function KratosMyPage({ children }: KratosMyPageProps) {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const profile = useUserStore((state) => state.userInfo)
  const isLoggedIn = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const aiEnabled = useSettingStore((state) => state.aiEnabled)
  const avatarText = (profile?.nick_name || 'U').slice(0, 1)
  /** 个人卡账号行：登录展示账号，未登录提示去登录。 */
  const accountText =
    isLoggedIn && profile ? profile.user_name || profile.nick_name || '' : t('system.profile.login_prompt')

  /** 打开需要登录的公共页面。 */
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
      className={`my-viewport ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <View className='my-content'>
        <KratosNavBar
          title={t('core.navigation.my')}
          variant='tab'
          action={{ icon: 'settings', label: t('system.settings.title'), onClick: () => openAuthenticatedPage('app/settings') }}
        />

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

          {children}
        </View>
      </View>
    </ScrollView>
  )
}
