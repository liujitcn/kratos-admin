<script setup lang="ts">
import KratosEmpty from '@liujitcn/kratos-uni-app-core/components/KratosEmpty.vue'
import KratosNavBar from '@liujitcn/kratos-uni-app-core/components/KratosNavBar.vue'
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { navigateAppView, t } from '@liujitcn/kratos-uni-app-core'
import { useThemeStore } from '@liujitcn/kratos-uni-app-core/stores'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification, NotificationCategory } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { notificationUnreadTotal, refreshNotificationSummary } from '../../../notification'
import { resolvePersistedMessage } from '../../../utils/persisted-message'
import { resolveMessageCategoryIcon } from './icons'

const themeStore = useThemeStore()
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

/** 空态提示语随当前视图切换。 */
const emptyHintKey = computed(() => {
  if (selectedView.value === NotificationView.NOTIFICATION_VIEW_UNREAD)
    return 'system.notification.empty_unread_hint'
  if (selectedView.value === NotificationView.NOTIFICATION_VIEW_ARCHIVED)
    return 'system.notification.empty_archived_hint'
  return 'system.notification.empty_inbox_hint'
})
</script>

<template>
  <view class="message-page" :class="themeStore.themeClass">
    <KratosNavBar
      :title="t('system.notification.title')"
      variant="tab"
      :action="
        selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && notificationUnreadTotal > 0
          ? {
              display: 'text',
              label: t('system.notification.mark_all_read'),
              onClick: markAllRead,
            }
          : undefined
      "
    />
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
            <text class="k-msg-item__title">{{ resolvePersistedMessage(item.title) }}</text>
            <view class="k-msg-item__foot">
              <text class="k-msg-item__meta">{{ item.category_name }}</text>
              <text class="k-msg-item__time">{{ item.received_at }}</text>
            </view>
          </view>
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
    <KratosEmpty
      v-if="finished && items.length === 0"
      icon="notification-filled"
      :title="t('system.notification.empty')"
      :description="t(emptyHintKey)"
    />
  </view>
</template>

<style scoped lang="scss">
/* 仅页面局部布局；视觉词汇统一来自 core 的 .k-* 全局类。 */
.message-page {
  box-sizing: border-box;
  min-height: 100vh;
  padding: 0 20rpx calc(200rpx + env(safe-area-inset-bottom));
  background: var(--kratos-bg-gradient);
  color: var(--kratos-color-text);
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
  border: 1px solid var(--kratos-color-border);
  border-radius: 16rpx;
  background: var(--kratos-color-surface-2);
  color: var(--kratos-color-text);
  font-size: 26rpx;
  line-height: 74rpx;
}
.load-more::after {
  border: 0;
}
</style>
