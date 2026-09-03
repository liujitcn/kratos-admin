import { Button, Image, Input, Picker, Text, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useEffect, useMemo, useRef, useState } from 'react'
import defaultLogo from '@liujitcn/kratos-taro-app-core/static/images/logo_icon.png'
import { defLoginService } from '../../../api/base/v1/login'
import { defMfaService } from '../../../api/base/v1/mfa'
import { navigateAppRoute } from '../../../navigation'
import { LoginStatus, type LoginRequest, type LoginResponse } from '../../../rpc/base/v1/login'
import { useSettingStore, useUserStore } from '../../../stores'
import { restoreLoginRedirect } from '../../../utils/navigation'
import { encryptPassword, PASSWORD_CRYPTO_SCENE } from '../../../utils/passwordCrypto'
import { createWebAuthnCredential, getWebAuthnAssertion } from '../../../utils/webauthn'
import MfaRecoveryCodesDialog from '../../../components/MfaRecoveryCodesDialog'
import MfaSetupPanel from '../../../components/MfaSetupPanel'
import BehaviorCaptcha from './components/BehaviorCaptcha'
import type { BehaviorCaptchaData, BehaviorCaptchaPoint } from './components/types'
import './login.scss'
import { t as translate, useLocaleStore, useI18n, type SupportedLocale } from '../../../locales'

const behaviorCaptchaTypes = new Set(['slide', 'click', 'rotate'])
const wechatMiniProvider = 'wechatmini'

type BehaviorPayload = BehaviorCaptchaData & {
  type?: string
  width?: number
  height?: number
}

const emptyLoginForm = (): LoginRequest => ({
  tenant_code: '',
  user_name: '',
  password: undefined,
  captcha_id: '',
  captcha_code: '',
})

function isWechatUnboundError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const response = error as {
    data?: {
      reason?: string | number
      message?: string
      binding_required?: boolean
      error?: { reason?: string | number; message?: string }
    }
  }
  const reason = response.data?.reason ?? response.data?.error?.reason
  return response.data?.binding_required === true || String(reason || '') === 'UNAUTHENTICATED'
}

/** 提取小程序请求失败时后端返回的可展示消息。 */
function getWechatErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message
  if (!error || typeof error !== 'object') return fallback
  const response = error as {
    data?: {
      message?: string
      error?: { message?: string }
    }
  }
  return response.data?.message || response.data?.error?.message || fallback
}

async function resolveMiniCaptchaImage(payload: string, captchaId: string): Promise<string> {
  if (!payload.startsWith('data:image/')) return payload
  const commaIndex = payload.indexOf(',')
  if (commaIndex < 0 || !Taro.env.USER_DATA_PATH) return payload
  const filePath = `${Taro.env.USER_DATA_PATH}/kratos-login-captcha-${captchaId}.png`
  await new Promise<void>((resolve, reject) => {
    Taro.getFileSystemManager().writeFile({
      filePath,
      data: payload.slice(commaIndex + 1),
      encoding: 'base64',
      success: () => resolve(),
      fail: () => reject(new Error(translate('core.login.captcha_write_failed'))),
    })
  })
  return filePath
}

