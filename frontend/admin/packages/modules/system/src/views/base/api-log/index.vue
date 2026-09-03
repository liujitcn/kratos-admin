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
  formatLogBytes,
  formatLogDuration,
  formatLogStatusCode,
  logDateSearch,
  logDetailColumn,
  logEnumLabel,
  requestLogTrace
} from "@liujitcn/kratos-admin-system/components/log";
import { defBaseApiLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api_log";
import { BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import type { BaseApiLog, PageBaseApiLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_log";

defineOptions({ name: "BaseApiLog", inheritAttrs: false });

const page = ref<InstanceType<typeof LogTable>>();
const resultOptions = computed(() =>
  createLogEnumOptions([
    [BaseLogResult.BASE_LOG_RESULT_UNSPECIFIED, t("system.base.log.result.unspecified")],
    [BaseLogResult.BASE_LOG_RESULT_SUCCESS, t("system.base.log.result.success")],
    [BaseLogResult.BASE_LOG_RESULT_FAILURE, t("system.base.log.result.failure")],
    [BaseLogResult.BASE_LOG_RESULT_ERROR, t("system.base.log.result.error")]
  ])
);
const columns = computed<ColumnProps[]>(() => [
  { prop: "operation", label: t("system.base.log.field.operation"), minWidth: 320, search: { el: "input" } },
  { prop: "method", label: t("system.base.log.field.method"), minWidth: 90 },
  { prop: "status_code", label: t("system.base.log.field.status_code"), minWidth: 100 },
  {
    prop: "result",
    label: t("system.base.log.field.result"),
    minWidth: 110,
    search: { el: "select", enum: resultOptions.value },
    render: scope => logEnumLabel(resultOptions.value, (scope.row as BaseApiLog).result)
  },
  { prop: "latency_ms", label: t("system.base.log.field.latency_ms"), minWidth: 110 },
  { prop: "user_name", label: t("system.base.log.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.log.field.occurred_at"), minWidth: 190, search: logDateSearch(t) },
  logDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<LogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.log.api.title.detail"),
  closeText: t("common.action.close"),
  trace: requestLogTrace,
  detailFields: [
    { key: "id", label: t("system.base.log.field.id") },
    { key: "tenant_id", label: t("system.base.log.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.log.field.tenant_code") },
    { key: "user_id", label: t("system.base.log.field.user_id") },
    { key: "user_name", label: t("system.base.log.field.user_name") },
    { key: "service_name", label: t("system.base.log.field.service_name") },
    { key: "operation", label: t("system.base.log.field.operation"), span: 2 },
    { key: "method", label: t("system.base.log.field.method") },
    { key: "path", label: t("system.base.log.field.path"), span: 2 },
    { key: "status_code", label: t("system.base.log.field.status_code"), format: value => formatLogStatusCode(value, t) },
    { key: "result", label: t("system.base.log.field.result"), enum: resultOptions.value },
    { key: "reason_code", label: t("system.base.log.field.reason_code") },
    { key: "reason", label: t("system.base.log.field.reason"), span: 2 },
    { key: "latency_ms", label: t("system.base.log.field.latency_ms"), format: value => formatLogDuration(value, t) },
    { key: "request_size", label: t("system.base.log.field.request_size"), format: value => formatLogBytes(value, t) },
    { key: "response_size", label: t("system.base.log.field.response_size"), format: value => formatLogBytes(value, t) },
    { key: "client_ip", label: t("system.base.log.field.client_ip") },
    { key: "user_agent", label: t("system.base.log.field.user_agent"), span: 2 },
    { key: "request_id", label: t("system.base.log.field.request_id") },
    { key: "trace_id", label: t("system.base.log.field.trace_id") },
    { key: "occurred_at", label: t("system.base.log.field.occurred_at") },
    { key: "created_at", label: t("system.base.log.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseApiLogService.PageBaseApiLog(buildPageRequest(params as unknown as PageBaseApiLogRequest));
  return { list: response.base_api_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: string) {
  return (await defBaseApiLogService.GetBaseApiLog({ id })) as unknown as Record<string, unknown>;
}
</script>
