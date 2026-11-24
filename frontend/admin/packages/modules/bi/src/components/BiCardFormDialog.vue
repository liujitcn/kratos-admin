<template>
  <ProDialog
    class="bi-card-editor-dialog"
    :model-value="modelValue"
    :title="card ? t('bi.card.edit') : t('bi.card.create')"
    width="1160px"
    align-center
    destroy-on-close
    :close-on-click-modal="!saving && !pendingSave"
    :close-on-press-escape="!saving && !pendingSave"
    :show-close="!saving && !pendingSave"
    @update:model-value="value => { if (!saving && !pendingSave) emit('update:modelValue', value); }"
    @closed="reset"
  >
    <div class="bi-card-editor" :class="{ 'is-narrow': narrow }">
      <div class="bi-card-editor__settings">
        <ProForm ref="nameFormRef" :model="form" :fields="nameFields" :rules="rules" label-position="top" :disabled="editingLocked" />
        <el-tabs v-model="activeTab" class="bi-card-editor__tabs">
          <el-tab-pane name="query" :label="t('bi.card.query_tab')">
            <ProForm
              ref="queryFormRef"
              class="bi-card-editor__query"
              :model="form"
              :fields="queryFields"
              :rules="rules"
              :gutter="12"
              label-position="top"
              :disabled="editingLocked"
            >
              <template #measures>
                <div class="bi-card-measures">
                  <div v-for="(measure, index) in form.measures" :key="index" class="bi-card-measures__row">
                    <el-select disabled v-model="measure.code" :aria-label="t('bi.card.measure')">
                      <el-option v-for="item in source?.measures ?? []" :key="item.code" :label="item.name" :value="item.code" />
                    </el-select>
                    <span>{{ source?.measures.find(item => item.code === measure.code)?.unit }}</span>
                  </div>
                </div>
              </template>
            </ProForm>
            <div v-if="querySummary" class="bi-card-query-summary">
              <el-text>{{ t("bi.card.dimensions") }}：{{ querySummary.dimensions }}</el-text>
              <el-text>{{ t("bi.refine.period") }}：{{ querySummary.period }}</el-text>
            </div>
            <BiQueryRefinement
              v-if="!card && source && baseQuery && sourceRecordID"
              :source="source"
              :query="baseQuery"
              :record-id="sourceRecordID"
              :disabled="editingLocked || previewing"
              @busy="refining = $event"
              @applied="applyRefinement"
            />
          </el-tab-pane>
          <el-tab-pane name="chart" :label="t('bi.card.chart_tab')">
            <BiCardChartEditor v-model="chartConfig" v-model:chart-type="form.chart_type" :query="query" :result="previewResult" :disabled="editingLocked" />
          </el-tab-pane>
          <el-tab-pane name="sql" :label="t('bi.sql.tab')" lazy>
            <div v-loading="loadingDetails" class="bi-card-formulas">
              <el-radio-group v-model="codeView" size="small">
                <el-radio-button value="formula">{{ t('bi.sql.expression') }}</el-radio-button>
                <el-radio-button value="sql">{{ t('bi.sql.full_sql') }}</el-radio-button>
              </el-radio-group>
              <template v-if="codeView === 'formula'">
                <el-alert :title="t(canEditMetrics ? 'bi.sql.shared_notice' : 'bi.sql.readonly_notice')" type="info" show-icon :closable="false" />
                <div v-for="metric in editedMetrics" :key="metric.code" class="bi-card-formulas__metric">
                  <el-text tag="strong">{{ metric.name }} <el-text tag="small">{{ metric.unit }}</el-text></el-text>
                  <CodeEditor :key="editingLocked || !canEditMetrics ? 'readonly' : 'editable'" v-model="metric.expression" language="sql" height="80px" :readonly="editingLocked || !canEditMetrics" />
                </div>
              </template>
              <CodeEditor v-else-if="sql" :model-value="sql" language="sql" height="clamp(120px, 32dvh, 320px)" readonly />
              <el-empty v-else :image-size="36" :description="t('bi.sql.preview_empty')" />
              <div v-for="change in impact?.changes ?? []" :key="change.after?.code ?? change.before?.code" class="bi-card-formulas__metric">
                <el-text tag="strong">{{ change.after?.name ?? change.before?.name }}</el-text>
                <code>{{ change.before?.expression }} → {{ change.after?.expression }}</code>
              </div>
              <el-button v-if="card" :disabled="editingLocked || previewing" @click="loadDetails(true)">{{ t('bi.sql.refresh_model') }}</el-button>
            </div>
          </el-tab-pane>
          <el-tab-pane v-if="narrow" name="preview" :label="t('bi.card.preview')" class="bi-card-editor__preview-pane">
            <div ref="previewTarget" class="bi-card-editor__preview-pane" />
          </el-tab-pane>
        </el-tabs>
      </div>
      <Teleport :to="previewTarget ?? 'body'" :disabled="!narrow || !previewTarget">
        <section class="bi-card-editor__preview" :aria-label="t('bi.card.preview')">
          <el-text v-if="!narrow" tag="strong">{{ t('bi.card.preview') }}</el-text>
          <BiDataCard
            v-if="previewResult"
            :card="previewCard"
            :preview-result="previewResult"
            :source-name="narrow ? undefined : source?.name"
            :show-category="false"
          />
          <el-empty v-else :image-size="36" :description="t('bi.card.preview_empty')" />
        </section>
      </Teleport>
    </div>
    <template #footer>
      <div class="bi-card-editor__footer">
        <el-alert v-if="pendingSave && !saving" :title="t('bi.sql.save_pending')" type="warning" show-icon :closable="false" />
        <el-alert
          v-else-if="impact?.changes?.length"
          :title="t('bi.sql.impact', { cards: impact.affected_cards ?? 0, dashboards: impact.affected_dashboards ?? 0 })"
          type="warning"
          show-icon
          :closable="false"
        />
        <div class="bi-card-editor__actions">
          <el-button :disabled="saving || Boolean(pendingSave)" @click="emit('update:modelValue', false)">{{ t('common.action.cancel') }}</el-button>
          <el-button :loading="previewing" :disabled="editingLocked || refining" @click="preview(true)">{{ t('bi.card.preview') }}</el-button>
          <el-button type="primary" :loading="saving" :disabled="loadingDetails || refining" @click="submit">{{ t('bi.card.save') }}</el-button>
        </div>
      </div>
    </template>
  </ProDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useMediaQuery } from "@vueuse/core";
