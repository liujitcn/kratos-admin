<script setup lang="ts">
import { useSettingStore, useThemeStore, useUserStore } from '../../../stores'
import type { LoginRequest } from '../../../rpc/base/v1/login'
import { LoginStatus, type LoginResponse } from '../../../rpc/base/v1/login'
import { onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { computed, defineAsyncComponent, nextTick, reactive, ref, watch } from 'vue'
import { defLoginService } from '../../../api/base/v1/login'
import { defMfaService } from '../../../api/base/v1/mfa'
import defaultLogo from '../../../static/images/logo_icon.png'
import { formatSrc } from '../../../utils'
import { prepareLoginPage, restoreLoginRedirect, returnFromLogin } from '../../../utils/navigation'
import { encryptPassword } from '../../../utils/secretCrypto'
import { createWebAuthnCredential, getWebAuthnAssertion } from '../../../utils/webauthn'
import { navigateAppRoute } from '../../../navigation'
import { getLanguageOptions, useI18n, type SupportedLocale } from '../../../locales'
import MfaRecoveryCodesDialog from '../../../components/MfaRecoveryCodesDialog.vue'
import MfaSetupPanel from '../../../components/MfaSetupPanel.vue'

// go-captcha-uni 仅在 H5 端使用，小程序端不支持 defineAsyncComponent，需按平台裁剪。
// #ifdef H5
const GoCaptchaUni = defineAsyncComponent(() => import('go-captcha-uni'))
// #endif

const userStore = useUserStore()
const settingStore = useSettingStore()
const themeStore = useThemeStore()
const { locale, setLocale, t } = useI18n()
const localeOptions = computed(() => getLanguageOptions())
const localeLabels = computed(() => localeOptions.value.map((item) => item.native_name))
const localeIndex = computed(() =>
  Math.max(
    0,
    localeOptions.value.findIndex((item) => item.language_code === locale.value),
  ),
)
const wechatMiniProvider = 'wechatmini'

/** 参考管理端 View、Hide 图标视觉的本端 SVG 路径。 */
const passwordIconPaths = {
  view: 'M512 160c320 0 512 352 512 352S832 864 512 864 0 512 0 512s192-352 512-352m0 64c-225.28 0-384.128 208.064-436.8 288 52.608 79.872 211.456 288 436.8 288 225.28 0 384.128-208.064 436.8-288-52.608-79.872-211.456-288-436.8-288m0 64a224 224 0 1 1 0 448 224 224 0 0 1 0-448m0 64a160.19 160.19 0 0 0-160 160c0 88.192 71.744 160 160 160s160-71.808 160-160-71.744-160-160-160',
  hide: 'M876.8 156.8c0-9.6-3.2-16-9.6-22.4s-12.8-9.6-22.4-9.6-16 3.2-22.4 9.6L736 220.8c-64-32-137.6-51.2-224-60.8-160 16-288 73.6-377.6 176S0 496 0 512s48 73.6 134.4 176c22.4 25.6 44.8 48 73.6 67.2l-86.4 89.6c-6.4 6.4-9.6 12.8-9.6 22.4s3.2 16 9.6 22.4 12.8 9.6 22.4 9.6 16-3.2 22.4-9.6l704-710.4c3.2-6.4 6.4-12.8 6.4-22.4m-646.4 528Q115.2 579.2 76.8 512q43.2-72 153.6-172.8C304 272 400 230.4 512 224c64 3.2 124.8 19.2 176 44.8l-54.4 54.4C598.4 300.8 560 288 512 288c-64 0-115.2 22.4-160 64s-64 96-64 160c0 48 12.8 89.6 35.2 124.8L256 707.2c-9.6-6.4-19.2-16-25.6-22.4m140.8-96Q352 555.2 352 512c0-44.8 16-83.2 48-112s67.2-48 112-48c28.8 0 54.4 6.4 73.6 19.2zM889.599 336c-12.8-16-28.8-28.8-41.6-41.6l-48 48c73.6 67.2 124.8 124.8 150.4 169.6q-43.2 72-153.6 172.8c-73.6 67.2-172.8 108.8-284.8 115.2-51.2-3.2-99.2-12.8-140.8-28.8l-48 48c57.6 22.4 118.4 38.4 188.8 44.8 160-16 288-73.6 377.6-176S1024 528 1024 512s-48.001-73.6-134.401-176M511.998 672c-12.8 0-25.6-3.2-38.4-6.4l-51.2 51.2c28.8 12.8 57.6 19.2 89.6 19.2 64 0 115.2-22.4 160-64 41.6-41.6 64-96 64-160 0-32-6.4-64-19.2-89.6l-51.2 51.2c3.2 12.8 6.4 25.6 6.4 38.4 0 44.8-16 83.2-48 112s-67.2 48-112 48',
} as const
const loginSettingsReady = ref(false)
let loginSettingsPromise: Promise<boolean> | undefined

const configuredMainTitle = computed(
  () => settingStore.getData('mainTitle') || t('core.home.main_title'),
)
const configuredSubTitle = computed(
  () => settingStore.getData('subTitle') || t('core.login.default_sub_title'),
)
const configuredAppLogo = computed(() => {
  const configuredLogo = settingStore.getData('appLogo')
  return configuredLogo ? formatSrc(configuredLogo) : defaultLogo
})
const appLogo = ref(defaultLogo)
const showTenantCode = computed(() => settingStore.getData('showTenantCode') !== 'false')
/** 按管理端 View、Hide 视觉生成本端密码可见性图标。 */
const passwordIconSrc = (visible: boolean) => {
  const color = themeStore.colors.textMuted
  const path = visible ? passwordIconPaths.hide : passwordIconPaths.view
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" fill="${color}"><path d="${path}"/></svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

watch(
  configuredAppLogo,
  (value) => {
    appLogo.value = value
  },
  { immediate: true },
)

/** Logo 加载失败时回退到本地默认图片。 */
const handleLogoError = () => {
  if (appLogo.value !== defaultLogo) {
    appLogo.value = defaultLogo
  }
}

// 是否同意协议
const isAgreePrivacy = ref(false)
const isAgreePrivacyShakeY = ref(false)
const loading = ref(false)
const mfaVisible = ref(false)
const mfaLoading = ref(false)
const mfaChallengeId = ref('')
const mfaCode = ref('')
const mfaRecoveryCode = ref('')
const mfaMethod = ref('totp')
const mfaRememberDays = ref(0)
const rememberMfaDevice = ref(false)
const mfaWebAuthnOptionsJson = ref('')
const mfaSetupVisible = ref(false)
const mfaSetupTicket = ref('')
const mfaSetupUri = ref('')
const mfaSetupCode = ref('')
const mfaSetupMethod = ref('totp')
const mfaSetupWebAuthnOptionsJson = ref('')
const mfaRecoveryCodesVisible = ref(false)
const mfaRecoveryCodes = ref<string[]>([])

/** 切换登录页语言。 */
const onLocaleChange = (event: { detail: { value: string | number } }) => {
  const nextLocale = localeOptions.value[Number(event.detail.value)]?.language_code as
    | SupportedLocale
    | undefined
  if (nextLocale) void setLocale(nextLocale)
}

const toggleAgreePrivacy = () => {
  isAgreePrivacy.value = !isAgreePrivacy.value
}

let agreePrivacyShakeTimer: ReturnType<typeof setTimeout> | undefined
let loginSuccessTimer: ReturnType<typeof setTimeout> | undefined

const triggerAgreePrivacyShake = () => {
  isAgreePrivacyShakeY.value = true
  if (agreePrivacyShakeTimer) clearTimeout(agreePrivacyShakeTimer)
  agreePrivacyShakeTimer = setTimeout(() => {
    isAgreePrivacyShakeY.value = false
    agreePrivacyShakeTimer = undefined
  }, 500)
}

// 打开服务条款
const onOpenServiceProtocol = () => {
  navigateAppRoute('app/protocol/service')
}

// 打开隐私协议
const onOpenPrivacyContract = () => {
  navigateAppRoute('app/protocol/privacy')
}

/** 提取请求失败时后端返回的可展示消息。 */
const getWechatErrorMessage = (error: unknown, fallback: string) => {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (!error || typeof error !== 'object') {
    return fallback
  }
  const response = error as {
    data?: {
      message?: string
      error?: { message?: string }
    }
  }
  return response.data?.message || response.data?.error?.message || fallback
}

// #ifdef MP-WEIXIN
const miniBinding = ref(false)
const miniCaptchaImage = ref('')
const miniForm = reactive({
  tenant_code: '',
  user_name: '',
  captcha_code: '',
  captcha_id: '',
})
const miniPasswordValue = ref('')
const miniPasswordVisible = ref(false)

type MiniFileSystemManager = {
  writeFile(options: {
    filePath: string
    data: string
    encoding: 'base64'
    success: () => void
    fail: () => void
  }): void
}

type MiniWxRuntime = typeof wx & {
  env: { USER_DATA_PATH: string }
  getFileSystemManager: () => MiniFileSystemManager
}

/** 将接口返回的验证码 Data URI 转为小程序可渲染的临时文件。 */
const resolveMiniCaptchaImage = (payload: string, captchaId: string) => {
  if (!payload.startsWith('data:image/')) {
    return Promise.resolve(payload)
  }
  const commaIndex = payload.indexOf(',')
  if (commaIndex < 0) {
    return Promise.resolve(payload)
  }
  const wxRuntime = wx as MiniWxRuntime
  // 每次使用不同路径，避免微信小程序复用同一 image src 的缓存。
  const filePath = `${wxRuntime.env.USER_DATA_PATH}/kratos-login-captcha-${captchaId}.png`
  return new Promise<string>((resolve, reject) => {
    wxRuntime.getFileSystemManager().writeFile({
      filePath,
      data: payload.slice(commaIndex + 1),
      encoding: 'base64',
      success: () => resolve(filePath),
      fail: () => reject(new Error(t('core.login.captcha_write_failed'))),
    })
  })
}

/** 加载小程序绑定所需的数字验证码。 */
const refreshMiniCaptcha = async () => {
  try {
    const captcha = await defLoginService.Captcha({ type: 'digit' })
    miniCaptchaImage.value = await resolveMiniCaptchaImage(
      captcha.captcha_base64,
      captcha.captcha_id,
    )
    miniForm.captcha_id = captcha.captcha_id
    miniForm.captcha_code = ''
  } catch {
    await uni.showToast({ icon: 'none', title: t('core.login.captcha_load_failed') })
  }
}

/** 仅将后端明确返回的未绑定错误转为账号绑定流程。 */
const isWechatUnboundError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const response = error as { data?: { binding_required?: boolean } }
  return response.data?.binding_required === true
}

// 微信授权失败时保留请求层提示，避免异步登录异常冒泡到小程序调试器。
const wxLogin = async () => {
  const isAgreed = await checkedAgreePrivacy()
  if (!isAgreed) {
    return
  }
  if (loading.value) {
    return
  }
  loading.value = true
  let loginCode = ''
  try {
    loginCode = (await wx.login()).code
  } catch {
    await uni.showToast({ icon: 'none', title: t('core.login.wechat_failed') })
    loading.value = false
    return
  }
  try {
    const session = await userStore.createOauthSession({
      provider: wechatMiniProvider,
      code: loginCode,
    })
    if (session.binding_required) {
      miniBinding.value = true
      void refreshMiniCaptcha()
      return
    }
    if (await handleLoginResponse(session)) await loginSuccess()
  } catch (error) {
    if (!isWechatUnboundError(error)) {
      await uni.showToast({
        icon: 'none',
        title: getWechatErrorMessage(error, t('core.login.wechat_failed')),
      })
      return
    }
    miniBinding.value = true
    void refreshMiniCaptcha()
  } finally {
    loading.value = false
  }
}

// 提交已有账号并绑定当前微信。授权码在每次调用后失效，绑定时重新获取。
const bindMiniAccount = async () => {
  if (loading.value) return
  if (!(await checkedAgreePrivacy())) return
  if (showTenantCode.value && !miniForm.tenant_code) {
    await uni.showToast({ icon: 'none', title: t('core.login.tenant') })
    return
  }
  if (!miniForm.user_name || !miniPasswordValue.value) {
    await uni.showToast({ icon: 'none', title: t('core.login.user_name_password') })
    return
  }
  if (!miniForm.captcha_id || !miniForm.captcha_code) {
    await uni.showToast({ icon: 'none', title: t('core.login.captcha') })
    return
  }
  loading.value = true
  try {
    const password = await encryptPassword(miniPasswordValue.value)
    const code = (await wx.login()).code
    const response = await userStore.bindOauthSession({
      provider: wechatMiniProvider,
      code,
      tenant_code: showTenantCode.value ? miniForm.tenant_code : '0000',
      user_name: miniForm.user_name,
      password,
      captcha_code: miniForm.captcha_code,
      captcha_id: miniForm.captcha_id,
    })
    if (await handleLoginResponse(response)) await loginSuccess()
  } catch (error) {
    await refreshMiniCaptcha()
    await uni.showToast({
      icon: 'none',
      title: getWechatErrorMessage(error, t('core.login.login_failed')),
    })
  } finally {
    loading.value = false
  }
}
// #endif

// 处理登录接口返回的 MFA 状态。
const handleLoginResponse = async (response: LoginResponse) => {
  if (response.status === LoginStatus.LOGIN_STATUS_MFA_REQUIRED) {
    mfaChallengeId.value = response.mfa_challenge_id
    mfaCode.value = ''
    mfaRecoveryCode.value = ''
    mfaMethod.value = response.mfa_method || 'totp'
    mfaRememberDays.value = response.mfa_remember_days || 0
    rememberMfaDevice.value = false
    mfaWebAuthnOptionsJson.value = response.mfa_webauthn_options_json || ''
    mfaVisible.value = true
    return false
  }
  if (response.status === LoginStatus.LOGIN_STATUS_MFA_ENROLLMENT_REQUIRED) {
    mfaSetupTicket.value = response.mfa_setup_ticket
    mfaSetupMethod.value = response.mfa_method || 'totp'
    await beginMfaSetup(response.mfa_setup_ticket)
    return false
  }
  return true
}

// 校验登录阶段 MFA。
const verifyMfaLogin = async () => {
  if (
    mfaLoading.value ||
    !mfaChallengeId.value ||
    (mfaMethod.value !== 'webauthn' && !mfaCode.value && !mfaRecoveryCode.value)
  )
    return
  mfaLoading.value = true
  try {
    const webauthnResponseJson =
      mfaMethod.value === 'webauthn' && !mfaRecoveryCode.value
        ? await getWebAuthnAssertion(mfaWebAuthnOptionsJson.value)
        : ''
    const response = await userStore.verifyMfa({
      challenge_id: mfaChallengeId.value,
      code: mfaRecoveryCode.value ? '' : mfaCode.value,
      recovery_code: mfaRecoveryCode.value,
      webauthn_response_json: webauthnResponseJson,
      remember_device: rememberMfaDevice.value,
    })
    mfaVisible.value = false
    if (await handleLoginResponse(response)) await loginSuccess()
  } catch (error) {
    await uni.showToast({
      icon: 'none',
      title: getWechatErrorMessage(error, t('core.login.login_failed')),
    })
  } finally {
    mfaLoading.value = false
  }
}

// 开始强制绑定 MFA。
const beginMfaSetup = async (ticket: string) => {
  mfaLoading.value = true
  try {
    const response = await defMfaService.BeginMfaEnrollment({ setup_ticket: ticket })
    mfaSetupTicket.value = response.setup_ticket
    mfaSetupUri.value = response.otpauth_uri
    mfaSetupCode.value = ''
    mfaSetupMethod.value = response.method || 'totp'
    mfaSetupWebAuthnOptionsJson.value = response.webauthn_options_json || ''
    mfaSetupVisible.value = true
  } finally {
    mfaLoading.value = false
  }
}

// 确认强制绑定 MFA。
const confirmMfaSetup = async () => {
  if (
    mfaLoading.value ||
    !mfaSetupTicket.value ||
    (mfaSetupMethod.value !== 'webauthn' && !mfaSetupCode.value)
  )
    return
  mfaLoading.value = true
  try {
    const webauthnResponseJson =
      mfaSetupMethod.value === 'webauthn'
        ? await createWebAuthnCredential(mfaSetupWebAuthnOptionsJson.value)
        : ''
    const response = await defMfaService.ConfirmMfaEnrollment({
      setup_ticket: mfaSetupTicket.value,
      code: mfaSetupCode.value,
      webauthn_response_json: webauthnResponseJson,
    })
    mfaSetupVisible.value = false
    mfaRecoveryCodes.value = response.recovery_codes
    mfaRecoveryCodesVisible.value = true
  } catch (error) {
    await uni.showToast({
      icon: 'none',
      title: getWechatErrorMessage(error, t('core.login.login_failed')),
    })
  } finally {
    mfaLoading.value = false
  }
}

/** 确认已保存恢复码，刷新验证码并重新进入登录流程。 */
const finishMfaEnrollment = async () => {
  mfaRecoveryCodesVisible.value = false
  await nextTick()
  mfaRecoveryCodes.value = []
  await uni.showToast({ icon: 'none', title: t('core.login.mfa_setup_success') })
  // #ifdef H5
  await loadPageCaptcha()
  // #endif
  // 小程序没有可复用的账号密码会话，绑定完成后重新走微信登录以进入 MFA 挑战。
  // #ifdef MP-WEIXIN
  miniBinding.value = false
  await wxLogin()
  // #endif
}

// #ifdef H5
const captcha_base64 = ref() // 验证码图片Base64字符串
const defaultCaptchaImageWidth = 170
const captchaImageWidth = ref(`${defaultCaptchaImageWidth}rpx`)
const behaviorDialogVisible = ref(false)
const behaviorLoading = ref(false)
type CaptchaImageLoadDetail = {
  detail: {
    width: number
    height: number
  }
}
type BehaviorCaptchaPayload = {
  type?: string
  image: string
  thumb: string
  width?: number
  height?: number
  thumbX?: number
  thumbY?: number
  thumbWidth?: number
  thumbHeight?: number
  thumbSize?: number
}
type CaptchaPoint = {
  x: number
  y: number
}
type ClickCaptchaPoint = CaptchaPoint & {
  key?: number
  index?: number
}
type BehaviorCaptchaData = {
  image: string
  thumb: string
  thumbX?: number
  thumbY?: number
  thumbWidth?: number
  thumbHeight?: number
  thumbSize?: number
  angle?: number
}
const behaviorCaptchaTypeSet = new Set(['slide', 'click', 'rotate'])
const configuredCaptchaType = computed(() => settingStore.getData('captchaType'))
const currentCaptchaType = ref('')
const isBehaviorCaptcha = computed(() => behaviorCaptchaTypeSet.has(currentCaptchaType.value))
const behaviorCaptchaData = reactive<BehaviorCaptchaData>({
  image: '',
  thumb: '',
})
const behaviorCaptchaWindowWidth = uni.getSystemInfoSync().windowWidth || 375
const behaviorCaptchaImageWidth = Math.round(
  Math.min(320, Math.max(280, behaviorCaptchaWindowWidth * 0.86)),
)
const rotateCaptchaImageSize = Math.round(Math.min(300, behaviorCaptchaImageWidth * 0.94))
const behaviorCaptchaSource = reactive({
  width: 300,
  height: 220,
  thumbWidth: 60,
  thumbHeight: 60,
  thumbSize: 150,
})
const behaviorCaptchaImageHeight = computed(() =>
  Math.round(
    (behaviorCaptchaSource.height * behaviorCaptchaImageWidth) / behaviorCaptchaSource.width,
  ),
)
const behaviorCaptchaScaleX = computed(
  () => behaviorCaptchaImageWidth / behaviorCaptchaSource.width,
)
const behaviorCaptchaScaleY = computed(
  () => behaviorCaptchaImageHeight.value / behaviorCaptchaSource.height,
)
const rotateCaptchaScale = computed(() => rotateCaptchaImageSize / behaviorCaptchaSource.width)

// 行为验证码按展示尺寸渲染，提交前再换算回服务端原始坐标。
const toDisplayCaptchaX = (value: number) => Math.round(value * behaviorCaptchaScaleX.value)
const toDisplayCaptchaY = (value: number) => Math.round(value * behaviorCaptchaScaleY.value)
const toDisplayRotateSize = (value: number) => Math.round(value * rotateCaptchaScale.value)
const toOriginalCaptchaX = (value: number) =>
  Math.min(
    behaviorCaptchaSource.width,
    Math.max(0, Math.round(value / behaviorCaptchaScaleX.value)),
  )
const toOriginalCaptchaY = (value: number) =>
  Math.min(
    behaviorCaptchaSource.height,
    Math.max(0, Math.round(value / behaviorCaptchaScaleY.value)),
  )
const behaviorCaptchaConfig = computed(() => {
  if (currentCaptchaType.value === 'rotate') {
    return {
      width: behaviorCaptchaImageWidth,
      height: rotateCaptchaImageSize,
      size: rotateCaptchaImageSize,
      showTheme: false,
      verticalPadding: 0,
      horizontalPadding: 0,
      iconSize: 20,
      title: t('core.login.behavior_rotate'),
    }
  }
  return {
    width: behaviorCaptchaImageWidth,
    height: behaviorCaptchaImageHeight.value,
    thumbWidth: toDisplayCaptchaX(behaviorCaptchaSource.thumbWidth),
    thumbHeight: toDisplayCaptchaY(behaviorCaptchaSource.thumbHeight),
    showTheme: false,
    verticalPadding: 0,
    horizontalPadding: 0,
    buttonText: t('common.action.confirm'),
    iconSize: 20,
    dotSize: 20,
    title:
      currentCaptchaType.value === 'click'
        ? t('core.login.behavior_click')
        : t('core.login.behavior_puzzle'),
  }
})
const behaviorCaptchaTheme = {
  textColor: '#1f2937',
  iconColor: '#4b5563',
  btnBgColor: '#28bb9c',
  btnBorderColor: '#28bb9c',
  activeColor: '#28bb9c',
  dragBarColor: '#dbe7e3',
  dragBgColor: '#28bb9c',
  dragIconColor: '#ffffff',
  loadingIconColor: '#28bb9c',
  bodyBgColor: '#f1f5f9',
  dotColor: '#ffffff',
  dotBgColor: 'rgba(40, 187, 156, 0.68)',
  dotBorderColor: 'rgba(255, 255, 255, 0.92)',
}
// 获取验证码
const getCaptcha = async () => {
  const requestedCaptchaType = configuredCaptchaType.value
  if (!requestedCaptchaType) {
    throw new Error(t('core.login.captcha_type_missing'))
  }
  const data = await defLoginService.Captcha({ type: requestedCaptchaType })
  if (!data.type) {
    throw new Error(t('core.login.captcha_type_response_missing'))
  }
  currentCaptchaType.value = data.type
  if (!isBehaviorCaptcha.value) {
    behaviorDialogVisible.value = false
  }
  form.value.captcha_id = data.captcha_id
  form.value.captcha_code = ''
  captchaImageWidth.value = `${defaultCaptchaImageWidth}rpx`
  captcha_base64.value = isBehaviorCaptcha.value ? '' : data.captcha_base64
  if (isBehaviorCaptcha.value) {
    applyBehaviorCaptchaPayload(data.captcha_base64)
  }
}

/** 刷新验证码并将接口错误转为页面提示。 */
const refreshCaptcha = async () => {
  try {
    await getCaptcha()
    return true
  } catch (error) {
    behaviorDialogVisible.value = false
    await uni.showToast({
      icon: 'none',
      title: error instanceof Error ? error.message : t('core.login.captcha_load_failed'),
    })
    return false
  }
}

/** 返回首页并结束本次登录回跳，避免上一页再次要求登录。 */
const goBack = () => {
  void returnFromLogin()
}

// 页面加载或普通表单刷新验证码，行为验证码延迟到登录弹窗打开时再请求。
const loadPageCaptcha = async () => {
  if (isBehaviorCaptcha.value) {
    form.value.captcha_id = ''
    form.value.captcha_code = ''
    captcha_base64.value = ''
    return
  }
  await getCaptcha()
}
// 解析行为验证码图片载荷并映射为官方组件数据。
const applyBehaviorCaptchaPayload = (payloadText: string) => {
  const payload = JSON.parse(payloadText || '{}') as BehaviorCaptchaPayload
  const payloadType = payload.type || currentCaptchaType.value
  behaviorCaptchaSource.width = payload.width || 300
  behaviorCaptchaSource.height = payload.height || (payloadType === 'rotate' ? 300 : 220)
  behaviorCaptchaSource.thumbWidth = payload.thumbWidth || (payloadType === 'click' ? 180 : 60)
  behaviorCaptchaSource.thumbHeight = payload.thumbHeight || (payloadType === 'click' ? 48 : 60)
  behaviorCaptchaSource.thumbSize = payload.thumbSize || 150
  behaviorCaptchaData.image = payload.image || ''
  behaviorCaptchaData.thumb = payload.thumb || ''
  behaviorCaptchaData.thumbX = toDisplayCaptchaX(payload.thumbX ?? 0)
  behaviorCaptchaData.thumbY = toDisplayCaptchaY(payload.thumbY ?? 0)
  behaviorCaptchaData.thumbWidth = toDisplayCaptchaX(behaviorCaptchaSource.thumbWidth)
  behaviorCaptchaData.thumbHeight = toDisplayCaptchaY(behaviorCaptchaSource.thumbHeight)
  behaviorCaptchaData.thumbSize = toDisplayRotateSize(behaviorCaptchaSource.thumbSize)
  behaviorCaptchaData.angle = 0
}
// 根据验证码图片原始比例更新展示宽度。
const handleCaptchaImageLoad = (event: Event) => {
  const { width, height } = (event as unknown as CaptchaImageLoadDetail).detail
  if (!width || !height) {
    return
  }
  // 验证码固定展示高度，宽度按图片比例自适应，避免算术验证码横向内容被裁剪。
  const imageWidth = Math.round((60 * width) / height)
  captchaImageWidth.value = `${Math.min(Math.max(imageWidth, defaultCaptchaImageWidth), 300)}rpx`
}
// 传统表单登录。
const form = ref<LoginRequest>({
  tenant_code: '',
  user_name: '',
  password: undefined,
  captcha_id: '',
  captcha_code: '',
})
const passwordValue = ref('')
const passwordVisible = ref(false)
// 校验账号密码与协议勾选状态。
const validateLoginForm = async () => {
  if (!(await ensureLoginSettings())) {
    return false
  }
  if (showTenantCode.value && !form.value.tenant_code) {
    await uni.showToast({
      icon: 'none',
      title: t('core.login.tenant'),
    })
    return false
  }
  if (!form.value.user_name) {
    await uni.showToast({
      icon: 'none',
      title: t('core.login.user_name'),
    })
    return false
  }
  if (!passwordValue.value) {
    await uni.showToast({
      icon: 'none',
      title: t('core.login.password'),
    })
    return false
  }
  if (!isBehaviorCaptcha.value && !form.value.captcha_code) {
    await uni.showToast({
      icon: 'none',
      title: t('core.login.captcha'),
    })
    return false
  }
  return checkedAgreePrivacy()
}
// 预校验行为验证码并返回可用于登录的一次性令牌。
const verifyCaptchaToken = async (captchaCode: string) => {
  const result = await defLoginService.VerifyCaptcha({
    captcha_id: form.value.captcha_id,
    captcha_code: captchaCode,
  })
  return result.captcha_token
}

// 执行真正的账号登录流程。
const submitLogin = async (captchaCode: string) => {
  const password = await encryptPassword(passwordValue.value)
  return userStore.login({
    ...form.value,
    tenant_code: showTenantCode.value ? form.value.tenant_code : '0000',
    password,
    captcha_code: captchaCode,
  })
}
// 打开行为验证码弹窗。
const openBehaviorCaptcha = async () => {
  behaviorDialogVisible.value = true
  behaviorLoading.value = true
  try {
    await refreshCaptcha()
  } finally {
    behaviorLoading.value = false
  }
}
// 关闭行为验证码弹窗。
const closeBehaviorCaptcha = () => {
  behaviorDialogVisible.value = false
}
// 验证行为验证码并继续登录。
const verifyBehaviorCaptcha = async (captchaCode: string, reset: () => void) => {
  if (behaviorLoading.value) {
    return
  }
  behaviorLoading.value = true
  try {
    const captchaToken = await verifyCaptchaToken(captchaCode)
    behaviorDialogVisible.value = false
    const response = await submitLogin(captchaToken)
    if (await handleLoginResponse(response)) await loginSuccess()
  } catch {
    reset()
    await refreshCaptcha()
  } finally {
    behaviorLoading.value = false
  }
}
// 行为验证码确认回调。
const onBehaviorConfirm = (
  value: CaptchaPoint | ClickCaptchaPoint[] | number,
  reset: () => void,
) => {
  if (currentCaptchaType.value === 'click') {
    const dots = value as ClickCaptchaPoint[]
    void verifyBehaviorCaptcha(
      JSON.stringify(
        dots.map((dot) => ({ x: toOriginalCaptchaX(dot.x), y: toOriginalCaptchaY(dot.y) })),
      ),
      reset,
    )
    return
  }
  if (currentCaptchaType.value === 'slide') {
    const point = value as CaptchaPoint
    void verifyBehaviorCaptcha(String(toOriginalCaptchaX(point.x)), reset)
    return
  }
  void verifyBehaviorCaptcha(String(Math.round(value as number)), reset)
}
// 刷新行为验证码。
const onBehaviorRefresh = () => {
  void refreshCaptcha()
}
// 表单提交
const onSubmit = async () => {
  const valid = await validateLoginForm()
  if (!valid) {
    return
  }
  if (isBehaviorCaptcha.value) {
    await openBehaviorCaptcha()
    return
  }
  try {
    const response = await submitLogin(form.value.captcha_code)
    if (await handleLoginResponse(response)) await loginSuccess()
  } catch {
    form.value.captcha_code = ''
    await refreshCaptcha()
  }
}
// #endif
/** 完成用户资料加载并替换到本次登录目标，直接登录默认进入首页。 */
const loginSuccess = async () => {
  await userStore.getUserProfile()
  // 成功提示
  await uni.showToast({ icon: 'success', title: t('core.login.login_success') })
  if (loginSuccessTimer) clearTimeout(loginSuccessTimer)
  loginSuccessTimer = setTimeout(() => {
    loginSuccessTimer = undefined
    void restoreLoginRedirect()
  }, 500)
}

// 请先阅读并勾选协议
const checkedAgreePrivacy = async () => {
  if (!(await ensureLoginSettings())) {
    return false
  }

  if (!settingStore.getData('serviceProtocol') || !settingStore.getData('privacyProtocol')) {
    await uni.showToast({ icon: 'none', title: t('core.login.protocol_missing') })
    return false
  }

  if (isAgreePrivacy.value) {
    return true
  }

  triggerAgreePrivacyShake()

  return new Promise<boolean>((resolve) => {
    uni.showModal({
      title: t('common.title.notice'),
      content: t('core.login.protocol_prompt'),
      confirmText: t('common.action.confirm'),
      cancelText: t('common.action.cancel'),
      confirmColor: themeStore.colors.primary,
      cancelColor: themeStore.colors.textMuted,
      success: ({ confirm }) => {
        if (confirm) {
          isAgreePrivacy.value = true
          resolve(true)
          return
        }
        resolve(false)
      },
      fail: () => resolve(false),
    })
  })
}

/** 加载登录所需的移动端配置。 */
const loadLoginSettings = () => {
  if (loginSettingsReady.value) {
    return Promise.resolve(true)
  }
  if (loginSettingsPromise) {
    return loginSettingsPromise
  }

  loginSettingsPromise = (async () => {
    try {
      await settingStore.loadData()
      // #ifdef H5
      const captchaType = configuredCaptchaType.value
      if (!captchaType) {
        throw new Error(t('core.login.captcha_type_missing'))
      }
      currentCaptchaType.value = captchaType
      await loadPageCaptcha()
      // #endif
      // #ifdef MP-WEIXIN
      await refreshMiniCaptcha()
      // #endif
      loginSettingsReady.value = true
      return true
    } catch (error) {
      await uni.showToast({
        icon: 'none',
        title: error instanceof Error ? error.message : t('core.protocol.load_failed'),
      })
      loginSettingsPromise = undefined
      return false
    }
  })()

  return loginSettingsPromise
}

/** 确保登录前已完成移动端配置加载。 */
const ensureLoginSettings = () => loadLoginSettings()

let loginRedirect: string | undefined
onLoad((options) => {
  loginRedirect = options?.redirect
})

onShow(() => {
  if (!prepareLoginPage(loginRedirect)) return
  void loadLoginSettings()
})

onUnload(() => {
  if (agreePrivacyShakeTimer) clearTimeout(agreePrivacyShakeTimer)
  agreePrivacyShakeTimer = undefined
  if (loginSuccessTimer) clearTimeout(loginSuccessTimer)
  loginSuccessTimer = undefined
})
</script>

<template>
  <view class="login-page" :class="themeStore.themeClass">
    <view class="login-header">
      <view class="k-ic login-back" aria-hidden="true" @tap="goBack">‹</view>
      <picker
        class="login-locale"
        :range="localeLabels"
        :value="localeIndex"
        @change="onLocaleChange"
      >
        <view class="k-lg-lang login-locale__value">{{ localeLabels[localeIndex] }}</view>
      </picker>
    </view>
    <view class="login-hero">
      <view class="k-lg-logo login-logo-shell">
        <image class="login-logo" :src="appLogo" @error="handleLogoError" />
      </view>
      <view class="login-hero-copy">
        <text class="k-lg-title login-title">{{ configuredMainTitle }}</text>
        <text class="k-lg-sub login-subtitle">{{ configuredSubTitle }}</text>
      </view>
    </view>
    <view class="k-glass k-glass--lit login-panel">
      <view class="login-form">
        <!-- 网页端表单登录 -->
        <!-- #ifdef H5 -->
        <view v-if="showTenantCode" class="k-lg-field">
          <input
            v-model="form.tenant_code"
            class="k-lg-field__input"
            type="text"
            confirm-type="next"
            :placeholder="t('core.login.tenant')"
          />
        </view>
        <view class="k-lg-field">
          <input
            v-model="form.user_name"
            class="k-lg-field__input"
            type="text"
            confirm-type="next"
            :placeholder="t('core.login.user_name_mobile')"
            @confirm="onSubmit"
          />
        </view>
        <view class="k-lg-field">
          <input
            v-model="passwordValue"
            class="k-lg-field__input"
            type="text"
            :password="!passwordVisible"
            confirm-type="done"
            :placeholder="t('core.login.password')"
            @confirm="onSubmit"
          />
          <view
            class="password-toggle"
            :aria-label="
              passwordVisible ? t('core.login.password_hide') : t('core.login.password_show')
            "
            @tap="passwordVisible = !passwordVisible"
          >
            <image
              class="password-toggle-icon"
              :src="passwordIconSrc(passwordVisible)"
              mode="aspectFit"
            ></image>
          </view>
        </view>
        <view v-if="!isBehaviorCaptcha" class="k-lg-field">
          <input
            v-model="form.captcha_code"
            class="k-lg-field__input"
            type="text"
            confirm-type="done"
            :placeholder="t('core.login.captcha')"
            @confirm="onSubmit"
          />
          <view class="captcha-divider"></view>
          <view
            class="k-lg-cap captcha-trigger"
            :style="{ width: captchaImageWidth }"
            @tap="getCaptcha"
          >
            <image
              class="captcha-image"
              :style="{ width: captchaImageWidth }"
              :src="captcha_base64"
              mode="aspectFit"
              @load="handleCaptchaImageLoad"
            />
          </view>
        </view>
        <button @tap="onSubmit" class="k-lg-btn k-lg-btn--pri">
          {{ t('common.action.login') }}
        </button>
        <view v-if="behaviorDialogVisible" class="login-behavior-mask">
          <view class="login-behavior-panel">
            <view v-if="behaviorLoading" class="login-behavior-loading">{{
              t('core.login.behavior_loading')
            }}</view>
            <GoCaptchaUni
              :type="currentCaptchaType"
              :data="behaviorCaptchaData"
              :config="behaviorCaptchaConfig"
              :theme="behaviorCaptchaTheme"
              @event-confirm="onBehaviorConfirm"
              @event-refresh="onBehaviorRefresh"
              @event-close="closeBehaviorCaptcha"
            />
          </view>
        </view>
        <!-- #endif -->

        <!-- 小程序端授权登录 -->
        <!-- #ifdef MP-WEIXIN -->
        <button v-if="!miniBinding" class="k-lg-btn k-lg-btn--wx" :loading="loading" @tap="wxLogin">
          <text class="icon icon-phone"></text>
          {{ t('core.login.wechat') }}
        </button>
        <view v-else class="login-form">
          <view class="login-bind-tip">{{ t('core.login.wechat_bind_tip') }}</view>
          <view v-if="showTenantCode" class="k-lg-field">
            <input
              v-model="miniForm.tenant_code"
              class="k-lg-field__input"
              type="text"
              :placeholder="t('core.login.tenant')"
            />
          </view>
          <view class="k-lg-field">
            <input
              v-model="miniForm.user_name"
              class="k-lg-field__input"
              type="text"
              :placeholder="t('core.login.user_name_mobile')"
            />
          </view>
          <view class="k-lg-field">
            <input
              v-model="miniPasswordValue"
              class="k-lg-field__input"
              type="text"
              :password="!miniPasswordVisible"
              :placeholder="t('core.login.password')"
            />
            <view
              class="password-toggle"
              :aria-label="
                miniPasswordVisible ? t('core.login.password_hide') : t('core.login.password_show')
              "
              @tap="miniPasswordVisible = !miniPasswordVisible"
            >
              <image
                class="password-toggle-icon"
                :src="passwordIconSrc(miniPasswordVisible)"
                mode="aspectFit"
              ></image>
            </view>
          </view>
          <view class="k-lg-field">
            <input
              v-model="miniForm.captcha_code"
              class="k-lg-field__input"
              type="text"
              :placeholder="t('core.login.captcha')"
            />
            <view class="captcha-divider"></view>
            <view class="k-lg-cap captcha-trigger" @tap="refreshMiniCaptcha">
              <image class="captcha-image" :src="miniCaptchaImage" mode="aspectFit" />
            </view>
          </view>
          <button class="k-lg-btn k-lg-btn--pri" :loading="loading" @tap="bindMiniAccount">
            {{ t('core.login.bind_wechat') }}
          </button>
        </view>
        <!-- #endif -->
      </view>
      <view class="login-tips" :class="{ animate__shakeY: isAgreePrivacyShakeY }">
        <view class="k-lg-agree login-agreement" @tap="toggleAgreePrivacy">
          <view class="k-lg-chk" :class="{ 'k-lg-chk--off': !isAgreePrivacy }">✓</view>
          <text class="login-agree-desc">{{ t('core.login.agree_prefix') }}</text>
          <text class="k-lg-link login-agree-link" @tap.stop="onOpenServiceProtocol">{{
            t('core.login.service')
          }}</text>
          <text class="login-agree-separator">{{ t('core.login.agree_separator') }}</text>
          <text class="k-lg-link login-agree-link" @tap.stop="onOpenPrivacyContract">{{
            t('core.login.privacy')
          }}</text>
        </view>
      </view>
      <view v-if="mfaVisible" class="mfa-mask">
        <view class="mfa-panel">
          <text class="mfa-title">{{ t('core.login.mfa_title') }}</text>
          <input
            v-if="mfaMethod !== 'webauthn'"
            v-model="mfaCode"
            class="login-input"
            type="number"
            maxlength="8"
            :placeholder="t('core.login.mfa_code')"
          />
          <input
            v-model="mfaRecoveryCode"
            class="login-input"
            :placeholder="t('core.login.mfa_recovery_code')"
          />
          <checkbox-group
            v-if="mfaRememberDays > 0"
            @change="rememberMfaDevice = $event.detail.value.includes('remember')"
          >
            <label class="mfa-remember-device">
              <checkbox
                class="mfa-remember-device__checkbox"
                value="remember"
                :checked="rememberMfaDevice"
              />
              <text>{{ t('core.login.mfa_remember_device', { days: mfaRememberDays }) }}</text>
            </label>
          </checkbox-group>
          <button
            class="login-button login-button-primary"
            :loading="mfaLoading"
            @tap="verifyMfaLogin"
          >
            {{
              mfaMethod === 'webauthn' && !mfaRecoveryCode
                ? t('core.login.mfa_webauthn_action')
                : t('common.action.confirm')
            }}
          </button>
        </view>
      </view>
      <view v-if="mfaSetupVisible" class="mfa-mask">
        <view class="mfa-panel">
          <text class="mfa-title">{{ t('core.login.mfa_setup_title') }}</text>
          <MfaSetupPanel :uri="mfaSetupUri" />
          <input
            v-if="mfaSetupMethod !== 'webauthn'"
            v-model="mfaSetupCode"
            class="login-input"
            type="number"
            maxlength="8"
            :placeholder="t('core.login.mfa_code')"
          />
          <button
            class="login-button login-button-primary"
            :loading="mfaLoading"
            @tap="confirmMfaSetup"
          >
            {{
              mfaSetupMethod === 'webauthn'
                ? t('core.login.mfa_webauthn_action')
                : t('common.action.confirm')
            }}
          </button>
        </view>
      </view>
      <MfaRecoveryCodesDialog
        v-model="mfaRecoveryCodesVisible"
        :codes="mfaRecoveryCodes"
        @confirm="finishMfaEnrollment"
      />
    </view>
  </view>
</template>

<style lang="scss">
.login-page {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 100vh;
  padding: 48rpx 20rpx calc(56rpx + env(safe-area-inset-bottom));
  background: var(--kratos-bg-gradient);
  box-sizing: border-box;
}

/* 自绘头部：左返回（复用 k-ic）、右上角语言（复用 k-lg-lang），替代原生导航栏 */
.login-header {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: env(safe-area-inset-top) 20rpx 0;
  box-sizing: content-box;
}

/* #ifdef MP-WEIXIN */
.login-header {
  /* 右上角避让小程序胶囊 */
  padding-right: 220rpx;
}
/* #endif */

/* k-ic 已提供方形图标按钮视觉，这里只补字号以容纳返回符号 */
.login-back {
  font-size: 36rpx;
  line-height: 1;
}

.login-locale {
  max-width: calc(100% - 120rpx);
}

/* k-lg-lang 已提供视觉样式，这里只补多语言文案的截断处理 */
.login-locale__value {
  max-width: 240rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.login-hero {
  margin-bottom: 44rpx;
  text-align: center;

  /* k-lg-logo 已提供渐变底板，这里只约束内部实际 Logo 图片尺寸 */
  .login-logo {
    width: 100rpx;
    height: 100rpx;
  }

  .login-hero-copy {
    display: flex;
    flex-direction: column;
  }
}

/* k-glass/k-glass--lit 已提供卡片底色与描边，这里只补内部留白 */
.login-panel {
  padding: 36rpx 30rpx 30rpx;
}

.login-form {
  display: flex;
  flex-direction: column;

  .login-bind-tip {
    margin-bottom: 20rpx;
    color: var(--kratos-color-text-muted);
    font-size: 24rpx;
    line-height: 1.5;
    text-align: center;
  }

  /* 密码/验证码行内尾随的小按钮： k-lg-field 没有对应词汇，保留页面局部样式 */
  .password-toggle {
    display: flex;
    flex: 0 0 52rpx;
    align-items: center;
    justify-content: center;
    width: 52rpx;
    height: 52rpx;
    margin-left: 12rpx;
  }

  .password-toggle-icon {
    width: 24rpx;
    height: 24rpx;
  }

  .icon {
    margin-right: 10rpx;
    font-size: 32rpx;
  }

  .icon-phone::before {
    content: '☎';
  }

  .captcha-divider {
    flex-shrink: 0;
    width: 1rpx;
    height: 44rpx;
    background: var(--kratos-color-border);
  }

  /* k-lg-cap 已提供验证码色块视觉，这里只补行内间距 */
  .captcha-trigger {
    margin-left: 14rpx;
  }

  .captcha-image {
    width: 170rpx;
    flex-shrink: 0;
    height: 60rpx;
  }
}
.mfa-mask {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40rpx;
  background: var(--kratos-color-mask);
}
.mfa-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 620rpx;
  padding: 36rpx;
  background: var(--kratos-color-surface-solid);
  border: 1rpx solid var(--kratos-color-border-accent);
  border-radius: 24rpx;
  box-sizing: border-box;
}
.mfa-title {
  margin-bottom: 22rpx;
  font-size: 34rpx;
  font-weight: 700;
  color: var(--kratos-color-text);
}

.mfa-remember-device {
  display: flex;
  align-items: center;
  margin-bottom: 20rpx;
  font-size: 26rpx;
  line-height: 1.5;
  color: var(--kratos-color-text);
}

.mfa-remember-device__checkbox {
  flex-shrink: 0;
  margin-right: 12rpx;
}

.mfa-panel .login-input {
  width: 100%;
  height: 96rpx;
  margin-bottom: 20rpx;
  padding: 0 30rpx;
  color: var(--kratos-color-text);
  font-size: 28rpx;
  background: var(--kratos-color-surface-2);
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 20rpx;
  box-sizing: border-box;
}

.mfa-panel .login-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 96rpx;
  margin-top: 12rpx;
  padding: 0;
  color: var(--kratos-color-on-primary);
  font-size: 30rpx;
  font-weight: 800;
  background: transparent;
  border: 0;
  border-radius: 20rpx;
  box-shadow: 0 0 40rpx rgba(40, 240, 212, 0.35);
}

