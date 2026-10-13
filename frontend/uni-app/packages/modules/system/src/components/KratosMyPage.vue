<script setup lang="ts">
import { useSettingStore, useThemeStore, useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import { computed } from 'vue'
import { formatSrc, getMpCapsuleRightGap } from '@liujitcn/kratos-uni-app-core/utils/index'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import { navigateAppRoute } from '@liujitcn/kratos-uni-app-core'
import defaultAvatar from '@liujitcn/kratos-uni-app-core/static/images/avatar.png'
import { useI18n } from '@liujitcn/kratos-uni-app-core'

defineOptions({ name: 'KratosMyPage' })

defineSlots<{
  /** 在个人信息和 AI 助手之后渲染的项目业务内容。 */
  extensions?: () => unknown
}>()

// 公共个人中心统一复用用户、设置和主题状态。
const userStore = useUserStore()
const settingStore = useSettingStore()
const themeStore = useThemeStore()
const { t } = useI18n()
const isLoggedIn = computed(() => userStore.isAuthenticated())
const profile = computed(() => userStore.userInfo)

/** MP-WEIXIN 原生胶囊按钮右侧避让宽度，写入 --mp-capsule-gap 供 .k-top 预留空间。 */
const capsuleGap = getMpCapsuleRightGap()

/** 昵称或占位账号首字符，用于渐变头像兜底。 */
const avatarText = computed(() => (profile.value?.nick_name || 'U').slice(0, 1))

/** 设计稿齿轮线稿，随主题取色的 data URI，替代 CSS 手绘图标。 */
const gearIcon = computed(() => {
  const color = themeStore.theme === 'light' ? '#102232' : '#e9f4f6'
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8 2 2 0 1 1-2.8 2.8 1.6 1.6 0 0 0-2.7 1.1 2 2 0 1 1-4 0 1.6 1.6 0 0 0-2.6-1.1 2 2 0 1 1-2.8-2.8A1.6 1.6 0 0 0 3 13.9a2 2 0 1 1 0-4 1.6 1.6 0 0 0 1.5-2.7 2 2 0 1 1 2.8-2.8A1.6 1.6 0 0 0 10 4.6a2 2 0 1 1 4 0 1.6 1.6 0 0 0 2.7.6 2 2 0 1 1 2.8 2.8 1.6 1.6 0 0 0-.1 2.6a2.2 2.2 0 1 1 0 4.4Z"/></svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
})

/** 个人卡账号行：登录展示账号，未登录提示去登录。 */
const accountText = computed(() => {
  if (isLoggedIn.value && profile.value) {
    return profile.value.user_name || profile.value.nick_name || ''
  }
  return t('system.profile.login_prompt')
})

/** 打开移动端 AI 助手静态页，未登录时先进入登录流程。 */
const navigateToAi = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  navigateAppRoute('app/ai')
}

/** 打开设置页，未登录时先进入登录流程。 */
const navigateToSettings = () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  navigateAppRoute('app/settings')
}

/** 打开当前用户资料。 */
const navigateToProfile = () => navigateAppRoute('app/profile')
</script>

<template>
  <scroll-view enable-back-to-top class="my-viewport" scroll-y :class="themeStore.themeClass">
    <view class="my-content">
      <!-- 顶栏：标题 + 设置入口（Tab 根页，无返回） -->
      <view class="k-top" :style="{ '--mp-capsule-gap': `${capsuleGap}px` }">
        <view class="k-top__title">{{ t('core.navigation.my') }}</view>
        <view class="k-ic" @tap="navigateToSettings">
          <image class="my-gear" :src="gearIcon" mode="aspectFit" />
        </view>
      </view>

      <view class="my-stack">
        <!-- 个人资料 -->
        <view class="k-glass k-glass--lit k-pfx my-profile" @tap="navigateToProfile">
          <view class="my-pfx-main">
            <image
              v-if="isLoggedIn && profile"
              class="k-pfx__av"
              :src="profile.avatar ? formatSrc(profile.avatar) : defaultAvatar"
              mode="aspectFill"
            ></image>
            <view v-else class="k-pfx__av">{{ avatarText }}</view>
            <view class="my-pfx-meta">
              <view class="k-pfx__nm">
                {{ isLoggedIn && profile ? profile.nick_name : t('system.profile.not_logged_in') }}
              </view>
              <view class="k-pfx__ac">{{ accountText }}</view>
            </view>
          </view>
          <view class="k-pfx__ent">{{ t('system.profile.title') }} ›</view>
        </view>

        <!-- AI 助手入口 -->
        <view v-if="settingStore.aiEnabled" class="k-aix" @tap="navigateToAi">
          <view class="k-aix__ic">AI</view>
          <view class="my-aix-copy">
            <view class="k-aix__t">{{ t('system.settings.ai_title') }}</view>
            <view class="k-aix__d">{{ t('system.settings.ai_description') }}</view>
          </view>
          <view class="k-aix__b">{{ t('system.settings.go_ask') }}</view>
        </view>

        <slot name="extensions" />
      </view>
    </view>
  </scroll-view>
</template>

<style lang="scss">
.my-viewport {
  height: 100%;
  min-height: 100vh;
  box-sizing: border-box;
  background: var(--kratos-bg-gradient);
}

.my-content {
  padding: calc(env(safe-area-inset-top) + 24rpx) 32rpx calc(40rpx + env(safe-area-inset-bottom));

  /* #ifdef MP-WEIXIN */
  /* k-top 在小程序端自带 44px 状态栏高度的导航带，页面不再额外让出顶部空间 */
  padding-top: 0;
  /* #endif */
}

.my-stack {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
  margin-top: 16rpx;
}

.my-gear {
  width: 34rpx;
  height: 34rpx;
}

/* k-pfx 仅定义 flex 排布，卡片内边距交由页面控制 */
.my-profile {
  padding: 36rpx 30rpx;
}

.my-pfx-main {
  display: flex;
  align-items: center;
  gap: 24rpx;
  min-width: 0;
}

.my-pfx-meta {
  min-width: 0;
}

/* k-aix 中部文案占位伸展 */
.my-aix-copy {
  flex: 1;
  min-width: 0;
}
</style>
