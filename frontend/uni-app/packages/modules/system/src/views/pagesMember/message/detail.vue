<script setup lang="ts">
import KratosNavBar from '@liujitcn/kratos-uni-app-core/components/KratosNavBar.vue'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { t } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore } from '@liujitcn/kratos-uni-app-core/stores'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { MessageContentFormat } from '../../../rpc/base/v1/notification'
import { resolvePersistedMessage } from '../../../utils/persisted-message'
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

</script>

<template>
  <view class="detail-page" :class="themeStore.themeClass">
    <KratosNavBar :title="t('system.notification.title')" />
    <template v-if="detail">
      <text class="detail-title">{{ resolvePersistedMessage(detail.title) }}</text>
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
        <text>{{ resolvePersistedMessage(detail.sender_name) }}</text>
        <text>{{ detail.received_at }}</text>
      </view>
      <rich-text
        v-if="detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT"
        class="detail-content-rich"
        :nodes="resolvePersistedMessage(detail.content)"
      />
      <text v-else class="detail-content">{{ resolvePersistedMessage(detail.content) }}</text>
    </template>
    <view v-else class="detail-loading">{{ t('core.status.loading') }}</view>
  </view>
</template>

<style scoped lang="scss">
.detail-page {
  box-sizing: border-box;
  min-height: 100vh;
  padding: 0 20rpx calc(96rpx + env(safe-area-inset-bottom));
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
