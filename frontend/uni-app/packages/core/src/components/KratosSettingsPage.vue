<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useThemeStore, useUserStore, type ThemeMode } from '../stores'
import { navigateToLogin } from '../utils/navigation'
import { getLanguageOptions, useI18n, type SupportedLocale } from '../locales'
import { isAuthExpiredError } from '../utils/http'
import { defMfaService } from '../api/base/v1/mfa'
import MfaManageDialog from './MfaManageDialog.vue'

defineOptions({ name: 'KratosSettingsPage' })

defineSlots<{
  /** 在所有公共设置卡片之前渲染的业务扩展内容。 */
  extensions?: () => unknown
}>()

const userStore = useUserStore()
const themeStore = useThemeStore()
const { locale, setLocale, t } = useI18n()

const logoutLoading = ref(false)
const mfaDialogVisible = ref(false)
const mfaDialogMode = ref<'setup' | 'disable'>('setup')
const mfaEnabled = ref(false)
const mfaMethod = ref('totp')
const mfaStatusLoading = ref(false)
const localeOptions = computed(() => getLanguageOptions())
const themeModeOptions: ThemeMode[] = ['auto', 'dark', 'light']
const themeModeLabels = computed(() =>
  themeModeOptions.map((mode) =>
    mode === 'auto'
      ? t('core.theme.auto')
      : mode === 'dark'
        ? t('core.theme.dark')
        : t('core.theme.light'),
  ),
)
const themeModeIndex = computed(() => Math.max(0, themeModeOptions.indexOf(themeStore.mode)))
const localeLabels = computed(() => localeOptions.value.map((item) => item.native_name))
const localeIndex = computed(() =>
  Math.max(
    0,
    localeOptions.value.findIndex((item) => item.language_code === locale.value),
  ),
)

