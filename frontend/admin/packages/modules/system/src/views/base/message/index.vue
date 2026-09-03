<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey)"
      width="900px"
      destroy-on-close
      :model="formState"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @closed="handleClose"
    >
      <template #audiences>
        <div class="audiences-editor">
          <div v-for="(audience, index) in formState.audiences" :key="index" class="audience-row">
            <el-select v-model="audience.type" class="audience-type" @change="onAudienceTypeChange(audience)">
              <el-option
                v-for="option in audienceOptions"
                :key="String(option.value)"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-input-number
              v-model="audience.id"
              class="audience-id"
              :min="audience.type === MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT ? 0 : 1"
              :precision="0"
              :disabled="audience.type === MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT"
            />
            <el-switch
              v-if="audience.type === MessageAudienceType.MESSAGE_AUDIENCE_TYPE_DEPT"
              v-model="audience.include_children"
            />
            <el-button v-if="formState.audiences.length > 1" link type="danger" @click="removeAudience(index)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button link type="primary" @click="addAudience">
            <el-icon><CirclePlus /></el-icon>
            {{ t("common.action.create") }}
          </el-button>
        </div>
      </template>
    </FormDialog>
    <ProDialog
      v-model="content.visible"
      :title="content.data?.base_message?.title || t('system.base.message.content.title')"
      width="760px"
      destroy-on-close
      :show-footer="false"
    >
      <el-skeleton v-if="content.loading" :rows="8" animated />
      <template v-else-if="content.data">
        <div class="message-content-meta">
          <span>{{ content.data.base_message?.category_name || "-" }}</span>
          <span>{{ content.data.base_message?.created_at || "-" }}</span>
        </div>
        <RichTextPreview
          v-if="content.data.form?.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT"
          class="message-content-body"
          :model-value="content.data.form?.content ?? ''"
        />
        <pre v-else class="message-content-body">{{ content.data.form?.content ?? "" }}</pre>
      </template>
    </ProDialog>
    <ProDialog
      v-model="detail.visible"
      :title="t('system.base.message.send_detail.title')"
      width="min(1200px, calc(100vw - 32px))"
      :show-footer="false"
    >
      <template v-if="detail.data">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="t('system.base.message.field.title')">{{
            detail.data.base_message?.title
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.category')">{{
            detail.data.base_message?.category_name
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('common.field.status')">{{
            optionLabel(statusOptions, detail.data.base_message?.status)
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.sender')">{{
            detail.data.base_message?.sender_name
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.delivered_total')">{{
            detail.data.base_message?.delivered_total
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.recipient_total')">{{
            detail.data.base_message?.recipient_total
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.failed_total')">{{
            detail.data.base_message?.failed_total
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.published_at')">{{
            detail.data.base_message?.published_at || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.message.field.scheduled_at')">{{
            detail.data.base_message?.scheduled_at || "-"
          }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.data.dispatches" style="margin-top: 16px" size="small">
          <el-table-column prop="id" label="ID" width="90" />
          <el-table-column prop="audience_type" :label="t('system.base.message.field.audience_type')" width="120">
            <template #default="scope">{{ optionLabel(audienceOptions, scope.row.audience_type) }}</template>
          </el-table-column>
          <el-table-column prop="audience_id" :label="t('system.base.message.field.audience_id')" width="110" />
          <el-table-column prop="include_children" :label="t('system.base.message.field.include_children')" width="120">
            <template #default="scope">{{ scope.row.include_children ? t("common.value.yes") : t("common.value.no") }}</template>
          </el-table-column>
          <el-table-column prop="status" :label="t('common.field.status')" width="120" />
          <el-table-column prop="matched_total" :label="t('system.base.message.field.matched_total')" width="110" />
          <el-table-column prop="inserted_total" :label="t('system.base.message.field.inserted_total')" width="110" />
          <el-table-column prop="attempt_count" :label="t('system.base.message.field.attempt_count')" width="90" />
          <el-table-column
            prop="last_error"
            :label="t('system.base.message.field.last_error')"
            min-width="220"
            show-overflow-tooltip
          />
          <el-table-column :label="t('common.field.operation')" width="90">
            <template #default="scope">
              <el-button
                v-if="scope.row.status === MessageDispatchStatus.MESSAGE_DISPATCH_STATUS_FAILED && BUTTONS['base:message:retry']"
                link
                type="primary"
                @click="retryDispatch(scope.row.id)"
              >
                {{ t("system.base.message.action.retry") }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import RichTextPreview from "@liujitcn/kratos-admin-core/components/RichTextPreview/index.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@liujitcn/kratos-admin-core/tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseMessageService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message";
import { defBaseMessageCategoryService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message_category";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant";
import type {
  BaseMessage,
  BaseMessageDetail,
  BaseMessageForm,
  BaseMessageAudienceForm,
  PageBaseMessageRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message";
import {
  MessageActionType,
  MessageAudienceType,
  MessageContentFormat,
  MessageDispatchStatus,
  MessagePriority,
  MessageStatus
} from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";

defineOptions({ name: "BaseMessage", inheritAttrs: false });

/** 消息管理页面表单状态。 */
type MessageFormState = Omit<BaseMessageForm, "tenant_id" | "category_id"> & {
  /** 租户ID。 */
  tenant_id?: number;
  /** 消息分类ID。 */
  category_id?: number;
  /** 单项受众类型。 */
  audience_type: MessageAudienceType;
  /** 单项受众ID。 */
  audience_id: number;
  /** 是否包含子部门。 */
  include_children: boolean;
};

const { BUTTONS } = useAuthButtons();
const userStore = useUserStore();
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const categoryOptions = ref<ProFormOption[]>([]);
const tenantOptions = ref<ProFormOption[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create" });
const content = reactive<{ visible: boolean; loading: boolean; data?: BaseMessageDetail }>({ visible: false, loading: false });
const detail = reactive<{ visible: boolean; data?: BaseMessageDetail }>({ visible: false });
const formState = reactive<MessageFormState>(defaultForm());

const priorityOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message.priority.normal"), value: MessagePriority.MESSAGE_PRIORITY_NORMAL },
  { label: t("system.base.message.priority.important"), value: MessagePriority.MESSAGE_PRIORITY_IMPORTANT },
  { label: t("system.base.message.priority.urgent"), value: MessagePriority.MESSAGE_PRIORITY_URGENT }
]);
const actionTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message.action_type.none"), value: MessageActionType.MESSAGE_ACTION_TYPE_UNSPECIFIED },
  { label: t("system.base.message.action_type.view_key"), value: MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY }
]);
const audienceOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message.audience.tenant"), value: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT },
  { label: t("system.base.message.audience.user"), value: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_USER },
  { label: t("system.base.message.audience.role"), value: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_ROLE },
  { label: t("system.base.message.audience.dept"), value: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_DEPT },
  { label: t("system.base.message.audience.post"), value: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_POST }
]);
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message.status.draft"), value: MessageStatus.MESSAGE_STATUS_DRAFT },
  { label: t("system.base.message.status.scheduled"), value: MessageStatus.MESSAGE_STATUS_SCHEDULED },
  { label: t("system.base.message.status.publishing"), value: MessageStatus.MESSAGE_STATUS_PUBLISHING },
  { label: t("system.base.message.status.published"), value: MessageStatus.MESSAGE_STATUS_PUBLISHED },
  { label: t("system.base.message.status.revoked"), value: MessageStatus.MESSAGE_STATUS_REVOKED }
]);
onMounted(() => {
  void loadCategoryOptions();
  void loadTenantOptions();
});
watch(
  () => formState.action_type,
  value => {
    if (value === MessageActionType.MESSAGE_ACTION_TYPE_UNSPECIFIED) formState.action_target = "";
  }
);

