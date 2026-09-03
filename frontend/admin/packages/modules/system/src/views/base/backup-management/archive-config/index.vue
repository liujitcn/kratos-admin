<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      ref="formDialogRef"
      v-model="dialog.visible"
      :title="t(dialog.editing ? 'common.action.edit_resource' : 'common.action.create_resource', { resource: t('system.backup.archive.config') })"
      :model="formData"
      :fields="formFields"
      :rules="formRules"
      @confirm="handleSubmit"
      @close="resetForm"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTableArchiveService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_archive";
import { defBaseTableSourceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_source";
import { BaseTableArchiveMode, type BaseTableArchiveForm, type PageBaseTableArchiveRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseTableArchiveConfig", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, editing: false });
const formData = reactive<BaseTableArchiveForm>(defaultForm());
const sourceOptions = ref<ProFormOption[]>([]);
const tableOptions = ref<ProFormOption[]>([]);
const loadingSources = ref(false);
const loadingTables = ref(false);
const modeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.backup.archive.mode.internal"), value: BaseTableArchiveMode.BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE },
  { label: t("system.backup.archive.mode.oss"), value: BaseTableArchiveMode.BASE_TABLE_ARCHIVE_MODE_OSS }
]);
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const formFields = computed<ProFormField[]>(() => [
  { prop: "source_name", label: t("system.backup.field.source_name"), component: "select", options: sourceOptions.value, props: { loading: loadingSources.value, filterable: true, placeholder: t("system.backup.placeholder.source_name"), onChange: handleSourceChange } },
  { prop: "table_name", label: t("system.backup.field.table_name"), component: "select", options: tableOptions.value, props: { loading: loadingTables.value, filterable: true, placeholder: t("system.backup.placeholder.table_name"), disabled: !formData.source_name } },
  { prop: "archive_mode", label: t("system.backup.field.archive_mode"), component: "select", options: modeOptions.value },
  { prop: "online_retention_days", label: t("system.backup.field.online_retention_days"), component: "input-number", props: { min: 0, max: 36500 } },
  { prop: "archive_retention_days", label: t("system.backup.field.archive_retention_days"), component: "input-number", props: { min: 0, max: 36500 } },
  { prop: "batch_size", label: t("system.backup.field.batch_size"), component: "input-number", props: { min: 1, max: 100000 } },
  { prop: "delete_after_verify", label: t("system.backup.field.delete_after_verify"), component: "switch" },
  { prop: "oss_prefix", label: t("system.backup.field.oss_prefix"), component: "input" },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);
const formRules = computed(() => ({
  source_name: [{ required: true, message: t("system.backup.validation.source_name"), trigger: "change" }],
  table_name: [{ required: true, message: t("system.backup.validation.table_name"), trigger: "change" }]
}));
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 140 },
  { prop: "table_name", label: t("system.backup.field.table_name"), minWidth: 180 },
  { prop: "archive_mode", label: t("system.backup.field.archive_mode"), minWidth: 130, render: scope => modeLabel(scope.row.archive_mode) },
  { prop: "online_retention_days", label: t("system.backup.field.online_retention_days"), width: 120 },
  { prop: "archive_retention_days", label: t("system.backup.field.archive_retention_days"), width: 120 },
  { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:table-archive:status"], beforeChange: scope => setStatus(scope.row as BaseTableArchiveForm) } },
  { prop: "operation", label: t("common.field.operation"), width: 180, fixed: "right", cellType: "actions", actions: [{ label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:table-archive:update"], onClick: scope => openDialog((scope.row as BaseTableArchiveForm).id) }, { label: t("common.action.delete"), link: true, type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:table-archive:delete"], onClick: scope => remove(scope.row as BaseTableArchiveForm) }] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !BUTTONS.value["base:table-archive:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !BUTTONS.value["base:table-archive:delete"], disabled: scope => !scope.isSelected, onClick: scope => remove(scope.selectedListIds as number[]) }]);

async function requestTable(params: PageBaseTableArchiveRequest) {
  const data = await defBaseTableArchiveService.PageBaseTableArchive(buildPageRequest(params));
  return { data: { list: data.base_table_archives ?? [], total: data.total } };
}
function refresh() { proTable.value?.getTableList(); }
function defaultForm(): BaseTableArchiveForm { return { id: 0, source_name: "", table_name: "", archive_mode: BaseTableArchiveMode.BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE, online_retention_days: 180, archive_retention_days: 3650, batch_size: 5000, delete_after_verify: false, oss_prefix: "archive", status: Status.STATUS_ENABLE }; }
function resetForm() { dialog.visible = false; formDialogRef.value?.resetFields(); Object.assign(formData, defaultForm()); }
async function openDialog(id?: number) {
  await loadSourceOptions();
  Object.assign(formData, defaultForm());
  dialog.editing = Boolean(id);
  if (id) Object.assign(formData, await defBaseTableArchiveService.GetBaseTableArchive({ id }));
  if (!formData.source_name) formData.source_name = String(sourceOptions.value[0]?.value ?? "");
  await loadTableOptions(formData.source_name);
  dialog.visible = true;
}
async function loadSourceOptions() {
  if (sourceOptions.value.length || loadingSources.value) return;
  loadingSources.value = true;
  try {
    const data = await defBaseTableSourceService.OptionBaseTableSource({});
    sourceOptions.value = (data.value ?? []).map(value => ({ label: value, value }));
  } finally {
    loadingSources.value = false;
  }
}
async function handleSourceChange(value: string | number | boolean | undefined) {
  formData.source_name = String(value ?? "");
  formData.table_name = "";
  await loadTableOptions(formData.source_name);
}
async function loadTableOptions(sourceName: string) {
  tableOptions.value = [];
  if (!sourceName) return;
  loadingTables.value = true;
  try {
    const data = await defBaseTableSourceService.OptionBaseTable({ source_name: sourceName });
    tableOptions.value = (data.value ?? []).map(value => ({ label: value, value }));
  } finally {
    loadingTables.value = false;
  }
}
async function handleSubmit() { const valid = await formDialogRef.value?.validate(); if (!valid) return; const payload = JSON.parse(JSON.stringify(formData)) as BaseTableArchiveForm; if (payload.id) await defBaseTableArchiveService.UpdateBaseTableArchive({ base_table_archive: payload }); else await defBaseTableArchiveService.CreateBaseTableArchive({ base_table_archive: payload }); ElMessage.success(t("common.message.save_success")); resetForm(); refresh(); }
/** 确认并切换归档配置状态。 */
async function setStatus(row: BaseTableArchiveForm) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  const name = row.table_name || row.source_name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action,
        resource: t("system.backup.archive.config"),
        field: t("system.backup.field.table_name"),
        value: name
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseTableArchiveService.SetBaseTableArchiveStatus({ id: row.id, status });
    ElMessage.success(t("common.message.status_success", { action }));
    await proTable.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}
async function remove(selected: number | number[] | BaseTableArchiveForm) { let ids: number[]; if (Array.isArray(selected)) ids = selected.map(Number); else if (typeof selected === "object") ids = [selected.id]; else ids = normalizeSelectedIds(selected).map(Number); if (!ids.length) return; await ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.backup.archive.config") }), t("common.title.warning"), { type: "warning" }); await defBaseTableArchiveService.DeleteBaseTableArchive({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.backup.archive.config") })); refresh(); }
function modeLabel(value: BaseTableArchiveMode) { return modeOptions.value.find(item => item.value === value)?.label ?? String(value); }
</script>
