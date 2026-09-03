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
import { defBasePolicyEvaluationLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_policy_evaluation_log";
import { BasePolicyDecision, BasePolicyEvaluationType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_policy_evaluation_log";
import type { BasePolicyEvaluationLog, PageBasePolicyEvaluationLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_policy_evaluation_log";

defineOptions({ name: "BasePolicyEvaluationLog", inheritAttrs: false });

const page = ref<InstanceType<typeof LogTable>>();
const decisionOptions = computed(() => createLogEnumOptions([
  [BasePolicyDecision.BASE_POLICY_DECISION_UNSPECIFIED, t("system.base.log.policy_decision.unspecified")],
  [BasePolicyDecision.BASE_POLICY_DECISION_ALLOW, t("system.base.log.policy_decision.allow")],
  [BasePolicyDecision.BASE_POLICY_DECISION_DENY, t("system.base.log.policy_decision.deny")],
  [BasePolicyDecision.BASE_POLICY_DECISION_ERROR, t("system.base.log.policy_decision.error")]
]));
const evaluationTypeOptions = computed(() => createLogEnumOptions([
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_UNSPECIFIED, t("system.base.log.evaluation_type.unspecified")],
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_IS_AUTHORIZED, t("system.base.log.evaluation_type.is_authorized")],
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_FILTER_PAIRS, t("system.base.log.evaluation_type.filter_pairs")],
  [BasePolicyEvaluationType.BASE_POLICY_EVALUATION_TYPE_FILTER_PROJECTS, t("system.base.log.evaluation_type.filter_projects")]
]));
const columns = computed<ColumnProps[]>(() => [
  { prop: "resource", label: t("system.base.log.field.resource"), minWidth: 320, search: { el: "input" } },
  { prop: "action", label: t("system.base.log.field.action"), minWidth: 100 },
  { prop: "engine", label: t("system.base.log.field.engine"), minWidth: 100 },
  { prop: "evaluation_type", label: t("system.base.log.field.evaluation_type"), minWidth: 160, search: { el: "select", enum: evaluationTypeOptions.value }, render: scope => logEnumLabel(evaluationTypeOptions.value, (scope.row as BasePolicyEvaluationLog).evaluation_type) },
  { prop: "decision", label: t("system.base.log.field.decision"), minWidth: 110, search: { el: "select", enum: decisionOptions.value }, render: scope => logEnumLabel(decisionOptions.value, (scope.row as BasePolicyEvaluationLog).decision) },
  { prop: "duration_ms", label: t("system.base.log.field.duration_ms"), minWidth: 110 },
  { prop: "occurred_at", label: t("system.base.log.field.occurred_at"), minWidth: 190, search: logDateSearch(t) },
  logDetailColumn(t("common.action.view"), id => page.value?.handleOpenDialog(id))
]);

const config = computed<LogTableConfig>(() => ({
  columns: columns.value,
  detailTitle: t("system.base.log.policy.title.detail"),
  closeText: t("common.action.close"),
  trace: requestLogTrace,
  detailFields: [
    { key: "id", label: t("system.base.log.field.id") },
    { key: "tenant_id", label: t("system.base.log.field.tenant_id") },
    { key: "tenant_code", label: t("system.base.log.field.tenant_code") },
    { key: "user_id", label: t("system.base.log.field.user_id") },
    { key: "user_name", label: t("system.base.log.field.user_name") },
    { key: "role_id", label: t("system.base.log.field.role_id") },
    { key: "role_code", label: t("system.base.log.field.role_code") },
    { key: "engine", label: t("system.base.log.field.engine") },
    { key: "evaluation_type", label: t("system.base.log.field.evaluation_type"), enum: evaluationTypeOptions.value },
    { key: "resource", label: t("system.base.log.field.resource"), span: 2 },
    { key: "action", label: t("system.base.log.field.action") },
    { key: "project", label: t("system.base.log.field.project") },
    { key: "decision", label: t("system.base.log.field.decision"), enum: decisionOptions.value },
    { key: "reason_code", label: t("system.base.log.field.reason_code") },
    { key: "reason", label: t("system.base.log.field.reason"), span: 2 },
    { key: "duration_ms", label: t("system.base.log.field.duration_ms"), format: value => formatLogDuration(value, t) },
    { key: "candidate_count", label: t("system.base.log.field.candidate_count"), format: value => formatLogCount(value, t) },
    { key: "matched_count", label: t("system.base.log.field.matched_count"), format: value => formatLogCount(value, t) },
    { key: "input_hash", label: t("system.base.log.field.input_hash") },
    { key: "client_ip", label: t("system.base.log.field.client_ip") },
    { key: "request_id", label: t("system.base.log.field.request_id") },
    { key: "trace_id", label: t("system.base.log.field.trace_id") },
    { key: "occurred_at", label: t("system.base.log.field.occurred_at") },
    { key: "created_at", label: t("system.base.log.field.created_at") }
  ],
  request: requestTable,
  get: getDetail
}));

async function requestTable(params: Record<string, unknown>) {
  const response = await defBasePolicyEvaluationLogService.PageBasePolicyEvaluationLog(buildPageRequest(params as unknown as PageBasePolicyEvaluationLogRequest));
  return { list: response.base_policy_evaluation_logs as unknown as Record<string, unknown>[], total: response.total };
}

async function getDetail(id: string) {
  return (await defBasePolicyEvaluationLogService.GetBasePolicyEvaluationLog({ id })) as unknown as Record<string, unknown>;
}
</script>
