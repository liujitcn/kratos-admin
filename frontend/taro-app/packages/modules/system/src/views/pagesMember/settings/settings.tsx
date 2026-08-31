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
import { defMfaService } from '@liujitcn/kratos-taro-app-core/api/base/mfa'
import PasswordVerifyDialog from './components/PasswordVerifyDialog'
import './settings.scss'

/** 应用设置页。 */
export default function SettingsPage() {
  const { locale, setLocale, t } = useI18n()
  const languageOptions = useLocaleStore((state) => state.languageOptions)
  const [logoutLoading, setLogoutLoading] = useState(false)
  const [mfaStatusLoading, setMfaStatusLoading] = useState(false)
  const [mfaDialogOpen, setMfaDialogOpen] = useState(false)
  const [mfaDialogMode, setMfaDialogMode] = useState<'setup' | 'disable'>('setup')
  const [mfaEnabled, setMfaEnabled] = useState(false)
  const [mfaMethod, setMfaMethod] = useState('totp')
  const authenticated = useUserStore((state) => state.isAuthenticated())
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const logout = useUserStore((state) => state.logout)
  const localeLabels = useMemo(
    () => languageOptions.map((item) => item.native_name),
    [languageOptions],
  )
  const localeIndex = Math.max(
    0,
    languageOptions.findIndex((item) => item.language_code === locale),
  )

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('system.settings.title') })
  }, [locale, t])

  const loadMfaStatus = async (force = false) => {
    if (!useUserStore.getState().isAuthenticated() || (mfaStatusLoading && !force)) return
    setMfaStatusLoading(true)
    try {
      const result = await defMfaService.GetMfaStatus({})
      setMfaEnabled(result.enabled)
      setMfaMethod(result.method || 'totp')
      return result
    } finally {
      setMfaStatusLoading(false)
    }
  }

  const mfaStatusText = mfaEnabled
    ? t('system.settings.mfa_enabled_method', {
        method:
          mfaMethod === 'webauthn'
            ? t('system.settings.mfa_method_webauthn')
            : t('system.settings.mfa_method_totp'),
      })
    : t('system.settings.mfa_disabled')

  const onMfaTap = async () => {
    if (!ensureAuthenticated()) {
      navigateToLogin()
      return
    }
    const currentMfaEnabled = mfaEnabled
    let latestMfaEnabled = currentMfaEnabled
    try {
      const result = await loadMfaStatus(true)
      latestMfaEnabled = result?.enabled ?? currentMfaEnabled
    } catch {
      // 状态刷新失败时保留页面已有状态，避免已启用用户无法进入禁用流程。
      setMfaEnabled(currentMfaEnabled)
    }
    if (latestMfaEnabled) {
      setMfaDialogMode('disable')
      setMfaDialogOpen(true)
      return
    }
    setMfaDialogMode('setup')
    setMfaDialogOpen(true)
  }

  const onMfaSuccess = async (method: string) => {
    setMfaDialogOpen(false)
    setMfaEnabled(true)
    setMfaMethod(method || 'totp')
    await Taro.showToast({ icon: 'none', title: t('system.settings.mfa_bind_success') })
    await useUserStore.getState().clearUserData()
    await Taro.reLaunch({ url: '/pages/login/login' })
  }

  /** 禁用成功后清理旧会话并重新登录。 */
  const onMfaDisabled = async () => {
    setMfaDialogOpen(false)
    setMfaEnabled(false)
    await Taro.showToast({ icon: 'none', title: t('system.settings.mfa_disable_success') })
    await useUserStore.getState().clearUserData()
    await Taro.reLaunch({ url: '/pages/login/login' })
  }

  useLoad(() => {
    if (!ensureAuthenticated()) {
      if (process.env.TARO_ENV !== 'weapp') navigateToLogin()
      return
    }
    void loadMfaStatus()
  })

  const onLogout = async () => {
    if (logoutLoading) return
    const result = await Taro.showModal({
      content: t('system.settings.logout_confirm'),
      confirmColor: '#27BA9B',
    })
    if (!result.confirm) return
    setLogoutLoading(true)
    try {
      await logout()
      await Taro.navigateBack()
    } catch {
      await Taro.showToast({ icon: 'none', title: t('system.settings.logout_failed') })
    } finally {
      setLogoutLoading(false)
    }
  }

  return (
    <>
      <View className='settings-viewport'>
        {process.env.TARO_ENV === 'weapp' ? (
          <View className='settings-list'>
            <Button
              className='settings-item settings-arrow'
              hoverClass='none'
              openType='openSetting'
            >
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
          {authenticated ? (
            <View className='settings-item settings-arrow settings-mfa' onClick={onMfaTap}>
              <Text>{t('system.settings.mfa_title')}</Text>
              <Text className='settings-mfa__value'>{mfaStatusText}</Text>
            </View>
          ) : null}
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
      <PasswordVerifyDialog
        open={mfaDialogOpen}
        mode={mfaDialogMode}
        method={mfaMethod}
        onClose={() => setMfaDialogOpen(false)}
        onSuccess={onMfaSuccess}
        onDisabled={onMfaDisabled}
      />
    </>
  )
}
