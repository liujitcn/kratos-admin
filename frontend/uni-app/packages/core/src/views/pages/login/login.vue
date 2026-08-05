<script setup lang="ts">
import { useSettingStore, useUserStore } from '../../../stores'
import type { LoginRequest } from '../../../rpc/base/v1/login'
import { onLoad } from '@dcloudio/uni-app'
import { computed, defineAsyncComponent, reactive, ref, watch } from 'vue'
import { defLoginService } from '../../../api/base/login'
import defaultLogo from '../../../static/images/logo_icon.png'
import { homeTabPage } from '../../../utils/navigation'
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from '../../../utils/passwordCrypto'
import { navigateAppRoute } from '../../../navigation'
import { getLanguageOptions, useI18n, type SupportedLocale } from '../../../locales'

// go-captcha-uni 仅在 H5 端使用，小程序端不支持 defineAsyncComponent，需按平台裁剪。
// #ifdef H5
const GoCaptchaUni = defineAsyncComponent(() => import('go-captcha-uni'))
// #endif

const userStore = useUserStore()
const settingStore = useSettingStore()
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
const loginSettingsReady = ref(false)
let loginSettingsPromise: Promise<boolean> | undefined

const configuredMainTitle = computed(
  () => settingStore.getData('mainTitle') || t('core.home.main_title'),
)
const configuredSubTitle = computed(
  () => settingStore.getData('subTitle') || t('core.login.default_sub_title'),
)
const configuredAppLogo = computed(() => settingStore.getData('appLogo') || defaultLogo)
const showTenantCode = computed(() => settingStore.getData('showTenantCode') !== 'false')

// 是否同意协议
const isAgreePrivacy = ref(false)
const isAgreePrivacyShakeY = ref(false)
const loading = ref(false)

watch(locale, () => void uni.setNavigationBarTitle({ title: t('common.action.login') }), {
  immediate: true,
})

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

