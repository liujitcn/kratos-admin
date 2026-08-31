<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey)"
      width="680px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleClose"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, type Component } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { DEFAULT_TENANT_CODE } from "@liujitcn/kratos-admin-core/tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseMessageCategoryService } from "@liujitcn/kratos-admin-system/api/system/base_message_category";
import type {
  BaseMessageCategory,
  BaseMessageCategoryForm,
  PageBaseMessageCategoryRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_category";
import { MessagePriority } from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseMessageCategory", inheritAttrs: false });

/** 消息分类页面表单状态。 */
const { BUTTONS } = useAuthButtons();
const userStore = useUserStore();
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, titleKey: "common.action.create" });
const formData = reactive<BaseMessageCategoryForm>(defaultForm());

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const priorityOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message.priority.normal"), value: MessagePriority.MESSAGE_PRIORITY_NORMAL },
  { label: t("system.base.message.priority.important"), value: MessagePriority.MESSAGE_PRIORITY_IMPORTANT },
  { label: t("system.base.message.priority.urgent"), value: MessagePriority.MESSAGE_PRIORITY_URGENT }
]);
const iconOptions = computed<ProFormOption[]>(() => {
  const options: ProFormOption[] = [
    { label: `${t("system.base.message_category.icon.bell")} (Bell)`, value: "Bell", icon: Bell },
    { label: `${t("system.base.message_category.icon.lock")} (Lock)`, value: "Lock", icon: Lock },
    { label: `${t("system.base.message_category.icon.list")} (List)`, value: "List", icon: List },
    { label: `${t("system.base.message_category.icon.message")} (Message)`, value: "Message", icon: Message },
    { label: `${t("system.base.message_category.icon.info")} (InfoFilled)`, value: "InfoFilled", icon: InfoFilled },
    { label: `${t("system.base.message_category.icon.warning")} (WarningFilled)`, value: "WarningFilled", icon: WarningFilled },
    {
      label: `${t("system.base.message_category.icon.success")} (CircleCheckFilled)`,
      value: "CircleCheckFilled",
      icon: CircleCheckFilled
    },
    { label: `${t("system.base.message_category.icon.promotion")} (Promotion)`, value: "Promotion", icon: Promotion },
    { label: `${t("system.base.message_category.icon.setting")} (Setting)`, value: "Setting", icon: Setting },
    { label: `${t("system.base.message_category.icon.user")} (User)`, value: "User", icon: User },
    { label: `${t("system.base.message_category.icon.calendar")} (Calendar)`, value: "Calendar", icon: Calendar },
    { label: `${t("system.base.message_category.icon.chat")} (ChatDotRound)`, value: "ChatDotRound", icon: ChatDotRound },
    { label: `${t("system.base.message_category.icon.analysis")} (DataAnalysis)`, value: "DataAnalysis", icon: DataAnalysis }
  ];
  if (formData.icon && !options.some(option => option.value === formData.icon)) {
    options.push({ label: formData.icon, value: formData.icon, icon: CollectionTag });
  }
  return options;
});
const presetColors = ["#3a7afe", "#e5484d", "#f59e0b", "#16a34a", "#8b5cf6", "#0ea5e9", "#64748b"];
const categoryIcons: Record<string, Component> = {
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

const rules = computed(() => ({
  code: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.message_category.field.code") }),
      trigger: "blur"
    }
  ],
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.message_category.field.name") }),
      trigger: "blur"
    }
  ]
}));