const settingsIcons = computed(() => {
  const primary = themeStore.theme === 'light' ? '#039e8c' : '#28f0d4'
  const warning = themeStore.theme === 'light' ? '#d9880f' : '#ffb648'
  const violet = themeStore.theme === 'light' ? '#5f4fe0' : '#9b8cff'
  const createIcon = (body: string, color: string) =>
    `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${body}</svg>`)}`

  return {
    theme: createIcon('<path d="M21 12.8A8.5 8.5 0 1 1 11.2 3a6.5 6.5 0 0 0 9.8 9.8Z"/>', primary),
    mfa: createIcon(
      '<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6l-8-3Z"/><path d="m9 12 2 2 4-4"/>',
      primary,
    ),
    language: createIcon(
      '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18"/>',
      primary,
    ),
    authorization: createIcon(
      '<circle cx="8" cy="15" r="4"/><path d="m11 12 7-7 3 3-7 7"/><path d="m15 8 2 2"/>',
      warning,
    ),
    feedback: createIcon('<path d="M4 5h16v12H8l-4 4z"/>', violet),
    contact: createIcon(
      '<path d="M4 5c0 9 6 15 15 15 1 0 2-1 2-2v-2l-4-2-2 2a11 11 0 0 1-5-5l2-2-2-4H6c-1 0-2 1-2 2Z"/>',
      primary,
    ),
  }
})

/** 处理主题外观选择并持久化。 */
const onThemeModeChange = (event: { detail: { value: string | number } }) => {
  const nextMode = themeModeOptions[Number(event.detail.value)]
  if (nextMode) themeStore.setMode(nextMode)
}

/** 处理语言选择并刷新动态本地化数据。 */
const onLocaleChange = (event: { detail: { value: string | number } }) => {
  const nextLocale = localeOptions.value[Number(event.detail.value)]?.language_code as
    | SupportedLocale
    | undefined
  if (nextLocale) void setLocale(nextLocale)
}

/** 加载当前用户的多因素认证状态。 */
const loadMfaStatus = async (force = false) => {
  if (!userStore.isAuthenticated() || (mfaStatusLoading.value && !force)) return
  mfaStatusLoading.value = true
  try {
    const result = await defMfaService.GetMfaStatus({})
    mfaEnabled.value = result.enabled
    mfaMethod.value = result.method || 'totp'
  } finally {
    mfaStatusLoading.value = false
  }
}

/** 当前多因素认证状态展示文案。 */
const mfaStatusText = computed(() => {
  if (!mfaEnabled.value) return t('core.settings.mfa_disabled')
  const method =
    mfaMethod.value === 'webauthn'
      ? t('core.settings.mfa_method_webauthn')
      : t('core.settings.mfa_method_totp')
  return t('core.settings.mfa_enabled_method', { method })
})

/** 根据当前状态打开多因素认证操作组件。 */
const onMfaTap = async () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  const currentMfaEnabled = mfaEnabled.value
  try {
    await loadMfaStatus(true)
  } catch {
    // 状态刷新失败时保留页面已有状态，避免已启用用户无法进入禁用流程。
    mfaEnabled.value = currentMfaEnabled
  }
  if (mfaEnabled.value) {
    mfaDialogMode.value = 'disable'
    mfaDialogVisible.value = true
    return
  }
  mfaDialogMode.value = 'setup'
  mfaDialogVisible.value = true
}

/** 绑定成功后清理旧会话并重新登录。 */
const onMfaSuccess = async (method: string) => {
  mfaEnabled.value = true
  mfaMethod.value = method || 'totp'
  await uni.showToast({ icon: 'none', title: t('core.settings.mfa_bind_success') })
  // 绑定 MFA 后后端会撤销当前会话，不能再调用登出接口或继续使用旧 token。
  await userStore.clearUserData()
  uni.reLaunch({ url: '/pages/login/login' })
}

/** 禁用成功后清理旧会话并重新登录。 */
const onMfaDisabled = async () => {
  mfaEnabled.value = false
  await uni.showToast({ icon: 'none', title: t('core.settings.mfa_disable_success') })
  await userStore.clearUserData()
  uni.reLaunch({ url: '/pages/login/login' })
}

/** 返回个人中心，页面栈不可返回时回到首页。 */
const onBack = () => {
  uni.navigateBack({
    fail: () => {
      uni.reLaunch({ url: '/pages/index/index' })
    },
  })
}

// 公共设置模块挂载时加载 MFA 状态，兼容页面包装和业务插槽场景。
onMounted(() => {
  if (!userStore.ensureAuthenticated()) {
    // #ifndef MP-WEIXIN
    navigateToLogin()
    // #endif
    return
  }
  void loadMfaStatus()
})

/** 确认后退出登录并返回上一页。 */
const onLogout = () => {
  if (logoutLoading.value) {
    return
  }
  // 模态弹窗
  uni.showModal({
    content: t('core.settings.logout_confirm'),
    confirmColor: themeStore.theme === 'light' ? '#039e8c' : '#28f0d4',
    cancelColor: themeStore.theme === 'light' ? '#5a7083' : '#8aa0b4',
    success: async (res) => {
      if (!res.confirm) {
        return
      }

      logoutLoading.value = true
      try {
        // 先完成退出和本地登录态清理，再返回个人中心，避免 onShow 读取到旧登录态。
        await userStore.logout()
        uni.navigateBack()
      } catch (error) {
        if (!isAuthExpiredError(error)) {
          await uni.showToast({
            icon: 'none',
            title: t('core.settings.logout_failed'),
          })
        }
      } finally {
        logoutLoading.value = false
      }
    },
  })
}
</script>

<template>
  <view class="settings-page" :class="themeStore.themeClass">
    <slot name="extensions" />
    <view class="k-page-nav">
      <view class="k-page-nav__back" :aria-label="t('common.action.back')" @tap="onBack"></view>
      <view class="k-page-nav__title">{{ t('core.settings.title') }}</view>
      <view class="k-page-nav__side" aria-hidden="true"></view>
    </view>
    <view class="settings-content">
      <view class="k-set-cat">{{ t('core.settings.section_appearance') }}</view>
      <picker :range="themeModeLabels" :value="themeModeIndex" @change="onThemeModeChange">
        <view class="k-set-row k-set-row--arrow">
          <view class="k-set-row__lead">
            <view class="k-set-row__icon">
              <image class="settings-row__icon" :src="settingsIcons.theme" mode="aspectFit" />
            </view>
            <text class="k-set-row__title">{{ t('core.settings.theme') }}</text>
          </view>
          <text class="k-set-row__value settings-row__value">{{
            themeModeLabels[themeModeIndex]
          }}</text>
        </view>
      </picker>

      <view class="k-set-cat">{{ t('core.settings.section_security') }}</view>
      <view v-if="userStore.isAuthenticated()" class="k-set-row k-set-row--arrow" @tap="onMfaTap">
        <view class="k-set-row__lead">
          <view class="k-set-row__icon">
            <image class="settings-row__icon" :src="settingsIcons.mfa" mode="aspectFit" />
          </view>
          <text class="k-set-row__title">{{ t('core.settings.mfa_title') }}</text>
        </view>
        <text
          class="k-set-row__value settings-row__value"
          :class="{ 'k-set-row__value--on': mfaEnabled }"
          >{{ mfaStatusText }}</text
        >
      </view>
      <picker :range="localeLabels" :value="localeIndex" @change="onLocaleChange">
        <view class="k-set-row k-set-row--arrow">
          <view class="k-set-row__lead">
            <view class="k-set-row__icon">
              <image class="settings-row__icon" :src="settingsIcons.language" mode="aspectFit" />
            </view>
            <text class="k-set-row__title">{{ t('common.field.language') }}</text>
          </view>
          <text class="k-set-row__value settings-row__value">{{ localeLabels[localeIndex] }}</text>
        </view>
      </picker>

      <!-- #ifdef MP-WEIXIN -->
      <view class="k-set-cat">{{ t('core.settings.section_more') }}</view>
      <button
        hover-class="none"
        class="k-set-row k-set-row--arrow settings-row--button"
        open-type="openSetting"
      >
        <view class="k-set-row__lead">
          <view class="k-set-row__icon">
            <image class="settings-row__icon" :src="settingsIcons.authorization" mode="aspectFit" />
          </view>
          <text class="k-set-row__title">{{ t('core.settings.authorization') }}</text>
        </view>
      </button>
      <button
        hover-class="none"
        class="k-set-row k-set-row--arrow settings-row--button"
        open-type="feedback"
      >
        <view class="k-set-row__lead">
          <view class="k-set-row__icon">
            <image class="settings-row__icon" :src="settingsIcons.feedback" mode="aspectFit" />
          </view>
          <text class="k-set-row__title">{{ t('core.settings.feedback') }}</text>
        </view>
      </button>
      <button
        hover-class="none"
        class="k-set-row k-set-row--arrow settings-row--button"
        open-type="contact"
      >
        <view class="k-set-row__lead">
          <view class="k-set-row__icon">
            <image class="settings-row__icon" :src="settingsIcons.contact" mode="aspectFit" />
          </view>
          <text class="k-set-row__title">{{ t('core.settings.contact') }}</text>
        </view>
      </button>
      <!-- #endif -->

      <view v-if="userStore.isAuthenticated()" class="k-set-logout" @tap="onLogout">{{
        t('common.action.logout')
      }}</view>
    </view>
  </view>
  <MfaManageDialog
    v-model="mfaDialogVisible"
    :mode="mfaDialogMode"
    :method="mfaMethod"
    @success="onMfaSuccess"
    @disabled="onMfaDisabled"
  />
</template>

<style scoped lang="scss">
.settings-page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 0 32rpx calc(40rpx + env(safe-area-inset-bottom));
  background: var(--kratos-bg-gradient);
}
.settings-content {
  padding-bottom: 20rpx;
}

/* 行图标图片尺寸（挂在 .k-set-row__icon 内的 image 上） */
.settings-row__icon {
  width: 38rpx;
  height: 38rpx;
}

/* 行尾值文本截断（配合全局 .k-set-row__value 使用） */
.settings-row__value {
  min-width: 0;
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 小程序 button 皮肤为 .k-set-row：去掉原生边框，保留箭头 ::after */
.settings-row--button {
  line-height: normal;
}

.settings-row--button::after {
  position: static;
  border-left: 0;
  border-bottom: 0;
}
</style>
