<template>
  <template v-for="(block, index) in blocks" :key="`${block.type}:${block.query_record_id ?? index}`">
    <el-alert
      v-if="block.type === 'bi_permission_denied' && block.code === 'PERMISSION_DENIED'"
      class="bi-chat-alert"
      :title="t('bi.chat.permission_title')"
      :description="t('bi.chat.permission_description')"
      type="warning"
      :closable="false"
      show-icon
    />
    <section v-else-if="block.type === 'bi_query_clarification'" class="bi-chat-block">
      <header class="bi-chat-block__header">
        <strong>{{ t('bi.chat.choose_metric') }}</strong>
        <span>{{ block.source_name }}</span>
      </header>
      <div class="bi-chat-block__choices">
        <button
          v-for="option in block.options"
          :key="option.code"
          type="button"
          class="bi-chat-block__choice"
          :disabled="!canFillDraft || !block.question"
          @click="chooseMetric(block, option)"
        >
          <span>{{ option.name || option.code }}</span>
          <small v-if="option.unit">{{ option.unit }}</small>
          <el-icon><ArrowRight /></el-icon>
        </button>
      </div>
    </section>
    <section v-else-if="block.type === 'bi_query_empty'" class="bi-chat-block">
      <strong>{{ t('bi.chat.empty') }}</strong>
      <ul v-if="block.query?.time_range || block.query?.filters?.length" class="bi-chat-block__scope">
        <li v-if="block.query?.time_range">{{ describeBiTimeRange(block.query.time_range, t) }}</li>
        <li v-for="(filter, filterIndex) in block.query?.filters" :key="filterIndex">{{ filterLabel(block, filter) }}</li>
      </ul>
      <div v-if="block.question && canFillDraft" class="bi-chat-block__actions">
        <el-button size="small" @click="emit('fill-draft', block.question)">{{ t('bi.chat.adjust_query') }}</el-button>
        <el-button v-if="canAdjustToRecentWeek(block)" size="small" @click="emit('fill-draft', t('bi.chat.last_week_draft', { question: block.question }))">
          {{ t('bi.refine.last_7') }}
        </el-button>
      </div>
    </section>
    <section v-else-if="block.type === 'bi_query_result' || block.type === 'bi_card_guide'" class="bi-chat-block">
      <header class="bi-chat-block__header">
        <strong>{{ resultTitle(block) }}</strong>
        <span v-if="block.truncated">{{ t('bi.query.truncated') }}</span>
      </header>
      <dl v-if="block.rows?.length === 1 && block.columns?.length && !block.query?.dimensions?.length" class="bi-chat-block__metrics">
        <div v-for="(column, columnIndex) in block.columns" :key="column.code">
          <dt>{{ column.name || column.code }}</dt>
          <dd>{{ block.rows[0]?.values?.[columnIndex] ?? '—' }} <small v-if="column.unit">{{ column.unit }}</small></dd>
        </div>
      </dl>
      <div v-else-if="block.rows?.length && block.columns?.length" class="bi-chat-block__table" role="region" :aria-label="resultTitle(block)" tabindex="0">
        <table>
          <thead>
            <tr v-if="block.row_count && block.row_count > block.rows.length" class="bi-chat-block__table-count">
              <td :colspan="block.columns.length">{{ t('bi.chat.preview_rows', { shown: block.rows.length, returned: block.row_count }) }}</td>
            </tr>
            <tr>
              <th v-for="column in block.columns" :key="column.code" scope="col" :class="{ 'is-numeric': column.type === 'number' }">
                {{ column.name || column.code }} <small v-if="column.unit">({{ column.unit }})</small>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, rowIndex) in block.rows" :key="rowIndex">
              <td v-for="(column, columnIndex) in block.columns" :key="column.code" :class="{ 'is-numeric': column.type === 'number' }">
                {{ row.values?.[columnIndex] ?? '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="block.query_record_id" class="bi-chat-block__actions">
        <span v-if="savedRecords.has(block.query_record_id)" role="status" class="bi-chat-block__saved">
          <el-icon><CircleCheck /></el-icon>{{ t('bi.card.saved') }}
        </span>
        <el-button v-else-if="block.query" :disabled="dialogVisible" size="small" @click="openCardDialog(block)">
          <el-icon class="el-icon--left"><DataAnalysis /></el-icon>
          {{ t('bi.chat.create_card') }}
        </el-button>
        <el-button
          v-if="block.type === 'bi_query_result' && (!exportFailures.has(block.query_record_id) || exportFailures.get(block.query_record_id) === 'retry')"
          :loading="exportingRecords.has(block.query_record_id)"
          size="small"
          @click="exportResult(block, block.query_record_id)"
        >
          <el-icon v-if="!exportingRecords.has(block.query_record_id)" class="el-icon--left"><Download /></el-icon>
          {{ t(exportActionLabel(block)) }}
        </el-button>
      </div>
      <el-alert
        v-if="block.query_record_id && exportFailures.has(block.query_record_id)"
        :title="t(exportFailureLabels[exportFailures.get(block.query_record_id) ?? 'retry'])"
        :description="exportFailures.get(block.query_record_id) === 'retry' ? t('bi.chat.export_summary_preserved') : undefined"
        type="warning"
        :closable="false"
        show-icon
      />
    </section>
  </template>
  <BiCardGuideDialog v-if="dialogGuide" v-model="dialogVisible" :guide="dialogGuide" @saved="saveCard" />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { ChatMessageItem } from "@liujitcn/kratos-admin-system/components/ai/chat";
import { defBiQueryRecordService } from "../api/bi/admin/v1/bi_query_record";
import { resolveBiChatBlocks, type BiCardGuide, type BiChatBlock, type BiMetricOption } from "../bi-chat";
import { classifyBiCsvExportFailure, downloadBiCsv, type BiCsvExportFailure } from "../bi-csv";
import { normalizeBiQuerySpec } from "../bi-response";
import { describeBiQuery, describeBiTimeRange } from "../query-summary";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiFilterSpec } from "../rpc/bi/admin/v1/bi_query";

const BiCardGuideDialog = defineAsyncComponent(() => import("../views/query/components/BiCardGuideDialog.vue"));
const props = defineProps<{ message: ChatMessageItem; activeFlowMessageId: string }>();
const emit = defineEmits<{ "fill-draft": [text: string]; "block-saved": [card: BiCard] }>();
const blocks = computed(() => resolveBiChatBlocks((props.message.blocks ?? []) as unknown as BiChatBlock[], props.message.tools));
const canFillDraft = computed(() => Boolean(props.activeFlowMessageId && props.activeFlowMessageId === String(props.message.id)));
const dialogVisible = ref(false);
const dialogGuide = ref<BiCardGuide>();
const savedRecords = ref(new Set<number>());
const exportingRecords = ref(new Set<number>());
const exportedRecords = ref(new Set<number>());
const exportFailures = ref(new Map<number, BiCsvExportFailure>());
const exportFailureLabels: Record<BiCsvExportFailure, string> = {
  denied: "bi.chat.export_denied",
  unavailable: "bi.chat.export_unavailable",
  retry: "bi.chat.export_failed"
};

/** 区分首次导出、失败重试以及再次下载同一结果。 */
function exportActionLabel(block: BiChatBlock) {
  const recordID = block.query_record_id;
  if (recordID && exportFailures.value.has(recordID)) return "bi.chat.export_retry";
  if (recordID && exportedRecords.value.has(recordID)) return "bi.chat.export_again";
  return block.truncated ? "bi.chat.export_returned_csv" : "bi.chat.export_csv";
}

/** 使用模型建议标题，缺省时使用实际查询指标及维度的名称。 */
function resultTitle(block: BiChatBlock) {
  if (block.default_name) return block.default_name;
  return block.query ? describeBiQuery(normalizeBiQuerySpec(block.query), t, undefined, block.columns).title : t("bi.query.result");
}

/** 显示本次执行的过滤条件，结果返回列名称时优先显示名称。 */
function filterLabel(block: BiChatBlock, filter: BiFilterSpec) {
  const field = block.columns?.find(column => column.code === filter.field)?.name || filter.field;
  return `${field} ${t(`bi.chat.filter.${filter.operator}`)} ${filter.values?.join("、") || "—"}`;
}

/** 已经查询过去七天时不重复推荐同一查询时间范围。 */
function canAdjustToRecentWeek(block: BiChatBlock) {
  const range = block.query?.time_range;
  return range && !(range.start === "now-7d" && range.end === "now");
}

/** 用目录口径补全原提问，发送时由现有查询入口重新校验。 */
function chooseMetric(block: BiChatBlock, option: BiMetricOption) {
  if (!canFillDraft.value || !block.question) return;
  emit("fill-draft", t("bi.chat.metric_draft", {
    question: block.question,
    source: block.source_name || block.source_code || "",
    sourceCode: block.source_code || "",
    metric: option.name || option.code,
    metricCode: option.code
  }));
}

/** 创建卡片时使用原查询的可追溯记录和推荐图型。 */
function openCardDialog(block: BiChatBlock) {
  dialogGuide.value = { ...block, default_name: resultTitle(block) };
  dialogVisible.value = true;
}

/** 记录本次已保存的来源并把实际卡片回传给工作台。 */
function saveCard(card: BiCard) {
  const recordID = dialogGuide.value?.query_record_id;
  if (recordID) savedRecords.value.add(recordID);
  dialogVisible.value = false;
  emit("block-saved", card);
}

/** 导出服务端已保存的问数快照，避免重复执行查询。 */
async function exportResult(block: BiChatBlock, recordID: number) {
  if (exportingRecords.value.has(recordID)) return;
  exportingRecords.value.add(recordID);
  try {
    const { result } = await defBiQueryRecordService.GetBiQuery({ id: recordID });
    if (!result?.rows?.length) {
      exportFailures.value.set(recordID, "unavailable");
      return;
    }
    const title = resultTitle(block);
    downloadBiCsv(result, title);
    exportedRecords.value.add(recordID);
    exportFailures.value.delete(recordID);
    ElMessage.success(t("bi.card.export_success", { name: title }));
  } catch (error) {
    exportFailures.value.set(recordID, classifyBiCsvExportFailure(error));
  } finally {
    exportingRecords.value.delete(recordID);
  }
}
</script>

<style scoped>
.bi-chat-block,
.bi-chat-alert {
  margin-block: 12px;
}

.bi-chat-block {
  display: grid;
  gap: 12px;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
  background: var(--el-bg-color);
}

.bi-chat-block__header,
.bi-chat-block__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.bi-chat-block__header {
  justify-content: space-between;
}

.bi-chat-block__header span,
.bi-chat-block small {
  color: var(--el-text-color-secondary);
}

.bi-chat-block__choices {
  display: grid;
  gap: 6px;
}

.bi-chat-block__choice {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 8px 10px;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: start;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
  cursor: pointer;
}

.bi-chat-block__choice:disabled {
  color: var(--el-text-color-secondary);
  cursor: default;
}

.bi-chat-block__choice:not(:disabled):is(:hover, :focus-visible) {
  border-color: var(--el-color-primary);
}

.bi-chat-block__choice .el-icon {
  margin-inline-start: auto;
}

.bi-chat-block__scope,
.bi-chat-block__metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.bi-chat-block__scope li,
.bi-chat-block__metrics div {
  padding: 4px 8px;
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-light);
  overflow-wrap: anywhere;
}

.bi-chat-block__metrics dd {
  margin: 0;
  font-size: 1.15em;
  font-weight: 600;
}

.bi-chat-block__table {
  overflow-x: auto;
}

.bi-chat-block__table table {
  width: 100%;
  border-collapse: collapse;
  white-space: nowrap;
}

.bi-chat-block__table :is(th, td) {
  padding: 4px 8px;
  text-align: start;
  border-block-end: 1px solid var(--el-border-color-lighter);
}

.bi-chat-block__table .is-numeric {
  font-variant-numeric: tabular-nums;
  text-align: end;
}

.bi-chat-block__table-count {
  color: var(--el-text-color-secondary);
  font-size: .85em;
}

.bi-chat-block__actions :deep(.el-button + .el-button) {
  margin-inline-start: 0;
}

.bi-chat-block__saved {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--el-color-success);
}
</style>