import { t } from "@liujitcn/kratos-admin-core";
import CodeEditor from "@liujitcn/kratos-admin-core/components/CodeEditor/index.vue";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BiCard, SaveBiCardRequest } from "../rpc/bi/admin/v1/bi_card";
import type { AskBiQueryResponse, BiMeasureSpec, BiQueryResult, BiQuerySpec, BiSourceDescriptor } from "../rpc/bi/admin/v1/bi_query";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import type { BiModelImpact, BiSourceMeasure } from "../rpc/bi/admin/v1/bi_source";
import { metricChanges, mergeMetricDrafts, appliedCardID } from "../metric-edit";
import BiDataCard from "./BiDataCard.vue";
import BiCardChartEditor from "./BiCardChartEditor.vue";
import BiQueryRefinement from "./BiQueryRefinement.vue";
import { parseCardChart } from "../card-schema";
import { normalizeBiQuerySpec } from "../bi-response";
import { describeBiQuery } from "../query-summary";
import type { CardChartConfig } from "../card-schema";

/** 卡片编辑状态，查询条件沿用原语义计划。 */
interface BiCardEditState {
  name: string;
  source_code: string;
  dimensions: string[];
  measures: BiMeasureSpec[];
  chart_type: string;
  refresh_seconds: number;
  is_shared: boolean;
}

