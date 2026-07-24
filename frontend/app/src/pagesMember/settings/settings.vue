<script setup lang="ts">
import { useUserStore } from '@/stores'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateToLogin } from '@/utils/navigation'

const userStore = useUserStore()
const logoutLoading = ref(false)

// #ifndef MP-WEIXIN
// 非微信小程序端未登录时没有可用设置项，直接引导登录以避免显示空白页面。
onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
  }
})
// #endif

// 退出登录
const onLogout = () => {
  if (logoutLoading.value) {
    return
  }
  // 模态弹窗
  uni.showModal({
    content: '是否退出登录？',
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
          title: '退出登录失败',
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
    <!-- #ifdef MP-WEIXIN -->
    <!-- 列表2 -->
    <view class="list">
      <button hover-class="none" class="item arrow" open-type="openSetting">授权管理</button>
      <button hover-class="none" class="item arrow" open-type="feedback">问题反馈</button>
      <button hover-class="none" class="item arrow" open-type="contact">联系我们</button>
    </view>
    <!-- #endif -->
    <!-- 操作按钮 -->
    <view class="action" v-if="userStore.isAuthenticated()">
      <view @tap="onLogout" class="button">退出登录</view>
    </view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f4f4f4;
}

.viewport {
  padding: 20rpx;
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
    &:first-child {
      border: none;
    }
    &::after {
      right: 5rpx;
    }
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
