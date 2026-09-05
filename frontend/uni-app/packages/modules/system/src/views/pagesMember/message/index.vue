<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateAppView, t } from '@liujitcn/kratos-uni-app-core'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification, NotificationCategory } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { notificationUnreadTotal, refreshNotificationSummary } from '../../../notification'
import { resolveMessageCategoryIcon } from './icons'

const items = ref<Notification[]>([])
const loading = ref(false)
const cursorId = ref(0)
const finished = ref(false)
const selectedView = ref(NotificationView.NOTIFICATION_VIEW_INBOX)
const categoryId = ref<number>()
const categoryOptions = ref<NotificationCategory[]>([])
const viewOptions = [
  { value: NotificationView.NOTIFICATION_VIEW_INBOX, key: 'system.notification.view.inbox' },
  { value: NotificationView.NOTIFICATION_VIEW_UNREAD, key: 'system.notification.view.unread' },
  { value: NotificationView.NOTIFICATION_VIEW_ARCHIVED, key: 'system.notification.view.archived' },
]

onShow(() => {
  void loadCategories().catch(() => undefined)
  void refresh()
  void refreshNotificationSummary()
})

/** 加载管理端维护的消息分类。 */
async function loadCategories() {
  const result = await defNotificationService.ListNotificationCategories({})
  categoryOptions.value = result.categories ?? []
}

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

/** 切换消息分类。 */
function changeCategory(id?: number) {
  categoryId.value = id
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
  navigateAppView('MESSAGE_DETAIL', { id: String(item.id) })
}
</script>