const rules = computed(() => ({
  tenant_id: isDefaultTenant.value
    ? [
        {
          required: true,
          message: t("common.validation.required_select", { field: t("common.field.tenant") }),
          trigger: "change"
        }
      ]
    : [],
  category_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.message.field.category") }),
      trigger: "change"
    }
  ],
  title: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.message.field.title") }),
      trigger: "blur"
    }
  ],
  content: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.message.field.content") }),
      trigger: "blur"
    }
  ],
  audiences: [{ required: true, message: t("system.base.message.validation.audience_id"), trigger: "change" }]
}));

const formFields = computed<ProFormField[]>(() => [
  {
    prop: "tenant_id",
    label: t("common.field.tenant"),
    labelTooltip: t("system.base.message.tooltip.tenant"),
    component: "select",
    visible: () => isDefaultTenant.value,
    options: tenantOptions.value,
    props: { filterable: true, disabled: Boolean(formState.id), placeholder: t("system.base.message.placeholder.tenant") }
  },
  {
    prop: "category_id",
    label: t("system.base.message.field.category"),
    labelTooltip: t("system.base.message.tooltip.category"),
    component: "select",
    options: categoryOptions.value,
    props: { filterable: true, placeholder: t("system.base.message.placeholder.category") }
  },
  {
    prop: "title",
    label: t("system.base.message.field.title"),
    labelTooltip: t("system.base.message.tooltip.title"),
    component: "input",
    props: { maxlength: 200, showWordLimit: true, placeholder: t("system.base.message.placeholder.title") },
    colSpan: 24
  },
  {
    prop: "content",
    label: t("system.base.message.field.content"),
    labelTooltip: t("system.base.message.tooltip.content"),
    component: "rich-text",
    props: {
      height: "320px",
      uploadType: "message",
      editorConfig: { placeholder: t("system.base.message.placeholder.content") }
    },
    colSpan: 24
  },
  {
    prop: "priority",
    label: t("system.base.message.field.priority"),
    labelTooltip: t("system.base.message.tooltip.priority"),
    component: "select",
    options: priorityOptions.value
  },
  {
    prop: "audiences",
    label: t("system.base.message.field.audience_type"),
    labelTooltip: t("system.base.message.tooltip.audience_type"),
    component: "slot",
    slotName: "audiences",
    colSpan: 24,
    rules: [{ required: true, message: t("system.base.message.validation.audience_id"), trigger: "change" }]
  },
  {
    prop: "action_type",
    label: t("system.base.message.field.action_type"),
    labelTooltip: t("system.base.message.tooltip.action_type"),
    component: "select",
    options: actionTypeOptions.value
  },
  {
    prop: "action_target",
    label: t("system.base.message.field.action_target"),
    labelTooltip: t("system.base.message.tooltip.action_target"),
    component: "input",
    visible: () => formState.action_type === MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY,
    props: { placeholder: t("system.base.message.placeholder.action_target") },
    colSpan: 24
  },
  {
    prop: "scheduled_at",
    label: t("system.base.message.field.scheduled_at"),
    component: "date-picker",
    props: {
      type: "datetime",
      valueFormat: "YYYY-MM-DD HH:mm:ss",
      placeholder: t("system.base.message.placeholder.scheduled_at")
    }
  },
  {
    prop: "expires_at",
    label: t("system.base.message.field.expires_at"),
    component: "date-picker",
    props: { type: "datetime", valueFormat: "YYYY-MM-DD HH:mm:ss", placeholder: t("system.base.message.placeholder.expires_at") }
  }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...(isDefaultTenant.value
    ? ([
        {
          prop: "tenant_id",
          label: t("common.field.tenant"),
          minWidth: 120,
          search: { el: "select" },
          enum: requestTenantOptions
        }
      ] satisfies ColumnProps[])
    : []),
  {
    prop: "title",
    label: t("system.base.message.field.title"),
    minWidth: 220,
    search: { el: "input" },
    render: scope => {
      const row = scope.row as BaseMessage;
      return h(
        "a",
        {
          href: "#",
          title: row.title,
          style: {
            display: "block",
            maxWidth: "100%",
            overflow: "hidden",
            color: "var(--el-color-primary)",
            textDecoration: "none",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap"
          },
          onClick: (event: MouseEvent) => {
            event.preventDefault();
            void openContent(row.id);
          }
        },
        row.title
      );
    }
  },
  {
    prop: "category_name",
    label: t("system.base.message.field.category"),
    minWidth: 130,
    search: { el: "select", key: "category_id", enum: categoryOptions.value }
  },
  {
    prop: "priority",
    label: t("system.base.message.field.priority"),
    width: 100,
    render: scope => optionLabel(priorityOptions.value, (scope.row as BaseMessage).priority)
  },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 110,
    search: { el: "select", enum: statusOptions.value },
    render: scope => optionLabel(statusOptions.value, (scope.row as BaseMessage).status)
  },
  { prop: "delivered_total", label: t("system.base.message.field.delivered_total"), width: 100, align: "right" },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 380,
    className: "message-operation-column",
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        link: true,
        icon: EditPen,
        hidden: scope =>
          !BUTTONS.value["base:message:update"] || (scope.row as BaseMessage).status !== MessageStatus.MESSAGE_STATUS_DRAFT,
        onClick: scope => openDialog((scope.row as BaseMessage).id)
      },
      {
        label: t("system.base.message.send_detail.action"),
        link: true,
        icon: View,
        onClick: scope => openDetail((scope.row as BaseMessage).id)
      },
      {
        label: t("system.base.message.action.publish"),
        type: "primary",
        link: true,
        icon: Promotion,
        hidden: scope =>
          !BUTTONS.value["base:message:publish"] || (scope.row as BaseMessage).status !== MessageStatus.MESSAGE_STATUS_DRAFT,
        onClick: scope => publishMessage(scope.row as BaseMessage)
      },
      {
        label: t("system.base.message.action.cancel_schedule"),
        link: true,
        icon: RefreshRight,
        hidden: scope =>
          !BUTTONS.value["base:message:schedule"] || (scope.row as BaseMessage).status !== MessageStatus.MESSAGE_STATUS_SCHEDULED,
        onClick: scope => cancelSchedule(scope.row as BaseMessage)
      },
      {
        label: t("system.base.message.action.revoke"),
        type: "danger",
        link: true,
        icon: SwitchButton,
        hidden: scope =>
          !BUTTONS.value["base:message:revoke"] ||
          ![MessageStatus.MESSAGE_STATUS_PUBLISHING, MessageStatus.MESSAGE_STATUS_PUBLISHED].includes(
            (scope.row as BaseMessage).status
          ),
        onClick: scope => revokeMessage(scope.row as BaseMessage)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope =>
          !BUTTONS.value["base:message:delete"] || (scope.row as BaseMessage).status !== MessageStatus.MESSAGE_STATUS_DRAFT,
        onClick: scope => handleDelete(scope.row as BaseMessage)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "primary",
    icon: CirclePlus,
    hidden: !BUTTONS.value["base:message:create"],
    onClick: () => openDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: !BUTTONS.value["base:message:delete"],
    disabled: scope => !scope.isSelected,
    onClick: scope => handleDelete(scope.selectedListIds as number[])
  }
]);

