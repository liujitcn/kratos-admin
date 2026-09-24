<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestTable" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { formatDateTime } from "@liujitcn/kratos-admin-core/format";
import { resolvePersistedMessage } from "../../../../utils/persisted-message";
import { defBaseTableBackupRecordService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_backup_record";
import { BaseTableBackupRecordStatus, type PageBaseTableBackupRecordRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_record";

defineOptions({ name: "BaseTableBackupRecord", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const statusOptions = computed(() => [
  { label: t("system.backup.status.running"), value: BaseTableBackupRecordStatus.BASE_TABLE_BACKUP_RECORD_STATUS_RUNNING },
  { label: t("system.backup.status.success"), value: BaseTableBackupRecordStatus.BASE_TABLE_BACKUP_RECORD_STATUS_SUCCESS },
  { label: t("system.backup.status.failed"), value: BaseTableBackupRecordStatus.BASE_TABLE_BACKUP_RECORD_STATUS_FAILED },
  { label: t("system.backup.status.deleted"), value: BaseTableBackupRecordStatus.BASE_TABLE_BACKUP_RECORD_STATUS_DELETED }
]);
const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.backup.field.id"), width: 90, align: "right" },
  { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 140 },
  { prop: "database_name", label: t("system.backup.field.database_name"), minWidth: 160 },
  { prop: "object_key", label: t("system.backup.field.object_key"), minWidth: 260 },
  { prop: "size_bytes", label: t("system.backup.field.size_bytes"), width: 110 },
  { prop: "status", label: t("common.field.status"), width: 110, search: { el: "select", enum: statusOptions.value }, render: scope => statusOptions.value.find(item => item.value === scope.row.status)?.label ?? String(scope.row.status) },
  { prop: "error", label: t("system.backup.field.error"), minWidth: 220, render: scope => resolvePersistedMessage(scope.row.error) },
  { prop: "started_at", align: "center", label: t("system.backup.field.started_at"), minWidth: 170, render: scope => formatDateTime(scope.row.started_at) },
  { prop: "finished_at", align: "center", label: t("system.backup.field.finished_at"), minWidth: 170, render: scope => formatDateTime(scope.row.finished_at) },
  { prop: "verified_at", label: t("system.backup.field.verified_at"), minWidth: 170 }
]);

/** 请求备份记录分页列表。 */
async function requestTable(params: PageBaseTableBackupRecordRequest) {
  const data = await defBaseTableBackupRecordService.PageBaseTableBackupRecord(buildPageRequest(params));
  return { data: { list: data.base_table_backup_records ?? [], total: data.total } };
}
</script>
