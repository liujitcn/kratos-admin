import { Button, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import {
  navigateToLogin,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './settings.scss'

/** 应用设置页。 */
export default function SettingsPage() {
  const [logoutLoading, setLogoutLoading] = useState(false)
  const authenticated = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const logout = useUserStore((state) => state.logout)

  useLoad(() => {
    if (process.env.TARO_ENV !== 'weapp' && !ensureAuthenticated()) navigateToLogin()
  })

  const onLogout = async () => {
    if (logoutLoading) return
    const result = await Taro.showModal({ content: '是否退出登录？', confirmColor: '#27BA9B' })
    if (!result.confirm) return
    setLogoutLoading(true)
    try {
      await logout()
      await Taro.navigateBack()
    } catch {
      await Taro.showToast({ icon: 'none', title: '退出登录失败' })
    } finally {
      setLogoutLoading(false)
    }
  }

  return (
    <View className='settings-viewport'>
      {process.env.TARO_ENV === 'weapp' ? (
        <View className='settings-list'>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='openSetting'>
            授权管理
          </Button>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='feedback'>
            问题反馈
          </Button>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='contact'>
            联系我们
          </Button>
        </View>
      ) : null}
      {authenticated ? (
        <View className='settings-action'>
          <View className='settings-button' onClick={() => void onLogout()}>
            退出登录
          </View>
        </View>
      ) : null}
    </View>
  )
}
