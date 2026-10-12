<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateAppView, t } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore } from '@liujitcn/kratos-uni-app-core/stores'
import { homeTabPage } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { MessageActionType, MessageContentFormat } from '../../../rpc/base/v1/notification'
import { resolveMessageCategoryIcon } from './icons'

const themeStore = useThemeStore()
const detail = ref<Notification>()

onLoad((options) => {
  const id = Number(options?.id)
  if (id > 0) void loadDetail(id)
})

/** 加载站内信详情。 */
async function loadDetail(id: number) {
  detail.value = await defNotificationService.GetNotification({ id })
  if (!detail.value.read_at) await defNotificationService.MarkNotificationRead({ ids: [id] })
}

/** 将系统通知发送者映射到当前语言。 */
function resolveSenderName(value?: string) {
  if (value === '系统' || value === '__I18N__:system.notification.sender.system') {
    return t('system.notification.sender.system')
  }
  return value ?? ''
}

/** 执行通知携带的稳定 viewKey 动作。 */
function openAction() {
  if (!detail.value || detail.value.action_type !== MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY)
    return
  let params: Record<string, string> = {}
  if (detail.value.action_params) {
    try {
      const value = JSON.parse(detail.value.action_params) as Record<string, unknown>
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        params = Object.fromEntries(
          Object.entries(value)
            .filter(([, item]) => typeof item === 'string' || typeof item === 'number')
            .map(([key, item]) => [key, String(item)]),
        )
      }
    } catch {
      params = {}
    }
  }
  navigateAppView(detail.value.action_target, params)
}

/** 返回消息列表，页面栈为空时回到首页。 */
function onBack() {
  uni.navigateBack({ fail: () => uni.reLaunch({ url: homeTabPage }) })
}
</script>

<template>
  <view class="detail-page" :class="themeStore.themeClass">
    <view class="k-page-nav">
      <view class="k-page-nav__back" :aria-label="t('common.action.back')" @tap="onBack"></view>
      <view class="k-page-nav__title">{{ t('system.notification.title') }}</view>
      <view class="k-page-nav__side" aria-hidden="true"></view>
    </view>
    <template v-if="detail">
      <text class="detail-title">{{ detail.title }}</text>
      <view class="detail-meta">
        <view class="detail-category" :style="{ color: detail.category_color || undefined }">
          <uni-icons
            class="detail-category__icon"
            :type="resolveMessageCategoryIcon(detail.category_icon)"
            size="18"
            :color="detail.category_color || '#64748b'"
            aria-hidden="true"
          />
          <text>{{ detail.category_name }}</text>
        </view>
        <text>{{ resolveSenderName(detail.sender_name) }}</text>
        <text>{{ detail.received_at }}</text>
      </view>
      <rich-text
        v-if="detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT"
        class="detail-content-rich"
        :nodes="detail.content"
      />
      <text v-else class="detail-content">{{ detail.content }}</text>
      <button
        v-if="detail.action_type === MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY"
        class="detail-action"
        @tap="openAction"
      >
        {{ t('common.action.view') }}
      </button>
    </template>
    <view v-else class="detail-loading">{{ t('core.status.loading') }}</view>
  </view>
</template>

<style scoped lang="scss">
.detail-page {
  box-sizing: border-box;
  min-height: 100vh;
  padding: 24rpx 30rpx calc(96rpx + env(safe-area-inset-bottom));
  background: var(--kratos-bg-gradient);
  color: var(--kratos-color-text);
}
.detail-loading {
  padding: 120rpx 0;
  color: var(--kratos-color-text-dim);
  font-family: var(--kratos-font-mono);
  text-align: center;
}
.detail-title {
  display: block;
  padding-top: 16rpx;
  color: var(--kratos-color-text);
  font-size: 42rpx;
  font-weight: 800;
  line-height: 1.35;
}
.detail-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  margin: 24rpx 0 36rpx;
  padding-bottom: 28rpx;
  border-bottom: 1px solid var(--kratos-color-border);
  color: var(--kratos-color-text-muted);
  font-size: 22rpx;
  font-family: var(--kratos-font-mono);
}
.detail-category {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.detail-category__icon {
  margin-right: 8rpx;
}
.detail-content {
  display: block;
  padding: 28rpx;
  border: 1px solid var(--kratos-color-border);
  border-radius: 20rpx;
  background: var(--kratos-color-surface);
  color: var(--kratos-color-text);
  font-size: 30rpx;
  line-height: 1.8;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.detail-content-rich {
  display: block;
  padding: 28rpx;
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 20rpx;
  background: var(--kratos-color-surface);
  color: var(--kratos-color-text);
  font-size: 30rpx;
  line-height: 1.8;
  overflow-wrap: anywhere;
}
.detail-action {
  margin-top: 36rpx;
  border: 1px solid var(--kratos-color-border-accent);
  border-radius: 20rpx;
  background: var(--kratos-color-primary-soft);
  color: var(--kratos-color-primary);
  font-size: 28rpx;
  font-weight: 700;
  line-height: 88rpx;
}
.empty {
  box-sizing: border-box;
  min-height: 100vh;
  padding-top: 180rpx;
  background: var(--kratos-bg-gradient);
  color: var(--kratos-color-text-dim);
  font-family: var(--kratos-font-mono);
  text-align: center;
}
</style>
