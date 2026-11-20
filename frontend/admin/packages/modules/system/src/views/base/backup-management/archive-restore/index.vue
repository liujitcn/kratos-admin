<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestArchiveRestoreTable"
    />
    <FormDialog
      ref="formDialogRef"
      v-model="dialogVisible"
      :title="t('system.backup.archive.restore.manual')"
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
import { defBaseTableArchiveRestoreService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_archive_restore";
import {
  BaseTableArchiveRestoreMode,
  BaseTableArchiveRestoreStatus,
  type BaseTableArchiveRestore,
  type PageBaseTableArchiveRestoreRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_restore";

defineOptions({ name: "BaseTableArchiveRestore", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const { BUTTONS } = useAuthButtons();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialogVisible = ref(false);
const submitting = ref(false);
const formData = reactive<BaseTableArchiveRestore>(defaultForm());
const modeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.backup.restore.mode.all"), value: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL },
  { label: t("system.backup.restore.mode.selected"), value: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED }
]);
const statusOptions = computed(() => [
  {
    label: t("system.backup.status.pending"),
    value: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_PENDING,
    tagType: "info"
  },
  {
    label: t("system.backup.status.running"),
    value: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_RUNNING,
    tagType: "warning"
  },
  {
    label: t("system.backup.status.success"),
    value: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_SUCCESS,
    tagType: "success"
  },
  {
    label: t("system.backup.status.failed"),
    value: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_FAILED,
    tagType: "danger"
  }
]);
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "archive_record_id",
    label: t("system.backup.field.archive_record_id"),
    component: "input-number",
    props: { min: 0, precision: 0 }
  },
  { prop: "restore_mode", label: t("system.backup.field.restore_mode"), component: "select", options: modeOptions.value },
  {
    prop: "restore_range",
    label: t("system.backup.field.restore_range"),
    component: "input",
    visible: model => model.restore_mode === BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED,
    props: { placeholder: '{"start_id":1,"end_id":100}' }
  }
]);
const formRules = computed(() => ({
  archive_record_id: [
    { required: true, message: t("system.backup.validation.archive_record_id"), trigger: "blur" },
    { type: "number", min: 1, message: t("system.backup.validation.archive_record_id_positive"), trigger: "change" }
  ]
}));
const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.backup.field.id"), width: 90 },
  { prop: "archive_record_id", label: t("system.backup.field.archive_record_id"), width: 130 },
  { prop: "table_name", label: t("system.backup.field.table_name"), minWidth: 180 },
  { prop: "restore_mode", label: t("system.backup.field.restore_mode"), width: 120, enum: modeOptions.value },
  { prop: "restored_rows", label: t("system.backup.field.restored_rows"), width: 110 },
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
    label: t("system.backup.archive.restore.manual"),
    type: "primary",
    icon: CirclePlus,
    hidden: !BUTTONS.value["base:table-archive-restore:execute"],
    onClick: openDialog
  }
]);

/** 请求归档恢复记录分页列表。 */
async function requestArchiveRestoreTable(params: PageBaseTableArchiveRestoreRequest) {
  const data = await defBaseTableArchiveRestoreService.PageBaseTableArchiveRestore(buildPageRequest(params));
  return { data: { list: data.base_table_archive_restores ?? [], total: data.total } };
}
/** 生成归档恢复表单默认值。 */
function defaultForm(): BaseTableArchiveRestore {
  return {
    id: 0,
    archive_record_id: 0,
    table_name: "",
    restore_mode: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL,
    restore_range: "",
    restored_rows: 0,
    operator_id: 0,
    status: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_PENDING,
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
/** 打开手工恢复归档弹窗。 */
function openDialog() {
  dialogVisible.value = true;
}
/** 手工提交归档恢复请求。 */
async function submitRestore() {
  if (!(await formDialogRef.value?.validate())) return;
  if (!Number.isInteger(formData.archive_record_id) || formData.archive_record_id < 1) {
    ElMessage.warning(t("system.backup.validation.archive_record_id_positive"));
    return;
  }
  submitting.value = true;
  try {
    await defBaseTableArchiveRestoreService.ExecuteBaseTableArchiveRestore({ base_table_archive_restore: { ...formData } });
    ElMessage.success(t("system.backup.message.restore_submitted"));
    resetForm();
    proTable.value?.getTableList();
  } finally {
    submitting.value = false;
  }
}
</script>
