<script setup lang="ts">
import { useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import { getLanguageOptions, useI18n, type SupportedLocale } from '@liujitcn/kratos-uni-app-core'
import { computed, watch } from 'vue'

const userStore = useUserStore()
const { locale, setLocale, t } = useI18n()
const logoutLoading = ref(false)
const localeOptions = computed(() => getLanguageOptions())
const localeLabels = computed(() => localeOptions.value.map((item) => item.language_name))
const localeIndex = computed(() =>
  Math.max(
    0,
    localeOptions.value.findIndex((item) => item.language_code === locale.value),
  ),
)

watch(locale, () => void uni.setNavigationBarTitle({ title: t('system.settings.title') }), {
  immediate: true,
})

/** 处理语言选择并刷新动态本地化数据。 */
const onLocaleChange = (event: { detail: { value: string | number } }) => {
  const nextLocale = localeOptions.value[Number(event.detail.value)]?.language_code as
    | SupportedLocale
    | undefined
  if (nextLocale) void setLocale(nextLocale)
}

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
    content: t('system.settings.logoutConfirm'),
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
          title: t('system.settings.logoutFailed'),
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
      <button hover-class="none" class="item arrow" open-type="openSetting">
        {{ t('system.settings.authorization') }}
      </button>
      <button hover-class="none" class="item arrow" open-type="feedback">
        {{ t('system.settings.feedback') }}
      </button>
      <button hover-class="none" class="item arrow" open-type="contact">
        {{ t('system.settings.contact') }}
      </button>
    </view>
    <!-- #endif -->
    <view class="list">
      <picker :range="localeLabels" :value="localeIndex" @change="onLocaleChange">
        <view class="item arrow locale-item">
          <text>{{ t('common.field.language') }}</text>
          <text class="locale-value">{{ localeLabels[localeIndex] }}</text>
        </view>
      </picker>
    </view>
    <!-- 操作按钮 -->
    <view class="action" v-if="userStore.isAuthenticated()">
      <view @tap="onLogout" class="button">{{ t('common.action.logout') }}</view>
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

.locale-item {
  display: flex;
  justify-content: space-between;
  padding-right: 42rpx;
}

.locale-value {
  color: #888;
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
