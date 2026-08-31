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
import { defBaseOperationLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BaseAuditResult, BaseOperationAction } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BaseOperationLog, PageBaseOperationLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BaseOperationLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const resultOptions = computed(() => createAuditEnumOptions([
  [BaseAuditResult.BASE_AUDIT_RESULT_SUCCESS, t("system.base.audit.result.success")],
  [BaseAuditResult.BASE_AUDIT_RESULT_FAILURE, t("system.base.audit.result.failure")],
  [BaseAuditResult.BASE_AUDIT_RESULT_ERROR, t("system.base.audit.result.error")]
]));
const actionOptions = computed(() => createAuditEnumOptions([
  [BaseOperationAction.BASE_OPERATION_ACTION_CREATE, t("system.base.audit.operation_action.create")],
  [BaseOperationAction.BASE_OPERATION_ACTION_UPDATE, t("system.base.audit.operation_action.update")],
  [BaseOperationAction.BASE_OPERATION_ACTION_DELETE, t("system.base.audit.operation_action.delete")],
  [BaseOperationAction.BASE_OPERATION_ACTION_PUBLISH, t("system.base.audit.operation_action.publish")],
  [BaseOperationAction.BASE_OPERATION_ACTION_REVOKE, t("system.base.audit.operation_action.revoke")],
  [BaseOperationAction.BASE_OPERATION_ACTION_IMPORT, t("system.base.audit.operation_action.import")],
  [BaseOperationAction.BASE_OPERATION_ACTION_EXPORT, t("system.base.audit.operation_action.export")],
  [BaseOperationAction.BASE_OPERATION_ACTION_OTHER, t("system.base.audit.operation_action.other")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource_type", label: t("system.base.audit.field.resource_type"), minWidth: 150, search: { el: "input" } },
  { prop: "resource_id", label: t("system.base.audit.field.resource_id"), minWidth: 120 },
  { prop: "resource_name", label: t("system.base.audit.field.resource_name"), minWidth: 160 },
  { prop: "action", label: t("system.base.audit.field.action"), minWidth: 110, search: { el: "select", enum: actionOptions.value }, render: scope => auditEnumLabel(actionOptions.value, (scope.row as BaseOperationLog).action) },
  { prop: "result", label: t("system.base.audit.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => auditEnumLabel(resultOptions.value, (scope.row as BaseOperationLog).result) },
  { prop: "user_name", label: t("system.base.audit.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.operation.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "resource_type", label: t("system.base.audit.field.resource_type") },
    { key: "resource_id", label: t("system.base.audit.field.resource_id") },
    { key: "resource_name", label: t("system.base.audit.field.resource_name") },
    { key: "action", label: t("system.base.audit.field.action") },
    { key: "result", label: t("system.base.audit.field.result") },
    { key: "changed_fields", label: t("system.base.audit.field.changed_fields"), span: 2, code: true },
    { key: "before_data", label: t("system.base.audit.field.before_data"), span: 2, code: true },
    { key: "after_data", label: t("system.base.audit.field.after_data"), span: 2, code: true },
    { key: "reason_code", label: t("system.base.audit.field.reason_code") },
    { key: "reason", label: t("system.base.audit.field.reason"), span: 2 },
    { key: "request_id", label: t("system.base.audit.field.request_id") },
    { key: "trace_id", label: t("system.base.audit.field.trace_id") },
    { key: "occurred_at", label: t("system.base.audit.field.occurred_at") },
    { key: "created_at", label: t("system.base.audit.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseOperationLogService.PageBaseOperationLog(buildPageRequest(params as unknown as PageBaseOperationLogRequest));
  return { list: response.base_operation_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBaseOperationLogService.GetBaseOperationLog({ id })) as unknown as Record<string, unknown>;
}
</script>
