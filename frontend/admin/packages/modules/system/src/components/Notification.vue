<template>
  <el-popover placement="bottom-end" :width="360" trigger="click" @show="refresh">
    <template #reference>
      <el-badge class="notification-trigger" :value="badgeValue" :hidden="unreadTotal === 0" :max="99">
        <el-icon :size="24"><Bell /></el-icon>
      </el-badge>
    </template>
    <div class="notification-panel">
      <div class="notification-header">
        <strong>{{ t("system.notification.title") }}</strong>
        <el-button link type="primary" @click="markAllRead">{{ t("system.notification.action.mark_all_read") }}</el-button>
      </div>
      <div class="notification-list">
        <button
          v-for="item in notifications"
          :key="item.id"
          class="notification-item"
          :class="{ 'is-unread': !item.read_at }"
          type="button"
          @click="openInbox"
        >
          <span class="notification-category-icon" :style="{ color: item.category_color || undefined }" aria-hidden="true">
            <component :is="resolveNotificationIcon(item.category_icon)" />
          </span>
          <span class="notification-status" :style="{ backgroundColor: item.category_color || undefined }" aria-hidden="true"></span>
          <span class="notification-title">{{ item.title }}</span>
          <span class="notification-time">{{ item.received_at }}</span>
        </button>
        <el-empty v-if="notifications.length === 0" :description="t('system.notification.empty')" :image-size="72" />
      </div>
      <el-button class="notification-more" text @click="openInbox">{{ t("system.notification.action.view_all") }}</el-button>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defNotificationService } from "@liujitcn/kratos-admin-system/api/base/notification";
import { subscribeSseEvent, type SseStop } from "@liujitcn/kratos-admin-system/api/base/sse";
import type { Notification } from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";
import { NotificationView } from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";

const router = useRouter();
const unreadTotal = ref(0);
const latestDeliveryID = ref(0);
const notifications = ref<Notification[]>([]);
let stopSse: SseStop | undefined;

const badgeValue = computed(() => Math.min(unreadTotal.value, 99));
const notificationIcons = {
  Bell,
  Calendar,
  ChatDotRound,
  CircleCheckFilled,
  CollectionTag,
  DataAnalysis,
  InfoFilled,
  List,
  Lock,
  Message,
  Promotion,
  Setting,
  User,
  WarningFilled
};

onMounted(() => {
  void refresh();
  stopSse = subscribeSseEvent(
    { stream: "base.notification", channel_id: undefined },
    "inbox.changed",
    raw => {
      try {
        return JSON.parse(raw) as Record<string, unknown>;
      } catch {
        return null;
      }
    },
    () => void refresh()
  );
});

onBeforeUnmount(() => stopSse?.());

/** 刷新未读汇总和最近消息。 */
async function refresh() {
  const [summary, page] = await Promise.all([
    defNotificationService.GetNotificationSummary({}),
    defNotificationService.PageNotification({ view: NotificationView.NOTIFICATION_VIEW_INBOX, category_id: undefined, priority: undefined, cursor_id: 0, page_num: 1, page_size: 5 })
  ]);
  unreadTotal.value = summary.unread_total;
  latestDeliveryID.value = summary.latest_delivery_id;
  notifications.value = page.notifications;
}

/** 标记全部消息为已读。 */
async function markAllRead() {
  if (!latestDeliveryID.value) return;
  await defNotificationService.MarkAllNotificationRead({ before_delivery_id: latestDeliveryID.value });
  await refresh();
}

/** 打开完整收件箱。 */
function openInbox() {
  void router.push({ name: "NotificationInbox" });
}

/** 解析收件箱分类图标，兼容历史数据中的未知图标名。 */
function resolveNotificationIcon(icon: string) {
  return notificationIcons[icon as keyof typeof notificationIcons] ?? CollectionTag;
}
</script>

<style scoped lang="scss">
.notification-trigger {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.notification-panel {
  min-height: 160px;
}

.notification-list {
  max-height: min(440px, calc(100vh - 240px));
  overflow-y: auto;
  overscroll-behavior: contain;
}

.notification-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.notification-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 10px 0;
  border: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.notification-status {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  border-radius: 50%;
  background: var(--el-color-primary);
  visibility: hidden;
}

.notification-category-icon {
  display: inline-flex;
  flex: 0 0 18px;
  align-items: center;
  justify-content: center;
  font-size: 17px;
}

.notification-item.is-unread .notification-status {
  visibility: visible;
}

.notification-title {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.notification-more {
  width: 100%;
  margin-top: 8px;
}
</style>
