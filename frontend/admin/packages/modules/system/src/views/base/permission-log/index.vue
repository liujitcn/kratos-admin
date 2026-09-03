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
import { defBasePermissionLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_permission_log";
import { BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import { BasePermissionAction, BasePermissionTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_permission_log";
import type { BasePermissionLog, PageBasePermissionLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_permission_log";

defineOptions({ name: "BasePermissionLog", inheritAttrs: false });

const page = ref<InstanceType<typeof LogTable>>();
const resultOptions = computed(() => createLogEnumOptions([
  [BaseLogResult.BASE_LOG_RESULT_UNSPECIFIED, t("system.base.log.result.unspecified")],
  [BaseLogResult.BASE_LOG_RESULT_SUCCESS, t("system.base.log.result.success")],
  [BaseLogResult.BASE_LOG_RESULT_FAILURE, t("system.base.log.result.failure")],
  [BaseLogResult.BASE_LOG_RESULT_ERROR, t("system.base.log.result.error")]
]));
const targetOptions = computed(() => createLogEnumOptions([
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_UNSPECIFIED, t("system.base.log.target_type.unspecified")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_USER, t("common.field.user")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_ROLE, t("common.field.role")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_MENU, t("system.base.log.target_type.menu")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_API, t("system.base.log.target_type.api")],
  [BasePermissionTargetType.BASE_PERMISSION_TARGET_TYPE_TENANT, t("common.field.tenant")]
]));
const actionOptions = computed(() => createLogEnumOptions([
  [BasePermissionAction.BASE_PERMISSION_ACTION_UNSPECIFIED, t("system.base.log.permission_action.unspecified")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_GRANT, t("system.base.log.permission_action.grant")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_REVOKE, t("system.base.log.permission_action.revoke")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_CREATE, t("system.base.log.permission_action.create")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_UPDATE, t("system.base.log.permission_action.update")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_DELETE, t("system.base.log.permission_action.delete")],
  [BasePermissionAction.BASE_PERMISSION_ACTION_ASSIGN, t("system.base.log.permission_action.assign")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "target_type", label: t("system.base.log.field.target_type"), minWidth: 120, search: { el: "select", enum: targetOptions.value }, render: scope => logEnumLabel(targetOptions.value, (scope.row as BasePermissionLog).target_type) },
  { prop: "target_name", label: t("system.base.log.field.target_name"), minWidth: 180, search: { el: "input" } },
  { prop: "action", label: t("system.base.log.field.action"), minWidth: 110, search: { el: "select", enum: actionOptions.value }, render: scope => logEnumLabel(actionOptions.value, (scope.row as BasePermissionLog).action) },
  { prop: "result", label: t("system.base.log.field.result"), minWidth: 110, search: { el: "select", enum: resultOptions.value }, render: scope => logEnumLabel(resultOptions.value, (scope.row as BasePermissionLog).result) },
  { prop: "user_name", label: t("system.base.log.field.user_name"), minWidth: 130 },
  { prop: "occurred_at", label: t("system.base.log.field.occurred_at"), minWidth: 190, search: logDateSearch(t) },
  logDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<LogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.log.permission.title.detail"),
  closeText: t("common.action.close"),
  trace: requestLogTrace,
  detailFields: [
    { key: "id", label: t("system.base.log.field.id") },
    { key: "tenant_id", label: t("system.base.log.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.log.field.tenant_code") },
    { key: "user_id", label: t("system.base.log.field.user_id") },
    { key: "user_name", label: t("system.base.log.field.user_name") },
    { key: "target_type", label: t("system.base.log.field.target_type"), enum: targetOptions.value },
    { key: "target_id", label: t("system.base.log.field.target_id") },
    { key: "target_name", label: t("system.base.log.field.target_name") },
    { key: "action", label: t("system.base.log.field.action"), enum: actionOptions.value },
    { key: "old_value", label: t("system.base.log.field.old_value"), span: 2, code: true },
    { key: "new_value", label: t("system.base.log.field.new_value"), span: 2, code: true },
    { key: "result", label: t("system.base.log.field.result"), enum: resultOptions.value },
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
  const response = await defBasePermissionLogService.PageBasePermissionLog(buildPageRequest(params as unknown as PageBasePermissionLogRequest));
  return { list: response.base_permission_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: string) {
  return (await defBasePermissionLogService.GetBasePermissionLog({ id })) as unknown as Record<string, unknown>;
}
</script>