/** 创建默认消息表单。 */
function defaultForm(): MessageFormState {
  return {
    id: 0,
    tenant_id: undefined,
    category_id: undefined,
    title: "",
    content: "",
    content_format: MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT,
    priority: MessagePriority.MESSAGE_PRIORITY_NORMAL,
    action_type: MessageActionType.MESSAGE_ACTION_TYPE_UNSPECIFIED,
    action_target: "",
    action_params: "",
    scheduled_at: "",
    expires_at: "",
    audiences: [defaultAudience()],
    version: 1,
    audience_type: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT,
    audience_id: 0,
    include_children: false
  };
}

/** 创建默认受众行。 */
function defaultAudience(): BaseMessageAudienceForm {
  return { type: MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT, id: 0, include_children: false };
}

/** 添加一条消息受众。 */
function addAudience() {
  formState.audiences.push(defaultAudience());
}

/** 删除一条消息受众。 */
function removeAudience(index: number) {
  if (formState.audiences.length > 1) formState.audiences.splice(index, 1);
}

/** 切换受众类型时清理不适用的字段。 */
function onAudienceTypeChange(audience: BaseMessageAudienceForm) {
  if (audience.type === MessageAudienceType.MESSAGE_AUDIENCE_TYPE_TENANT) audience.id = 0;
  if (audience.type !== MessageAudienceType.MESSAGE_AUDIENCE_TYPE_DEPT) audience.include_children = false;
}

