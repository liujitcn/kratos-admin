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
import { defBaseTableArchiveRecordService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_archive_record";
import { BaseTableArchiveMode } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive";
import { BaseTableArchiveRecordStatus, type PageBaseTableArchiveRecordRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_record";

defineOptions({ name: "BaseTableArchiveRecord", inheritAttrs: false });
const proTable = ref<ProTableInstance>();
const statusOptions = computed(() => [
  { label: t("system.backup.status.running"), value: BaseTableArchiveRecordStatus.BASE_TABLE_ARCHIVE_RECORD_STATUS_RUNNING },
  { label: t("system.backup.status.success"), value: BaseTableArchiveRecordStatus.BASE_TABLE_ARCHIVE_RECORD_STATUS_SUCCESS },
  { label: t("system.backup.status.failed"), value: BaseTableArchiveRecordStatus.BASE_TABLE_ARCHIVE_RECORD_STATUS_FAILED },
  { label: t("system.backup.status.deleted"), value: BaseTableArchiveRecordStatus.BASE_TABLE_ARCHIVE_RECORD_STATUS_DELETED }
]);
const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.backup.field.id"), width: 90 },
  { prop: "source_name", label: t("system.backup.field.source_name"), minWidth: 140 },
  { prop: "table_name", label: t("system.backup.field.table_name"), minWidth: 180 },
  { prop: "archive_mode", label: t("system.backup.field.archive_mode"), width: 130, render: scope => scope.row.archive_mode === BaseTableArchiveMode.BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE ? t("system.backup.archive.mode.internal") : t("system.backup.archive.mode.oss") },
  { prop: "cutoff_at", label: t("system.backup.field.cutoff_at"), minWidth: 170 },
  { prop: "archived_rows", label: t("system.backup.field.archived_rows"), width: 110 },
  { prop: "deleted_rows", label: t("system.backup.field.deleted_rows"), width: 110 },
  { prop: "object_key", label: t("system.backup.field.object_key"), minWidth: 220 },
  { prop: "status", label: t("common.field.status"), width: 110, search: { el: "select", enum: statusOptions.value }, render: scope => statusOptions.value.find(item => item.value === scope.row.status)?.label ?? String(scope.row.status) },
  { prop: "started_at", label: t("system.backup.field.started_at"), minWidth: 170 },
  { prop: "finished_at", label: t("system.backup.field.finished_at"), minWidth: 170 }
]);

/** 请求归档记录分页列表。 */
async function requestTable(params: PageBaseTableArchiveRecordRequest) {
  const data = await defBaseTableArchiveRecordService.PageBaseTableArchiveRecord(buildPageRequest(params));
  return { data: { list: data.base_table_archive_records ?? [], total: data.total } };
}
</script>
