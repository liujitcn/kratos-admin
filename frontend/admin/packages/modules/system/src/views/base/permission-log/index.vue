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
import { defBasePermissionLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BaseAuditResult, BasePermissionAction, BasePermissionTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BasePermissionLog, PageBasePermissionLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BasePermissionLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const resultOptions = computed(() => createAuditEnumOptions([
  [BaseAuditResult.BASE_AUDIT_RESULT_SUCCESS, t("system.base.audit.result.success")],
  [BaseAuditResult.BASE_AUDIT_RESULT_FAILURE, t("system.base.audit.result.failure")],
  [BaseAuditResult.BASE_AUDIT_RESULT_ERROR, t("system.base.audit.result.error")]
]));
const targetOptions = computed(() => createAuditEnumOptions([
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_USER, t("system.base.audit.target_type.user")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_ROLE, t("system.base.audit.target_type.role")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_MENU, t("system.base.audit.target_type.menu")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_API, t("system.base.audit.target_type.api")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_TENANT, t("system.base.audit.target_type.tenant")]
]));
const actionOptions = computed(() => createAuditEnumOptions([
  [BasePermissionAction.BASE_PERMISSION_ACTION_GRANT, t("system.base.audit.permission_action.grant")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_REVOKE, t("system.base.audit.permission_action.revoke")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_CREATE, t("system.base.audit.permission_action.create")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_UPDATE, t("system.base.audit.permission_action.update")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_DELETE, t("system.base.audit.permission_action.delete")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_ASSIGN, t("system.base.audit.permission_action.assign")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "target_type", label: t("system.base.audit.field.target_type"), minWidth: 120, search: { el: "select", enum: targetOptions.value }, render: scope => auditEnumLabel(targetOptions.value, (scope.row as BasePermissionLog).target_type) },
  { prop: "target_name", label: t("system.base.audit.field.target_name"), minWidth: 180, search: { el: "input" } },
  { prop: "action", label: t("system.base.audit.field.action"), minWidth: 110, search: { el: "select", enum: actionOptions.value }, render: scope => auditEnumLabel(actionOptions.value, (scope.row as BasePermissionLog).action) },
  { prop: "result", label: t("system.base.audit.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => auditEnumLabel(resultOptions.value, (scope.row as BasePermissionLog).result) },
  { prop: "user_name", label: t("system.base.audit.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.permission.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "target_type", label: t("system.base.audit.field.target_type") },
    { key: "target_id", label: t("system.base.audit.field.target_id") },
    { key: "target_name", label: t("system.base.audit.field.target_name") },
    { key: "action", label: t("system.base.audit.field.action") },
    { key: "old_value", label: t("system.base.audit.field.old_value"), span: 2, code: true },
    { key: "new_value", label: t("system.base.audit.field.new_value"), span: 2, code: true },
    { key: "result", label: t("system.base.audit.field.result") },
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
  const response = await defBasePermissionLogService.PageBasePermissionLog(buildPageRequest(params as unknown as PageBasePermissionLogRequest));
  return { list: response.base_permission_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBasePermissionLogService.GetBasePermissionLog({ id })) as unknown as Record<string, unknown>;
}
</script>