const props = defineProps<{
  modelValue: boolean;
  sources: BiSourceDescriptor[];
  card?: BiCard;
  initialQuery?: BiQuerySpec;
  initialResult?: BiQueryResult;
  initialTitle?: string;
  initialChartType?: string;
}>();
const emit = defineEmits<{ "update:modelValue": [value: boolean]; saved: [card: BiCard] }>();
const nameFormRef = ref<ProFormInstance>();
const queryFormRef = ref<ProFormInstance>();
const previewTarget = ref<HTMLElement>();
const narrow = useMediaQuery("(max-width: 680px)");
const activeTab = ref("chart");
const codeView = ref("formula");
const form = reactive<BiCardEditState>(defaultForm());
const baseQuery = ref<BiQuerySpec>();
const previewResult = ref<BiQueryResult>();
const sourceRecordID = ref(0);
const chartConfig = ref<CardChartConfig>({ encoding: {}, style: { showLegend: true, showLabel: false } });
const saving = ref(false);
const pendingSave = ref<SaveBiCardRequest>();
const previewing = ref(false);
const refining = ref(false);
const loadingDetails = ref(false);
const canEditMetrics = ref(false);
const editingLocked = computed(() => saving.value || loadingDetails.value || Boolean(pendingSave.value));
const originalMetrics = ref<BiSourceMeasure[]>([]);
const editedMetrics = ref<BiSourceMeasure[]>([]);
const sql = ref("");
const impact = ref<BiModelImpact>();
const previewRecordID = ref(0);
const cardVersion = ref(0);
const sourceVersion = ref(0);
const changes = computed(() => metricChanges(originalMetrics.value, editedMetrics.value));
let previewRequest = 0;
let session = 0;
let suggestedName = "";
const source = computed(() => props.sources.find(item => item.code === form.source_code));
const query = computed<BiQuerySpec>(() => ({
  source_code: form.source_code,
  dimensions: [...form.dimensions],
  measures: form.measures.map(item => ({ ...item })),
  filters: baseQuery.value?.filters ?? [],
  limit: baseQuery.value?.limit || 100,
  time_range: baseQuery.value?.time_range,
  orders: baseQuery.value?.orders ?? [],
  grouping_options: baseQuery.value?.grouping_options ?? []
}));
const querySummary = computed(() => baseQuery.value && describeBiQuery(baseQuery.value, t, source.value, previewResult.value?.columns ?? props.card?.columns));
const previewCard = computed<BiCard>(() => ({
  id: 0,
  tenant_id: 0,
  project_id: 0,
  owner_id: 0,
  name: form.name,
  source_code: form.source_code,
  query: query.value,
  chart_type: form.chart_type,
  chart_json: JSON.stringify(chartConfig.value),
  query_record_id: sourceRecordID.value,
  refresh_seconds: form.refresh_seconds,
  is_shared: form.is_shared,
  updated_at: "",
  category: "mine",
  version: 0, validation_query_record_id: 0, execution_kind: "", source_version: 0, can_edit_metrics: false, metric_definitions: [], sql: "", sync_summary: undefined, columns: [],
  can_edit: true
}));
const nameFields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("bi.card.name"), component: "input", props: { maxlength: 128 } }
]);
const queryFields = computed<ProFormField[]>(() => [
  {
    prop: "source_code",
    label: t("bi.card.source"),
    component: "select",
    colSpan: 12,
    options: props.sources.map(item => ({ label: item.name, value: item.code })),
    props: { disabled: true }
  },
  {
    prop: "dimensions",
    label: t("bi.card.dimensions"),
    component: "select",
    colSpan: 12,
    options: (source.value?.dimensions ?? []).map(item => ({ label: item.name, value: item.code })),
    props: {
      multiple: true,
      multipleLimit: 2,
      disabled: true
    }
  },
  { prop: "measures", label: t("bi.card.measure"), component: "slot", slotName: "measures", colSpan: 12 },
  {
    prop: "refresh_seconds",
    label: t("bi.card.refresh"),
    component: "select",
    colSpan: 12,
    rowBreakBefore: true,
    options: [0, 300, 86400].map(value => ({ label: t(`bi.card.refresh.${value}`), value }))
  },
  { prop: "is_shared", label: t("bi.card.share"), component: "switch", colSpan: 12 }
]);
const rules = computed(() => ({
  name: [{ required: true, whitespace: true, message: t("bi.card.validation.name"), trigger: "blur" }],
  source_code: [{ required: true, message: t("bi.card.validation.source"), trigger: "change" }],
  measures: [{ type: "array" as const, required: true, min: 1, message: t("bi.card.validation.measures"), trigger: "change" }]
}));

/** 创建独立表单状态，关闭后不保留上次卡片。 */
function defaultForm(): BiCardEditState {
  return {
    name: "",
    source_code: "",
    dimensions: [],
    measures: [],
    chart_type: "bar",
    refresh_seconds: 0,
    is_shared: false
  };
}

