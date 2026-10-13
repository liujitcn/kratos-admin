<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import KratosNavBar from '../../../components/KratosNavBar.vue'
import { computed, ref, watch } from 'vue'
import { useSettingStore, useThemeStore } from '../../../stores'
import defaultLogo from '../../../static/images/logo_icon.png'
import { formatSrc } from '../../../utils'
import { useI18n } from '../../../locales'

const settingStore = useSettingStore()
const themeStore = useThemeStore()
const { t } = useI18n()
const mainTitle = computed(() => t('core.navigation.home'))
const subTitle = computed(() => settingStore.getData('subTitle') || t('core.home.sub_title'))
const configuredAppLogo = computed(() => {
  const configuredLogo = settingStore.getData('appLogo')
  return configuredLogo ? formatSrc(configuredLogo) : defaultLogo
})
const appLogo = ref(defaultLogo)

watch(
  configuredAppLogo,
  (value) => {
    appLogo.value = value
  },
  { immediate: true },
)

/** Logo 加载失败时回退到本地默认图片。 */
const handleLogoError = () => {
  if (appLogo.value !== defaultLogo) {
    appLogo.value = defaultLogo
  }
}

onLoad(() => {
  void settingStore.loadData().catch(() => undefined)
})
</script>

<template>
  <view class="home-page" :class="themeStore.themeClass">
    <view class="k-grid-bg" aria-hidden="true"></view>
    <view class="home-content">
      <KratosNavBar :title="mainTitle" variant="tab" />

      <view class="k-glass k-glass--lit home-hero">
        <view class="home-hero__main">
          <image class="home-logo" :src="appLogo" mode="aspectFit" @error="handleLogoError" />
          <view class="home-hero__copy">
            <text class="home-subtitle">{{ subTitle }}</text>
          </view>
        </view>
      </view>

      <view class="home-section">
        <text class="home-section__title">{{ t('core.home.tech_stack') }}</text>
        <view class="k-glass home-info-list">
          <view class="k-field">
            <text class="k-field__k">{{ t('core.home.framework') }}</text>
            <text class="k-field__v">uni-app + Vue 3</text>
          </view>
          <view class="k-field">
            <text class="k-field__k">{{ t('core.home.language') }}</text>
            <text class="k-field__v">TypeScript</text>
          </view>
          <view class="k-field">
            <text class="k-field__k">{{ t('core.home.state_management') }}</text>
            <text class="k-field__v">Pinia</text>
          </view>
          <view class="k-field">
            <text class="k-field__k">{{ t('core.home.style_solution') }}</text>
            <text class="k-field__v">Sass + rpx</text>
          </view>
        </view>
      </view>

      <view class="home-section">
        <text class="home-section__title">{{ t('core.home.demo_scope') }}</text>
        <view class="k-glass home-demo-list">
          <view class="home-demo-item">
            <view class="home-demo-dot">1</view>
            <view class="home-demo-copy">
              <text class="home-demo-title">{{ t('core.home.cross_platform') }}</text>
              <text class="home-demo-desc">{{ t('core.home.cross_platform_description') }}</text>
            </view>
          </view>
          <view class="home-demo-item">
            <view class="home-demo-dot">2</view>
            <view class="home-demo-copy">
              <text class="home-demo-title">{{ t('core.home.account_capability') }}</text>
              <text class="home-demo-desc">{{
                t('core.home.account_capability_description')
              }}</text>
            </view>
          </view>
          <view class="home-demo-item">
            <view class="home-demo-dot">3</view>
            <view class="home-demo-copy">
              <text class="home-demo-title">{{ t('core.home.message_center') }}</text>
              <text class="home-demo-desc">{{ t('core.home.message_center_description') }}</text>
            </view>
          </view>
          <view class="home-demo-item">
            <view class="home-demo-dot">4</view>
            <view class="home-demo-copy">
              <text class="home-demo-title">{{ t('core.home.dynamic_config') }}</text>
              <text class="home-demo-desc">{{ t('core.home.dynamic_config_description') }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.home-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background: var(--kratos-bg-gradient);
}

.home-content {
  position: relative;
  z-index: 1;
  padding: 0 20rpx 220rpx;
}

.home-hero {
  padding: 40rpx 36rpx;
}

.home-hero__main {
  display: flex;
  align-items: center;
}

/* 简洁 Logo：圆角方块 + 细描边 */
.home-logo {
  flex: 0 0 auto;
  width: 104rpx;
  height: 104rpx;
  border-radius: 26rpx;
  border: 1px solid var(--kratos-color-border-accent);
  background: var(--kratos-color-surface-2);
}

.home-hero__copy {
  min-width: 0;
  margin-left: 26rpx;
}

.home-subtitle {
  display: block;
  color: var(--kratos-color-text-muted);
  font-size: 26rpx;
  line-height: 1.6;
}

.home-section {
  margin-top: 36rpx;
}

.home-section__title {
  display: block;
  margin: 0 4rpx 16rpx;
  color: var(--kratos-color-text-muted);
  font-size: 26rpx;
  font-weight: 700;
}

.home-info-list {
  padding: 8rpx 28rpx;
}

.home-demo-list {
  padding: 10rpx 28rpx;
}

.home-demo-item {
  display: flex;
  align-items: flex-start;
  padding: 28rpx 0;
  border-top: 1px solid var(--kratos-color-border);

  &:first-child {
    border-top: 0;
  }
}

.home-demo-dot {
  flex: 0 0 auto;
  width: 48rpx;
  height: 48rpx;
  border-radius: 14rpx;
  background: var(--kratos-color-primary-soft);
  border: 1px solid var(--kratos-color-primary-edge);
  color: var(--kratos-color-primary);
  font-size: 24rpx;
  font-family: var(--kratos-font-mono);
  font-weight: 700;
  line-height: 46rpx;
  text-align: center;
}

.home-demo-copy {
  min-width: 0;
  margin-left: 22rpx;
}

.home-demo-title,
.home-demo-desc {
  display: block;
}

.home-demo-title {
  color: var(--kratos-color-text);
  font-size: 28rpx;
  font-weight: 600;
}

.home-demo-desc {
  margin-top: 8rpx;
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
  line-height: 1.5;
}
</style>