/** 账号和微信授权登录页。 */
export default function LoginPage() {
  const { locale, setLocale, t } = useI18n()
  const languageOptions = useLocaleStore((state) => state.languageOptions)
  const settings = useSettingStore((state) => state.data)
  const loadSettings = useSettingStore((state) => state.loadData)
  const userStore = useUserStore.getState()
  const [agreed, setAgreed] = useState(false)
  const [shake, setShake] = useState(false)
  const [loading, setLoading] = useState(false)
  const [settingsReady, setSettingsReady] = useState(false)
  const settingsPromise = useRef<Promise<boolean>>()
  const [form, setForm] = useState<LoginRequest>(emptyLoginForm)
  const [password, setPassword] = useState('')
  const [captchaImage, setCaptchaImage] = useState('')
  const [captchaType, setCaptchaType] = useState('')
  const [behaviorVisible, setBehaviorVisible] = useState(false)
  const [behaviorLoading, setBehaviorLoading] = useState(false)
  const [behaviorData, setBehaviorData] = useState<BehaviorCaptchaData>({ image: '', thumb: '' })
  const [behaviorSource, setBehaviorSource] = useState({ width: 300, height: 220 })
  const [miniBinding, setMiniBinding] = useState(false)
  const [miniCaptchaImage, setMiniCaptchaImage] = useState('')
  const [miniForm, setMiniForm] = useState(emptyLoginForm)
  const [miniPassword, setMiniPassword] = useState('')
  const [mfaVisible, setMfaVisible] = useState(false)
  const [mfaLoading, setMfaLoading] = useState(false)
  const [mfaChallengeId, setMfaChallengeId] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [mfaRecoveryCode, setMfaRecoveryCode] = useState('')
  const [mfaMethod, setMfaMethod] = useState('totp')
  const [mfaWebAuthnOptionsJson, setMfaWebAuthnOptionsJson] = useState('')
  const [mfaSetupVisible, setMfaSetupVisible] = useState(false)
  const [mfaSetupTicket, setMfaSetupTicket] = useState('')
  const [mfaSetupUri, setMfaSetupUri] = useState('')
  const [mfaSetupCode, setMfaSetupCode] = useState('')
  const [mfaSetupMethod, setMfaSetupMethod] = useState('totp')
  const [mfaSetupWebAuthnOptionsJson, setMfaSetupWebAuthnOptionsJson] = useState('')
  const [mfaRecoveryCodesVisible, setMfaRecoveryCodesVisible] = useState(false)
  const [mfaRecoveryCodes, setMfaRecoveryCodes] = useState<string[]>([])

  const currentLanguageName =
    languageOptions.find((item) => item.language_code === locale)?.native_name || locale
  const mainTitle = settings?.get('mainTitle') || t('core.home.main_title')
  const subTitle = settings?.get('subTitle') || t('core.login.default_sub_title')
  const appLogo = settings?.get('appLogo') || defaultLogo
  const showTenantCode = settings?.get('showTenantCode') !== 'false'
  const configuredCaptchaType = settings?.get('captchaType') || ''
  const isBehaviorCaptcha = behaviorCaptchaTypes.has(captchaType)
  const behaviorWidth = Math.round(
    Math.min(320, Math.max(280, (Taro.getSystemInfoSync().windowWidth || 375) * 0.86)),
  )
  const behaviorHeight = Math.round((behaviorSource.height * behaviorWidth) / behaviorSource.width)
  const behaviorConfig = useMemo(() => {
    if (captchaType === 'rotate') {
      const size = Math.round(Math.min(300, behaviorWidth * 0.94))
      return {
        width: behaviorWidth,
        height: size,
        size,
        showTheme: false,
        verticalPadding: 0,
        horizontalPadding: 0,
        iconSize: 20,
        title: t('core.login.behavior_rotate'),
      }
    }
    return {
      width: behaviorWidth,
      height: behaviorHeight,
      thumbWidth: behaviorData.thumbWidth ?? 60,
      thumbHeight: behaviorData.thumbHeight ?? 60,
      showTheme: false,
      verticalPadding: 0,
      horizontalPadding: 0,
      buttonText: t('common.action.confirm'),
      iconSize: 20,
      dotSize: 20,
      title:
        captchaType === 'click' ? t('core.login.behavior_click') : t('core.login.behavior_puzzle'),
    }
  }, [
    behaviorData.thumbHeight,
    behaviorData.thumbWidth,
    behaviorHeight,
    behaviorWidth,
    captchaType,
    t,
  ]) as unknown as Record<string, string | number | boolean>

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('common.action.login') })
  }, [locale, t])

  const updateForm = (key: keyof LoginRequest, value: string) => {
    setForm((current) => ({ ...current, [key]: value }))
  }
  const updateMiniForm = (key: keyof LoginRequest, value: string) => {
    setMiniForm((current) => ({ ...current, [key]: value }))
  }

  const refreshCaptcha = async (requestedType = captchaType || configuredCaptchaType) => {
    if (!requestedType) throw new Error(t('core.login.captcha_type_missing'))
    const data = await defLoginService.Captcha({ type: requestedType })
    if (!data.type) throw new Error(t('core.login.captcha_type_response_missing'))
    setCaptchaType(data.type)
    setForm((current) => ({ ...current, captcha_id: data.captcha_id, captcha_code: '' }))
    if (!behaviorCaptchaTypes.has(data.type)) {
      setBehaviorVisible(false)
      setCaptchaImage(data.captcha_base64)
      return true
    }
    const payload = JSON.parse(data.captcha_base64 || '{}') as BehaviorPayload
    const source = {
      width: payload.width || 300,
      height: payload.height || (data.type === 'rotate' ? 300 : 220),
    }
    const displayHeight = Math.round((source.height * behaviorWidth) / source.width)
    const scaleX = behaviorWidth / source.width
    const scaleY = displayHeight / source.height
    setBehaviorSource(source)
    setCaptchaImage('')
    setBehaviorData({
      image: payload.image || '',
      thumb: payload.thumb || '',
      thumbX: Math.round((payload.thumbX ?? 0) * scaleX),
      thumbY: Math.round((payload.thumbY ?? 0) * scaleY),
      thumbWidth: Math.round((payload.thumbWidth ?? (data.type === 'click' ? 180 : 60)) * scaleX),
      thumbHeight: Math.round((payload.thumbHeight ?? (data.type === 'click' ? 48 : 60)) * scaleY),
      thumbSize: Math.round((payload.thumbSize ?? 150) * (behaviorWidth / source.width)),
      angle: 0,
    })
    return true
  }

  const refreshMiniCaptcha = async () => {
    try {
      const captcha = await defLoginService.Captcha({ type: 'digit' })
      setMiniCaptchaImage(await resolveMiniCaptchaImage(captcha.captcha_base64, captcha.captcha_id))
      setMiniForm((current) => ({
        ...current,
        captcha_id: captcha.captcha_id,
        captcha_code: '',
      }))
    } catch {
      await Taro.showToast({ icon: 'none', title: t('core.login.captcha_load_failed') })
    }
  }

  const loadLoginSettings = (): Promise<boolean> => {
    if (settingsReady) return Promise.resolve(true)
    if (settingsPromise.current) return settingsPromise.current
    settingsPromise.current = (async () => {
      try {
        await loadSettings()
        const nextCaptchaType = useSettingStore.getState().getData('captchaType')
        if (process.env.TARO_ENV === 'h5') {
          if (!nextCaptchaType) throw new Error(t('core.login.captcha_type_missing'))
          setCaptchaType(nextCaptchaType)
          if (!behaviorCaptchaTypes.has(nextCaptchaType)) await refreshCaptcha(nextCaptchaType)
        } else {
          await refreshMiniCaptcha()
        }
        setSettingsReady(true)
        return true
      } catch (error) {
        await Taro.showToast({
          icon: 'none',
          title: error instanceof Error ? error.message : t('core.protocol.load_failed'),
        })
        settingsPromise.current = undefined
        return false
      }
    })()
    return settingsPromise.current
  }

  useLoad(() => {
    void loadLoginSettings()
  })

  const checkedAgreePrivacy = async (): Promise<boolean> => {
    if (!(await loadLoginSettings())) return false
    const currentSettings = useSettingStore.getState()
    if (
      !currentSettings.getData('serviceProtocol') ||
      !currentSettings.getData('privacyProtocol')
    ) {
      await Taro.showToast({ icon: 'none', title: t('core.login.protocol_missing') })
      return false
    }
    if (agreed) return true
    setShake(true)
    setTimeout(() => setShake(false), 500)
    const result = await Taro.showModal({
      title: t('common.title.notice'),
      content: t('core.login.protocol_prompt'),
      confirmText: t('common.action.confirm'),
      cancelText: t('common.action.cancel'),
    })
    if (result.confirm) setAgreed(true)
    return result.confirm
  }

  /** 完成用户资料加载并恢复登录前页面。 */
  const loginSuccess = async () => {
    await userStore.getUserProfile()
    await Taro.showToast({ icon: 'success', title: t('core.login.login_success') })
    await new Promise<void>((resolve) => setTimeout(resolve, 500))
    await restoreLoginRedirect()
  }

  const validateLoginForm = async () => {
    if (!(await loadLoginSettings())) return false
    const validations: Array<[boolean, string]> = [
      [!showTenantCode || Boolean(form.tenant_code), t('core.login.tenant')],
      [Boolean(form.user_name), t('core.login.user_name')],
      [Boolean(password), t('core.login.password')],
      [isBehaviorCaptcha || Boolean(form.captcha_code), t('core.login.captcha')],
    ]
    const failed = validations.find(([valid]) => !valid)
    if (failed) {
      await Taro.showToast({ icon: 'none', title: failed[1] })
      return false
    }
    return checkedAgreePrivacy()
  }

  const beginMfaSetup = async (ticket: string) => {
    setMfaLoading(true)
    try {
      const response = await defMfaService.BeginMfaEnrollment({ setup_ticket: ticket })
      setMfaSetupTicket(response.setup_ticket)
      setMfaSetupUri(response.otpauth_uri)
      setMfaSetupCode('')
      setMfaSetupMethod(response.method || 'totp')
      setMfaSetupWebAuthnOptionsJson(response.webauthn_options_json || '')
      setMfaSetupVisible(true)
    } finally {
      setMfaLoading(false)
    }
  }

  const handleLoginResponse = async (response: LoginResponse): Promise<boolean> => {
    if (response.status === LoginStatus.LOGIN_STATUS_MFA_REQUIRED) {
      setMfaChallengeId(response.mfa_challenge_id)
      setMfaCode('')
      setMfaRecoveryCode('')
      setMfaMethod(response.mfa_method || 'totp')
      setMfaWebAuthnOptionsJson(response.mfa_webauthn_options_json || '')
      setMfaVisible(true)
      return false
    }
    if (response.status === LoginStatus.LOGIN_STATUS_MFA_ENROLLMENT_REQUIRED) {
      setMfaSetupTicket(response.mfa_setup_ticket)
      await beginMfaSetup(response.mfa_setup_ticket)
      return false
    }
    return true
  }

  const verifyMfaLogin = async () => {
    if (mfaLoading || !mfaChallengeId || (mfaMethod !== 'webauthn' && !mfaCode && !mfaRecoveryCode))
      return
    setMfaLoading(true)
    try {
      const webauthnResponseJson =
        mfaMethod === 'webauthn' && !mfaRecoveryCode
          ? await getWebAuthnAssertion(mfaWebAuthnOptionsJson)
          : ''
      const response = await userStore.verifyMfa({
        challenge_id: mfaChallengeId,
        code: mfaRecoveryCode ? '' : mfaCode,
        recovery_code: mfaRecoveryCode,
        webauthn_response_json: webauthnResponseJson,
      })
      setMfaVisible(false)
      if (await handleLoginResponse(response)) await loginSuccess()
    } catch (error) {
      await Taro.showToast({
        icon: 'none',
        title: getWechatErrorMessage(error, t('core.login.login_failed')),
      })
    } finally {
      setMfaLoading(false)
    }
  }

  const confirmMfaSetup = async () => {
    if (mfaLoading || !mfaSetupTicket || (mfaSetupMethod !== 'webauthn' && !mfaSetupCode)) return
    setMfaLoading(true)
    try {
      const webauthnResponseJson =
        mfaSetupMethod === 'webauthn'
          ? await createWebAuthnCredential(mfaSetupWebAuthnOptionsJson)
          : ''
      const response = await defMfaService.ConfirmMfaEnrollment({
        setup_ticket: mfaSetupTicket,
        code: mfaSetupCode,
        webauthn_response_json: webauthnResponseJson,
      })
      setMfaSetupVisible(false)
      setMfaRecoveryCodes(response.recovery_codes)
      setMfaRecoveryCodesVisible(true)
    } catch (error) {
      await Taro.showToast({
        icon: 'none',
        title: getWechatErrorMessage(error, t('core.login.login_failed')),
      })
    } finally {
      setMfaLoading(false)
    }
  }

  /** 确认已保存恢复码，刷新验证码并重新进入登录流程。 */
  const finishMfaEnrollment = async () => {
    setMfaRecoveryCodesVisible(false)
    setMfaRecoveryCodes([])
    await Taro.showToast({ icon: 'none', title: t('core.login.mfa_setup_success') })
    if (process.env.TARO_ENV === 'weapp') {
      // 小程序没有可复用的账号密码会话，绑定完成后重新走微信登录以进入 MFA 挑战。
      setMiniBinding(false)
      await wxLogin()
      return
    }
    await refreshCaptcha().catch(() => undefined)
  }

  const submitLogin = async (captchaCode: string) => {
    const encrypted = await encryptPassword(
      password,
      PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_LOGIN,
    )
    return userStore.login({
      ...form,
      tenant_code: showTenantCode ? form.tenant_code : '0000',
      password: encrypted,
      captcha_code: captchaCode,
    })
  }

  const onSubmit = async () => {
    if (!(await validateLoginForm()) || loading) return
    if (isBehaviorCaptcha) {
      setBehaviorVisible(true)
      setBehaviorLoading(true)
      try {
        await refreshCaptcha()
      } finally {
        setBehaviorLoading(false)
      }
      return
    }
    setLoading(true)
    try {
      const response = await submitLogin(form.captcha_code)
      if (await handleLoginResponse(response)) await loginSuccess()
    } catch (error) {
      await refreshCaptcha().catch(() => undefined)
      if (error instanceof Error) {
        await Taro.showToast({ icon: 'none', title: error.message || t('core.login.login_failed') })
      }
    } finally {
      setLoading(false)
    }
  }

  const verifyBehaviorCaptcha = async (
    value: BehaviorCaptchaPoint | BehaviorCaptchaPoint[] | number,
    reset: () => void,
  ) => {
    if (behaviorLoading) return
    setBehaviorLoading(true)
    try {
      let captchaCode = String(value)
      if (captchaType === 'click') {
        captchaCode = JSON.stringify(
          (value as BehaviorCaptchaPoint[]).map((dot) => ({
            x: Math.round(dot.x * (behaviorSource.width / behaviorWidth)),
            y: Math.round(dot.y * (behaviorSource.height / behaviorHeight)),
          })),
        )
      } else if (captchaType === 'slide') {
        captchaCode = String(
          Math.round((value as BehaviorCaptchaPoint).x * (behaviorSource.width / behaviorWidth)),
        )
      } else {
        captchaCode = String(Math.round(value as number))
      }
      const verified = await defLoginService.VerifyCaptcha({
        captcha_id: form.captcha_id,
        captcha_code: captchaCode,
      })
      setBehaviorVisible(false)
      const response = await submitLogin(verified.captcha_token)
      if (await handleLoginResponse(response)) await loginSuccess()
    } catch {
      reset()
      await refreshCaptcha().catch(() => undefined)
    } finally {
      setBehaviorLoading(false)
    }
  }

  const wxLogin = async () => {
    if (!(await checkedAgreePrivacy()) || loading) return
    setLoading(true)
    try {
      const code = (await Taro.login()).code
      const session = await userStore.createOauthSession({ provider: wechatMiniProvider, code })
      if (session.binding_required) {
        setMiniBinding(true)
        await refreshMiniCaptcha()
        return
      }
      if (await handleLoginResponse(session)) await loginSuccess()
    } catch (error) {
      if (isWechatUnboundError(error)) {
        setMiniBinding(true)
        await refreshMiniCaptcha()
      } else {
        await Taro.showToast({
          icon: 'none',
          title: getWechatErrorMessage(error, t('core.login.wechat_failed')),
        })
      }
    } finally {
      setLoading(false)
    }
  }

  const bindMiniAccount = async () => {
    if (loading || !(await checkedAgreePrivacy())) return
    const failed = [
      [showTenantCode && !miniForm.tenant_code, t('core.login.tenant')],
      [!miniForm.user_name || !miniPassword, t('core.login.user_name_password')],
      [!miniForm.captcha_id || !miniForm.captcha_code, t('core.login.captcha')],
    ].find(([invalid]) => invalid)
    if (failed) {
      await Taro.showToast({ icon: 'none', title: String(failed[1]) })
      return
    }
    setLoading(true)
    try {
      const encrypted = await encryptPassword(
        miniPassword,
        PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_LOGIN,
      )
      const code = (await Taro.login()).code
      const response = await userStore.bindOauthSession({
        provider: wechatMiniProvider,
        code,
        tenant_code: showTenantCode ? miniForm.tenant_code : '0000',
        user_name: miniForm.user_name,
        password: encrypted,
        captcha_code: miniForm.captcha_code,
        captcha_id: miniForm.captcha_id,
      })
      if (await handleLoginResponse(response)) await loginSuccess()
    } catch (error) {
      await refreshMiniCaptcha()
      await Taro.showToast({
        icon: 'none',
        title: getWechatErrorMessage(error, t('core.login.login_failed')),
      })
    } finally {
      setLoading(false)
    }
  }

  const agreement = (
    <View className={`login-tips${shake ? ' login-tips--shake' : ''}`}>
      <View className='login-agreement' onClick={() => setAgreed((value) => !value)}>
        <View className={`login-agree-icon${agreed ? ' checked' : ''}`} />
        <Text className='login-agree-desc'>{t('core.login.agree_prefix')}</Text>
        <Text
          className='login-agree-link'
          onClick={(event) => {
            event.stopPropagation()
            navigateAppRoute('app/protocol/service')
          }}
        >
          {t('core.login.service')}
        </Text>
        <Text className='login-agree-separator'>{t('core.login.agree_separator')}</Text>
        <Text
          className='login-agree-link'
          onClick={(event) => {
            event.stopPropagation()
            navigateAppRoute('app/protocol/privacy')
          }}
        >
          {t('core.login.privacy')}
        </Text>
      </View>
    </View>
  )

  return (
    <View className='login-page'>
      <Picker
        className='login-locale'
        mode='selector'
        range={languageOptions.map((item) => item.native_name)}
        value={Math.max(
          0,
          languageOptions.findIndex((item) => item.language_code === locale),
        )}
        onChange={(event) => {
          const nextLocale = languageOptions[Number(event.detail.value)]?.language_code as
            | SupportedLocale
            | undefined
          if (nextLocale) void setLocale(nextLocale)
        }}
      >
        <View className='login-locale__value'>{currentLanguageName}</View>
      </Picker>
      <View className='login-hero'>
        <View className='login-logo-shell'>
          <Image className='login-logo' src={appLogo} />
        </View>
        <View className='login-hero-copy'>
          <Text className='login-title'>{mainTitle}</Text>
          <Text className='login-subtitle'>{subTitle}</Text>
        </View>
      </View>
      <View className='login-panel'>
        {process.env.TARO_ENV === 'h5' ? (
          <View className='login-form'>
            {showTenantCode ? (
              <Input
                className='login-input'
                placeholder={t('core.login.tenant')}
                value={form.tenant_code}
                onInput={(event) => updateForm('tenant_code', event.detail.value)}
              />
            ) : null}
            <Input
              className='login-input'
              placeholder={t('core.login.user_name_mobile')}
              value={form.user_name}
              onInput={(event) => updateForm('user_name', event.detail.value)}
            />
            <Input
              className='login-input'
              password
              placeholder={t('core.login.password')}
              value={password}
              onInput={(event) => setPassword(event.detail.value)}
              onConfirm={() => void onSubmit()}
            />
            {!isBehaviorCaptcha ? (
              <View className='captcha-row'>
                <Input
                  className='login-input captcha-input'
                  placeholder={t('core.login.captcha')}
                  value={form.captcha_code}
                  onInput={(event) => updateForm('captcha_code', event.detail.value)}
                  onConfirm={() => void onSubmit()}
                />
                <View className='captcha-divider' />
                <View className='captcha-trigger' onClick={() => void refreshCaptcha()}>
                  <Image className='captcha-image' src={captchaImage} mode='aspectFit' />
                </View>
              </View>
            ) : null}
            <Button
              className='login-button login-button-primary'
              loading={loading}
              onClick={() => void onSubmit()}
            >
              {t('common.action.login')}
            </Button>
            {behaviorVisible ? (
              <View className='login-behavior-mask'>
                <View className='login-behavior-panel'>
                  {behaviorLoading ? (
                    <View className='login-behavior-loading'>
                      {t('core.login.behavior_loading')}
                    </View>
                  ) : null}
                  <BehaviorCaptcha
                    type={captchaType}
                    data={behaviorData}
                    config={behaviorConfig}
                    onConfirm={(value, reset) => void verifyBehaviorCaptcha(value, reset)}
                    onRefresh={() => void refreshCaptcha()}
                    onClose={() => setBehaviorVisible(false)}
                  />
                </View>
              </View>
            ) : null}
          </View>
        ) : !miniBinding ? (
          <Button
            className='login-button login-button-primary'
            loading={loading}
            onClick={() => void wxLogin()}
          >
            <Text className='login-phone-icon'>☎</Text>
            {t('core.login.wechat')}
          </Button>
        ) : (
          <View className='login-form'>
            <View className='login-bind-tip'>{t('core.login.wechat_bind_tip')}</View>
            {showTenantCode ? (
              <Input
                className='login-input'
                placeholder={t('core.login.tenant')}
                value={miniForm.tenant_code}
                onInput={(event) => updateMiniForm('tenant_code', event.detail.value)}
              />
            ) : null}
            <Input
              className='login-input'
              placeholder={t('core.login.user_name_mobile')}
              value={miniForm.user_name}
              onInput={(event) => updateMiniForm('user_name', event.detail.value)}
            />
            <Input
              className='login-input'
              password
              placeholder={t('core.login.password')}
              value={miniPassword}
              onInput={(event) => setMiniPassword(event.detail.value)}
            />
            <View className='captcha-row'>
              <Input
                className='login-input captcha-input'
                placeholder={t('core.login.captcha')}
                value={miniForm.captcha_code}
                onInput={(event) => updateMiniForm('captcha_code', event.detail.value)}
              />
              <View className='captcha-divider' />
              <View className='captcha-trigger' onClick={() => void refreshMiniCaptcha()}>
                <Image className='captcha-image' src={miniCaptchaImage} mode='aspectFit' />
              </View>
            </View>
            <Button
              className='login-button login-button-primary'
              loading={loading}
              onClick={() => void bindMiniAccount()}
            >
              {t('core.login.bind_wechat')}
            </Button>
          </View>
        )}
        {mfaVisible ? (
          <View className='mfa-mask'>
            <View className='mfa-panel'>
              <Text className='mfa-title'>{t('core.login.mfa_title')}</Text>
              {mfaMethod !== 'webauthn' ? (
                <Input
                  className='login-input'
                  type='number'
                  maxlength={8}
                  value={mfaCode}
                  placeholder={t('core.login.mfa_code')}
                  onInput={(event) => setMfaCode(event.detail.value)}
                />
              ) : null}
              <Input
                className='login-input'
                value={mfaRecoveryCode}
                placeholder={t('core.login.mfa_recovery_code')}
                onInput={(event) => setMfaRecoveryCode(event.detail.value)}
              />
              <Button
                className='login-button login-button-primary'
                loading={mfaLoading}
                onClick={() => void verifyMfaLogin()}
              >
                {mfaMethod === 'webauthn' && !mfaRecoveryCode
                  ? t('core.login.mfa_webauthn_action')
                  : t('common.action.confirm')}
              </Button>
            </View>
          </View>
        ) : null}
        {mfaSetupVisible ? (
          <View className='mfa-mask'>
            <View className='mfa-panel'>
              <Text className='mfa-title'>{t('core.login.mfa_setup_title')}</Text>
              {mfaSetupMethod !== 'webauthn' ? (
                <>
                  <MfaSetupPanel uri={mfaSetupUri} />
                  <Input
                    className='login-input'
                    type='number'
                    maxlength={8}
                    value={mfaSetupCode}
                    placeholder={t('core.login.mfa_code')}
                    onInput={(event) => setMfaSetupCode(event.detail.value)}
                  />
                </>
              ) : null}
              <Button
                className='login-button login-button-primary'
                loading={mfaLoading}
                onClick={() => void confirmMfaSetup()}
              >
                {mfaSetupMethod === 'webauthn'
                  ? t('core.login.mfa_webauthn_action')
                  : t('common.action.confirm')}
              </Button>
            </View>
          </View>
        ) : null}
        <MfaRecoveryCodesDialog
          open={mfaRecoveryCodesVisible}
          codes={mfaRecoveryCodes}
          onConfirm={finishMfaEnrollment}
        />
        {agreement}
      </View>
    </View>
  )
}
