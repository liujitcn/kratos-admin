import { Button, Picker, Text, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useEffect, useMemo, useState } from 'react'
import {
  useLocaleStore,
  navigateToLogin,
  type SupportedLocale,
  useI18n,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './settings.scss'

/** 应用设置页。 */
export default function SettingsPage() {
  const { locale, setLocale, t } = useI18n()
  const languageOptions = useLocaleStore((state) => state.languageOptions)
  const [logoutLoading, setLogoutLoading] = useState(false)
  const authenticated = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const logout = useUserStore((state) => state.logout)
  const localeLabels = useMemo(() => languageOptions.map((item) => item.native_name), [languageOptions])
  const localeIndex = Math.max(0, languageOptions.findIndex((item) => item.language_code === locale))

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('system.settings.title') })
  }, [locale, t])

  useLoad(() => {
    if (process.env.TARO_ENV !== 'weapp' && !ensureAuthenticated()) navigateToLogin()
  })

  const onLogout = async () => {
    if (logoutLoading) return
    const result = await Taro.showModal({
      content: t('system.settings.logoutConfirm'),
      confirmColor: '#27BA9B',
    })
    if (!result.confirm) return
    setLogoutLoading(true)
    try {
      await logout()
      await Taro.navigateBack()
    } catch {
      await Taro.showToast({ icon: 'none', title: t('system.settings.logoutFailed') })
    } finally {
      setLogoutLoading(false)
    }
  }

  return (
    <View className='settings-viewport'>
      {process.env.TARO_ENV === 'weapp' ? (
        <View className='settings-list'>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='openSetting'>
            {t('system.settings.authorization')}
          </Button>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='feedback'>
            {t('system.settings.feedback')}
          </Button>
          <Button className='settings-item settings-arrow' hoverClass='none' openType='contact'>
            {t('system.settings.contact')}
          </Button>
        </View>
      ) : null}
      <View className='settings-list'>
        <Picker
          mode='selector'
          range={localeLabels}
          value={localeIndex}
          onChange={(event) => {
            const nextLocale = languageOptions[Number(event.detail.value)]?.language_code as
              | SupportedLocale
              | undefined
            if (nextLocale) void setLocale(nextLocale)
          }}
        >
          <View className='settings-item settings-arrow settings-locale'>
            <Text>{t('common.field.language')}</Text>
            <Text className='settings-locale__value'>{localeLabels[localeIndex]}</Text>
          </View>
        </Picker>
      </View>
      {authenticated ? (
        <View className='settings-action'>
          <View className='settings-button' onClick={() => void onLogout()}>
            {t('common.action.logout')}
          </View>
        </View>
      ) : null}
    </View>
  )
}
