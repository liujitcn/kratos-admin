import { Image, ScrollView, Text, View } from '@tarojs/components'
import centerBackground from '@liujitcn/kratos-taro-app-core/static/images/center_bg.png'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import {
  formatSrc,
  navigateAppRoute,
  navigateToLogin,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './my.scss'

/** 个人中心首页。 */
export default function MyPage() {
  const profile = useUserStore((state) => state.userInfo)
  const isLoggedIn = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)

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
                <Text className='my-update'>更新头像昵称</Text>
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
                未登录
              </View>
              <View className='my-extra'>
                <Text className='my-tips'>点击登录账号</Text>
              </View>
            </View>
          </View>
        )}
        <View className='my-settings' onClick={() => openAuthenticatedPage('app/settings')}>
          设置
        </View>
      </View>

      <View className='my-ai-entry' onClick={() => openAuthenticatedPage('app/ai')}>
        <View className='my-ai-entry__icon'>AI</View>
        <View className='my-ai-entry__content'>
          <View className='my-ai-entry__title'>智能助手</View>
          <View className='my-ai-entry__desc'>帮你整理信息、回答问题并处理日常任务</View>
        </View>
        <View className='my-ai-entry__action'>去提问</View>
      </View>
    </ScrollView>
  )
}
