<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :request-api="requestTable" />

    <el-drawer
      v-model="detail.visible"
      :title="t('system.ai.query.detail.title')"
      size="min(720px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="detail.row" v-loading="detail.loading" class="ai-query-detail">
        <div class="ai-query-detail__meta">
          <span>{{ t("system.ai.query.field.user") }}：{{ displayUser(detail.row) }}</span>
          <el-tag :type="statusTagType(detail.row.status)" size="small" effect="plain">
            {{ statusLabel(detail.row.status) }}
          </el-tag>
          <span>{{ t("system.ai.query.field.row_count") }}：{{ detail.row.row_count }}</span>
          <span>{{ t("system.ai.query.field.elapsed_ms") }}：{{ detail.row.elapsed_ms }}</span>
          <span>{{ t("common.field.created_at") }}：{{ detail.row.created_at }}</span>
        </div>
        <div class="ai-query-detail__question">{{ detail.row.question }}</div>
        <div v-if="detail.row.error_msg" class="ai-query-detail__error">
          {{ t("system.ai.query.field.error_msg") }}：{{ detail.row.error_msg }}
        </div>
        <div class="ai-query-detail__sql-title">{{ t("system.ai.query.field.sql") }}</div>
        <CodeBlock class="ai-query-detail__sql" :code="detail.row.sql || '-'" language="sql" />
        <template v-if="resultRows.length">
          <div class="ai-query-detail__sql-title">{{ t("system.ai.query.detail.result") }}</div>
          <el-table :data="resultRows" size="small" max-height="320" border class="ai-query-detail__result">
            <el-table-column
              v-for="(column, index) in detail.row.result_columns"
              :key="`${column}-${index}`"
              :prop="String(index)"
              :label="column"
              show-overflow-tooltip
              min-width="120"
            />
          </el-table>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { View } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import CodeBlock from "@liujitcn/kratos-admin-core/components/CodeBlock/index.vue";
import type { ColumnProps, EnumProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiQueryService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_query";
import type { AiQueryRecord, PageAiQueryRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_query";
import { AiQueryStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_query";

defineOptions({ name: "AiQuery", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const detail = reactive({
  visible: false,
  row: null as AiQueryRecord | null,
  loading: false
});

/** 问数状态枚举选项。 */
const statusEnum = computed<EnumProps[]>(() => [
  { label: t("system.ai.query.status.success"), value: AiQueryStatus.AI_QUERY_STATUS_SUCCESS, tagType: "success" },
  { label: t("system.ai.query.status.failed"), value: AiQueryStatus.AI_QUERY_STATUS_FAILED, tagType: "danger" }
]);

const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.ai.query.field.id"), width: 90, align: "center" },
  {
    prop: "question",
    label: t("system.ai.query.field.question"),
    minWidth: 220,
    showOverflowTooltip: true,
    search: { el: "input" }
  },
  {
    prop: "user",
    label: t("system.ai.query.field.user"),
    minWidth: 120,
    showOverflowTooltip: true,
    render: scope => displayUser(scope.row as AiQueryRecord)
  },
  {
    prop: "status",
    label: t("system.ai.query.field.status"),
    width: 110,
    align: "center",
    enum: statusEnum,
    search: { el: "select", props: { clearable: true } }
  },
  { prop: "row_count", label: t("system.ai.query.field.row_count"), width: 100, align: "right" },
  { prop: "elapsed_ms", label: t("system.ai.query.field.elapsed_ms"), width: 110, align: "right" },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 170, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    fixed: "right",
    width: 120,
    actions: [
      {
        label: t("system.ai.query.action.detail"),
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:ai-query:detail"],
        onClick: scope => openDetail(scope.row as AiQueryRecord)
      }
    ]
  }
]);

/** 请求智能问数记录表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const request = buildPageRequest<PageAiQueryRequest>(params as unknown as PageAiQueryRequest);
  if (!request.status) delete request.status;
  const data = await defAiQueryService.PageAiQuery(request);
  return { data: { list: data.queries ?? [], total: data.total } };
}

/** 展示记录所属用户，昵称缺失时回退账号。 */
function displayUser(row: AiQueryRecord): string {
  return row.user_name || String(row.user_id ?? "");
}

/** 转换问数状态为展示文案。 */
function statusLabel(status?: AiQueryStatus): string {
  return status === AiQueryStatus.AI_QUERY_STATUS_SUCCESS
    ? t("system.ai.query.status.success")
    : status === AiQueryStatus.AI_QUERY_STATUS_FAILED
      ? t("system.ai.query.status.failed")
      : "";
}

/** 转换问数状态为标签类型。 */
function statusTagType(status?: AiQueryStatus): "success" | "danger" | "info" {
  switch (status) {
    case AiQueryStatus.AI_QUERY_STATUS_SUCCESS:
      return "success";
    case AiQueryStatus.AI_QUERY_STATUS_FAILED:
      return "danger";
    default:
      return "info";
  }
}

/** 打开问数详情抽屉。 */
async function openDetail(row: AiQueryRecord) {
  detail.row = row;
  detail.visible = true;
  detail.loading = true;
  try {
    detail.row = await defAiQueryService.GetAiQuery({ id: row.id });
  } finally {
    detail.loading = false;
  }
}

/** 将结果快照行转换为表格行数据，按列下标取值。 */
const resultRows = computed<Record<string, string>[]>(() => {
  const columns = detail.row?.result_columns ?? [];
  return (detail.row?.result_rows ?? []).map(row => {
    const item: Record<string, string> = {};
    columns.forEach((_, index) => {
      item[String(index)] = row.values[index] ?? "";
    });
    return item;
  });
});
</script>

<style scoped>
.ai-query-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ai-query-detail__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.ai-query-detail__question {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.6;
}

.ai-query-detail__error {
  padding: 8px 12px;
  color: var(--el-color-danger);
  font-size: 13px;
  border: 1px solid var(--el-color-danger-light-8);
  border-radius: 6px;
  background-color: var(--el-color-danger-light-9);
}

.ai-query-detail__sql-title {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.ai-query-detail__result {
  width: 100%;
}

:deep(.ai-query-detail__result .el-table__cell) {
  font-size: 12px;
}
</style>