/** 打开弹窗时复制原查询，保留问数生成的过滤、排序和时间范围。 */
watch(
  () => props.modelValue,
  visible => {
    session++;
    previewRequest++;
    if (!visible) return;
    activeTab.value = "chart";
    codeView.value = "formula";
    saving.value = false;
    previewing.value = false;
    refining.value = false;
    loadingDetails.value = false;
    const original = props.card?.query ?? props.initialQuery;
    const current = original ? normalizeBiQuerySpec(original) : undefined;
    baseQuery.value = current ? (JSON.parse(JSON.stringify(current)) as BiQuerySpec) : undefined;
    const initialSource = props.sources.find(item => item.code === current?.source_code);
    suggestedName = (current?.grouping_options.length
      ? describeBiQuery(current, t, initialSource, props.initialResult?.columns).title
      : props.initialTitle ?? "").slice(0, 128);
    Object.assign(form, defaultForm(), {
      name: props.card?.name ?? suggestedName,
      source_code: current?.source_code || props.sources[0]?.code || "",
      dimensions: [...(current?.dimensions ?? [])],
      measures: (current?.measures ?? []).map(item => ({ ...item })),
      chart_type: props.card?.chart_type || props.initialChartType || (current && !current.dimensions.length ? "metric" : "bar"),
      refresh_seconds: props.card?.refresh_seconds ?? 0,
      is_shared: props.card?.is_shared ?? false
    });
    previewResult.value = props.initialResult;
    sourceRecordID.value = props.card?.query_record_id || props.initialResult?.query_record_id || 0;
    chartConfig.value = parseCardChart(props.card ?? { chart_json: "" });
    cardVersion.value = props.card?.version ?? 0;
    sourceVersion.value = source.value?.version ?? 0;
    previewRecordID.value = 0;
    impact.value = undefined;
    originalMetrics.value = [];
    editedMetrics.value = [];
    sql.value = "";
    canEditMetrics.value = false;
    if (props.card) void loadDetails(false);
    else void preview();
  }
);
watch(narrow, compact => {
  if (!compact && activeTab.value === "preview") activeTab.value = "chart";
});
watch(
  query,
  () => {
    previewRequest++;
    previewResult.value = undefined;
    sourceRecordID.value = 0;
    chartConfig.value = { encoding: {}, style: { ...chartConfig.value.style } };
  },
  { deep: true, flush: "sync" }
);

/** 公式变化后使成功预览失效；样式编辑不影响公式预览。 */
watch(editedMetrics, () => {
  previewRequest++;
  previewRecordID.value = 0;
  impact.value = undefined;
  sql.value = "";
  previewResult.value = undefined;
}, { deep: true, flush: "sync" });

/** 读取当前卡片的源版本和授权公式，冲突刷新保留本地公式输入。 */
async function loadDetails(preserve: boolean) {
  if (!props.card || pendingSave.value) return;
  const request = ++previewRequest;
  const currentSession = session;
  loadingDetails.value = true;
  try {
    const detail = await defBiCardService.GetBiCard({ id: props.card.id });
    if (request !== previewRequest || !props.modelValue) return;
    cardVersion.value = detail.version;
    sourceVersion.value = detail.source_version;
    canEditMetrics.value = detail.can_edit_metrics;
    const definitions = detail.metric_definitions ?? [];
    originalMetrics.value = definitions.map(item => ({ ...item }));
    editedMetrics.value = preserve ? mergeMetricDrafts(definitions, editedMetrics.value) : definitions.map(item => ({ ...item }));
    sql.value = changes.value.length ? "" : detail.sql;
    if (preserve) ElMessage.info(t("bi.sql.repreview"));
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (currentSession === session) loadingDetails.value = false;
  }
}

/** 更新卡片来源并清除与旧查询绑定的预览。 */
function applyRefinement(next: AskBiQueryResponse) {
  if (!next.query || !next.result?.query_record_id || !props.modelValue) return;
  const useSuggestedName = !form.name.trim() || form.name === suggestedName;
  baseQuery.value = next.query;
  form.dimensions = [...next.query.dimensions];
  form.chart_type = next.chart_type;
  suggestedName = describeBiQuery(next.query, t, source.value, next.result.columns).title.slice(0, 128);
  if (useSuggestedName) form.name = suggestedName;
  previewRecordID.value = 0;
  sql.value = "";
  impact.value = undefined;
  sourceRecordID.value = next.result.query_record_id;
  previewResult.value = next.result;
}