<template>
  <view class="message-page">
    <view class="message-header">
      <view class="message-header__title">
        <text class="message-title">{{ t('system.notification.title') }}</text>
        <view v-if="notificationUnreadTotal > 0" class="message-unread-dot" aria-hidden="true" />
      </view>
      <button
        v-if="
          selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED &&
          notificationUnreadTotal > 0
        "
        class="message-action message-action--read message-header__action"
        @tap="markAllRead"
      >
        {{ t('system.notification.mark_all_read') }}
      </button>
    </view>
    <view class="message-controls">
      <scroll-view class="message-category-scroll" scroll-x :show-scrollbar="false">
        <view class="message-category-list">
          <button
            class="message-category"
            :class="{ active: categoryId === undefined }"
            @tap="changeCategory()"
          >
            {{ t('system.notification.view.inbox') }}
          </button>
          <button
            v-for="category in categoryOptions"
            :key="category.id"
            class="message-category"
            :class="{ active: categoryId === category.id }"
            @tap="changeCategory(category.id)"
          >
            <uni-icons
              class="message-category__icon"
              :type="resolveMessageCategoryIcon(category.icon)"
              size="16"
              :color="categoryId === category.id ? '#fff' : category.color || '#64748b'"
              aria-hidden="true"
            />
            {{ category.name }}
          </button>
        </view>
      </scroll-view>
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
    </view>
    <view v-for="item in items" :key="item.id" class="message-item" @tap="openDetail(item)">
      <view class="message-item__main">
        <text :class="['message-item__title', { unread: !item.read_at }]">{{ item.title }}</text>
        <text class="message-item__time">{{ item.received_at }}</text>
      </view>
      <view class="message-item__category">
        <uni-icons
          class="message-item__category-icon"
          :type="resolveMessageCategoryIcon(item.category_icon)"
          size="16"
          :color="item.category_color || '#64748b'"
          aria-hidden="true"
        />
        <text>{{ item.category_name }}</text>
      </view>
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
  box-sizing: border-box;
  min-height: 100vh;
  padding: 16rpx 24rpx calc(120rpx + env(safe-area-inset-bottom));
  background: var(--kratos-color-background, #f7f8fa);
  color: var(--kratos-color-text, #1f2937);
}
.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 20rpx 4rpx 24rpx;
}
.message-header__title {
  display: flex;
  align-items: center;
  min-width: 0;
}
.message-title {
  display: block;
  font-size: 40rpx;
  font-weight: 700;
  line-height: 1.35;
  color: var(--kratos-color-text, #1f2937);
}
.message-unread-dot {
  width: 12rpx;
  height: 12rpx;
  margin-left: 8rpx;
  border-radius: 50%;
  background: #e5484d;
}
.message-controls {
  margin-bottom: 24rpx;
}
.message-category-scroll {
  width: 100%;
  white-space: nowrap;
}
.message-category-list {
  display: flex;
  gap: 12rpx;
  width: max-content;
  padding-bottom: 4rpx;
}
.message-category,
.message-tabs,
.message-item__actions {
  display: flex;
  align-items: center;
}
.message-category {
  flex-shrink: 0;
  height: 64rpx;
  margin: 0;
  padding: 0 22rpx;
  border: 1rpx solid var(--kratos-color-border, #e2e8f0);
  border-radius: 32rpx;
  background: #fff;
  color: var(--kratos-color-text-muted, #6b7280);
  font-size: 24rpx;
  line-height: 64rpx;
  white-space: nowrap;
}
.message-category.active {
  border-color: var(--kratos-color-primary, #27ba9b);
  background: var(--kratos-color-primary, #27ba9b);
  color: #fff;
}
.message-tabs {
  gap: 12rpx;
  margin-top: 18rpx;
  padding: 6rpx;
  border-radius: 12rpx;
  background: #eef1f3;
}
.message-item__actions {
  gap: 12rpx;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.message-tab {
  box-sizing: border-box;
  flex: 1;
  min-width: 0;
  height: 68rpx;
  margin: 0;
  padding: 0 12rpx;
  border: 0;
  border-radius: 8rpx;
  background: transparent;
  color: var(--kratos-color-text-muted, #6b7280);
  font-size: 26rpx;
  line-height: 68rpx;
  white-space: nowrap;
}
.message-tab::after,
.message-category::after,
.message-action::after,
.message-item__actions button::after,
.load-more::after {
  border: 0;
}
.message-tab.active {
  background: var(--kratos-color-primary, #27ba9b);
  color: #fff;
}
.message-action,
.message-item__actions button {
  box-sizing: border-box;
  flex-shrink: 0;
  width: auto;
  height: 68rpx;
  margin: 0;
  padding: 0 20rpx;
  border: 0;
  border-radius: 8rpx;
  background: #eef1f3;
  color: var(--kratos-color-text, #1f2937);
  font-size: 24rpx;
  line-height: 68rpx;
  white-space: nowrap;
}
.message-action--read {
  background: #e8f8f4;
  color: var(--kratos-color-primary, #16806d);
}
.message-header__action {
  height: 56rpx;
  padding: 0 18rpx;
  line-height: 56rpx;
}
.message-item__actions button {
  height: 56rpx;
  padding: 0 16rpx;
  font-size: 22rpx;
  line-height: 56rpx;
}
.message-item {
  margin-bottom: 16rpx;
  padding: 28rpx;
  border-radius: 12rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(31, 41, 55, 0.05);
}
.message-item__main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
}
.message-item__title {
  display: block;
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--kratos-color-text, #1f2937);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.message-item__title.unread {
  color: var(--kratos-color-text, #1f2937);
  font-weight: 700;
}
.message-item__time,
.message-item__category {
  color: var(--kratos-color-text-muted, #6b7280);
  font-size: 24rpx;
}
.message-item__time {
  flex-shrink: 0;
}
.message-item__category {
  display: flex;
  align-items: center;
  margin-top: 14rpx;
}
.load-more {
  box-sizing: border-box;
  width: 100%;
  height: 72rpx;
  margin-top: 24rpx;
  padding: 0 24rpx;
  border: 0;
  border-radius: 8rpx;
  background: #eef1f3;
  color: var(--kratos-color-text, #1f2937);
  font-size: 28rpx;
  line-height: 72rpx;
}
.empty {
  padding: 120rpx 0;
  color: var(--kratos-color-text-muted, #6b7280);
  text-align: center;
}
</style>
