<script setup lang="ts">
import { useI18n } from '../locales'
import { useThemeStore } from '../stores'

defineProps<{ title?: string; detail?: string }>()
const { t } = useI18n()
const themeStore = useThemeStore()
</script>

<template>
  <view class="bootstrap-status" :class="themeStore.themeClass">
    <image class="bootstrap-status__logo" src="../static/images/logo_icon.png" mode="aspectFit" />
    <text class="bootstrap-status__title">{{ title || t('core.status.loading') }}</text>
    <text v-if="detail" class="bootstrap-status__detail">{{ detail }}</text>
  </view>
</template>

<style scoped>
.bootstrap-status {
  display: flex;
  min-height: 100vh;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 48rpx;
  background: var(--kratos-bg-gradient);
  box-sizing: border-box;
}
.bootstrap-status__logo {
  width: 112rpx;
  height: 112rpx;
  margin-bottom: 32rpx;
}
.bootstrap-status__title {
  color: var(--kratos-color-text);
  font-size: 32rpx;
}
.bootstrap-status__detail {
  margin-top: 16rpx;
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
  text-align: center;
}
</style>
