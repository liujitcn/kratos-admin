<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import KratosNavBar from '@liujitcn/kratos-uni-app-core/components/KratosNavBar.vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore, useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import { launchAppStatus, resolveAppRoute } from '@liujitcn/kratos-uni-app-core'
import {
  encryptWithPublicKey,
  fetchSecretPublicKey,
} from '@liujitcn/kratos-uni-app-core/utils/secretCrypto'
import { defAccountService } from '@liujitcn/kratos-uni-app-core/api/base/v1/account'

const { t } = useI18n()
const themeStore = useThemeStore()
const userStore = useUserStore()
const submitting = ref(false)
const policyLoading = ref(false)
const errorMessage = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordPolicy = ref<Awaited<ReturnType<typeof defAccountService.GetCurrentPasswordPolicy>>>()

const strengthScore = computed(() => {
  if (!newPassword.value) return 0
  let score = 0
  if (/[a-z]/.test(newPassword.value)) score += 1
  if (/[A-Z]/.test(newPassword.value)) score += 1
  if (/\d/.test(newPassword.value)) score += 1
  if (/[^A-Za-z0-9]/.test(newPassword.value)) score += 1
  if (score >= 4) return 3
  if (score === 3) return 2
  return 1
})
const strengthLevel = computed(() => ['empty', 'low', 'medium', 'high'][strengthScore.value])
const strengthLabelKey = computed(() => {
  if (strengthLevel.value === 'low') return 'system.settings.password_strength_low'
  if (strengthLevel.value === 'medium') return 'system.settings.password_strength_medium'
  if (strengthLevel.value === 'high') return 'system.settings.password_strength_high'
  return 'system.settings.password_strength_empty'
})
const policyTip = computed(() => {
  const policy = passwordPolicy.value
  if (!policy) return t('system.settings.password_policy_default')
  const requirements = [
    t('system.settings.password_policy_min_length', { count: policy.min_length }),
    t('system.settings.password_policy_complexity', { count: policy.min_complexity_classes }),
  ]
  if (policy.history_count > 0) {
    requirements.push(t('system.settings.password_policy_history', { count: policy.history_count }))
  }
  requirements.push(
    policy.max_age_days > 0
      ? t('system.settings.password_policy_max_age', { count: policy.max_age_days })
      : t('system.settings.password_policy_no_expiry'),
  )
  return requirements.join(t('system.settings.password_policy_separator'))
})

/** 页面加载时校验登录态并读取当前密码策略。 */
onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  if (!resolveAppRoute('app/password')) {
    launchAppStatus('NOT_FOUND')
    return
  }
  void loadPasswordPolicy()
})

/** 读取当前用户生效的密码策略。 */
async function loadPasswordPolicy() {
  policyLoading.value = true
  try {
    passwordPolicy.value = await defAccountService.GetCurrentPasswordPolicy({})
  } catch {
    passwordPolicy.value = undefined
  } finally {
    policyLoading.value = false
  }
}

