<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { navigateAppView, t } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore } from '@liujitcn/kratos-uni-app-core/stores'
import { getMpCapsuleRightGap } from '@liujitcn/kratos-uni-app-core/utils/index'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification, NotificationCategory } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { notificationUnreadTotal, refreshNotificationSummary } from '../../../notification'
import { resolveMessageCategoryIcon } from './icons'

const themeStore = useThemeStore()
/** MP-WEIXIN 原生胶囊按钮右侧避让宽度，写入 --mp-capsule-gap 供 .k-top 预留空间。 */
const capsuleGap = getMpCapsuleRightGap()
const items = ref<Notification[]>([])
const loading = ref(false)
const cursorId = ref(0)
const finished = ref(false)
const selectedView = ref(NotificationView.NOTIFICATION_VIEW_INBOX)
const categoryId = ref<number>()
const categoryOptions = ref<NotificationCategory[]>([])
let requestSeq = 0
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
  requestSeq++
  cursorId.value = 0
  finished.value = false
  items.value = []
  // 先重置 loading，再发起新请求，避免清空列表后 loadMore 因旧 loading 直接返回。
  loading.value = false
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
  const seq = requestSeq
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
    if (seq !== requestSeq) return
    items.value.push(...result.notifications)
    finished.value = !result.has_more
    cursorId.value = result.next_cursor_id
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

/** 打开站内信详情。 */
async function openDetail(item: Notification) {
  if (!item.read_at) await defNotificationService.MarkNotificationRead({ ids: [item.id] })
  navigateAppView('MESSAGE_DETAIL', { id: String(item.id) })
}
</script>

<template>
  <view class="message-page" :class="themeStore.themeClass">
    <view class="k-top message-header" :style="{ '--mp-capsule-gap': `${capsuleGap}px` }">
      <view class="message-header__title">
        <text class="k-top__t2">{{ t('system.notification.title') }}</text>
        <view v-if="notificationUnreadTotal > 0" class="message-unread-dot" aria-hidden="true" />
      </view>
      <view
        v-if="
          selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED &&
          notificationUnreadTotal > 0
        "
        class="k-ch k-ch--pri"
        @tap="markAllRead"
      >
        {{ t('system.notification.mark_all_read') }}
      </view>
    </view>
    <view class="k-chips">
      <view class="k-ch" :class="{ 'k-ch--pri': categoryId === undefined }" @tap="changeCategory()">
        {{ t('system.notification.view.inbox') }}
      </view>
      <view
        v-for="category in categoryOptions"
        :key="category.id"
        class="k-ch"
        :class="{ 'k-ch--pri': categoryId === category.id }"
        @tap="changeCategory(category.id)"
      >
        <uni-icons
          :type="resolveMessageCategoryIcon(category.icon)"
          size="16"
          :color="category.color || '#64748b'"
          aria-hidden="true"
        />
        {{ category.name }}
      </view>
    </view>
    <view class="k-pills message-tabs">
      <view
        v-for="option in viewOptions"
        :key="option.value"
        class="k-pills__item message-tab"
        :class="{ 'k-pills__item--on': selectedView === option.value }"
        @tap="changeView(option.value)"
      >
        {{ t(option.key) }}
      </view>
    </view>
    <view class="k-msg-list">
      <view
        v-for="item in items"
        :key="item.id"
        class="k-msg-item"
        :class="{ 'k-msg-item--unread': !item.read_at }"
        @tap="openDetail(item)"
      >
        <view class="k-msg-item__head">
          <view class="k-msg-item__icon">
            <uni-icons
              :type="resolveMessageCategoryIcon(item.category_icon)"
              size="18"
              :color="item.category_color || '#64748b'"
              aria-hidden="true"
            />
          </view>
          <view class="message-item__body">
            <text class="k-msg-item__title">{{ item.title }}</text>
            <text class="k-msg-item__meta">{{ item.category_name }}</text>
          </view>
          <text class="k-msg-item__time">{{ item.received_at }}</text>
        </view>
        <view
          v-if="
            selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED ||
            item.allow_archive ||
            item.allow_delete
          "
          class="k-msg-item__actions"
        >
          <text
            v-if="
              selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && item.allow_archive
            "
            class="k-msg-item__action"
            @tap.stop="toggleArchive(item)"
          >
            {{ t('system.notification.action.archive') }}
          </text>
          <text
            v-if="selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED"
            class="k-msg-item__action"
            @tap.stop="toggleArchive(item)"
          >
            {{ t('system.notification.action.restore') }}
          </text>
          <text v-if="item.allow_delete" class="k-msg-item__action" @tap.stop="deleteItem(item)">
            {{ t('system.notification.action.delete') }}
          </text>
        </view>
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
/* 仅页面局部布局；视觉词汇统一来自 core 的 .k-* 全局类。 */
.message-page {
  box-sizing: border-box;
  min-height: 100vh;
  padding: calc(16rpx + env(safe-area-inset-top)) 24rpx calc(200rpx + env(safe-area-inset-bottom));
  background: var(--kratos-bg-gradient);
  color: var(--kratos-color-text);
}
/* #ifdef MP-WEIXIN */
.message-page {
  padding-top: calc(44px + 16rpx);
}
/* #endif */
.message-header {
  padding: 20rpx 4rpx 24rpx;
}
.message-header__title {
  display: flex;
  align-items: center;
  min-width: 0;
}
.message-unread-dot {
  width: 12rpx;
  height: 12rpx;
  margin-left: 12rpx;
  border-radius: 50%;
  background: var(--kratos-color-danger);
  box-shadow: 0 0 16rpx var(--kratos-color-danger);
}
.message-tabs {
  display: flex;
  margin-top: 4rpx;
}
.message-tab {
  flex: 1;
}
.message-item__body {
  flex: 1;
  min-width: 0;
}
.load-more {
  box-sizing: border-box;
  width: 100%;
  height: 76rpx;
  margin-top: 24rpx;
  padding: 0 24rpx;
  border: 1rpx solid var(--kratos-color-border);
  border-radius: 16rpx;
  background: var(--kratos-color-surface-2);
  color: var(--kratos-color-text);
  font-size: 26rpx;
  line-height: 74rpx;
}
.load-more::after {
  border: 0;
}
.empty {
  padding: 120rpx 0;
  color: var(--kratos-color-text-dim);
  font-family: var(--kratos-font-mono);
  text-align: center;
}
</style>
