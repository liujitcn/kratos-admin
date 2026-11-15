import { Button, Input, Text, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useMemo, useState } from 'react'
import {
  useI18n,
  useThemeStore,
  useUserStore,
  KratosNavBar,
  navigateToLogin,
} from '@liujitcn/kratos-taro-app-core'
import { defAccountService } from '@liujitcn/kratos-taro-app-core/api/base/v1/account'
import {
  encryptWithPublicKey,
  fetchSecretPublicKey,
} from '@liujitcn/kratos-taro-app-core/utils/secretCrypto'
import type { CurrentPasswordPolicy } from '@liujitcn/kratos-taro-app-core/rpc/base/v1/account'
import './password.scss'

/** ChangePasswordPage 提供独立修改密码页面，与 uni 端 app/password 形态对齐。 */
export default function ChangePasswordPage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const clearUserData = useUserStore((state) => state.clearUserData)
  const [submitting, setSubmitting] = useState(false)
  const [policyLoading, setPolicyLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordPolicy, setPasswordPolicy] = useState<CurrentPasswordPolicy>()

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
  const strengthLabelKey =
    strengthLevel === 'low'
      ? 'system.settings.password_strength_low'
      : strengthLevel === 'medium'
        ? 'system.settings.password_strength_medium'
        : strengthLevel === 'high'
          ? 'system.settings.password_strength_high'
          : 'system.settings.password_strength_empty'
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

  useLoad(() => {
    if (!ensureAuthenticated()) {
      navigateToLogin()
      return
    }
    void loadPasswordPolicy()
  })

  /** 读取当前用户生效的密码策略。 */
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

  /** 校验并提交修改密码请求；成功后清空登录态重新登录。 */
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
      await Taro.showToast({ icon: 'none', title: t('system.settings.password_updated') })
      await clearUserData()
      await Taro.reLaunch({ url: '/pages/login/login' })
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('common.message.request_error'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View
      className={`change-password-page ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <KratosNavBar title={t('system.settings.password_title')} />
      <View className='change-password-page__content'>
        <View className='k-glass change-password-card'>
          <Text className='change-password-card__description'>
            {t('system.settings.password_description')}
          </Text>
          <Text className='change-password-card__label'>
            {t('system.settings.password_current')}
          </Text>
          <Input
            className='change-password-card__input'
            password
            value={oldPassword}
            placeholder={t('system.settings.password_current_placeholder')}
            onInput={(event) => {
              setOldPassword(event.detail.value)
              setErrorMessage('')
            }}
          />
          <Text className='change-password-card__label'>{t('system.settings.password_new')}</Text>
          <Input
            className='change-password-card__input'
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
                {t(strengthLabelKey)}
              </Text>
            </View>
            <View className='change-password-strength__bars'>
              {[1, 2, 3].map((segment) => (
                <View
                  key={segment}
                  className={`change-password-strength__bar ${
                    segment <= strengthScore ? `change-password-strength__bar--${strengthLevel}` : ''
                  }`}
                />
              ))}
            </View>
            <Text className='change-password-strength__tip'>{policyTip}</Text>
          </View>
          <Text className='change-password-card__label'>
            {t('system.settings.password_confirm')}
          </Text>
          <Input
            className='change-password-card__input'
            password
            value={confirmPassword}
            placeholder={t('system.settings.password_confirm_placeholder')}
            onInput={(event) => {
              setConfirmPassword(event.detail.value)
              setErrorMessage('')
            }}
          />
          {policyLoading ? (
            <Text className='change-password-card__hint'>
              {t('system.settings.password_policy_loading')}
            </Text>
          ) : null}
          {errorMessage ? (
            <Text className='change-password-card__error'>{errorMessage}</Text>
          ) : null}
        </View>
        <Button
          className='k-save change-password-page__submit'
          loading={submitting}
          onClick={handleSubmit}
        >
          {t('system.settings.password_submit')}
        </Button>
      </View>
    </View>
  )
}
