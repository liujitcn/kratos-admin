<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../locales'
import { useThemeStore } from '../stores'
import { getMpCapsuleRightGap } from '../utils'
import { homeTabPage } from '../utils/navigation'
import type { KratosNavBarProps } from '../types/nav-bar'

defineOptions({ name: 'KratosNavBar' })

const props = withDefaults(defineProps<KratosNavBarProps>(), {
  variant: 'page',
  back: undefined,
})
const { t } = useI18n()
const themeStore = useThemeStore()
const capsuleGap = getMpCapsuleRightGap()
const showBack = computed(() => props.back ?? props.variant === 'page')
const iconPaths: Record<string, string> = {
  settings:
    '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8 2 2 0 1 1-2.8 2.8 1.6 1.6 0 0 0-2.7 1.1 2 2 0 1 1-4 0 1.6 1.6 0 0 0-2.6-1.1 2 2 0 1 1-2.8-2.8A1.6 1.6 0 0 0 3 13.9a2 2 0 1 1 0-4 1.6 1.6 0 0 0 1.5-2.7 2 2 0 1 1 2.8-2.8A1.6 1.6 0 0 0 10 4.6a2 2 0 1 1 4 0 1.6 1.6 0 0 0 2.7.6 2 2 0 1 1 2.8 2.8 1.6 1.6 0 0 0-.1 2.6a2.2 2.2 0 1 1 0 4.4Z"/>',
  'read-all': '<path d="m3 12 4 4L17 6m-5 10 9-9"/>',
  history: '<path d="M4 6h16M4 12h16M4 18h16"/>',
}
const actionIcon = computed(() => {
  const icon = props.action?.icon
  if (!icon || !iconPaths[icon]) return icon
  const color = themeStore.theme === 'light' ? '#102232' : '#e9f4f6'
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${iconPaths[icon]}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
})
const actionDisplay = computed(() => props.action?.display ?? (actionIcon.value ? 'icon' : 'text'))
const showActionIcon = computed(() => !!actionIcon.value && actionDisplay.value !== 'text')
const showActionLabel = computed(() => !actionIcon.value || actionDisplay.value !== 'icon')

/** goBack 优先执行页面返回操作，否则返回上一页并在失败时回到首页。 */
function goBack() {
  if (props.onBack) {
    props.onBack()
    return
  }
  uni.navigateBack({ fail: () => uni.reLaunch({ url: homeTabPage }) })
}
</script>

<template>
  <view
    class="kratos-nav-bar"
    :class="{ 'kratos-nav-bar--tab': variant === 'tab' }"
    :style="{ '--kratos-nav-bar-capsule-gap': `${capsuleGap}px` }"
  >
    <view class="kratos-nav-bar__row">
      <view
        v-if="showBack"
        class="kratos-nav-bar__button kratos-nav-bar__back"
        :aria-label="t('common.action.back')"
        @tap="goBack"
      />
      <view class="kratos-nav-bar__title">{{ title }}</view>
      <view
        v-if="action"
        class="kratos-nav-bar__button kratos-nav-bar__action"
        :class="{ 'kratos-nav-bar__action--label': showActionLabel }"
        :aria-label="action.label"
        @tap="action.onClick"
      >
        <image
          v-if="showActionIcon"
          class="kratos-nav-bar__icon"
          :src="actionIcon"
          mode="aspectFit"
          aria-hidden="true"
        />
        <text v-if="showActionLabel" class="kratos-nav-bar__label">{{ action.label }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.kratos-nav-bar {
  flex-shrink: 0;
  box-sizing: border-box;
  padding-top: calc(env(safe-area-inset-top) + 24rpx);
}

.kratos-nav-bar--tab {
  margin-bottom: 16rpx;
}

.kratos-nav-bar__row {
  position: relative;
  display: flex;
  align-items: center;
  gap: 16rpx;
  height: 76rpx;
}

.kratos-nav-bar__title {
  position: absolute;
  top: 50%;
  left: 50%;
  max-width: calc(100% - 190rpx);
  overflow: hidden;
  color: var(--kratos-color-text);
  font-size: 32rpx;
  font-weight: 700;
  line-height: 44rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
  transform: translate(-50%, -50%);
}

.kratos-nav-bar__button {
  display: flex;
  flex: 0 0 76rpx;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  width: 76rpx;
  height: 76rpx;
  border: 1px solid var(--kratos-color-border);
  border-radius: 24rpx;
  background: var(--kratos-color-surface);
  color: var(--kratos-color-text);
}

.kratos-nav-bar__back::before {
  width: 16rpx;
  height: 16rpx;
  border-left: 2px solid var(--kratos-color-primary);
  border-bottom: 2px solid var(--kratos-color-primary);
  content: '';
  transform: rotate(45deg);
}

.kratos-nav-bar__action {
  margin-left: auto;
}

.kratos-nav-bar__action--label {
  flex: 0 1 auto;
  gap: 10rpx;
  min-width: 76rpx;
  max-width: calc(50% - 48rpx);
  width: auto;
  padding: 0 20rpx;
}

.kratos-nav-bar__icon {
  flex-shrink: 0;
  width: 34rpx;
  height: 34rpx;
}

.kratos-nav-bar__label {
  min-width: 0;
  overflow: hidden;
  font-size: 22rpx;
  line-height: 28rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* #ifdef MP-WEIXIN */
.kratos-nav-bar {
  padding-top: 44px;
}

.kratos-nav-bar__row {
  height: 88rpx;
}

/* 原生胶囊占据右侧，操作按钮与返回按钮统一放在左侧。 */
.kratos-nav-bar__action {
  margin-left: 0;
}

.kratos-nav-bar__title {
  max-width: calc(100% - 190rpx - var(--kratos-nav-bar-capsule-gap));
}
/* #endif */
</style>
