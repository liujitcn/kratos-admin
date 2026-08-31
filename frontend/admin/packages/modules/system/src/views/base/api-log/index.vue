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
import { defBaseApiLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BaseAuditResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BaseApiLog, PageBaseApiLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BaseApiLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const resultOptions = computed(() =>
  createAuditEnumOptions([
    [BaseAuditResult.BASE_AUDIT_RESULT_SUCCESS, t("system.base.audit.result.success")],
    [BaseAuditResult.BASE_AUDIT_RESULT_FAILURE, t("system.base.audit.result.failure")],
    [BaseAuditResult.BASE_AUDIT_RESULT_ERROR, t("system.base.audit.result.error")]
  ])
);
const columns = computed<ColumnProps[]>(() => [
  { prop: "operation", label: t("system.base.audit.field.operation"), minWidth: 320, search: { el: "input" } },
  { prop: "method", label: t("system.base.audit.field.method"), minWidth: 90 },
  { prop: "status_code", label: t("system.base.audit.field.status_code"), minWidth: 100 },
  {
    prop: "result",
    label: t("system.base.audit.field.result"),
    minWidth: 110,
    search: { el: "select", enum: resultOptions.value },
    render: scope => auditEnumLabel(resultOptions.value, (scope.row as BaseApiLog).result)
  },
  { prop: "latency_ms", label: t("system.base.audit.field.latency_ms"), minWidth: 110 },
  { prop: "user_name", label: t("system.base.audit.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.api.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "service_name", label: t("system.base.audit.field.service_name") },
    { key: "operation", label: t("system.base.audit.field.operation"), span: 2 },
    { key: "method", label: t("system.base.audit.field.method") },
    { key: "path", label: t("system.base.audit.field.path"), span: 2 },
    { key: "status_code", label: t("system.base.audit.field.status_code") },
    { key: "result", label: t("system.base.audit.field.result") },
    { key: "reason_code", label: t("system.base.audit.field.reason_code") },
    { key: "reason", label: t("system.base.audit.field.reason"), span: 2 },
    { key: "latency_ms", label: t("system.base.audit.field.latency_ms") },
    { key: "request_size", label: t("system.base.audit.field.request_size") },
    { key: "response_size", label: t("system.base.audit.field.response_size") },
    { key: "client_ip", label: t("system.base.audit.field.client_ip") },
    { key: "user_agent", label: t("system.base.audit.field.user_agent"), span: 2 },
    { key: "request_id", label: t("system.base.audit.field.request_id") },
    { key: "trace_id", label: t("system.base.audit.field.trace_id") },
    { key: "occurred_at", label: t("system.base.audit.field.occurred_at") },
    { key: "created_at", label: t("system.base.audit.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseApiLogService.PageBaseApiLog(buildPageRequest(params as unknown as PageBaseApiLogRequest));
  return { list: response.base_api_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBaseApiLogService.GetBaseApiLog({ id })) as unknown as Record<string, unknown>;
}
</script>
