<template>
  <AuditLogTable ref="page" :config="config" />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { t } from "@liujitcn/kratos-admin-core";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import AuditLogTable, { type AuditLogTableConfig } from "@liujitcn/kratos-admin-system/components/AuditLogTable.vue";
import { auditDateSearch, auditDetailColumn, auditEnumLabel, createAuditEnumOptions } from "@liujitcn/kratos-admin-system/components/auditLog";
import { defBaseDataAccessLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BaseAuditResult, BaseDataAccessType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BaseDataAccessLog, PageBaseDataAccessLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BaseDataAccessLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const resultOptions = computed(() => createAuditEnumOptions([
  [BaseAuditResult.BASE_AUDIT_RESULT_SUCCESS, t("system.base.audit.result.success")],
  [BaseAuditResult.BASE_AUDIT_RESULT_FAILURE, t("system.base.audit.result.failure")],
  [BaseAuditResult.BASE_AUDIT_RESULT_ERROR, t("system.base.audit.result.error")]
]));
const accessTypeOptions = computed(() => createAuditEnumOptions([
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_LIST, t("system.base.audit.access_type.list")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_DETAIL, t("system.base.audit.access_type.detail")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_QUERY, t("system.base.audit.access_type.query")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_EXPORT, t("system.base.audit.access_type.export")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_DOWNLOAD, t("system.base.audit.access_type.download")],
  [BaseDataAccessType.BASE_DATA_ACCESS_TYPE_IMPORT, t("system.base.audit.access_type.import")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource_type", label: t("system.base.audit.field.resource_type"), minWidth: 150, search: { el: "input" } },
  { prop: "table_name", label: t("system.base.audit.field.table_name"), minWidth: 150 },
  { prop: "access_type", label: t("system.base.audit.field.access_type"), minWidth: 120, search: { el: "select", enum: accessTypeOptions.value }, render: scope => auditEnumLabel(accessTypeOptions.value, (scope.row as BaseDataAccessLog).access_type) },
  { prop: "affected_rows", label: t("system.base.audit.field.affected_rows"), minWidth: 110 },
  { prop: "sensitive", label: t("system.base.audit.field.sensitive"), minWidth: 110, search: { el: "switch" }, render: scope => (scope.row as BaseDataAccessLog).sensitive ? t("common.value.yes") : t("common.value.no") },
  { prop: "result", label: t("system.base.audit.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => auditEnumLabel(resultOptions.value, (scope.row as BaseDataAccessLog).result) },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.data_access.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "resource_type", label: t("system.base.audit.field.resource_type") },
    { key: "resource_id", label: t("system.base.audit.field.resource_id") },
    { key: "access_type", label: t("system.base.audit.field.access_type") },
    { key: "data_source", label: t("system.base.audit.field.data_source") },
    { key: "table_name", label: t("system.base.audit.field.table_name") },
    { key: "field_scope", label: t("system.base.audit.field.field_scope"), span: 2, code: true },
    { key: "data_scope", label: t("system.base.audit.field.data_scope") },
    { key: "affected_rows", label: t("system.base.audit.field.affected_rows") },
    { key: "sensitive", label: t("system.base.audit.field.sensitive") },
    { key: "sql_digest", label: t("system.base.audit.field.sql_digest") },
    { key: "sql_text", label: t("system.base.audit.field.sql_text"), span: 2, code: true },
    { key: "result", label: t("system.base.audit.field.result") },
    { key: "reason_code", label: t("system.base.audit.field.reason_code") },
    { key: "latency_ms", label: t("system.base.audit.field.latency_ms") },
    { key: "request_id", label: t("system.base.audit.field.request_id") },
    { key: "trace_id", label: t("system.base.audit.field.trace_id") },
    { key: "occurred_at", label: t("system.base.audit.field.occurred_at") },
    { key: "created_at", label: t("system.base.audit.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseDataAccessLogService.PageBaseDataAccessLog(buildPageRequest(params as unknown as PageBaseDataAccessLogRequest));
  return { list: response.base_data_access_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBaseDataAccessLogService.GetBaseDataAccessLog({ id })) as unknown as Record<string, unknown>;
}
</script>
