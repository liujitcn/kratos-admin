<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <el-dialog v-model="detailVisible" :title="detail?.title" width="720px" destroy-on-close>
      <div v-if="detail" class="message-detail">
        <div class="message-meta">
          <span class="message-category" :style="{ color: detail.category_color || undefined }">
            <component :is="resolveNotificationIcon(detail.category_icon)" />
            {{ detail.category_name }}
          </span>
          <span>{{ detail.sender_name }}</span>
          <span>{{ detail.received_at }}</span>
        </div>
        <RichTextPreview
          v-if="detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT"
          class="message-content"
          :model-value="detail.content"
        />
        <pre v-else class="message-content">{{ detail.content }}</pre>
        <el-button
          v-if="detail.action_type === MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY && actionRouteName"
          class="message-action"
          type="primary"
          @click="openAction"
        >
          {{ t("common.action.view") }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref } from "vue";
import { useRouter } from "vue-router";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import RichTextPreview from "@liujitcn/kratos-admin-core/components/RichTextPreview/index.vue";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defNotificationService } from "@liujitcn/kratos-admin-system/api/base/notification";
import type { Notification, PageNotificationRequest } from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";
import {
  MessageActionType,
  MessageContentFormat,
  NotificationView
} from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";

defineOptions({ name: "NotificationInbox", inheritAttrs: false });

const proTable = ref<ProTableInstance>();
const router = useRouter();
const detailVisible = ref(false);
const detail = ref<Notification>();
const actionRouteName = computed(() => {
  const viewKey = detail.value?.action_target;
  if (!viewKey) return "";
  return ({ MESSAGE_INBOX: "NotificationInbox", PROFILE: "Profile", AI: "AiChat" } as Record<string, string>)[viewKey] ?? "";
});
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

const viewOptions = computed<ProFormOption[]>(() => [
  { label: t("system.notification.view.inbox"), value: NotificationView.NOTIFICATION_VIEW_INBOX },
  { label: t("system.notification.view.unread"), value: NotificationView.NOTIFICATION_VIEW_UNREAD },
  { label: t("system.notification.view.archived"), value: NotificationView.NOTIFICATION_VIEW_ARCHIVED }
]);

const columns = computed<ColumnProps[]>(() => [
  { prop: "title", label: t("system.base.message.field.title"), minWidth: 240 },
  {
    prop: "category_name",
    label: t("system.base.message.field.category"),
    minWidth: 150,
    render: scope => {
      const row = scope.row as Notification;
      return h("span", { class: "message-category", style: { color: row.category_color || undefined } }, [
        h(resolveNotificationIcon(row.category_icon)),
        h("span", { class: "message-category__name" }, row.category_name)
      ]);
    }
  },
  {
    prop: "read_at",
    label: t("system.notification.field.read_status"),
    width: 110,
    search: { el: "select", key: "view", enum: viewOptions.value },
    render: scope =>
      (scope.row as Notification).read_at ? t("system.notification.status.read") : t("system.notification.status.unread")
  },
  { prop: "sender_name", label: t("system.notification.field.sender"), minWidth: 130 },
  { prop: "received_at", label: t("system.notification.field.received_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 250,
    fixed: "right",
    cellType: "actions",
    actions: [
      { label: t("common.action.view"), link: true, icon: View, onClick: scope => openDetail(scope.row as Notification) },
      {
        label: t("system.notification.action.mark_unread"),
        link: true,
        icon: RefreshLeft,
        hidden: scope => !(scope.row as Notification).read_at,
        onClick: scope => markUnread(scope.row as Notification)
      },
      {
        label: t("system.notification.action.archive"),
        link: true,
        icon: Folder,
        hidden: scope => Boolean((scope.row as Notification).archived_at) || !(scope.row as Notification).allow_archive,
        onClick: scope => archive(scope.row as Notification)
      },
      {
        label: t("system.notification.action.restore"),
        link: true,
        icon: RefreshLeft,
        hidden: scope => !(scope.row as Notification).archived_at,
        onClick: scope => restore(scope.row as Notification)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope => !(scope.row as Notification).allow_delete,
        onClick: scope => remove(scope.row as Notification)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("system.notification.action.mark_all_read"), type: "primary", icon: Finished, onClick: () => markAllRead() }
]);

/** 请求当前用户收件箱。 */
async function requestTable(params: Record<string, unknown>) {
  const request = buildPageRequest<PageNotificationRequest>(params as unknown as PageNotificationRequest);
  request.view = Number(params.view ?? NotificationView.NOTIFICATION_VIEW_INBOX) as NotificationView;
  const data = await defNotificationService.PageNotification(request);
  return { data: { list: data.notifications ?? [], total: data.total } };
}

/** 打开消息详情并标记已读。 */
async function openDetail(row: Notification) {
  detail.value = await defNotificationService.GetNotification({ id: row.id });
  detailVisible.value = true;
  if (!row.read_at) {
    await defNotificationService.MarkNotificationRead({ ids: [row.id] });
    proTable.value?.getTableList();
  }
}

/** 执行通知携带的稳定 viewKey 动作。 */
function openAction() {
  if (!actionRouteName.value) return;
  void router.push({ name: actionRouteName.value });
}

/** 标记消息为未读。 */
async function markUnread(row: Notification) {
  await defNotificationService.MarkNotificationUnread({ ids: [row.id] });
  proTable.value?.getTableList();
}

/** 标记全部消息为已读。 */
async function markAllRead() {
  const summary = await defNotificationService.GetNotificationSummary({});
  if (!summary.latest_delivery_id) return;
  await defNotificationService.MarkAllNotificationRead({ before_delivery_id: summary.latest_delivery_id });
  ElMessage.success(t("common.message.operation_success"));
  proTable.value?.getTableList();
}

/** 归档消息。 */
async function archive(row: Notification) {
  await defNotificationService.ArchiveNotification({ id: row.id });
  proTable.value?.getTableList();
}

/** 恢复已归档消息。 */
async function restore(row: Notification) {
  await defNotificationService.RestoreNotification({ id: row.id });
  proTable.value?.getTableList();
}

/** 从个人收件箱删除消息。 */
async function remove(row: Notification) {
  await ElMessageBox.confirm(t("common.confirm.delete"), t("common.tips"), { type: "warning" });
  await defNotificationService.DeleteNotification({ id: row.id });
  proTable.value?.getTableList();
}

/** 解析收件箱分类图标，兼容历史数据中的未知图标名。 */
function resolveNotificationIcon(icon: string) {
  return notificationIcons[icon as keyof typeof notificationIcons] ?? CollectionTag;
}
</script>

<style scoped lang="scss">
.message-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 18px;
  color: var(--el-text-color-secondary);
}

.message-category {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.message-category__name {
  color: var(--el-text-color-primary);
}

.message-content {
  min-height: 180px;
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--el-text-color-primary);
  font-family: inherit;
  line-height: 1.7;
}

.message-content.core-rich-text-preview {
  white-space: normal;
}
</style>
