import { Button, Image, Picker, Text, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useMemo, useState } from 'react'
import {
  useLocaleStore,
  navigateToLogin,
  type SupportedLocale,
  useI18n,
  useThemeStore,
  useUserStore,
  type ThemeMode,
  KratosNavBar,
} from '@liujitcn/kratos-taro-app-core'
import { defMfaService } from '@liujitcn/kratos-taro-app-core/api/base/v1/mfa'
import { isAuthExpiredError } from '@liujitcn/kratos-taro-app-core/utils/http'
import PasswordVerifyDialog from './components/PasswordVerifyDialog'
import ChangePasswordEntry from './components/ChangePasswordEntry'
import './settings.scss'

/** SettingsPage 展示应用设置与账号安全操作。 */
export default function SettingsPage() {
  const { locale, setLocale, t } = useI18n()
  const themeStore = useThemeStore()
  const themeModeOptions: ThemeMode[] = ['auto', 'dark', 'light']
  const themeModeLabels = [
    t('system.settings.theme_auto'),
    t('system.settings.theme_dark'),
    t('system.settings.theme_light'),
  ]
  const themeModeIndex = Math.max(0, themeModeOptions.indexOf(themeStore.mode))
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
  const settingsIcons = useMemo(() => {
    const primary = themeStore.colors.aiPrimary
    const warning = themeStore.colors.aiWarning
    const violet = themeStore.colors.aiViolet
    const createIcon = (body: string, color: string) =>
      'data:image/svg+xml;charset=utf-8,' +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="' +
          color +
          '" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
          body +
          '</svg>',
      )

    return {
      theme: createIcon('<path d="M21 12.8A8.5 8.5 0 1 1 11.2 3a6.5 6.5 0 0 0 9.8 9.8Z"/>', primary),
      mfa: createIcon('<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6l-8-3Z"/><path d="m9 12 2 2 4-4"/>', primary),
      language: createIcon('<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18"/>', primary),
      authorization: createIcon('<circle cx="8" cy="15" r="4"/><path d="m11 12 7-7 3 3-7 7"/><path d="m15 8 2 2"/>', warning),
      feedback: createIcon('<path d="M4 5h16v12H8l-4 4z"/>', violet),
      contact: createIcon('<path d="M4 5c0 9 6 15 15 15 1 0 2-1 2-2v-2l-4-2-2 2a11 11 0 0 1-5-5l2-2-2-4H6c-1 0-2 1-2 2Z"/>', primary),
    }
  }, [themeStore.colors.aiPrimary, themeStore.colors.aiWarning, themeStore.colors.aiViolet])

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

  /** onLogout 确认后退出当前账号并返回来源页面。 */
  const onLogout = async () => {
    if (logoutLoading) return
    const result = await Taro.showModal({
      content: t('system.settings.logout_confirm'),
      confirmColor: themeStore.colors.aiPrimary,
      cancelColor: themeStore.colors.aiTextMuted,
    })
    if (!result.confirm) return
    setLogoutLoading(true)
    try {
      await logout()
      await Taro.navigateBack()
    } catch (error) {
      if (!isAuthExpiredError(error)) {
        await Taro.showToast({ icon: 'none', title: t('system.settings.logout_failed') })
      }
    } finally {
      setLogoutLoading(false)
    }
  }

  return (
    <>
      <View
        className={`settings-viewport ${
          themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
        }`}
      >
        <KratosNavBar title={t('system.settings.title')} />

        <View className='settings-content'>
          <View className='k-set-cat'>{t('system.settings.section_security')}</View>
          {authenticated ? (
            <View className='k-set-row k-set-row--arrow' onClick={onMfaTap}>
              <View className='k-set-row__lead'>
                <View className='k-set-row__icon'>
                  <Image className='settings-row__icon' src={settingsIcons.mfa} mode='aspectFit' />
                </View>
                <Text className='k-set-row__title'>{t('system.settings.mfa_title')}</Text>
              </View>
              <Text
                className={`k-set-row__value settings-row__value ${
                  mfaEnabled ? 'k-set-row__value--on' : ''
                }`}
              >
                {mfaStatusText}
              </Text>
            </View>
          ) : null}

          {process.env.TARO_ENV === 'h5' ? <ChangePasswordEntry /> : null}

          <View className='k-set-cat'>{t('system.settings.section_appearance')}</View>
          <Picker
            mode='selector'
            range={themeModeLabels}
            value={themeModeIndex}
            onChange={(event) => {
              const nextMode = themeModeOptions[Number(event.detail.value)]
              if (nextMode) themeStore.setMode(nextMode)
            }}
          >
            <View className='k-set-row k-set-row--arrow'>
              <View className='k-set-row__lead'>
                <View className='k-set-row__icon'>
                  <Image className='settings-row__icon' src={settingsIcons.theme} mode='aspectFit' />
                </View>
                <Text className='k-set-row__title'>{t('system.settings.theme')}</Text>
              </View>
              <Text className='k-set-row__value settings-row__value'>
                {themeModeLabels[themeModeIndex]}
              </Text>
            </View>
          </Picker>
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
            <View className='k-set-row k-set-row--arrow'>
              <View className='k-set-row__lead'>
                <View className='k-set-row__icon'>
                  <Image className='settings-row__icon' src={settingsIcons.language} mode='aspectFit' />
                </View>
                <Text className='k-set-row__title'>{t('common.field.language')}</Text>
              </View>
              <Text className='k-set-row__value settings-row__value'>{localeLabels[localeIndex]}</Text>
            </View>
          </Picker>

          {process.env.TARO_ENV === 'weapp' ? (
            <>
              <View className='k-set-cat'>{t('system.settings.section_more')}</View>
              <Button
                className='k-set-row k-set-row--arrow settings-row--button'
                hoverClass='none'
                openType='openSetting'
              >
                <View className='k-set-row__lead'>
                  <View className='k-set-row__icon'>
                    <Image
                      className='settings-row__icon'
                      src={settingsIcons.authorization}
                      mode='aspectFit'
                    />
                  </View>
                  <Text className='k-set-row__title'>{t('system.settings.authorization')}</Text>
                </View>
              </Button>
              <Button
                className='k-set-row k-set-row--arrow settings-row--button'
                hoverClass='none'
                openType='feedback'
              >
                <View className='k-set-row__lead'>
                  <View className='k-set-row__icon'>
                    <Image
                      className='settings-row__icon'
                      src={settingsIcons.feedback}
                      mode='aspectFit'
                    />
                  </View>
                  <Text className='k-set-row__title'>{t('system.settings.feedback')}</Text>
                </View>
              </Button>
              <Button
                className='k-set-row k-set-row--arrow settings-row--button'
                hoverClass='none'
                openType='contact'
              >
                <View className='k-set-row__lead'>
                  <View className='k-set-row__icon'>
                    <Image
                      className='settings-row__icon'
                      src={settingsIcons.contact}
                      mode='aspectFit'
                    />
                  </View>
                  <Text className='k-set-row__title'>{t('system.settings.contact')}</Text>
                </View>
              </Button>
            </>
          ) : null}

          {authenticated ? (
            <View className='k-set-logout' onClick={() => void onLogout()}>
              {t('common.action.logout')}
            </View>
          ) : null}
        </View>
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
