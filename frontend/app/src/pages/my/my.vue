<script setup lang="ts">
import { useUserStore } from '@/stores'
import { computed } from 'vue'
import { formatSrc } from '@/utils'
import { navigateToLogin } from '@/utils/navigation'
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
const AI_PAGE = '/pagesMember/ai/index'
const SETTINGS_PAGE = '/pagesMember/settings/settings'

// 获取会员信息
const userStore = useUserStore()
const isLoggedIn = computed(() => userStore.isAuthenticated())
const profile = computed(() => userStore.userInfo)

/** 打开移动端 AI 助手静态页，未登录时先进入登录流程。 */
const navigateToAi = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  uni.navigateTo({
    url: AI_PAGE,
  })
}

/** 打开设置页，未登录时先进入登录流程。 */
const navigateToSettings = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  uni.navigateTo({
    url: SETTINGS_PAGE,
  })
}
</script>

<template>
  <scroll-view enable-back-to-top class="viewport" scroll-y>
    <!-- 个人资料 -->
    <view class="profile" :style="{ paddingTop: safeAreaInsets!.top + 'px' }">
      <!-- 情况1：已登录 -->
      <view class="overview" v-if="isLoggedIn && profile">
        <navigator url="/pagesMember/profile/profile" hover-class="none">
          <image
            v-if="profile.avatar"
            class="avatar"
            :src="formatSrc(profile.avatar)"
            mode="aspectFill"
          ></image>
          <image v-else class="avatar" src="@/static/images/avatar.png" mode="aspectFill"></image>
        </navigator>
        <view class="meta">
          <view class="nickname">
            {{ profile.nick_name }}
          </view>
          <navigator class="extra" url="/pagesMember/profile/profile" hover-class="none">
            <text class="update">更新头像昵称</text>
          </navigator>
        </view>
      </view>
      <!-- 情况2：未登录 -->
      <view class="overview" v-else>
        <view @tap="navigateToLogin">
          <image class="avatar gray" mode="aspectFill" src="@/static/images/avatar.png"></image>
        </view>
        <view class="meta">
          <view @tap="navigateToLogin" class="nickname"> 未登录 </view>
          <view class="extra">
            <text class="tips">点击登录账号</text>
          </view>
        </view>
      </view>
      <view class="settings" @tap="navigateToSettings">设置</view>
    </view>
    <!-- AI 助手入口 -->
    <view class="ai-entry" @tap="navigateToAi">
      <view class="ai-entry__icon">AI</view>
      <view class="ai-entry__content">
        <view class="ai-entry__title">智能助手</view>
        <view class="ai-entry__desc">帮你整理信息、回答问题并处理日常任务</view>
      </view>
      <view class="ai-entry__action">去提问</view>
    </view>
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f7f7f8;
}

/* AI 助手入口 */
.ai-entry {
  position: relative;
  z-index: 99;
  display: flex;
  align-items: center;
  margin: 20rpx 20rpx 0;
  padding: 26rpx 24rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 4rpx 6rpx rgba(240, 240, 240, 0.6);
}

.ai-entry__icon {
  flex-shrink: 0;
  width: 92rpx;
  height: 92rpx;
  border-radius: 24rpx;
  color: #fff;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 92rpx;
  text-align: center;
  background-color: #27ba9b;
}

.ai-entry__content {
  flex: 1;
  min-width: 0;
  margin-left: 18rpx;
}

.ai-entry__title {
  color: #1e1e1e;
  font-size: 30rpx;
  font-weight: 600;
  line-height: 40rpx;
}

.ai-entry__desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 8rpx;
  color: #747f7c;
  font-size: 24rpx;
  line-height: 32rpx;
}

.ai-entry__action {
  flex-shrink: 0;
  margin-left: 16rpx;
  padding: 10rpx 18rpx;
  border-radius: 999rpx;
  color: #16806d;
  font-size: 24rpx;
  background-color: #e8f8f4;
}

.viewport {
  height: 100%;
  background-repeat: no-repeat;
  background-image: url(@/static/images/center_bg.png);
  background-size: 100% auto;
}

/* 用户信息 */
.profile {
  margin-top: 30rpx;
  position: relative;

  .overview {
    display: flex;
    height: 120rpx;
    padding: 0 36rpx;
    color: #fff;
  }

  .avatar {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background-color: #eee;
  }

  .gray {
    filter: grayscale(100%);
  }

  .meta {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: flex-start;
    line-height: 30rpx;
    padding: 16rpx 0;
    margin-left: 20rpx;
  }

  .nickname {
    max-width: 180rpx;
    margin-bottom: 16rpx;
    font-size: 30rpx;

    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .extra {
    display: flex;
    font-size: 20rpx;
  }

  .tips {
    font-size: 22rpx;
  }

  .update {
    padding: 3rpx 10rpx 1rpx;
    color: rgba(255, 255, 255, 0.8);
    border: 1rpx solid rgba(255, 255, 255, 0.8);
    margin-right: 10rpx;
    border-radius: 30rpx;
  }

  .settings {
    position: absolute;
    bottom: 0;
    right: 40rpx;
    font-size: 30rpx;
    color: #fff;
  }
}
</style>