/** 真实预览精确公共公式候选，并忽略编辑前的迟到响应。 */
async function preview(showResult = false) {
  if (previewing.value || refining.value || loadingDetails.value || pendingSave.value) return;
  if (!sourceRecordID.value) {
    ElMessage.warning(t("bi.card.ask_required"));
    return;
  }
  const request = ++previewRequest;
  const currentSession = session;
  previewing.value = true;
  try {
    const response = await defBiCardService.PreviewBiCardSql({
      id: props.card?.id ?? 0,
      expected_card_version: cardVersion.value,
      expected_source_version: sourceVersion.value,
      query_record_id: sourceRecordID.value,
      query: query.value,
      measure_changes: changes.value
    });
    if (request !== previewRequest || !props.modelValue) return;
    canEditMetrics.value = response.can_edit_metrics;
    sourceVersion.value = response.source_version;
    if (!originalMetrics.value.length) {
      originalMetrics.value = (response.metric_definitions ?? []).map(item => ({ ...item }));
      editedMetrics.value = originalMetrics.value.map(item => ({ ...item }));
    }
    previewResult.value = response.result;
    previewRecordID.value = response.preview_query_record_id;
    sql.value = response.sql;
    impact.value = response.impact;
    if (showResult && narrow.value && response.result) activeTab.value = "preview";
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (currentSession === session) previewing.value = false;
  }
}

/** 保存精确候选，结果未知时重试冻结请求，避免新预览绕过一次消费。 */
async function submit() {
  if (!pendingSave.value) {
    const [nameValid, queryValid] = await Promise.all([nameFormRef.value?.validate(), queryFormRef.value?.validate()]);
    if (!queryValid) activeTab.value = "query";
    if (!nameValid || !queryValid) return;
  }
  if (saving.value || previewing.value || refining.value || loadingDetails.value) return;
  if (!pendingSave.value && !sourceRecordID.value) {
    ElMessage.warning(t("bi.card.ask_required"));
    return;
  }
  if (!pendingSave.value && changes.value.length && !previewRecordID.value) {
    ElMessage.warning(t("bi.sql.repreview"));
    return;
  }
  saving.value = true;
  try {
    pendingSave.value ??= JSON.parse(JSON.stringify({
      id: props.card?.id ?? 0,
      expected_card_version: cardVersion.value,
      expected_source_version: sourceVersion.value,
      preview_query_record_id: previewRecordID.value,
      measure_changes: changes.value,
      card: {
        name: form.name,
        query: query.value,
        chart_type: form.chart_type,
        chart_json: JSON.stringify(chartConfig.value),
        query_record_id: sourceRecordID.value,
        refresh_seconds: form.refresh_seconds,
        is_shared: form.is_shared
      }
    })) as SaveBiCardRequest;
    const saved = await defBiCardService.SaveBiCard(pendingSave.value);
    pendingSave.value = undefined;
    emit("saved", saved);
    emit("update:modelValue", false);
    ElMessage.success(saved.sync_summary?.changes?.length ? t("bi.sql.synced", { cards: saved.sync_summary.affected_cards ?? 0, dashboards: saved.sync_summary.affected_dashboards ?? 0 }) : t("bi.card.saved"));
  } catch (cause) {
    const id = appliedCardID(cause);
    if (id) {
      try {
        const saved = await defBiCardService.GetBiCard({ id });
        pendingSave.value = undefined;
        emit("saved", saved);
        emit("update:modelValue", false);
        ElMessage.info(t("bi.sql.already_applied"));
        return;
      } catch (readError) {
        ElMessage.error(readError instanceof Error ? readError.message : t("bi.card.save_failed"));
        return;
      }
    }
    const status = (cause as { response?: { status?: number } } | null)?.response?.status;
    if (status && status >= 400 && status < 500) pendingSave.value = undefined;
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.card.save_failed"));
  } finally {
    saving.value = false;
  }
}

