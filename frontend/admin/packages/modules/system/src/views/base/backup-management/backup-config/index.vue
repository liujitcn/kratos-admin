<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      ref="formDialogRef"
      v-model="dialog.visible"
      :title="t(dialog.editing ? 'common.action.edit_resource' : 'common.action.create_resource', { resource: t('system.backup.backup.config') })"
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
import { defBaseTableBackupService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_backup";
import { defBaseTableSourceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_source";
import { BaseTableBackupType, type BaseTableBackupForm, type PageBaseTableBackupRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseTableBackupConfig", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, editing: false });
const formData = reactive<BaseTableBackupForm>(defaultForm());
const sourceOptions = ref<ProFormOption[]>([]);
const loadingSources = ref(false);
const backupTypeOptions = computed<ProFormOption[]>(() => [{ label: t("system.backup.backup.type.full"), value: BaseTableBackupType.BASE_TABLE_BACKUP_TYPE_FULL }]);
const statusOptions = computed<ProFormOption[]>(() => [{ label: t("common.status.enabled"), value: Status.STATUS_ENABLE }, { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }]);
const formFields = computed<ProFormField[]>(() => [
  { prop: "source_name", label: t("system.backup.field.source_name"), component: "select", options: sourceOptions.value, props: { loading: loadingSources.value, filterable: true, placeholder: t("system.backup.placeholder.source_name") } },
  { prop: "backup_type", label: t("system.backup.field.backup_type"), component: "select", options: backupTypeOptions.value },
  { prop: "oss_prefix", label: t("system.backup.field.oss_prefix"), component: "input" },
  { prop: "retention_count", label: t("system.backup.field.retention_count"), component: "input-number", props: { min: 1, max: 1000 } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);
const formRules = computed(() => ({ source_name: [{ required: true, message: t("system.backup.validation.source_name"), trigger: "change" }] }));
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 150 },
  { prop: "backup_type", label: t("system.backup.field.backup_type"), width: 110, render: scope => backupTypeLabel(scope.row.backup_type) },
  { prop: "oss_prefix", label: t("system.backup.field.oss_prefix"), minWidth: 180 },
  { prop: "retention_count", label: t("system.backup.field.retention_count"), width: 110 },
  { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:table-backup:status"], beforeChange: scope => setStatus(scope.row as BaseTableBackupForm) } },
  { prop: "operation", label: t("common.field.operation"), width: 180, fixed: "right", cellType: "actions", actions: [{ label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:table-backup:update"], onClick: scope => openDialog((scope.row as BaseTableBackupForm).id) }, { label: t("common.action.delete"), link: true, type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:table-backup:delete"], onClick: scope => remove(scope.row as BaseTableBackupForm) }] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !BUTTONS.value["base:table-backup:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !BUTTONS.value["base:table-backup:delete"], disabled: scope => !scope.isSelected, onClick: scope => remove(scope.selectedListIds as number[]) }]);

/** 请求备份配置分页列表。 */
async function requestTable(params: PageBaseTableBackupRequest) {
  const data = await defBaseTableBackupService.PageBaseTableBackup(buildPageRequest(params));
  return { data: { list: data.base_table_backups ?? [], total: data.total } };
}
function refresh() { proTable.value?.getTableList(); }
function defaultForm(): BaseTableBackupForm { return { id: 0, source_name: "", backup_type: BaseTableBackupType.BASE_TABLE_BACKUP_TYPE_FULL, oss_prefix: "backup/database", retention_count: 7, status: Status.STATUS_ENABLE }; }
function resetForm() { dialog.visible = false; formDialogRef.value?.resetFields(); Object.assign(formData, defaultForm()); }
async function openDialog(id?: number) {
  await loadSourceOptions();
  Object.assign(formData, defaultForm());
  dialog.editing = Boolean(id);
  if (id) Object.assign(formData, await defBaseTableBackupService.GetBaseTableBackup({ id }));
  if (!formData.source_name) formData.source_name = String(sourceOptions.value[0]?.value ?? "");
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
async function handleSubmit() { const valid = await formDialogRef.value?.validate(); if (!valid) return; const payload = JSON.parse(JSON.stringify(formData)) as BaseTableBackupForm; if (payload.id) await defBaseTableBackupService.UpdateBaseTableBackup({ base_table_backup: payload }); else await defBaseTableBackupService.CreateBaseTableBackup({ base_table_backup: payload }); ElMessage.success(t("common.message.save_success")); resetForm(); refresh(); }
/** 确认并切换备份配置状态。 */
async function setStatus(row: BaseTableBackupForm) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  const name = row.source_name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action,
        resource: t("system.backup.backup.config"),
        field: t("system.backup.field.source_name"),
        value: name
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseTableBackupService.SetBaseTableBackupStatus({ id: row.id, status });
    ElMessage.success(t("common.message.status_success", { action }));
    await proTable.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}
async function remove(selected: number | number[] | BaseTableBackupForm) { let ids: number[]; if (Array.isArray(selected)) ids = selected.map(Number); else if (typeof selected === "object") ids = [selected.id]; else ids = normalizeSelectedIds(selected).map(Number); if (!ids.length) return; await ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.backup.backup.config") }), t("common.title.warning"), { type: "warning" }); await defBaseTableBackupService.DeleteBaseTableBackup({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.backup.backup.config") })); refresh(); }
function backupTypeLabel(value: BaseTableBackupType) { return backupTypeOptions.value.find(item => item.value === value)?.label ?? String(value); }
</script>