const triggerAgreePrivacyShake = () => {
  isAgreePrivacyShakeY.value = true
  setTimeout(() => {
    isAgreePrivacyShakeY.value = false
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
  const response = error as {
    statusCode?: number
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
    await loginSuccess()
  } catch (error) {
    if (!isWechatUnboundError(error)) {
      await uni.showToast({ icon: 'none', title: t('core.login.wechat_failed') })
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
    const password = await encryptPassword(
      miniPasswordValue.value,
      PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_LOGIN,
    )
    const code = (await wx.login()).code
    await userStore.bindOauthSession({
      provider: wechatMiniProvider,
      code,
      tenant_code: showTenantCode.value ? miniForm.tenant_code : '0000',
      user_name: miniForm.user_name,
      password,
      captcha_code: miniForm.captcha_code,
      captcha_id: miniForm.captcha_id,
    })
    await loginSuccess()
  } catch (error) {
    await refreshMiniCaptcha()
    if (error instanceof Error && error.message) {
      await uni.showToast({ icon: 'none', title: error.message })
    }
  } finally {
    loading.value = false
  }
}
// #endif

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
  const password = await encryptPassword(
    passwordValue.value,
    PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_LOGIN,
  )
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
    await submitLogin(captchaToken)
    await loginSuccess()
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
    await submitLogin(form.value.captcha_code)
    await loginSuccess()
  } catch {
    form.value.captcha_code = ''
    await refreshCaptcha()
  }
}
// #endif
const loginSuccess = async () => {
  await userStore.getUserProfile()
  // 成功提示
  await uni.showToast({ icon: 'success', title: t('core.login.login_success') })
  setTimeout(() => {
    const lastRoute = uni.getStorageSync('lastRoute') || homeTabPage
    if (lastRoute.startsWith(homeTabPage)) {
      uni.setStorageSync('SwitchTabIndex', true)
    }
    uni.removeStorageSync('lastRoute')

    uni.reLaunch({ url: lastRoute })
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

// 获取 code 登录凭证
onLoad(() => {
  void loadLoginSettings()
})
</script>

<template>
  <view class="login-page">
    <picker
      class="login-locale"
      :range="localeLabels"
      :value="localeIndex"
      @change="onLocaleChange"
    >
      <view class="login-locale__value">{{ localeLabels[localeIndex] }}</view>
    </picker>
    <view class="login-hero">
      <view class="login-logo-shell">
        <image :src="configuredAppLogo" />
      </view>
      <view class="login-hero-copy">
        <text class="login-title">{{ configuredMainTitle }}</text>
        <text class="login-subtitle">{{ configuredSubTitle }}</text>
      </view>
    </view>
    <view class="login-panel">
      <view class="login-form">
        <!-- 网页端表单登录 -->
        <!-- #ifdef H5 -->
        <input
          v-if="showTenantCode"
          v-model="form.tenant_code"
          class="login-input"
          type="text"
          confirm-type="next"
          :placeholder="t('core.login.tenant')"
        />
        <input
          v-model="form.user_name"
          class="login-input"
          type="text"
          confirm-type="next"
          :placeholder="t('core.login.user_name_mobile')"
          @confirm="onSubmit"
        />
        <input
          v-model="passwordValue"
          class="login-input"
          type="text"
          password
          confirm-type="done"
          :placeholder="t('core.login.password')"
          @confirm="onSubmit"
        />
        <view v-if="!isBehaviorCaptcha" class="captcha-row">
          <input
            v-model="form.captcha_code"
            class="login-input captcha-input"
            type="text"
            confirm-type="done"
            :placeholder="t('core.login.captcha')"
            @confirm="onSubmit"
          />
          <view class="captcha-divider"></view>
          <view class="captcha-trigger" :style="{ width: captchaImageWidth }" @tap="getCaptcha">
            <image
              class="captcha-image"
              :style="{ width: captchaImageWidth }"
              :src="captcha_base64"
              mode="aspectFit"
              @load="handleCaptchaImageLoad"
            />
          </view>
        </view>
        <button @tap="onSubmit" class="login-button login-button-primary">
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
        <button
          v-if="!miniBinding"
          class="login-button login-button-primary"
          :loading="loading"
          @tap="wxLogin"
        >
          <text class="icon icon-phone"></text>
          {{ t('core.login.wechat') }}
        </button>
        <view v-else class="login-form">
          <view class="login-bind-tip">{{ t('core.login.wechat_bind_tip') }}</view>
          <input
            v-if="showTenantCode"
            v-model="miniForm.tenant_code"
            class="login-input"
            type="text"
            :placeholder="t('core.login.tenant')"
          />
          <input
            v-model="miniForm.user_name"
            class="login-input"
            type="text"
            :placeholder="t('core.login.user_name_mobile')"
          />
          <input
            v-model="miniPasswordValue"
            class="login-input"
            type="text"
            password
            :placeholder="t('core.login.password')"
          />
          <view class="captcha-row">
            <input
              v-model="miniForm.captcha_code"
              class="login-input captcha-input"
              type="text"
              :placeholder="t('core.login.captcha')"
            />
            <view class="captcha-divider"></view>
            <view class="captcha-trigger" @tap="refreshMiniCaptcha">
              <image class="captcha-image" :src="miniCaptchaImage" mode="aspectFit" />
            </view>
          </view>
          <button
            class="login-button login-button-primary"
            :loading="loading"
            @tap="bindMiniAccount"
          >
            {{ t('core.login.bind_wechat') }}
          </button>
        </view>
        <!-- #endif -->
      </view>
      <view class="login-tips" :class="{ animate__shakeY: isAgreePrivacyShakeY }">
        <view class="login-agreement" @tap="toggleAgreePrivacy">
          <view class="login-agree-icon" :class="{ checked: isAgreePrivacy }"></view>
          <text class="login-agree-desc">{{ t('core.login.agree_prefix') }}</text>
          <text class="login-agree-link" @tap.stop="onOpenServiceProtocol">{{
            t('core.login.service')
          }}</text>
          <text class="login-agree-separator">{{ t('core.login.agree_separator') }}</text>
          <text class="login-agree-link" @tap.stop="onOpenPrivacyContract">{{
            t('core.login.privacy')
          }}</text>
        </view>
      </view>
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
  padding: 48rpx 40rpx 56rpx;
  background: linear-gradient(180deg, #f4fbf8 0%, #ffffff 45%, #ffffff 100%);
  box-sizing: border-box;
}

.login-locale {
  position: absolute;
  top: calc(24rpx + env(safe-area-inset-top));
  right: 28rpx;
  z-index: 2;
}

.login-locale__value {
  padding: 12rpx 16rpx;
  color: #16806d;
  font-size: 24rpx;
}

.login-hero {
  margin-bottom: 40rpx;
  text-align: center;

  .login-logo-shell {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 180rpx;
    height: 180rpx;
    margin-bottom: 24rpx;
    border-radius: 44rpx;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: 0 18rpx 44rpx rgba(40, 187, 156, 0.12);
  }

  image {
    width: 120rpx;
    height: 120rpx;
  }

  .login-hero-copy {
    display: flex;
    flex-direction: column;
  }

  .login-title {
    font-size: 44rpx;
    font-weight: 600;
    color: #1f2937;
  }

  .login-subtitle {
    margin-top: 12rpx;
    color: #6b7280;
    font-size: 26rpx;
  }
}

.login-panel {
  padding: 36rpx 28rpx 30rpx;
  border-radius: 36rpx;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 24rpx 60rpx rgba(15, 23, 42, 0.08);
}

.login-form {
  display: flex;
  flex-direction: column;

  .login-bind-tip {
    margin-bottom: 20rpx;
    color: #6b7280;
    font-size: 24rpx;
    line-height: 1.5;
    text-align: center;
  }

  .login-input {
    width: 100%;
    height: 88rpx;
    font-size: 28rpx;
    color: #1f2937;
    border-radius: 24rpx;
    border: 1px solid #e5e7eb;
    background: #f9fbfa;
    padding: 0 30rpx;
    margin-bottom: 20rpx;
    box-sizing: border-box;
  }

  .login-button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 88rpx;
    margin-top: 12rpx;
    font-size: 30rpx;
    font-weight: 500;
    border-radius: 24rpx;
    color: #fff;
    box-shadow: 0 18rpx 36rpx rgba(40, 187, 156, 0.24);

    &::after {
      border: 0;
    }

    .icon {
      font-size: 40rpx;
      margin-right: 10rpx;
    }

    .icon-phone::before {
      content: '☎';
    }
  }

  .login-button-primary {
    background: linear-gradient(135deg, #34c8aa 0%, #28bb9c 100%);
  }

  .captcha-row {
    display: flex;
    align-items: center;
    width: 100%;
    height: 88rpx;
    margin-bottom: 20rpx;
    padding: 0 12rpx 0 30rpx;
    border: 1px solid #e5e7eb;
    border-radius: 24rpx;
    background: #f9fbfa;
    box-sizing: border-box;

    .captcha-input {
      flex: 1;
      height: 100%;
      margin-bottom: 0;
      padding: 0;
      border: 0;
      background: transparent;
    }

    .captcha-divider {
      width: 1rpx;
      height: 44rpx;
      background: #e7ebf0;
    }

    .captcha-trigger {
      display: flex;
      align-items: center;
      justify-content: center;
      flex: 0 0 auto;
      height: 72rpx;
      margin-left: 14rpx;
      border-radius: 18rpx;
      background: #fff;
      box-sizing: border-box;
    }

    .captcha-image {
      width: 170rpx;
      flex-shrink: 0;
      height: 60rpx;
    }
  }
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
  background: rgba(15, 23, 42, 0.38);
  box-sizing: border-box;
}

.login-behavior-panel {
  position: relative;
  width: auto;
  max-width: 100%;
  padding: 24rpx;
  border-radius: 28rpx;
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
  background: linear-gradient(135deg, #36d6b4 0%, #20aa8f 100%);
  border-radius: 999rpx;
  box-shadow: 0 14rpx 28rpx rgba(40, 187, 156, 0.28);
  color: #fff;
  fill: #fff;
}

.login-behavior-panel .go-captcha .gc-drag-block.disabled {
  background: #9adfce;
  box-shadow: none;
}

.login-behavior-panel .go-captcha .gc-drag-block .gc-icon {
  color: #fff;
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
  font-size: 22rpx;
  color: #999;
  line-height: 1.6;

  .login-agreement {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    justify-content: center;
    width: 100%;
    max-width: 100%;
    min-width: 0;
    gap: 6rpx;
    box-sizing: border-box;
  }

  .login-agree-separator {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .login-agree-icon {
    flex: 0 0 auto;
    width: 26rpx;
    height: 26rpx;
    margin-right: 2rpx;
    border: 2rpx solid #d1d5db;
    border-radius: 50%;
    background-color: #fff;
    box-sizing: border-box;
  }

  .login-agree-icon.checked {
    border-color: #28bb9c;
    background-color: #28bb9c;
    box-shadow: inset 0 0 0 6rpx #fff;
  }

  .login-agree-link {
    min-width: 0;
    overflow-wrap: anywhere;
    color: #28bb9c;
  }

  .login-agree-desc {
    min-width: 0;
    overflow-wrap: anywhere;
    color: #9ca3af;
  }
}
</style>