/** 校验并提交修改密码请求。 */
async function handleSubmit() {
  if (submitting.value) return
  if (!oldPassword.value || !newPassword.value || !confirmPassword.value) {
    errorMessage.value = t('system.settings.password_required')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    errorMessage.value = t('system.settings.password_mismatch')
    return
  }
  submitting.value = true
  errorMessage.value = ''
  try {
    const publicKey = await fetchSecretPublicKey()
    await defAccountService.UpdateUserPassword({
      user_password: {
        old_pwd: await encryptWithPublicKey(publicKey, oldPassword.value),
        new_pwd: await encryptWithPublicKey(publicKey, newPassword.value),
      },
    })
    await uni.showToast({ icon: 'none', title: t('system.settings.password_updated') })
    await userStore.clearUserData()
    uni.reLaunch({ url: '/pages/login/login' })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('common.message.request_error')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <view class="change-password-page" :class="themeStore.themeClass">
    <KratosNavBar :title="t('system.settings.password_title')" />
    <view class="change-password-page__content">
      <view class="k-glass change-password-card">
        <text class="change-password-card__description">{{
          t('system.settings.password_description')
        }}</text>
        <text class="change-password-card__label">{{ t('system.settings.password_current') }}</text>
        <input
          v-model="oldPassword"
          class="change-password-card__input"
          password
          autocomplete="current-password"
          :placeholder="t('system.settings.password_current_placeholder')"
          @input="errorMessage = ''"
        />
        <text class="change-password-card__label">{{ t('system.settings.password_new') }}</text>
        <input
          v-model="newPassword"
          class="change-password-card__input"
          password
          autocomplete="new-password"
          :placeholder="t('system.settings.password_new_placeholder')"
          @input="errorMessage = ''"
        />
        <view class="change-password-strength">
          <view class="change-password-strength__header">
            <text>{{ t('system.settings.password_strength') }}</text>
            <text :class="'change-password-strength__label--' + strengthLevel">
              {{ t(strengthLabelKey) }}
            </text>
          </view>
          <view class="change-password-strength__bars">
            <view
              v-for="segment in [1, 2, 3]"
              :key="segment"
              class="change-password-strength__bar"
              :class="{
                'change-password-strength__bar--active': segment <= strengthScore,
                ['change-password-strength__bar--' + strengthLevel]: segment <= strengthScore,
              }"
            />
          </view>
          <text class="change-password-strength__tip">{{ policyTip }}</text>
        </view>
        <text class="change-password-card__label">{{ t('system.settings.password_confirm') }}</text>
        <input
          v-model="confirmPassword"
          class="change-password-card__input"
          password
          autocomplete="new-password"
          :placeholder="t('system.settings.password_confirm_placeholder')"
          @confirm="handleSubmit"
          @input="errorMessage = ''"
        />
        <text v-if="policyLoading" class="change-password-card__hint">{{
          t('system.settings.password_policy_loading')
        }}</text>
        <text v-if="errorMessage" class="change-password-card__error">{{ errorMessage }}</text>
      </view>
      <button class="k-save change-password-page__submit" :loading="submitting" @tap="handleSubmit">
        {{ t('system.settings.password_submit') }}
      </button>
    </view>
  </view>
</template>

<style scoped lang="scss">
.change-password-page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 0 20rpx var(--kratos-tab-bar-space, calc(104rpx + env(safe-area-inset-bottom)));
  background: var(--kratos-bg-gradient);
}
.change-password-page__content {
  padding-bottom: 20rpx;
}
.change-password-card {
  padding: 32rpx;
}
.change-password-card__description,
.change-password-card__label,
.change-password-card__hint,
.change-password-card__error,
.change-password-strength__tip {
  display: block;
}
.change-password-card__description {
  margin-bottom: 28rpx;
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
  line-height: 1.6;
}
.change-password-card__label {
  margin: 24rpx 0 12rpx;
  color: var(--kratos-color-text);
  font-size: 28rpx;
  font-weight: 600;
}
.change-password-card__label:first-of-type {
  margin-top: 0;
}
.change-password-card__input {
  box-sizing: border-box;
  width: 100%;
  height: 84rpx;
  padding: 0 24rpx;
  color: var(--kratos-color-text);
  font-size: 30rpx;
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 8rpx;
  background: var(--kratos-color-surface-2);
}
.change-password-strength {
  margin-top: 20rpx;
  padding: 18rpx 20rpx;
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 10rpx;
  background: var(--kratos-color-surface-2);
}
.change-password-strength__header {
  display: flex;
  justify-content: space-between;
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
}
.change-password-strength__label--empty {
  color: var(--kratos-color-text-muted);
}
.change-password-strength__label--low {
  color: var(--kratos-color-danger);
}
.change-password-strength__label--medium {
  color: var(--kratos-color-warning);
}
.change-password-strength__label--high {
  color: var(--kratos-color-primary);
}
.change-password-strength__bars {
  display: flex;
  gap: 10rpx;
  margin-top: 14rpx;
}
.change-password-strength__bar {
  flex: 1;
  height: 10rpx;
  border-radius: 6rpx;
  background: var(--kratos-color-border);
}
.change-password-strength__bar--low {
  background: var(--kratos-color-danger);
}
.change-password-strength__bar--medium {
  background: var(--kratos-color-warning);
}
.change-password-strength__bar--high {
  background: var(--kratos-color-primary);
}
.change-password-strength__tip {
  margin-top: 12rpx;
  color: var(--kratos-color-text-muted);
  font-size: 22rpx;
  line-height: 1.6;
}
.change-password-card__hint,
.change-password-card__error {
  margin-top: 12rpx;
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
  line-height: 1.5;
}
.change-password-card__error {
  color: var(--kratos-color-danger);
}
.change-password-page__submit {
  margin-top: 32rpx;
}
</style>
