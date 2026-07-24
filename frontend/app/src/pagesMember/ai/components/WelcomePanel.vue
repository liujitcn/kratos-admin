<script setup lang="ts">
import type { AiShortcut } from '@/rpc/base/v1/ai_tool'

defineProps<{
  greetingMessage: string
  loading: boolean
  shortcuts: AiShortcut[]
  canRefresh: boolean
}>()

const emit = defineEmits<{
  refresh: []
  'shortcut-tap': [shortcut: AiShortcut]
}>()
</script>

<template>
  <view class="welcome-panel">
    <view class="welcome-row is-hello">
      <view class="ai-avatar">
        <view class="ai-avatar__halo"></view>
        <view class="ai-avatar__hair-back"></view>
        <view class="ai-avatar__face">
          <view class="ai-avatar__bang"></view>
          <view class="ai-avatar__eyes">
            <view></view>
            <view></view>
          </view>
          <view class="ai-avatar__blush is-left"></view>
          <view class="ai-avatar__blush is-right"></view>
          <view class="ai-avatar__smile"></view>
        </view>
        <view class="ai-avatar__hair-side is-left"></view>
        <view class="ai-avatar__hair-side is-right"></view>
        <view class="ai-avatar__body"></view>
        <view class="ai-avatar__bow"></view>
        <view class="ai-avatar__spark"></view>
      </view>
      <view class="welcome-bubble is-hello">您好，AI助手为您服务！</view>
    </view>
    <view class="welcome-bubble is-intro">{{ greetingMessage }}</view>

    <view class="prompt-card">
      <view class="prompt-card__head">
        <view>
          <view class="prompt-card__eyebrow">快捷操作</view>
          <view class="prompt-card__title">您可以这样问</view>
        </view>
        <button v-if="canRefresh" class="prompt-refresh" hover-class="none" @tap="emit('refresh')">
          <text>换一换</text>
          <uni-icons type="refresh" size="25" color="#00a96b" />
        </button>
      </view>
      <view v-if="loading" class="prompt-loading">正在加载...</view>
      <view v-else-if="!shortcuts.length" class="prompt-loading">暂无可用快捷助手</view>
      <template v-else>
        <button
          v-for="(shortcut, shortcutIndex) in shortcuts"
          :key="shortcut.key || shortcut.title"
          class="prompt-item"
          hover-class="none"
          @tap="emit('shortcut-tap', shortcut)"
        >
          <text class="prompt-index">{{ shortcutIndex + 1 }}</text>
          <view class="prompt-content">
            <text class="prompt-text">{{ shortcut.title }}</text>
            <text class="prompt-meta">{{ shortcut.group || '通用助手' }}</text>
          </view>
          <uni-icons type="right" size="20" color="#9aa0aa" />
        </button>
      </template>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.prompt-refresh,
.prompt-item {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.welcome-panel {
  padding-bottom: 32rpx;
}

.welcome-row {
  display: flex;
  align-items: center;
}

.welcome-row.is-hello {
  margin-left: 6rpx;
}

.ai-avatar {
  position: relative;
  z-index: 1;
  flex-shrink: 0;
  width: 114rpx;
  height: 114rpx;
  margin-right: -42rpx;
  overflow: hidden;
  border-radius: 34rpx;
  background: linear-gradient(180deg, #fff 0%, #f5f8fb 100%);
  box-shadow: 0 10rpx 26rpx rgba(15, 23, 42, 0.08);
  box-sizing: border-box;
  animation: ai-avatar-float 3.8s ease-in-out infinite;
}

.ai-avatar__halo {
  position: absolute;
  top: 8rpx;
  left: 16rpx;
  width: 82rpx;
  height: 82rpx;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 226, 218, 0.82) 0%, rgba(255, 255, 255, 0) 66%);
  animation: ai-avatar-glow 2.8s ease-in-out infinite;
}