/** 关闭弹窗后清空表单、预览与校验状态。 */
function reset() {
  previewRequest++;
  pendingSave.value = undefined;
  baseQuery.value = undefined;
  Object.assign(form, defaultForm());
  previewResult.value = undefined;
  previewRecordID.value = 0;
  originalMetrics.value = [];
  editedMetrics.value = [];
  impact.value = undefined;
  sql.value = "";
  sourceRecordID.value = 0;
  chartConfig.value = { encoding: {}, style: { showLegend: true, showLabel: false } };
  nameFormRef.value?.clearValidate();
  queryFormRef.value?.clearValidate();
}
</script>

<style scoped>
.bi-card-editor {
  display: grid;
  grid-template-columns: minmax(0, 60fr) minmax(0, 40fr);
  grid-template-rows: minmax(0, 1fr);
  width: 100%;
}
.bi-card-editor.is-narrow { grid-template-columns: minmax(0, 1fr); }
.bi-card-editor.is-narrow > .bi-card-editor__preview { display: none; }
.bi-card-editor__settings, .bi-card-editor__tabs {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
}
.bi-card-editor__settings { padding: 12px 16px 0; }
.bi-card-editor__settings :deep(.el-form-item) { margin-bottom: 12px; }
.bi-card-editor__settings :deep(.el-form-item--label-top .el-form-item__label) { margin-bottom: 4px; }
.bi-card-editor__tabs { flex: 1; }
.bi-card-editor__tabs :deep(.el-tabs__header) { flex: none; margin: 0 0 12px; }
.bi-card-editor__tabs :deep(.el-tabs__item) { padding-inline: 10px; }
.bi-card-editor__tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  padding: 0 1px 12px;
  overflow: auto;
  overscroll-behavior: contain;
}
/* 给查询表单 12px 栅格间距两侧的负边距留出空间。 */
.bi-card-editor__query { padding-inline: 6px; }
.bi-card-editor__query :deep(.el-col-12) { flex-basis: 50%; max-width: 50%; }
.bi-card-editor__preview-pane { height: 100%; }
.bi-card-editor__preview {
  display: flex;
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  gap: 10px;
  padding: 12px 16px;
  border-inline-start: 1px solid var(--el-border-color-lighter);
  background: var(--admin-page-card-bg-soft);
}
.bi-card-editor__preview-pane .bi-card-editor__preview { padding: 0; border: 0; background: transparent; }
.bi-card-editor__preview :deep(.bi-data-card) { flex: 1; min-height: 0; }
.bi-card-editor__preview :deep(.bi-data-card__header) { flex: none; }
.bi-card-editor__preview :deep(.bi-data-card__chart) { flex: 1; height: auto; min-height: 0; }
.bi-card-editor__preview > .el-empty { flex: 1; min-height: 0; }
.bi-card-editor__footer { display: grid; gap: 8px; }
.bi-card-editor__actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
.bi-card-editor__actions :deep(.el-button) { margin: 0; }
.bi-card-query-summary { display: flex; flex-wrap: wrap; gap: 4px 12px; }
.bi-card-query-summary :deep(.el-text) { overflow-wrap: anywhere; }
.bi-card-formulas { display: flex; min-width: 0; flex-direction: column; gap: 10px; }
.bi-card-formulas__metric { display: flex; min-width: 0; flex-direction: column; gap: 6px; }
.bi-card-formulas__metric code { overflow-wrap: anywhere; color: var(--el-text-color-regular); }
.bi-card-formulas :deep(.code-editor) { min-height: 0; }

.bi-card-measures {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 8px;
}
.bi-card-measures__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
}
@media (max-width: 680px) {
  .bi-card-editor__settings { padding-inline: 12px; }
  .bi-card-editor__tabs :deep(.el-tabs__item) { padding-inline: 6px; }
}
</style>

<style lang="scss">
.el-dialog.pro-dialog.bi-card-editor-dialog {
  display: flex;
  flex-direction: column;
  height: min(720px, calc(100dvh - 24px));
  max-width: calc(100vw - 24px);

  .el-dialog__header,
  .el-dialog__footer { flex: none; }

  .el-dialog__body {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    padding: 0;
  }

  .el-dialog__footer {
    padding: 10px 12px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}
</style>
