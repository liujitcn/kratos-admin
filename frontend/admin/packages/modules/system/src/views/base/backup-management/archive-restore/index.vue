<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestTable" />
    <el-card class="restore-card">
      <template #header>{{ t("system.backup.archive.restore.manual") }}</template>
      <ProForm ref="formRef" :model="formData" :fields="formFields" :rules="formRules" label-width="160px" />
      <el-button v-if="BUTTONS['base:table-archive-restore:execute']" type="primary" :loading="submitting" @click="submitRestore">{{ t("system.backup.action.restore") }}</el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormInstance, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTableArchiveRestoreService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_archive_restore";
import { BaseTableArchiveRestoreMode, BaseTableArchiveRestoreStatus, type BaseTableArchiveRestore } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_restore";

defineOptions({ name: "BaseTableArchiveRestore", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const { BUTTONS } = useAuthButtons();
const formRef = ref<ProFormInstance>();
const submitting = ref(false);
const formData = reactive<BaseTableArchiveRestore>({ id: 0, archive_record_id: 0, table_name: "", restore_mode: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL, restore_range: "", restored_rows: 0, operator_id: 0, status: BaseTableArchiveRestoreStatus.BASE_TABLE_ARCHIVE_RESTORE_STATUS_PENDING, error: "", started_at: "", finished_at: "" });
const modeOptions = computed<ProFormOption[]>(() => [{ label: t("system.backup.restore.mode.all"), value: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL }, { label: t("system.backup.restore.mode.selected"), value: BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED }]);
const formFields = computed<ProFormField[]>(() => [{ prop: "archive_record_id", label: t("system.backup.field.archive_record_id"), component: "input-number", props: { min: 0, precision: 0 } }, { prop: "restore_mode", label: t("system.backup.field.restore_mode"), component: "select", options: modeOptions.value }, { prop: "restore_range", label: t("system.backup.field.restore_range"), component: "input", visible: model => model.restore_mode === BaseTableArchiveRestoreMode.BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED, props: { placeholder: '{"start_id":1,"end_id":100}' } }]);
const formRules = computed(() => ({
  archive_record_id: [
    { required: true, message: t("system.backup.validation.archive_record_id"), trigger: "blur" },
    { type: "number", min: 1, message: t("system.backup.validation.archive_record_id_positive"), trigger: "change" }
  ]
}));
const columns = computed<ColumnProps[]>(() => [{ prop: "id", label: t("system.backup.field.id"), width: 90 }, { prop: "archive_record_id", label: t("system.backup.field.archive_record_id"), width: 130 }, { prop: "table_name", label: t("system.backup.field.table_name"), minWidth: 180 }, { prop: "restore_mode", label: t("system.backup.field.restore_mode"), width: 120, render: scope => modeOptions.value.find(item => item.value === scope.row.restore_mode)?.label ?? String(scope.row.restore_mode) }, { prop: "restored_rows", label: t("system.backup.field.restored_rows"), width: 110 }, { prop: "status", label: t("common.field.status"), width: 110 }, { prop: "error", label: t("system.backup.field.error"), minWidth: 220 }, { prop: "started_at", label: t("system.backup.field.started_at"), minWidth: 170 }, { prop: "finished_at", label: t("system.backup.field.finished_at"), minWidth: 170 }]);

/** 请求归档恢复记录分页列表。 */
async function requestTable(params: Record<string, unknown>) { const data = await defBaseTableArchiveRestoreService.PageBaseTableArchiveRestore(buildPageRequest(params)); return { data: { list: data.base_table_archive_restores ?? [], total: data.total } }; }
/** 手工提交归档恢复请求。 */
async function submitRestore() {
  if (!(await formRef.value?.validate())) return;
  if (!Number.isInteger(formData.archive_record_id) || formData.archive_record_id < 1) {
    ElMessage.warning(t("system.backup.validation.archive_record_id_positive"));
    return;
  }
  submitting.value = true;
  try {
    await defBaseTableArchiveRestoreService.ExecuteBaseTableArchiveRestore({ base_table_archive_restore: { ...formData } });
    ElMessage.success(t("system.backup.message.restore_submitted"));
    proTable.value?.getTableList();
  } finally {
    submitting.value = false;
  }
}
</script>
