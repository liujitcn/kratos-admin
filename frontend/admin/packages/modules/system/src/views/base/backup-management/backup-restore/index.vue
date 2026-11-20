<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBackupRestoreTable"
    />
    <FormDialog
      ref="formDialogRef"
      v-model="dialogVisible"
      :title="t('system.backup.backup.restore.manual')"
      :model="formData"
      :fields="formFields"
      :rules="formRules"
      :confirm-text="t('system.backup.action.restore')"
      :confirm-loading="submitting"
      label-width="160px"
      width="600px"
      @confirm="submitRestore"
      @close="resetForm"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { t } from "@liujitcn/kratos-admin-core";
import { formatDateTime } from "@liujitcn/kratos-admin-core/format";
import { resolvePersistedMessage } from "../../../../utils/persisted-message";
import { defBaseTableBackupRestoreService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_backup_restore";
import { defBaseTableSourceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_source";
import {
  BaseTableBackupRestoreMode,
  BaseTableBackupRestoreStatus,
  type BaseTableBackupRestore,
  type PageBaseTableBackupRestoreRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_restore";

defineOptions({ name: "BaseTableBackupRestore", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const { BUTTONS } = useAuthButtons();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialogVisible = ref(false);
const submitting = ref(false);
const sourceOptions = ref<ProFormOption[]>([]);
const formData = reactive<BaseTableBackupRestore>(defaultForm());
const modeOptions = computed<ProFormOption[]>(() => [
  {
    label: t("system.backup.restore.mode.verify_only"),
    value: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY
  },
  { label: t("system.backup.restore.mode.full"), value: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_FULL }
]);
const statusOptions = computed(() => [
  {
    label: t("system.backup.status.pending"),
    value: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_PENDING,
    tagType: "info"
  },
  {
    label: t("system.backup.status.running"),
    value: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_RUNNING,
    tagType: "warning"
  },
  {
    label: t("system.backup.status.success"),
    value: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_SUCCESS,
    tagType: "success"
  },
  {
    label: t("system.backup.status.failed"),
    value: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_FAILED,
    tagType: "danger"
  }
]);
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "backup_record_id",
    label: t("system.backup.field.backup_record_id"),
    component: "input-number",
    props: { min: 0, precision: 0 }
  },
  {
    prop: "target_source_name",
    label: t("system.backup.field.target_source_name"),
    component: "select",
    options: sourceOptions.value,
    props: { filterable: true, placeholder: t("system.backup.placeholder.source_name") }
  },
  { prop: "target_database", label: t("system.backup.field.target_database"), component: "input" },
  { prop: "restore_mode", label: t("system.backup.field.restore_mode"), component: "select", options: modeOptions.value }
]);
const formRules = computed(() => ({
  backup_record_id: [
    { required: true, message: t("system.backup.validation.backup_record_id"), trigger: "blur" },
    { type: "number", min: 1, message: t("system.backup.validation.backup_record_id_positive"), trigger: "change" }
  ],
  target_source_name: [{ required: true, message: t("system.backup.validation.target_source_name"), trigger: "blur" }],
  target_database: [{ required: true, message: t("system.backup.validation.target_database"), trigger: "blur" }]
}));
const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.backup.field.id"), width: 90 },
  { prop: "backup_record_id", label: t("system.backup.field.backup_record_id"), width: 130 },
  { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 130 },
  { prop: "target_source_name", label: t("system.backup.field.target_source_name"), minWidth: 130 },
  { prop: "target_database", label: t("system.backup.field.target_database"), minWidth: 150 },
  { prop: "restore_mode", label: t("system.backup.field.restore_mode"), width: 120, enum: modeOptions.value },
  { prop: "status", label: t("common.field.status"), width: 110, search: { el: "select" }, enum: statusOptions.value, tag: true },
  {
    prop: "error",
    label: t("system.backup.field.error"),
    minWidth: 220,
    render: scope => resolvePersistedMessage(scope.row.error)
  },
  {
    prop: "started_at",
    align: "center",
    label: t("system.backup.field.started_at"),
    minWidth: 170,
    render: scope => formatDateTime(scope.row.started_at)
  },
  {
    prop: "finished_at",
    align: "center",
    label: t("system.backup.field.finished_at"),
    minWidth: 170,
    render: scope => formatDateTime(scope.row.finished_at)
  }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("system.backup.backup.restore.manual"),
    type: "primary",
    icon: CirclePlus,
    hidden: !BUTTONS.value["base:table-backup-restore:execute"],
    onClick: openDialog
  }
]);

/** 请求备份恢复记录分页列表。 */
async function requestBackupRestoreTable(params: PageBaseTableBackupRestoreRequest) {
  const data = await defBaseTableBackupRestoreService.PageBaseTableBackupRestore(buildPageRequest(params));
  return { data: { list: data.base_table_backup_restores ?? [], total: data.total } };
}
/** 生成备份恢复表单默认值。 */
function defaultForm(): BaseTableBackupRestore {
  return {
    id: 0,
    backup_record_id: 0,
    source_name: "",
    target_source_name: "default",
    target_database: "",
    restore_mode: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY,
    operator_id: 0,
    status: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_PENDING,
    error: "",
    started_at: "",
    finished_at: ""
  };
}
/** 关闭恢复弹窗并重置表单和校验状态。 */
function resetForm() {
  dialogVisible.value = false;
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, defaultForm());
}
/** 加载目标数据源后打开手工恢复弹窗。 */
async function openDialog() {
  await formDialogRef.value?.open({
    load: () => defBaseTableSourceService.OptionBaseTableSource({}),
    commit: data => {
      sourceOptions.value = (data.value ?? []).map(value => ({ label: value, value }));
      if (!sourceOptions.value.some(item => item.value === formData.target_source_name))
        formData.target_source_name = String(sourceOptions.value[0]?.value ?? "");
    }
  });
}
/** 手工提交备份恢复请求。 */
async function submitRestore() {
  if (!(await formDialogRef.value?.validate())) return;
  if (!Number.isInteger(formData.backup_record_id) || formData.backup_record_id < 1) {
    ElMessage.warning(t("system.backup.validation.backup_record_id_positive"));
    return;
  }
  submitting.value = true;
  try {
    await defBaseTableBackupRestoreService.ExecuteBaseTableBackupRestore({ base_table_backup_restore: { ...formData } });
    ElMessage.success(t("system.backup.message.restore_submitted"));
    resetForm();
    proTable.value?.getTableList();
  } finally {
    submitting.value = false;
  }
}
</script>
