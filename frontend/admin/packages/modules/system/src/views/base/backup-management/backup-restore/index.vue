<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestTable" />
    <el-card class="restore-card">
      <template #header>{{ t("system.backup.backup.restore.manual") }}</template>
      <ProForm ref="formRef" :model="formData" :fields="formFields" :rules="formRules" label-width="160px" />
      <el-button v-if="BUTTONS['base:table-backup-restore:execute']" type="primary" :loading="submitting" @click="submitRestore">{{ t("system.backup.action.restore") }}</el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormInstance, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTableBackupRestoreService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_backup_restore";
import { defBaseTableSourceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_source";
import { BaseTableBackupRestoreMode, BaseTableBackupRestoreStatus, type BaseTableBackupRestore } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_restore";

defineOptions({ name: "BaseTableBackupRestore", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const { BUTTONS } = useAuthButtons();
const formRef = ref<ProFormInstance>();
const submitting = ref(false);
const sourceOptions = ref<ProFormOption[]>([]);
const formData = reactive<BaseTableBackupRestore>({ id: 0, backup_record_id: 0, source_name: "", target_source_name: "default", target_database: "", restore_mode: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY, operator_id: 0, status: BaseTableBackupRestoreStatus.BASE_TABLE_BACKUP_RESTORE_STATUS_PENDING, error: "", started_at: "", finished_at: "" });
const modeOptions = computed<ProFormOption[]>(() => [{ label: t("system.backup.restore.mode.verify_only"), value: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY }, { label: t("system.backup.restore.mode.full"), value: BaseTableBackupRestoreMode.BASE_TABLE_BACKUP_RESTORE_MODE_FULL }]);
const formFields = computed<ProFormField[]>(() => [{ prop: "backup_record_id", label: t("system.backup.field.backup_record_id"), component: "input-number", props: { min: 0, precision: 0 } }, { prop: "target_source_name", label: t("system.backup.field.target_source_name"), component: "select", options: sourceOptions.value, props: { filterable: true, placeholder: t("system.backup.placeholder.source_name") } }, { prop: "target_database", label: t("system.backup.field.target_database"), component: "input" }, { prop: "restore_mode", label: t("system.backup.field.restore_mode"), component: "select", options: modeOptions.value }]);
const formRules = computed(() => ({
  backup_record_id: [
    { required: true, message: t("system.backup.validation.backup_record_id"), trigger: "blur" },
    { type: "number", min: 1, message: t("system.backup.validation.backup_record_id_positive"), trigger: "change" }
  ],
  target_source_name: [{ required: true, message: t("system.backup.validation.target_source_name"), trigger: "blur" }],
  target_database: [{ required: true, message: t("system.backup.validation.target_database"), trigger: "blur" }]
}));
const columns = computed<ColumnProps[]>(() => [{ prop: "id", label: t("system.backup.field.id"), width: 90 }, { prop: "backup_record_id", label: t("system.backup.field.backup_record_id"), width: 130 }, { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 130 }, { prop: "target_source_name", label: t("system.backup.field.target_source_name"), minWidth: 130 }, { prop: "target_database", label: t("system.backup.field.target_database"), minWidth: 150 }, { prop: "restore_mode", label: t("system.backup.field.restore_mode"), width: 120, render: scope => modeOptions.value.find(item => item.value === scope.row.restore_mode)?.label ?? String(scope.row.restore_mode) }, { prop: "status", label: t("common.field.status"), width: 110 }, { prop: "error", label: t("system.backup.field.error"), minWidth: 220 }, { prop: "started_at", label: t("system.backup.field.started_at"), minWidth: 170 }, { prop: "finished_at", label: t("system.backup.field.finished_at"), minWidth: 170 }]);

/** 请求备份恢复记录分页列表。 */
async function requestTable(params: Record<string, unknown>) { const data = await defBaseTableBackupRestoreService.PageBaseTableBackupRestore(buildPageRequest(params)); return { data: { list: data.base_table_backup_restores ?? [], total: data.total } }; }
/** 手工提交备份恢复请求。 */
async function submitRestore() {
  if (!(await formRef.value?.validate())) return;
  if (!Number.isInteger(formData.backup_record_id) || formData.backup_record_id < 1) {
    ElMessage.warning(t("system.backup.validation.backup_record_id_positive"));
    return;
  }
  submitting.value = true;
  try {
    await defBaseTableBackupRestoreService.ExecuteBaseTableBackupRestore({ base_table_backup_restore: { ...formData } });
    ElMessage.success(t("system.backup.message.restore_submitted"));
    proTable.value?.getTableList();
  } finally {
    submitting.value = false;
  }
}
/** 加载备份恢复可用的目标数据源。 */
async function loadSourceOptions() { const data = await defBaseTableSourceService.OptionBaseTableSource({}); sourceOptions.value = (data.value ?? []).map(value => ({ label: value, value })); if (!sourceOptions.value.some(item => item.value === formData.target_source_name)) formData.target_source_name = String(sourceOptions.value[0]?.value ?? ""); }
onMounted(() => void loadSourceOptions());
</script>