const formFields = computed<ProFormField[]>(() => [
  {
    prop: "code",
    label: t("system.base.message_category.field.code"),
    labelTooltip: t("system.base.message_category.tooltip.code"),
    component: "input",
    props: { maxlength: 50, placeholder: t("system.base.message_category.placeholder.code") }
  },
  {
    prop: "name",
    label: t("system.base.message_category.field.name"),
    labelTooltip: t("system.base.message_category.tooltip.name"),
    component: "input",
    props: { maxlength: 50, placeholder: t("system.base.message_category.placeholder.name") }
  },
  {
    prop: "icon",
    label: t("system.base.message_category.field.icon"),
    labelTooltip: t("system.base.message_category.tooltip.icon"),
    component: "select",
    options: iconOptions.value,
    props: { filterable: true, clearable: true, placeholder: t("system.base.message_category.placeholder.icon") }
  },
  {
    prop: "color",
    label: t("system.base.message_category.field.color"),
    labelTooltip: t("system.base.message_category.tooltip.color"),
    component: "color-picker",
    props: { showAlpha: false, predefine: presetColors }
  },
  {
    prop: "sort",
    label: t("common.field.sort"),
    component: "input-number",
    props: { min: 0, precision: 0, placeholder: t("system.base.message_category.placeholder.sort") }
  },
  {
    prop: "default_priority",
    label: t("system.base.message_category.field.default_priority"),
    labelTooltip: t("system.base.message_category.tooltip.default_priority"),
    component: "select",
    options: priorityOptions.value
  },
  {
    prop: "retention_days",
    label: t("system.base.message_category.field.retention_days"),
    labelTooltip: t("system.base.message_category.tooltip.retention_days"),
    component: "input-number",
    props: { min: 0, precision: 0, placeholder: t("system.base.message_category.placeholder.retention_days") }
  },
  {
    prop: "allow_archive",
    label: t("system.base.message_category.field.allow_archive"),
    labelTooltip: t("system.base.message_category.tooltip.allow_archive"),
    component: "switch"
  },
  {
    prop: "allow_delete",
    label: t("system.base.message_category.field.allow_delete"),
    labelTooltip: t("system.base.message_category.tooltip.allow_delete"),
    component: "switch"
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: t("system.base.message_category.field.name"), minWidth: 150, search: { el: "input" } },
  { prop: "code", label: t("system.base.message_category.field.code"), minWidth: 130, search: { el: "input" } },
  {
    prop: "icon",
    label: t("system.base.message_category.field.icon"),
    width: 90,
    align: "center",
    render: scope => {
      const row = scope.row as BaseMessageCategory;
      return h(resolveCategoryIcon(row.icon), {
        style: { color: row.color || "var(--el-text-color-secondary)", width: "18px", height: "18px" }
      });
    }
  },
  {
    prop: "color",
    label: t("system.base.message_category.field.color"),
    width: 90,
    align: "center",
    render: scope =>
      h("span", {
        style: {
          display: "inline-block",
          width: "18px",
          height: "18px",
          borderRadius: "4px",
          backgroundColor: (scope.row as BaseMessageCategory).color || "var(--el-fill-color-light)"
        }
      })
  },
  {
    prop: "default_priority",
    label: t("system.base.message_category.field.default_priority"),
    minWidth: 120,
    render: scope => priorityLabel((scope.row as BaseMessageCategory).default_priority)
  },
  { prop: "retention_days", label: t("system.base.message_category.field.retention_days"), minWidth: 100, align: "right" },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 110,
    search: { el: "select", enum: statusOptions.value },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !isDefaultTenant.value || !BUTTONS.value["base:message-category:status"],
      beforeChange: scope => handleSetStatus(scope.row as BaseMessageCategory)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        link: true,
        icon: EditPen,
        hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-category:update"],
        onClick: scope => openDialog((scope.row as BaseMessageCategory).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-category:delete"],
        onClick: scope => handleDelete(scope.row as BaseMessageCategory)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "primary",
    icon: CirclePlus,
    hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-category:create"],
    onClick: () => openDialog()
  },
  {
    label: t("common.action.batch_delete"),
    type: "danger",
    icon: Delete,
    hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-category:delete"],
    disabled: scope => !scope.isSelected,
    onClick: scope => handleDelete(scope.selectedListIds as number[])
  }
]);

/** 创建默认消息分类表单。 */
function defaultForm(): BaseMessageCategoryForm {
  return {
    id: 0,
    code: "",
    name: "",
    icon: "Bell",
    color: "#3a7afe",
    sort: 0,
    default_priority: MessagePriority.MESSAGE_PRIORITY_NORMAL,
    retention_days: 180,
    allow_archive: true,
    allow_delete: true,
    status: Status.STATUS_ENABLE
  };
}

/** 请求消息分类表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defBaseMessageCategoryService.PageBaseMessageCategory(
    buildPageRequest<PageBaseMessageCategoryRequest>(params as unknown as PageBaseMessageCategoryRequest)
  );
  return { data: { list: data.base_message_categories ?? [], total: data.total } };
}

/** 打开消息分类表单。 */
async function openDialog(id?: number) {
  Object.assign(formData, defaultForm());
  dialog.titleKey = id ? "common.action.edit" : "common.action.create";
  if (id) Object.assign(formData, await defBaseMessageCategoryService.GetBaseMessageCategory({ id }));
  dialog.visible = true;
}

/** 提交消息分类表单。 */
async function handleSubmit() {
  const payload = formData as BaseMessageCategoryForm;
  if (payload.id) await defBaseMessageCategoryService.UpdateBaseMessageCategory({ base_message_category: payload });
  else await defBaseMessageCategoryService.CreateBaseMessageCategory({ base_message_category: payload });
  ElMessage.success(t("common.message.operation_success"));
  dialog.visible = false;
  proTable.value?.getTableList();
}

/** 删除消息分类。 */
async function handleDelete(value: BaseMessageCategory | BaseMessageCategory[] | number | number[]) {
  const ids = normalizeSelectedIds(value as Parameters<typeof normalizeSelectedIds>[0]);
  await ElMessageBox.confirm(t("common.confirm.delete"), t("common.tips"), { type: "warning" });
  await defBaseMessageCategoryService.DeleteBaseMessageCategory({ id: ids.join(",") });
  ElMessage.success(t("common.message.operation_success"));
  proTable.value?.getTableList();
}

/** 设置消息分类状态。 */
async function handleSetStatus(row: BaseMessageCategory) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  await defBaseMessageCategoryService.SetBaseMessageCategoryStatus({ id: row.id, status });
  return true;
}

/** 关闭消息分类表单。 */
function handleClose() {
  Object.assign(formData, defaultForm());
  formDialogRef.value?.resetFields();
}

/** 格式化消息优先级。 */
function priorityLabel(priority: MessagePriority) {
  return priorityOptions.value.find(item => item.value === priority)?.label ?? "";
}

/** 解析消息分类图标，兼容历史数据中的未知图标名。 */
function resolveCategoryIcon(icon: string) {
  return categoryIcons[icon] ?? CollectionTag;
}
</script>
