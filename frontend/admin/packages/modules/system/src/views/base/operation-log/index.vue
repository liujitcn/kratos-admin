<template>
  <LogTable ref="page" :config="config" />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { t } from "@liujitcn/kratos-admin-core";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import LogTable, { type LogTableConfig } from "@liujitcn/kratos-admin-system/components/LogTable.vue";
import { logDateSearch, logDetailColumn, logEnumLabel, createLogEnumOptions, requestLogTrace } from "@liujitcn/kratos-admin-system/components/log";
import { defBaseOperationLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_operation_log";
import { BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import { BaseOperationAction } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_operation_log";
import type { BaseOperationLog, PageBaseOperationLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_operation_log";

defineOptions({ name: "BaseOperationLog", inheritAttrs: false });

const page = ref<InstanceType<typeof LogTable>>();
const resultOptions = computed(() => createLogEnumOptions([
  [BaseLogResult.BASE_LOG_RESULT_UNSPECIFIED, t("system.base.log.result.unspecified")],
  [BaseLogResult.BASE_LOG_RESULT_SUCCESS, t("system.base.log.result.success")],
  [BaseLogResult.BASE_LOG_RESULT_FAILURE, t("system.base.log.result.failure")],
  [BaseLogResult.BASE_LOG_RESULT_ERROR, t("system.base.log.result.error")]
]));
const actionOptions = computed(() => createLogEnumOptions([
  [BaseOperationAction.BASE_OPERATION_ACTION_UNSPECIFIED, t("system.base.log.operation_action.unspecified")],
  [BaseOperationAction.BASE_OPERATION_ACTION_CREATE, t("system.base.log.operation_action.create")],
  [BaseOperationAction.BASE_OPERATION_ACTION_UPDATE, t("system.base.log.operation_action.update")],
  [BaseOperationAction.BASE_OPERATION_ACTION_DELETE, t("system.base.log.operation_action.delete")],
  [BaseOperationAction.BASE_OPERATION_ACTION_PUBLISH, t("system.base.log.operation_action.publish")],
  [BaseOperationAction.BASE_OPERATION_ACTION_REVOKE, t("system.base.log.operation_action.revoke")],
  [BaseOperationAction.BASE_OPERATION_ACTION_IMPORT, t("system.base.log.operation_action.import")],
  [BaseOperationAction.BASE_OPERATION_ACTION_EXPORT, t("system.base.log.operation_action.export")],
  [BaseOperationAction.BASE_OPERATION_ACTION_OTHER, t("system.base.log.operation_action.other")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource_type", label: t("system.base.log.field.resource_type"), minWidth: 150, search: { el: "input" } },
  { prop: "resource_id", label: t("system.base.log.field.resource_id"), minWidth: 120 },
  { prop: "resource_name", label: t("system.base.log.field.resource_name"), minWidth: 160 },
  { prop: "action", label: t("system.base.log.field.action"), minWidth: 110, search: { el: "select", enum: actionOptions.value }, render: scope => logEnumLabel(actionOptions.value, (scope.row as BaseOperationLog).action) },
  { prop: "result", label: t("system.base.log.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => logEnumLabel(resultOptions.value, (scope.row as BaseOperationLog).result) },
  { prop: "user_name", label: t("system.base.log.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.log.field.occurred_at"), minWidth: 190, search: logDateSearch(t) },
  logDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<LogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.log.operation.title.detail"),
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
    { key: "resource_name", label: t("system.base.log.field.resource_name") },
    { key: "action", label: t("system.base.log.field.action"), enum: actionOptions.value },
    { key: "result", label: t("system.base.log.field.result"), enum: resultOptions.value },
    { key: "changed_fields", label: t("system.base.log.field.changed_fields"), span: 2, code: true },
    { key: "before_data", label: t("system.base.log.field.before_data"), span: 2, code: true },
    { key: "after_data", label: t("system.base.log.field.after_data"), span: 2, code: true },
    { key: "reason_code", label: t("system.base.log.field.reason_code") },
    { key: "reason", label: t("system.base.log.field.reason"), span: 2 },
    { key: "request_id", label: t("system.base.log.field.request_id") },
    { key: "trace_id", label: t("system.base.log.field.trace_id") },
    { key: "occurred_at", label: t("system.base.log.field.occurred_at") },
    { key: "created_at", label: t("system.base.log.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseOperationLogService.PageBaseOperationLog(buildPageRequest(params as unknown as PageBaseOperationLogRequest));
  return { list: response.base_operation_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: string) {
  return (await defBaseOperationLogService.GetBaseOperationLog({ id })) as unknown as Record<string, unknown>;
}
</script>
