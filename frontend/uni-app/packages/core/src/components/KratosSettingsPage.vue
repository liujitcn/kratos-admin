<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useUserStore } from '../stores'
import { navigateToLogin } from '../utils/navigation'
import { getLanguageOptions, useI18n, type SupportedLocale } from '../locales'
import { defMfaService } from '../api/base/v1/mfa'
import MfaManageDialog from './MfaManageDialog.vue'

defineOptions({ name: 'KratosSettingsPage' })

defineSlots<{
  /** 在所有公共设置卡片之前渲染的业务扩展内容。 */
  extensions?: () => unknown
}>()

const userStore = useUserStore()
const { locale, setLocale, t } = useI18n()
const logoutLoading = ref(false)
const mfaDialogVisible = ref(false)
const mfaDialogMode = ref<'setup' | 'disable'>('setup')
const mfaEnabled = ref(false)
const mfaMethod = ref('totp')
const mfaStatusLoading = ref(false)
const localeOptions = computed(() => getLanguageOptions())
const localeLabels = computed(() => localeOptions.value.map((item) => item.native_name))
const localeIndex = computed(() =>
  Math.max(
    0,
    localeOptions.value.findIndex((item) => item.language_code === locale.value),
  ),
)

watch(locale, () => void uni.setNavigationBarTitle({ title: t('core.settings.title') }), {
  immediate: true,
})

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
    confirmColor: '#27BA9B',
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
        await uni.showToast({
          icon: 'none',
          title: t('core.settings.logout_failed'),
        })
      } finally {
        logoutLoading.value = false
      }
    },
  })
}
</script>

<template>
  <view class="viewport">
    <slot name="extensions" />
    <view class="list">
      <view v-if="userStore.isAuthenticated()" class="item arrow mfa-item" @tap="onMfaTap">
        <text>{{ t('core.settings.mfa_title') }}</text>
        <text class="mfa-value">{{ mfaStatusText }}</text>
      </view>
      <picker :range="localeLabels" :value="localeIndex" @change="onLocaleChange">
        <view class="item arrow locale-item">
          <text>{{ t('common.field.language') }}</text>
          <text class="locale-value">{{ localeLabels[localeIndex] }}</text>
        </view>
      </picker>
    </view>
    <!-- #ifdef MP-WEIXIN -->
    <!-- 列表2 -->
    <view class="list">
      <button hover-class="none" class="item arrow" open-type="openSetting">
        {{ t('core.settings.authorization') }}
      </button>
      <button hover-class="none" class="item arrow" open-type="feedback">
        {{ t('core.settings.feedback') }}
      </button>
      <button hover-class="none" class="item arrow" open-type="contact">
        {{ t('core.settings.contact') }}
      </button>
    </view>
    <!-- #endif -->
    <!-- 操作按钮 -->
    <view class="action" v-if="userStore.isAuthenticated()">
      <view @tap="onLogout" class="button">{{ t('common.action.logout') }}</view>
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
.viewport {
  min-height: 100vh;
  padding: 20rpx;
  box-sizing: border-box;
  background-color: #f4f4f4;
}

/* 列表 */
.list {
  padding: 0 20rpx;
  background-color: #fff;
  margin-bottom: 20rpx;
  border-radius: 10rpx;
  .item {
    line-height: 90rpx;
    padding-left: 10rpx;
    font-size: 30rpx;
    color: #333;
    border-top: 1rpx solid #ddd;
    position: relative;
    text-align: left;
    border-radius: 0;
    background-color: #fff;
    &::after {
      width: auto;
      height: auto;
      left: auto;
      border: none;
    }
    &::after {
      right: 5rpx;
    }
  }
  > .item:first-child {
    border: none;
  }
  .arrow::after {
    content: '›';
    position: absolute;
    top: 50%;
    color: #ccc;
    font-size: 36rpx;
    transform: translateY(-50%);
  }
}

.locale-item {
  display: flex;
  min-width: 0;
  justify-content: space-between;
  padding-right: 42rpx;

  > text:first-child {
    flex-shrink: 0;
  }
}

.locale-value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  margin-left: 24rpx;
  color: #888;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mfa-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-right: 42rpx;
}

.mfa-value {
  max-width: 55%;
  overflow: hidden;
  color: #888;
  font-size: 26rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 操作按钮 */
.action {
  text-align: center;
  line-height: 90rpx;
  margin-top: 40rpx;
  font-size: 32rpx;
  color: #333;
  .button {
    background-color: #fff;
    margin-bottom: 20rpx;
    border-radius: 10rpx;
  }
}
</style>
