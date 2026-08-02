<script setup lang="ts">
import { useI18n } from '../locales'

defineProps<{ title?: string; detail?: string }>()
const { t } = useI18n()
</script>

<template>
  <view class="bootstrap-status">
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
  background: #f8f8f8;
  box-sizing: border-box;
}
.bootstrap-status__logo {
  width: 112rpx;
  height: 112rpx;
  margin-bottom: 32rpx;
}
.bootstrap-status__title {
  color: #222;
  font-size: 32rpx;
}
.bootstrap-status__detail {
  margin-top: 16rpx;
  color: #777;
  font-size: 24rpx;
  text-align: center;
}
</style>
