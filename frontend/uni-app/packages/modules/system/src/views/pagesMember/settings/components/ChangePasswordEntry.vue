<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore, useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import { navigateAppRoute } from '@liujitcn/kratos-uni-app-core'

const { t } = useI18n()
const themeStore = useThemeStore()
const userStore = useUserStore()

const passwordIcon = computed(() => {
  const color = themeStore.colors.warning
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="10" width="16" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/><path d="M12 14v2"/></svg>`)}`
})

/** 跳转到独立的修改密码页面。 */
function openPasswordPage() {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  navigateAppRoute('app/password')
}
</script>

<template>
  <view
    v-if="userStore.isAuthenticated()"
    class="k-set-row k-set-row--arrow"
    @tap="openPasswordPage"
  >
    <view class="k-set-row__lead">
      <view class="k-set-row__icon">
        <image class="change-password-entry__icon" :src="passwordIcon" mode="aspectFit" />
      </view>
      <text class="k-set-row__title">{{ t('system.settings.password_title') }}</text>
    </view>
    <text class="k-set-row__value settings-row__value">{{
      t('system.settings.password_action')
    }}</text>
  </view>
</template>

<style scoped lang="scss">
.change-password-entry__icon {
  width: 38rpx;
  height: 38rpx;
}
</style>