/** 请求消息表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defBaseMessageService.PageBaseMessage(
    buildPageRequest<PageBaseMessageRequest>(params as unknown as PageBaseMessageRequest)
  );
  return { data: { list: data.base_messages ?? [], total: data.total } };
}

/** 加载租户选项。 */
async function loadTenantOptions() {
  if (!isDefaultTenant.value) return;
  const result = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  tenantOptions.value = result.list.map(item => ({ label: item.label, value: item.value }));
}

/** 加载消息分类选项。 */
async function loadCategoryOptions() {
  const result = await defBaseMessageCategoryService.OptionBaseMessageCategory({});
  categoryOptions.value = result.list.map(item => ({ label: item.label, value: item.value, disabled: item.disabled }));
}

/** 打开消息草稿表单。 */
async function openDialog(id?: number) {
  await loadTenantOptions();
  Object.assign(formState, defaultForm());
  dialog.titleKey = id ? "common.action.edit" : "common.action.create";
  if (id) {
    const detail = await defBaseMessageService.GetBaseMessage({ id });
    Object.assign(formState, detail.form);
    formState.audiences = detail.form?.audiences?.map(item => ({ ...item })) ?? [defaultAudience()];
  }
  await loadCategoryOptions();
  dialog.visible = true;
}

/** 提交消息草稿。 */
async function handleSubmit() {
  const audiences = formState.audiences.length > 0 ? formState.audiences.map(item => ({ ...item })) : [defaultAudience()];
  const payload: BaseMessageForm = {
    ...formState,
    tenant_id: formState.tenant_id ?? 0,
    category_id: formState.category_id ?? 0,
    content_format: MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT,
    audiences
  };
  if (payload.id) await defBaseMessageService.UpdateBaseMessage({ base_message: payload });
  else await defBaseMessageService.CreateBaseMessage({ base_message: payload });
  ElMessage.success(t("common.message.operation_success"));
  dialog.visible = false;
  proTable.value?.getTableList();
}

