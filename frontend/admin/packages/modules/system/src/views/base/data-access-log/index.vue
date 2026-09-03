<template>
  <LogTable ref="page" :config="config" />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { t } from "@liujitcn/kratos-admin-core";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import LogTable, { type LogTableConfig } from "@liujitcn/kratos-admin-system/components/LogTable.vue";
import {
  createLogEnumOptions,
  formatLogCount,
  formatLogDuration,
  logDateSearch,
  logDetailColumn,
  logEnumLabel,
  requestLogTrace
} from "@liujitcn/kratos-admin-system/components/log";
import { defBaseDataAccessLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_data_access_log";
import { BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import { BaseDataAccessType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_data_access_log";
import type { BaseDataAccessLog, PageBaseDataAccessLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_data_access_log";

defineOptions({ name: "BaseDataAccessLog", inheritAttrs: false });

const page = ref<InstanceType<typeof LogTable>>();
const resultOptions = computed(() => createLogEnumOptions([
  [BaseLogResult.BASE_LOG_RESULT_UNSPECIFIED, t("system.base.log.result.unspecified")],
  [BaseLogResult.BASE_LOG_RESULT_SUCCESS, t("system.base.log.result.success")],
  [BaseLogResult.BASE_LOG_RESULT_FAILURE, t("system.base.log.result.failure")],
  [BaseLogResult.BASE_LOG_RESULT_ERROR, t("system.base.log.result.error")]
]));
const accessTypeOptions = computed(() => createLogEnumOptions([
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_UNSPECIFIED, t("system.base.log.access_type.unspecified")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_LIST, t("system.base.log.access_type.list")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_DETAIL, t("system.base.log.access_type.detail")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_QUERY, t("system.base.log.access_type.query")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_EXPORT, t("system.base.log.access_type.export")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_DOWNLOAD, t("system.base.log.access_type.download")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_IMPORT, t("system.base.log.access_type.import")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource_type", label: t("system.base.log.field.resource_type"), minWidth: 150, search: { el: "input" } },
  { prop: "table_name", label: t("system.base.log.field.table_name"), minWidth: 150 },
  { prop: "access_type", label: t("system.base.log.field.access_type"), minWidth: 120, search: { el: "select", enum: accessTypeOptions.value }, render: scope => logEnumLabel(accessTypeOptions.value, (scope.row as BaseDataAccessLog).access_type) },
  { prop: "affected_rows", label: t("system.base.log.field.affected_rows"), minWidth: 110 },
  { prop: "sensitive", label: t("system.base.log.field.sensitive"), minWidth: 110, search: { el: "switch" }, render: scope => (scope.row as BaseDataAccessLog).sensitive ? t("common.value.yes") : t("common.value.no") },
  { prop: "result", label: t("system.base.log.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => logEnumLabel(resultOptions.value, (scope.row as BaseDataAccessLog).result) },
  { prop: "occurred_at", label: t("system.base.log.field.occurred_at"), minWidth: 190, search: logDateSearch(t) },
  logDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<LogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.log.data_access.title.detail"),
  closeText: t("common.action.close"),
  trace: requestLogTrace,
  detailFields: [
    { key: "id", label: t("system.base.log.field.id") },
    { key: "tenant_id", label: t("system.base.log.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.log.field.tenant_code") },
    { key: "user_id", label: t("system.base.log.field.user_id") },
    { key: "user_name", label: t("system.base.log.field.user_name") },
    { key: "resource_type", label: t("system.base.log.field.resource_type") },
    { key: "resource_id", label: t("system.base.log.field.resource_id") },
    { key: "access_type", label: t("system.base.log.field.access_type"), enum: accessTypeOptions.value },
    { key: "data_source", label: t("system.base.log.field.data_source") },
    { key: "table_name", label: t("system.base.log.field.table_name") },
    { key: "field_scope", label: t("system.base.log.field.field_scope"), span: 2, code: true },
    { key: "data_scope", label: t("system.base.log.field.data_scope") },
    { key: "affected_rows", label: t("system.base.log.field.affected_rows"), format: value => formatLogCount(value, t) },
    { key: "sensitive", label: t("system.base.log.field.sensitive"), format: value => (value ? t("common.value.yes") : t("common.value.no")) },
    { key: "sql_digest", label: t("system.base.log.field.sql_digest") },
    { key: "sql_text", label: t("system.base.log.field.sql_text"), span: 2, code: true },
    { key: "result", label: t("system.base.log.field.result"), enum: resultOptions.value },
    { key: "latency_ms", label: t("system.base.log.field.latency_ms"), format: value => formatLogDuration(value, t) },
    { key: "reason_code", label: t("system.base.log.field.reason_code") },
    { key: "request_id", label: t("system.base.log.field.request_id") },
    { key: "trace_id", label: t("system.base.log.field.trace_id") },
    { key: "occurred_at", label: t("system.base.log.field.occurred_at") },
    { key: "created_at", label: t("system.base.log.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseDataAccessLogService.PageBaseDataAccessLog(buildPageRequest(params as unknown as PageBaseDataAccessLogRequest));
  return { list: response.base_data_access_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: string) {
  return (await defBaseDataAccessLogService.GetBaseDataAccessLog({ id })) as unknown as Record<string, unknown>;
}
</script>
