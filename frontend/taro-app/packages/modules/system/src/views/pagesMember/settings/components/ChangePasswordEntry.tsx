import { Button, Image, Input, Text, View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { useMemo, useState } from 'react'
import {
  useI18n,
  useThemeStore,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import { defAccountService } from '@liujitcn/kratos-taro-app-core/api/base/v1/account'
import {
  encryptWithPublicKey,
  fetchSecretPublicKey,
} from '@liujitcn/kratos-taro-app-core/utils/secretCrypto'
import type { CurrentPasswordPolicy } from '@liujitcn/kratos-taro-app-core/rpc/base/v1/account'
import './ChangePasswordEntry.scss'

/** 设置页账户安全中的修改密码入口与弹层。 */
export default function ChangePasswordEntry() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const authenticated = useUserStore((state) => state.isAuthenticated())
  const clearUserData = useUserStore((state) => state.clearUserData)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [policyLoading, setPolicyLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordPolicy, setPasswordPolicy] = useState<CurrentPasswordPolicy>()

  const passwordIcon = useMemo(
    () =>
      `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${themeStore.colors.aiWarning}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="10" width="16" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/><path d="M12 14v2"/></svg>`)}`,
    [themeStore.colors.aiWarning],
  )

  const strengthScore = useMemo(() => {
    if (!newPassword) return 0
    let score = 0
    if (/[a-z]/.test(newPassword)) score += 1
    if (/[A-Z]/.test(newPassword)) score += 1
    if (/\d/.test(newPassword)) score += 1
    if (/[^A-Za-z0-9]/.test(newPassword)) score += 1
    if (score >= 4) return 3
    if (score === 3) return 2
    return 1
  }, [newPassword])

  const strengthLevel = strengthScore === 0 ? 'empty' : strengthScore === 1 ? 'low' : strengthScore === 2 ? 'medium' : 'high'
  const policyTip = passwordPolicy
    ? [
        t('system.settings.password_policy_min_length', { count: passwordPolicy.min_length }),
        t('system.settings.password_policy_complexity', {
          count: passwordPolicy.min_complexity_classes,
        }),
        ...(passwordPolicy.history_count > 0
          ? [t('system.settings.password_policy_history', { count: passwordPolicy.history_count })]
          : []),
        passwordPolicy.max_age_days > 0
          ? t('system.settings.password_policy_max_age', { count: passwordPolicy.max_age_days })
          : t('system.settings.password_policy_no_expiry'),
      ].join(t('system.settings.password_policy_separator'))
    : t('system.settings.password_policy_default')

  const loadPasswordPolicy = async () => {
    setPolicyLoading(true)
    try {
      setPasswordPolicy(await defAccountService.GetCurrentPasswordPolicy({}))
    } catch {
      setPasswordPolicy(undefined)
    } finally {
      setPolicyLoading(false)
    }
  }

  const openDialog = () => {
    if (!authenticated) return
    setDialogOpen(true)
    setErrorMessage('')
    if (!passwordPolicy && !policyLoading) void loadPasswordPolicy()
  }

  const closeDialog = () => {
    if (submitting) return
    setDialogOpen(false)
    setOldPassword('')
    setNewPassword('')
    setConfirmPassword('')
    setErrorMessage('')
  }

  const handleSubmit = async () => {
    if (submitting) return
    if (!oldPassword || !newPassword || !confirmPassword) {
      setErrorMessage(t('system.settings.password_required'))
      return
    }
    if (newPassword !== confirmPassword) {
      setErrorMessage(t('system.settings.password_mismatch'))
      return
    }
    setSubmitting(true)
    setErrorMessage('')
    try {
      const publicKey = await fetchSecretPublicKey()
      await defAccountService.UpdateUserPassword({
        user_password: {
          old_pwd: await encryptWithPublicKey(publicKey, oldPassword),
          new_pwd: await encryptWithPublicKey(publicKey, newPassword),
        },
      })
      setDialogOpen(false)
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      await Taro.showToast({ icon: 'none', title: t('system.settings.password_updated') })
      await clearUserData()
      await Taro.reLaunch({ url: '/pages/login/login' })
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('common.message.request_error'))
    } finally {
      setSubmitting(false)
    }
  }

  if (!authenticated) return null

  return (
    <>
      <View className='k-set-row k-set-row--arrow' onClick={openDialog}>
        <View className='k-set-row__lead'>
          <View className='k-set-row__icon'>
            <Image className='change-password-entry__icon' src={passwordIcon} mode='aspectFit' />
          </View>
          <Text className='k-set-row__title'>{t('system.settings.password_title')}</Text>
        </View>
        <Text className='k-set-row__value settings-row__value'>
          {t('system.settings.password_action')}
        </Text>
      </View>

      {dialogOpen ? (
        <View className='change-password-dialog' onClick={closeDialog}>
          <View className='change-password-dialog__panel' onClick={(event) => event.stopPropagation()}>
            <View className='change-password-dialog__header'>
              <View>
                <Text className='change-password-dialog__title'>{t('system.settings.password_title')}</Text>
                <Text className='change-password-dialog__description'>
                  {t('system.settings.password_description')}
                </Text>
              </View>
              <Text className='change-password-dialog__close' onClick={closeDialog}>×</Text>
            </View>
            <View className='change-password-dialog__body'>
              <Text className='change-password-dialog__label'>{t('system.settings.password_current')}</Text>
              <Input
                className='change-password-dialog__input'
                password
                value={oldPassword}
                placeholder={t('system.settings.password_current_placeholder')}
                onInput={(event) => {
                  setOldPassword(event.detail.value)
                  setErrorMessage('')
                }}
              />
              <Text className='change-password-dialog__label'>{t('system.settings.password_new')}</Text>
              <Input
                className='change-password-dialog__input'
                password
                value={newPassword}
                placeholder={t('system.settings.password_new_placeholder')}
                onInput={(event) => {
                  setNewPassword(event.detail.value)
                  setErrorMessage('')
                }}
              />
              <View className='change-password-strength'>
                <View className='change-password-strength__header'>
                  <Text>{t('system.settings.password_strength')}</Text>
                  <Text className={`change-password-strength__label--${strengthLevel}`}>
                    {t(`system.settings.password_strength_${strengthLevel}`)}
                  </Text>
                </View>
                <View className='change-password-strength__bars'>
                  {[1, 2, 3].map((segment) => (
                    <View
                      key={segment}
                      className={`change-password-strength__bar${segment <= strengthScore ? ` change-password-strength__bar--${strengthLevel}` : ''}`}
                    />
                  ))}
                </View>
                <Text className='change-password-strength__tip'>{policyTip}</Text>
              </View>
              <Text className='change-password-dialog__label'>{t('system.settings.password_confirm')}</Text>
              <Input
                className='change-password-dialog__input'
                password
                value={confirmPassword}
                placeholder={t('system.settings.password_confirm_placeholder')}
                onInput={(event) => {
                  setConfirmPassword(event.detail.value)
                  setErrorMessage('')
                }}
                onConfirm={() => void handleSubmit()}
              />
              {policyLoading ? (
                <Text className='change-password-dialog__hint'>
                  {t('system.settings.password_policy_loading')}
                </Text>
              ) : null}
              {errorMessage ? <Text className='change-password-dialog__error'>{errorMessage}</Text> : null}
            </View>
            <View className='change-password-dialog__footer'>
              <Button className='change-password-dialog__button change-password-dialog__button--cancel' disabled={submitting} onClick={closeDialog}>
                {t('common.action.cancel')}
              </Button>
              <Button className='change-password-dialog__button change-password-dialog__button--confirm' loading={submitting} onClick={() => void handleSubmit()}>
                {t('system.settings.password_submit')}
              </Button>
            </View>
          </View>
        </View>
      ) : null}
    </>
  )
}