/** 打开发送详情并加载投递进度。 */
async function openDetail(id: number) {
  detail.data = await defBaseMessageService.GetBaseMessage({ id });
  detail.visible = true;
}

/** 打开消息正文并单独展示内容。 */
async function openContent(id: number) {
  content.visible = true;
  content.loading = true;
  content.data = undefined;
  try {
    content.data = await defBaseMessageService.GetBaseMessage({ id });
  } catch {
    content.visible = false;
  } finally {
    content.loading = false;
  }
}

/** 重试失败的消息投递任务并刷新详情。 */
async function retryDispatch(id: number) {
  await defBaseMessageService.RetryBaseMessageDispatch({ id });
  if (detail.data?.base_message?.id)
    detail.data = await defBaseMessageService.GetBaseMessage({ id: detail.data.base_message.id });
  ElMessage.success(t("common.message.operation_success"));
}

/** 删除草稿消息。 */
async function handleDelete(value: BaseMessage | BaseMessage[] | number | number[]) {
  const ids = normalizeSelectedIds(value as Parameters<typeof normalizeSelectedIds>[0]);
  await ElMessageBox.confirm(t("common.confirm.delete"), t("common.title.warning"), { type: "warning" });
  await defBaseMessageService.DeleteBaseMessage({ id: ids.join(",") });
  ElMessage.success(t("common.message.operation_success"));
  proTable.value?.getTableList();
}

/** 发布消息。 */
async function publishMessage(row: BaseMessage) {
  await ElMessageBox.confirm(t("system.base.message.confirm.publish"), t("common.title.warning"), { type: "warning" });
  await defBaseMessageService.PublishBaseMessage({ id: row.id });
  proTable.value?.getTableList();
}

/** 取消定时发布。 */
async function cancelSchedule(row: BaseMessage) {
  await defBaseMessageService.CancelBaseMessageSchedule({ id: row.id });
  proTable.value?.getTableList();
}

/** 撤回消息。 */
async function revokeMessage(row: BaseMessage) {
  await ElMessageBox.confirm(t("system.base.message.confirm.revoke"), t("common.title.warning"), { type: "warning" });
  await defBaseMessageService.RevokeBaseMessage({ id: row.id });
  proTable.value?.getTableList();
}

/** 关闭消息表单。 */
function handleClose() {
  Object.assign(formState, defaultForm());
  formDialogRef.value?.resetFields();
}

/** 查询选项显示名称。 */
function optionLabel(options: ProFormOption[], value: unknown) {
  return options.find(item => item.value === value)?.label ?? "";
}
</script>

<style scoped>
.audiences-editor {
  width: 100%;
}

.audience-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.audience-type {
  width: 180px;
}

.audience-id {
  width: 160px;
}

.message-content-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.message-content-body {
  max-height: 60vh;
  margin: 0;
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

pre.message-content-body {
  white-space: pre-wrap;
}
</style>