.mfa-panel .login-button::after {
  border: 0;
}

.mfa-panel .login-button-primary {
  background: linear-gradient(
    135deg,
    var(--kratos-color-primary-deep) 0%,
    var(--kratos-color-primary) 100%
  );
}

.login-behavior-mask {
  position: fixed;
  z-index: 99;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24rpx;
  background: var(--kratos-color-mask);
  box-sizing: border-box;
}

.login-behavior-panel {
  position: relative;
  width: auto;
  max-width: 100%;
  padding: 24rpx;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 24rpx 70rpx rgba(15, 23, 42, 0.18);
  box-sizing: border-box;
}

.login-behavior-loading {
  position: absolute;
  z-index: 2;
  top: 24rpx;
  right: 24rpx;
  left: 24rpx;
  height: 44rpx;
  font-size: 24rpx;
  line-height: 44rpx;
  text-align: center;
  color: #28bb9c;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 999rpx;
}

.login-behavior-panel .go-captcha .gc-header {
  height: auto;
  min-height: 40rpx;
  margin-bottom: 12rpx;
}

.login-behavior-panel .go-captcha .gc-header .gc-text {
  font-size: 24rpx;
  font-weight: 500;
  line-height: 34rpx;
  white-space: nowrap;
}

.login-behavior-panel .go-captcha .gc-body {
  margin-top: 0;
  border-radius: 18rpx;
}

