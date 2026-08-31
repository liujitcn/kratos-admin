<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateAppRoute, t } from '@liujitcn/kratos-uni-app-core'
import { defNotificationService } from '../../../api/base/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { notificationUnreadTotal, refreshNotificationSummary } from '../../../notification'

const items = ref<Notification[]>([])
const loading = ref(false)
const cursorId = ref(0)
const finished = ref(false)
const selectedView = ref(NotificationView.NOTIFICATION_VIEW_INBOX)
const categoryInput = ref('')
const categoryId = ref<number>()
const viewOptions = [
  { value: NotificationView.NOTIFICATION_VIEW_INBOX, key: 'system.notification.view.inbox' },
  { value: NotificationView.NOTIFICATION_VIEW_UNREAD, key: 'system.notification.view.unread' },
  { value: NotificationView.NOTIFICATION_VIEW_ARCHIVED, key: 'system.notification.view.archived' },
]

onShow(() => {
  void refresh()
  void refreshNotificationSummary()
})

/** 刷新站内信列表。 */
async function refresh() {
  cursorId.value = 0
  finished.value = false
  items.value = []
  await loadMore()
}

/** 切换收件箱视图。 */
function changeView(view: NotificationView) {
  selectedView.value = view
  void refresh()
}

/** 应用分类筛选。 */
function applyCategoryFilter() {
  const value = Number(categoryInput.value)
  categoryId.value = Number.isInteger(value) && value > 0 ? value : undefined
  void refresh()
}

/** 标记当前水位线之前的消息为已读。 */
async function markAllRead() {
  const beforeDeliveryId = items.value.reduce((max, item) => Math.max(max, item.id), 0)
  if (beforeDeliveryId <= 0) return
  await defNotificationService.MarkAllNotificationRead({ before_delivery_id: beforeDeliveryId })
  await refresh()
  await refreshNotificationSummary()
}

/** 归档或恢复一条消息。 */
async function toggleArchive(item: Notification) {
  if (item.archived_at) await defNotificationService.RestoreNotification({ id: item.id })
  else if (item.allow_archive) await defNotificationService.ArchiveNotification({ id: item.id })
  await refresh()
  await refreshNotificationSummary()
}

/** 删除一条消息。 */
async function deleteItem(item: Notification) {
  if (!item.allow_delete) return
  await defNotificationService.DeleteNotification({ id: item.id })
  await refresh()
  await refreshNotificationSummary()
}

/** 加载下一页站内信。 */
async function loadMore() {
  if (loading.value || finished.value) return
  loading.value = true
  try {
    const result = await defNotificationService.PageNotification({
      view: selectedView.value,
      category_id: categoryId.value,
      priority: undefined,
      cursor_id: cursorId.value,
      page_num: 1,
      page_size: 20,
    })
    items.value.push(...result.notifications)
    finished.value = !result.has_more
    cursorId.value = result.next_cursor_id
  } finally {
    loading.value = false
  }
}

/** 打开站内信详情。 */
async function openDetail(item: Notification) {
  if (!item.read_at) await defNotificationService.MarkNotificationRead({ ids: [item.id] })
  navigateAppRoute(`app/message/detail?id=${item.id}`)
}
</script>

<template>
  <view class="message-page">
    <view class="message-header">
      <text class="message-title">{{ t('system.notification.title') }}</text>
      <text v-if="notificationUnreadTotal > 0" class="message-unread">{{
        notificationUnreadTotal > 99 ? '99+' : notificationUnreadTotal
      }}</text>
    </view>
    <view class="message-controls">
      <view class="message-tabs">
        <button
          v-for="option in viewOptions"
          :key="option.value"
          :class="['message-tab', { active: selectedView === option.value }]"
          @tap="changeView(option.value)"
        >
          {{ t(option.key) }}
        </button>
      </view>
      <view class="message-filter">
        <input
          v-model="categoryInput"
          type="number"
          :placeholder="t('system.notification.category_filter')"
          @confirm="applyCategoryFilter"
        />
        <button @tap="applyCategoryFilter">{{ t('common.action.confirm') }}</button>
        <button
          v-if="
            selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED &&
            notificationUnreadTotal > 0
          "
          @tap="markAllRead"
        >
          {{ t('system.notification.mark_all_read') }}
        </button>
      </view>
    </view>
    <view v-for="item in items" :key="item.id" class="message-item" @tap="openDetail(item)">
      <view class="message-item__main">
        <text :class="['message-item__title', { unread: !item.read_at }]">{{ item.title }}</text>
        <text class="message-item__time">{{ item.received_at }}</text>
      </view>
      <text class="message-item__category">{{ item.category_name }}</text>
      <view class="message-item__actions">
        <button
          v-if="selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && item.allow_archive"
          @tap.stop="toggleArchive(item)"
        >
          {{ t('system.notification.action.archive') }}
        </button>
        <button
          v-if="selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED"
          @tap.stop="toggleArchive(item)"
        >
          {{ t('system.notification.action.restore') }}
        </button>
        <button v-if="item.allow_delete" @tap.stop="deleteItem(item)">
          {{ t('system.notification.action.delete') }}
        </button>
      </view>
    </view>
    <button v-if="!finished" class="load-more" :loading="loading" @tap="loadMore">
      {{ t('common.action.load_more') }}
    </button>
    <view v-if="finished && items.length === 0" class="empty">{{
      t('system.notification.empty')
    }}</view>
  </view>
</template>

<style scoped lang="scss">
.message-page {
  min-height: 100vh;
  padding: 24rpx;
  background: var(--kratos-color-background);
}
.message-header {
  padding: 20rpx 4rpx 28rpx;
}
.message-title {
  font-size: 40rpx;
  font-weight: 700;
  color: var(--kratos-color-text);
}
.message-unread {
  min-width: 36rpx;
  padding: 4rpx 10rpx;
  border-radius: 18rpx;
  background: #e5484d;
  color: #fff;
  font-size: 22rpx;
  line-height: 28rpx;
  text-align: center;
}
.message-controls {
  margin-bottom: 18rpx;
}
.message-tabs,
.message-filter,
.message-item__actions {
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.message-tab {
  flex: 1;
  margin: 0;
  padding: 0 12rpx;
  background: transparent;
  color: var(--kratos-color-text-muted);
  font-size: 26rpx;
}
.message-tab.active {
  background: var(--kratos-color-primary);
  color: #fff;
}
.message-filter {
  margin-top: 14rpx;
}
.message-filter input {
  min-width: 0;
  flex: 1;
  padding: 12rpx 16rpx;
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 8rpx;
}
.message-filter button,
.message-item__actions button {
  margin: 0;
  padding: 0 14rpx;
  font-size: 24rpx;
}
.message-item__actions {
  justify-content: flex-end;
  margin-top: 14rpx;
}
.message-item {
  margin-bottom: 16rpx;
  padding: 28rpx;
  border-radius: 12rpx;
  background: #fff;
}
.message-item__main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
}
.message-item__title {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--kratos-color-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.message-item__title.unread {
  color: var(--kratos-color-text);
  font-weight: 700;
}
.message-item__time,
.message-item__category {
  color: var(--kratos-color-text-muted);
  font-size: 24rpx;
}
.message-item__category {
  display: block;
  margin-top: 14rpx;
}
.load-more {
  margin-top: 24rpx;
  font-size: 28rpx;
}
.empty {
  padding: 120rpx 0;
  color: var(--kratos-color-text-muted);
  text-align: center;
}
</style>
