<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'common.action.edit' : 'common.action.create')"
      width="min(1000px, calc(100vw - 32px))"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleClose"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseMessageProviderService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message_provider";
import type {
  BaseMessageProvider,
  BaseMessageProviderForm,
  PageBaseMessageProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_provider";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseMessageProvider", inheritAttrs: false });

interface ConfigItem {
  key: string;
  value: string;
}

interface FormState extends BaseMessageProviderForm {
  config_items: ConfigItem[];
}

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, editing: false });
const formData = reactive<FormState>(defaultForm());

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const rules = computed(() => ({
  provider: [{ required: true, message: t("system.base.message_provider.validation.provider"), trigger: "blur" }],
  name: [{ required: true, message: t("system.base.message_provider.validation.name"), trigger: "blur" }]
}));

const formFields = computed<ProFormField[]>(() => [
  { prop: "provider", label: t("system.base.message_provider.field.provider"), component: "input", props: { disabled: dialog.editing, maxlength: 32 } },
  { prop: "name", label: t("system.base.message_provider.field.name"), component: "input", props: { maxlength: 50 } },
  { prop: "description", label: t("system.base.message_provider.field.description"), component: "input", colSpan: 24, props: { maxlength: 255 } },
  { prop: "icon", label: t("system.base.message_provider.field.icon"), component: "input", props: { maxlength: 100 } },
  { prop: "client_id", label: t("system.base.message_provider.field.client_id"), component: "input", props: { maxlength: 255 } },
  { prop: "client_secret", label: t("system.base.message_provider.field.client_secret"), component: "input", props: { type: "password", showPassword: true, maxlength: 1024, placeholder: dialog.editing ? t("system.base.message_provider.placeholder.keep_secret") : "" } },
  { prop: "config_items", label: t("system.base.message_provider.field.config"), component: "kv-list", props: { keyInputProps: { maxlength: 128 } } },
  { prop: "sort", label: t("common.field.sort"), component: "input-number", props: { min: 0, precision: 0 } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: t("system.base.message_provider.field.name"), minWidth: 150, search: { el: "input" } },
  { prop: "provider", label: t("system.base.message_provider.field.provider"), minWidth: 130, search: { el: "input" } },
  { prop: "client_id", label: t("system.base.message_provider.field.client_id"), minWidth: 180 },
  { prop: "secret_configured", label: t("system.base.message_provider.field.secret_configured"), width: 120, render: scope => (scope.row as BaseMessageProvider).secret_configured ? t("common.status.configured") : t("common.status.not_configured") },
  { prop: "sort", label: t("common.field.sort"), width: 80, align: "right" },
  { prop: "status", label: t("common.field.status"), width: 110, search: { el: "select", enum: statusOptions.value }, cellType: "status", statusProps: {
    activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"),
    disabled: () => !isDefaultTenant.value || !BUTTONS.value["base:message-provider:status"], beforeChange: scope => handleSetStatus(scope.row as BaseMessageProvider)
  } },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
  { prop: "operation", label: t("common.field.operation"), cellType: "actions", actions: [
    { label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-provider:update"], onClick: scope => openDialog((scope.row as BaseMessageProvider).id) },
    { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-provider:delete"], onClick: scope => handleDelete(scope.row as BaseMessageProvider) }
  ] }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-provider:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-provider:delete"], disabled: scope => !scope.isSelected, onClick: scope => handleDelete(scope.selectedListIds as number[]) }
]);

function defaultForm(): FormState {
  return { id: 0, provider: "", name: "", description: "", icon: "", client_id: "", client_secret: "", secret_configured: false, config: {}, config_items: [], sort: 0, status: Status.STATUS_DISABLE };
}

async function requestTable(params: Record<string, unknown>) {
  const data = await defBaseMessageProviderService.PageBaseMessageProvider(buildPageRequest<PageBaseMessageProviderRequest>(params as unknown as PageBaseMessageProviderRequest));
  return { data: { list: data.base_message_providers ?? [], total: data.total } };
}

async function openDialog(id?: number) {
  await formDialogRef.value?.open({
    load: () => (id ? defBaseMessageProviderService.GetBaseMessageProvider({ id }) : undefined),
    commit: data => { Object.assign(formData, defaultForm(), data ?? {}); dialog.editing = Boolean(id); formData.config_items = configToItems(formData.config); }
  });
}

async function handleSubmit() {
  if (!(await formDialogRef.value?.validate())) return;
  const config = configItemsToMap(formData.config_items);
  if (!config) return;
  const payload = JSON.parse(JSON.stringify(formData)) as BaseMessageProviderForm & { config_items?: ConfigItem[] };
  delete payload.config_items;
  payload.config = config;
  if (payload.id) await defBaseMessageProviderService.UpdateBaseMessageProvider({ base_message_provider: payload });
  else await defBaseMessageProviderService.CreateBaseMessageProvider({ base_message_provider: payload });
  ElMessage.success(t("common.message.operation_success"));
  dialog.visible = false;
  await proTable.value?.getTableList();
}

async function handleDelete(value: BaseMessageProvider | number | number[]) {
  const ids = (Array.isArray(value) ? value : typeof value === "object" ? [value.id] : normalizeSelectedIds(value)).join(",");
  if (!ids) return;
  try {
    await ElMessageBox.confirm(t("common.confirm.delete"), t("common.title.warning"), { type: "warning" });
  } catch {
    return;
  }
  await defBaseMessageProviderService.DeleteBaseMessageProvider({ id: ids });
  await proTable.value?.getTableList();
  ElMessage.success(t("common.message.operation_success"));
}

async function handleSetStatus(row: BaseMessageProvider) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(t("common.dialog.status_change", { action: t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"), resource: t("system.base.message_provider.resource"), field: t("system.base.message_provider.field.name"), value: row.name }), t("common.title.notice"), { type: "warning" });
    await defBaseMessageProviderService.SetBaseMessageProviderStatus({ id: row.id, status });
    return true;
  } catch { return false; }
}

function handleClose() { Object.assign(formData, defaultForm()); formDialogRef.value?.resetFields(); }

function configToItems(config: Record<string, any> | undefined): ConfigItem[] { return Object.entries(config ?? {}).map(([key, value]) => ({ key, value: typeof value === "string" ? value : JSON.stringify(value) })); }

function configItemsToMap(items: ConfigItem[]): Record<string, unknown> | undefined {
  const result: Record<string, unknown> = {};
  for (const item of items) {
    const key = item.key.trim();
    if (!key || Object.prototype.hasOwnProperty.call(result, key)) { ElMessage.error(t("system.base.message_provider.message.config_invalid")); return undefined; }
    const value = item.value.trim();
    if (!value) { result[key] = ""; continue; }
    try { result[key] = JSON.parse(value); } catch { result[key] = value; }
  }
  return result;
}
</script>
