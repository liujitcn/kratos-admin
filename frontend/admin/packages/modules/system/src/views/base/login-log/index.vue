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
import { defBaseLoginLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BaseAuditResult, BaseLoginLogType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BaseLoginLog, PageBaseLoginLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BaseLoginLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const resultOptions = computed(() =>
  createAuditEnumOptions([
    [BaseAuditResult.BASE_AUDIT_RESULT_SUCCESS, t("system.base.audit.result.success")],
    [BaseAuditResult.BASE_AUDIT_RESULT_FAILURE, t("system.base.audit.result.failure")],
    [BaseAuditResult.BASE_AUDIT_RESULT_ERROR, t("system.base.audit.result.error")]
  ])
);
const loginTypeOptions = computed(() =>
  createAuditEnumOptions([
    [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_PASSWORD, t("system.base.audit.login_type.password")],
    [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_OAUTH, t("system.base.audit.login_type.oauth")],
    [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_MFA, t("system.base.audit.login_type.mfa")],
    [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_TOKEN_REFRESH, t("system.base.audit.login_type.token_refresh")],
    [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_LOGOUT, t("system.base.audit.login_type.logout")]
  ])
);

const columns = computed<ColumnProps[]>(() => [
  { prop: "user_name", label: t("system.base.audit.field.user_name"), minWidth: 130 },
  { prop: "tenant_code", label: t("system.base.audit.field.tenant_code"), minWidth: 120 },
  {
    prop: "login_type",
    label: t("system.base.audit.field.login_type"),
    minWidth: 130,
    search: { el: "select", enum: loginTypeOptions.value },
    render: scope => auditEnumLabel(loginTypeOptions.value, (scope.row as BaseLoginLog).login_type)
  },
  {
    prop: "result",
    label: t("system.base.audit.field.result"),
    minWidth: 110,
    search: { el: "select", enum: resultOptions.value },
    render: scope => auditEnumLabel(resultOptions.value, (scope.row as BaseLoginLog).result)
  },
  { prop: "client_ip", label: t("system.base.audit.field.client_ip"), minWidth: 140 },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.login.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "login_type", label: t("system.base.audit.field.login_type") },
    { key: "result", label: t("system.base.audit.field.result") },
    { key: "reason_code", label: t("system.base.audit.field.reason_code") },
    { key: "reason", label: t("system.base.audit.field.reason"), span: 2 },
    { key: "client_ip", label: t("system.base.audit.field.client_ip") },
    { key: "device_id", label: t("system.base.audit.field.device_id") },
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
  const response = await defBaseLoginLogService.PageBaseLoginLog(buildPageRequest(params as unknown as PageBaseLoginLogRequest));
  return { list: response.base_login_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBaseLoginLogService.GetBaseLoginLog({ id })) as unknown as Record<string, unknown>;
}
</script>
