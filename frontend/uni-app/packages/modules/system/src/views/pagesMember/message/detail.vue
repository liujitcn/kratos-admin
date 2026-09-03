<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateAppView, t } from '@liujitcn/kratos-uni-app-core'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { MessageActionType, MessageContentFormat } from '../../../rpc/base/v1/notification'

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
</script>

<template>
  <view class="detail-page" v-if="detail">
    <text class="detail-title">{{ detail.title }}</text>
    <view class="detail-meta">
      <text>{{ detail.category_name }}</text>
      <text>{{ detail.sender_name }}</text>
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
  </view>
  <view v-else class="empty">{{ t('common.status.loading') }}</view>
</template>

<style scoped lang="scss">
.detail-page {
  box-sizing: border-box;
  min-height: 100vh;
  padding: 24rpx 30rpx calc(96rpx + env(safe-area-inset-bottom));
  background: var(--kratos-color-background, #f7f8fa);
  color: var(--kratos-color-text, #1f2937);
}
.detail-title {
  display: block;
  color: var(--kratos-color-text, #1f2937);
  font-size: 42rpx;
  font-weight: 700;
  line-height: 1.35;
}
.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 18rpx;
  margin: 24rpx 0 36rpx;
  color: var(--kratos-color-text-muted, #6b7280);
  font-size: 24rpx;
}
.detail-content {
  color: var(--kratos-color-text, #1f2937);
  font-size: 30rpx;
  line-height: 1.8;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.detail-content-rich {
  display: block;
  color: var(--kratos-color-text, #1f2937);
  font-size: 30rpx;
  line-height: 1.8;
  overflow-wrap: anywhere;
}
.empty {
  padding-top: 180rpx;
  color: var(--kratos-color-text-muted, #6b7280);
  text-align: center;
}
</style>