.login-behavior-panel .go-captcha .gc-footer {
  padding-top: 22rpx;
}

.login-behavior-panel .go-captcha .gc-drag-slide-bar {
  height: 56rpx;
}

.login-behavior-panel .go-captcha .gc-drag-line {
  height: 18rpx;
  background: #dbe7e3;
  border-radius: 999rpx;
}

.login-behavior-panel .go-captcha .gc-drag-block {
  width: 92rpx;
  height: 56rpx;
  margin-top: -28rpx;
  background: linear-gradient(
    135deg,
    var(--kratos-color-primary-deep) 0%,
    var(--kratos-color-primary) 100%
  );
  border-radius: 999rpx;
  box-shadow: 0 14rpx 28rpx rgba(40, 240, 212, 0.28);
  color: var(--kratos-color-on-primary);
  fill: var(--kratos-color-on-primary);
}

.login-behavior-panel .go-captcha .gc-drag-block.disabled {
  background: #9adfce;
  box-shadow: none;
}

.login-behavior-panel .go-captcha .gc-drag-block .gc-icon {
  color: var(--kratos-color-on-primary);
  font-size: 42rpx;
}

.login-behavior-panel .go-captcha .gc-rotate-picture .gc-round {
  border-width: 8rpx;
  box-shadow: inset 0 0 0 2rpx rgba(255, 255, 255, 0.78);
}

.login-behavior-panel .go-captcha .gc-rotate-thumb-block {
  overflow: hidden;
  border: 4rpx solid rgba(255, 255, 255, 0.92);
  border-radius: 50%;
  box-shadow:
    0 14rpx 30rpx rgba(15, 23, 42, 0.2),
    0 0 0 2rpx rgba(15, 23, 42, 0.08);
}

.login-behavior-panel .go-captcha .gc-rotate-thumb-block .gc-img {
  border-radius: 50%;
}

.login-behavior-panel .go-captcha .gc-dots .gc-dot {
  font-size: 20rpx;
  font-weight: 600;
  border-width: 3rpx;
  box-shadow: 0 6rpx 14rpx rgba(15, 23, 42, 0.18);
}

.login-tips {
  margin-top: 26rpx;
}

/* k-lg-agree 已提供排版与字色，这里只补宽度约束，保证长文案正常换行 */
.login-agreement {
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.login-agree-separator {
  min-width: 0;
  overflow-wrap: anywhere;
}

.login-agree-link {
  min-width: 0;
  overflow-wrap: anywhere;
}

.login-agree-desc {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
