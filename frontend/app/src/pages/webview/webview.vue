<template>
  <view class="webview-container">
    <!-- 小程序/APP 使用 web-view 组件 -->
    <!-- #ifdef MP-WEIXIN || MP-ALIPAY || APP-PLUS -->
    <web-view :src="url" @message="handleMessage"></web-view>
    <!-- #endif -->

    <!-- H5 直接跳转 -->
    <!-- #ifdef H5 -->
    <iframe
      v-if="isH5 && url"
      :src="url"
      frameborder="0"
      class="h5-iframe"
      @load="handleIframeLoad"
    ></iframe>
    <!-- #endif -->

    <view v-if="showFallback" class="webview-empty">
      <view class="webview-empty__title">链接无法打开</view>
      <view class="webview-empty__desc">{{ emptyDesc }}</view>
      <!-- #ifdef H5 -->
      <button v-if="url" class="webview-empty__button" @tap="openInBrowser">新窗口打开</button>
      <!-- #endif -->
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'

const url = ref('')
const isH5 = ref(false)
const isIframeLoaded = ref(false)
const isIframeTimedOut = ref(false)

const showFallback = computed(
  () => !url.value || (isH5.value && isIframeTimedOut.value && !isIframeLoaded.value),
)
const emptyDesc = computed(() => {
  if (!url.value) {
    return '缺少有效链接地址'
  }
  return '当前 H5 页面可能被目标站点限制嵌入'
})

onLoad((query) => {
  url.value = decodeURIComponent(query?.url || '')
  const title = decodeURIComponent(query?.title || '')
  // 调用方传入标题时同步更新原生导航栏，避免继续显示页面注册时的空标题。
  if (title) {
    void uni.setNavigationBarTitle({ title })
  }

  // #ifdef H5
  isH5.value = true
  // H5 iframe 可能被 X-Frame-Options 拦截，超时后展示明确兜底。
  window.setTimeout(() => {
    isIframeTimedOut.value = true
  }, 800)
  // #endif
})

/** 标记 H5 iframe 已完成加载，避免正常外链被兜底层遮挡。 */
const handleIframeLoad = () => {
  isIframeLoaded.value = true
}

/** H5 下外链被禁止嵌入时，允许用户用新窗口继续访问。 */
const openInBrowser = () => {
  if (!url.value) return
  // #ifdef H5
  window.open(url.value, '_blank', 'noopener,noreferrer')
  // #endif
}

// 小程序接收消息
const handleMessage = (e: any) => {
  console.log('收到H5消息:', e.detail)
}
</script>

<style scoped>
.webview-container {
  position: relative;
  flex: 1;
  height: 100vh;
}
.h5-iframe {
  width: 100%;
  height: 100vh;
}

.webview-empty {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48rpx;
  text-align: center;
  background-color: #fff;
}

.webview-empty__title {
  font-size: 32rpx;
  font-weight: 600;
  color: #333;
}

.webview-empty__desc {
  margin-top: 16rpx;
  font-size: 26rpx;
  color: #888;
}

.webview-empty__button {
  margin-top: 36rpx;
  width: 240rpx;
  height: 72rpx;
  line-height: 72rpx;
  border-radius: 72rpx;
  background-color: #27ba9b;
  color: #fff;
  font-size: 26rpx;
}
</style>
