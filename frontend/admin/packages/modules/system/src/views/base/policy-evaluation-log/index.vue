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
import { defBasePolicyEvaluationLogService } from "@liujitcn/kratos-admin-system/api/system/base_audit_log";
import { BasePolicyDecision, BasePolicyEvaluationType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";
import type { BasePolicyEvaluationLog, PageBasePolicyEvaluationLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

defineOptions({ name: "BasePolicyEvaluationLog", inheritAttrs: false });

const page = ref<InstanceType<typeof AuditLogTable>>();
const decisionOptions = computed(() => createAuditEnumOptions([
  [BasePolicyDecision.BASE_POLICY_DECISION_ALLOW, t("system.base.audit.policy_decision.allow")],
  [BasePolicyDecision.BASE_POLICY_DECISION_DENY, t("system.base.audit.policy_decision.deny")],
  [BasePolicyDecision.BASE_POLICY_DECISION_ERROR, t("system.base.audit.policy_decision.error")]
]));
const evaluationTypeOptions = computed(() => createAuditEnumOptions([
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_IS_AUTHORIZED, t("system.base.audit.evaluation_type.is_authorized")],
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_FILTER_PAIRS, t("system.base.audit.evaluation_type.filter_pairs")],
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_FILTER_PROJECTS, t("system.base.audit.evaluation_type.filter_projects")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource", label: t("system.base.audit.field.resource"), minWidth: 320, search: { el: "input" } },
  { prop: "action", label: t("system.base.audit.field.action"), minWidth: 100 },
  { prop: "engine", label: t("system.base.audit.field.engine"), minWidth: 100 },
  { prop: "evaluation_type", label: t("system.base.audit.field.evaluation_type"), minWidth: 160, search: { el: "select", enum: evaluationTypeOptions.value }, render: scope => auditEnumLabel(evaluationTypeOptions.value, (scope.row as BasePolicyEvaluationLog).evaluation_type) },
  { prop: "decision", label: t("system.base.audit.field.decision"), minWidth: 110, search: { el: "select", enum: decisionOptions.value }, render: scope => auditEnumLabel(decisionOptions.value, (scope.row as BasePolicyEvaluationLog).decision) },
  { prop: "duration_ms", label: t("system.base.audit.field.duration_ms"), minWidth: 110 },
  { prop: "occurred_at", label: t("system.base.audit.field.occurred_at"), minWidth: 190, search: auditDateSearch(t) },
  auditDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<AuditLogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.audit.policy.title.detail"),
  closeText: t("common.action.close"),
  detailFields: [
    { key: "id", label: t("system.base.audit.field.id") },
    { key: "tenant_id", label: t("system.base.audit.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.audit.field.tenant_code") },
    { key: "user_id", label: t("system.base.audit.field.user_id") },
    { key: "user_name", label: t("system.base.audit.field.user_name") },
    { key: "role_id", label: t("system.base.audit.field.role_id") },
    { key: "role_code", label: t("system.base.audit.field.role_code") },
    { key: "engine", label: t("system.base.audit.field.engine") },
    { key: "evaluation_type", label: t("system.base.audit.field.evaluation_type") },
    { key: "resource", label: t("system.base.audit.field.resource"), span: 2 },
    { key: "action", label: t("system.base.audit.field.action") },
    { key: "project", label: t("system.base.audit.field.project") },
    { key: "decision", label: t("system.base.audit.field.decision") },
    { key: "reason_code", label: t("system.base.audit.field.reason_code") },
    { key: "reason", label: t("system.base.audit.field.reason"), span: 2 },
    { key: "duration_ms", label: t("system.base.audit.field.duration_ms") },
    { key: "candidate_count", label: t("system.base.audit.field.candidate_count") },
    { key: "matched_count", label: t("system.base.audit.field.matched_count") },
    { key: "input_hash", label: t("system.base.audit.field.input_hash") },
    { key: "client_ip", label: t("system.base.audit.field.client_ip") },
    { key: "request_id", label: t("system.base.audit.field.request_id") },
    { key: "trace_id", label: t("system.base.audit.field.trace_id") },
    { key: "occurred_at", label: t("system.base.audit.field.occurred_at") },
    { key: "created_at", label: t("system.base.audit.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBasePolicyEvaluationLogService.PageBasePolicyEvaluationLog(buildPageRequest(params as unknown as PageBasePolicyEvaluationLogRequest));
  return { list: response.base_policy_evaluation_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: number) {
  return (await defBasePolicyEvaluationLogService.GetBasePolicyEvaluationLog({ id })) as unknown as Record<string, unknown>;
}
</script>
