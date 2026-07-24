<template>
  <view class="page">
    <view class="hero">
      <image class="logo" src="@/static/images/logo_icon.png" mode="aspectFit" />
      <view class="hero-copy">
        <text class="eyebrow">DEMO APP</text>
        <text class="title">应用框架示例</text>
        <text class="subtitle">保留通用导航与个人中心体验</text>
      </view>
    </view>

    <view class="section">
      <text class="section-title">当前技术栈</text>
      <view class="info-list">
        <view class="info-row">
          <text class="info-label">应用框架</text>
          <text class="info-value">uni-app + Vue 3</text>
        </view>
        <view class="info-row">
          <text class="info-label">开发语言</text>
          <text class="info-value">TypeScript</text>
        </view>
        <view class="info-row">
          <text class="info-label">状态管理</text>
          <text class="info-value">Pinia</text>
        </view>
        <view class="info-row">
          <text class="info-label">样式方案</text>
          <text class="info-value">Sass + rpx</text>
        </view>
      </view>
    </view>

    <view class="section">
      <text class="section-title">演示范围</text>
      <view class="demo-list">
        <view class="demo-item">
          <view class="demo-dot">1</view>
          <view class="demo-copy">
            <text class="demo-title">跨端页面</text>
            <text class="demo-desc">同一套页面代码适配 H5、微信小程序和 App</text>
          </view>
        </view>
        <view class="demo-item">
          <view class="demo-dot">2</view>
          <view class="demo-copy">
            <text class="demo-title">账户能力</text>
            <text class="demo-desc">个人资料、应用设置与智能助手入口已保留</text>
          </view>
        </view>
        <view class="demo-item">
          <view class="demo-dot">3</view>
          <view class="demo-copy">
            <text class="demo-title">静态首页</text>
            <text class="demo-desc">当前首页用于展示框架信息，不依赖业务接口</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f7f7f8;
}

.page {
  min-height: 100vh;
  padding: 44rpx 28rpx 56rpx;
  box-sizing: border-box;
}

.hero {
  display: flex;
  align-items: center;
  padding: 34rpx 28rpx;
  border-radius: 12rpx;
  background-color: #27ba9b;
  box-shadow: 0 8rpx 18rpx rgba(39, 186, 155, 0.18);
}

.logo {
  flex: 0 0 auto;
  width: 92rpx;
  height: 92rpx;
  border-radius: 20rpx;
  background-color: #fff;
}

.hero-copy {
  min-width: 0;
  margin-left: 22rpx;
}

.eyebrow,
.title,
.subtitle {
  display: block;
}

.eyebrow {
  color: rgba(255, 255, 255, 0.8);
  font-size: 20rpx;
  letter-spacing: 2rpx;
}

.title {
  margin-top: 8rpx;
  color: #fff;
  font-size: 40rpx;
  font-weight: 600;
}

.subtitle {
  margin-top: 10rpx;
  color: rgba(255, 255, 255, 0.86);
  font-size: 24rpx;
}

.section {
  margin-top: 24rpx;
}

.section-title {
  display: block;
  margin: 0 8rpx 14rpx;
  color: #26332f;
  font-size: 28rpx;
  font-weight: 600;
}

.info-list,
.demo-list {
  overflow: hidden;
  border-radius: 10rpx;
  background-color: #fff;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 26rpx 24rpx;
  border-bottom: 1rpx solid #edf1ef;
}

.info-row:last-child {
  border-bottom: 0;
}

.info-label {
  color: #66736f;
  font-size: 26rpx;
}

.info-value {
  color: #26332f;
  font-size: 26rpx;
  font-weight: 500;
}

.demo-item {
  display: flex;
  align-items: flex-start;
  padding: 26rpx 24rpx;
  border-bottom: 1rpx solid #edf1ef;
}

.demo-item:last-child {
  border-bottom: 0;
}

.demo-dot {
  flex: 0 0 auto;
  width: 42rpx;
  height: 42rpx;
  border-radius: 50%;
  background-color: #e8f8f4;
  color: #16806d;
  font-size: 24rpx;
  line-height: 42rpx;
  text-align: center;
}

.demo-copy {
  min-width: 0;
  margin-left: 18rpx;
}

.demo-title,
.demo-desc {
  display: block;
}

.demo-title {
  color: #26332f;
  font-size: 28rpx;
  font-weight: 500;
}

.demo-desc {
  margin-top: 8rpx;
  color: #8a9691;
  font-size: 24rpx;
  line-height: 1.5;
}
</style>