.ai-avatar__hair-back {
  position: absolute;
  top: 15rpx;
  left: 28rpx;
  width: 58rpx;
  height: 66rpx;
  border-radius: 34rpx 34rpx 26rpx 26rpx;
  background: linear-gradient(160deg, #6b3a33 0%, #281c22 88%);
}

.ai-avatar__face {
  position: absolute;
  top: 27rpx;
  left: 31rpx;
  z-index: 2;
  width: 52rpx;
  height: 55rpx;
  border-radius: 24rpx 24rpx 26rpx 26rpx;
  background: linear-gradient(180deg, #ffe5d6 0%, #ffd3c2 100%);
  box-shadow: inset 0 -3rpx 0 rgba(219, 119, 103, 0.12);
}

.ai-avatar__bang {
  position: absolute;
  top: -12rpx;
  left: 0;
  width: 56rpx;
  height: 25rpx;
  border-radius: 28rpx 26rpx 18rpx 12rpx;
  background: linear-gradient(145deg, #5b342f 0%, #2a1f27 100%);
  transform: rotate(-5deg);
}

.ai-avatar__eyes {
  display: flex;
  justify-content: space-between;
  width: 26rpx;
  margin: 22rpx auto 0;
}

.ai-avatar__eyes view {
  width: 6rpx;
  height: 9rpx;
  border-radius: 50%;
  background-color: #38262b;
  animation: ai-avatar-blink 4.6s ease-in-out infinite;
}

.ai-avatar__blush {
  position: absolute;
  top: 34rpx;
  width: 10rpx;
  height: 5rpx;
  border-radius: 50%;
  background-color: rgba(246, 121, 118, 0.32);
}

.ai-avatar__blush.is-left {
  left: 9rpx;
}

.ai-avatar__blush.is-right {
  right: 9rpx;
}

.ai-avatar__smile {
  width: 18rpx;
  height: 9rpx;
  margin: 8rpx auto 0;
  border-bottom: 3rpx solid #d56f62;
  border-radius: 0 0 18rpx 18rpx;
}

.ai-avatar__hair-side {
  position: absolute;
  top: 42rpx;
  z-index: 1;
  width: 15rpx;
  height: 39rpx;
  border-radius: 16rpx;
  background: linear-gradient(180deg, #4d2d2b 0%, #231b23 100%);
}

.ai-avatar__hair-side.is-left {
  left: 22rpx;
  transform: rotate(9deg);
}

.ai-avatar__hair-side.is-right {
  right: 22rpx;
  transform: rotate(-9deg);
}

.ai-avatar__body {
  position: absolute;
  left: 29rpx;
  bottom: 4rpx;
  z-index: 1;
  width: 56rpx;
  height: 34rpx;
  border-radius: 26rpx 26rpx 10rpx 10rpx;
  background: linear-gradient(180deg, #ff5d58 0%, #df292d 100%);
}

.ai-avatar__bow {
  position: absolute;
  left: 46rpx;
  bottom: 24rpx;
  z-index: 3;
  width: 22rpx;
  height: 12rpx;
  border-radius: 12rpx;
  background: linear-gradient(90deg, #fff 0 40%, #f7dde0 40% 60%, #fff 60% 100%);
}

.ai-avatar__spark {
  position: absolute;
  top: 14rpx;
  right: 18rpx;
  z-index: 3;
  width: 10rpx;
  height: 10rpx;
  border-radius: 50%;
  background-color: #ffd35a;
  opacity: 0.9;
  animation: ai-avatar-spark 1.8s ease-in-out infinite;
}

@keyframes ai-avatar-float {
  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-4rpx);
  }
}

@keyframes ai-avatar-glow {
  0%,
  100% {
    opacity: 0.72;
    transform: scale(1);
  }

  50% {
    opacity: 1;
    transform: scale(1.04);
  }
}

@keyframes ai-avatar-blink {
  0%,
  88%,
  100% {
    transform: scaleY(1);
  }

  92%,
  95% {
    transform: scaleY(0.18);
  }
}

@keyframes ai-avatar-spark {
  0%,
  100% {
    opacity: 0.45;
    transform: scale(0.8);
  }

  50% {
    opacity: 1;
    transform: scale(1.18);
  }
}

.welcome-bubble {
  color: #111;
  background-color: #fff;
  box-shadow: 0 10rpx 32rpx rgba(15, 23, 42, 0.04);
  box-sizing: border-box;
}

.welcome-bubble.is-hello {
  min-width: 350rpx;
  height: 96rpx;
  padding: 0 34rpx 0 78rpx;
  border-radius: 18rpx;
  font-size: 30rpx;
  line-height: 96rpx;
}

.welcome-bubble.is-intro {
  display: flex;
  align-items: center;
  min-height: 96rpx;
  margin-top: 30rpx;
  padding: 24rpx 28rpx;
  border-radius: 18rpx;
  font-size: 29rpx;
  line-height: 44rpx;
}

.prompt-card {
  margin-top: 34rpx;
  padding: 30rpx 28rpx 24rpx;
  border: 1rpx solid #e7ecef;
  border-radius: 10rpx;
  background: linear-gradient(180deg, rgba(0, 169, 107, 0.05), rgba(255, 255, 255, 0) 38%), #fff;
  box-shadow: 0 10rpx 34rpx rgba(15, 23, 42, 0.05);
  box-sizing: border-box;
}

.prompt-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24rpx;
  margin-bottom: 18rpx;
}

.prompt-card__eyebrow {
  color: #00a96b;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 30rpx;
}

.prompt-card__title {
  color: #111;
  font-size: 31rpx;
  font-weight: 700;
  line-height: 42rpx;
}

.prompt-refresh {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 14rpx;
  height: 50rpx;
  color: #00a96b;
  font-size: 25rpx;
  line-height: 50rpx;
}

.prompt-loading {
  padding: 32rpx 0;
  color: #8d929c;
  font-size: 26rpx;
  line-height: 38rpx;
}

.prompt-item {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 96rpx;
  padding: 16rpx 16rpx 16rpx 0;
  border-top: 1rpx solid #edf0f2;
  color: #111;
  text-align: left;
  box-sizing: border-box;
}

.prompt-item:first-of-type {
  border-top: none;
}

.prompt-index {
  flex-shrink: 0;
  width: 36rpx;
  height: 36rpx;
  margin-right: 24rpx;
  border-radius: 8rpx;
  color: #00a96b;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 34rpx;
  text-align: center;
  background-color: #e7f7f2;
  box-sizing: border-box;
}

.prompt-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.prompt-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #111;
  font-size: 29rpx;
  line-height: 40rpx;
}

.prompt-meta {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #8d929c;
  font-size: 22rpx;
  line-height: 30rpx;
}

.prompt-item .uni-icons {
  flex-shrink: 0;
}
</style>
