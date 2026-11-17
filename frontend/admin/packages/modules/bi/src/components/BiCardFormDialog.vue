<template>
  <FormDialog
    ref="dialogRef"
    :model-value="modelValue"
    :title="card ? t('bi.card.edit') : t('bi.card.create')"
    width="min(1040px, 94vw)"
    label-width="7em"
    :model="form"
    :fields="fields"
    :rules="rules"
    :confirm-loading="saving || loadingDetails"
    :close-on-click-modal="!saving && !pendingSave"
    :close-on-press-escape="!saving && !pendingSave"
    :show-close="!saving && !pendingSave"
    :form-props="{ disabled: saving || loadingDetails || Boolean(pendingSave) }"
    @update:model-value="value => { if (!saving && !pendingSave) emit('update:modelValue', value); }"
    @confirm="submit"
    :destroy-on-close="true"
    @closed="reset"
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
    <template #formulas>
      <div v-loading="loadingDetails" class="bi-card-formulas">
        <el-alert :title="t(canEditMetrics ? 'bi.sql.shared_notice' : 'bi.sql.readonly_notice')" type="info" show-icon :closable="false" />
        <el-alert v-if="pendingSave && !saving" :title="t('bi.sql.save_pending')" type="warning" show-icon :closable="false" />
        <div v-for="metric in editedMetrics" :key="metric.code" class="bi-card-formulas__metric">
          <strong>{{ metric.name }} <small>{{ metric.unit }}</small></strong>
          <CodeEditor v-model="metric.expression" language="sql" height="120px" :readonly="!canEditMetrics || Boolean(pendingSave)" />
        </div>
        <template v-if="sql">
          <strong>{{ t('bi.sql.full_sql') }}</strong>
          <CodeEditor :model-value="sql" language="sql" height="220px" readonly />
        </template>
        <el-alert v-if="impact?.changes?.length" :title="t('bi.sql.impact', { cards: impact.affected_cards, dashboards: impact.affected_dashboards })" type="warning" show-icon :closable="false" />
        <div v-for="change in impact?.changes ?? []" :key="change.after?.code ?? change.before?.code" class="bi-card-formulas__diff">
          <strong>{{ change.after?.name ?? change.before?.name }}</strong>
          <code>{{ change.before?.expression }} → {{ change.after?.expression }}</code>
        </div>
        <el-button v-if="card" :disabled="saving || previewing || Boolean(pendingSave)" @click="loadDetails(true)">{{ t('bi.sql.refresh_model') }}</el-button>
      </div>
    </template>
    <template #chart>
      <BiCardChartEditor v-model="chartConfig" v-model:chart-type="form.chart_type" :query="query" :result="previewResult" />
    </template>
    <template #preview>
      <BiDataCard v-if="previewResult" :card="previewCard" :preview-result="previewResult" :source-name="source?.name" />
      <el-empty v-else :image-size="36" :description="t('bi.card.preview_empty')" />
    </template>
    <template #footer>
      <el-button :disabled="saving || Boolean(pendingSave)" @click="emit('update:modelValue', false)">{{ t("common.action.cancel") }}</el-button>
      <el-button :loading="previewing" :disabled="saving || loadingDetails || Boolean(pendingSave)" @click="preview">{{ t("bi.card.preview") }}</el-button>
      <el-button type="primary" :loading="saving" :disabled="loadingDetails" @click="submit">{{ t("bi.card.save") }}</el-button>
    </template>
  </FormDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import CodeEditor from "@liujitcn/kratos-admin-core/components/CodeEditor/index.vue";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BiCard, SaveBiCardRequest } from "../rpc/bi/admin/v1/bi_card";
import type { BiMeasureSpec, BiQueryResult, BiQuerySpec, BiSourceDescriptor } from "../rpc/bi/admin/v1/bi_query";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import type { BiModelImpact, BiSourceMeasure } from "../rpc/bi/admin/v1/bi_source";
import { metricChanges, mergeMetricDrafts, appliedCardID } from "../metric-edit";
import BiDataCard from "./BiDataCard.vue";
import BiCardChartEditor from "./BiCardChartEditor.vue";
import { parseCardChart } from "../card-schema";
import { normalizeBiQuerySpec } from "../bi-response";
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
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const form = reactive<BiCardEditState>(defaultForm());
const baseQuery = ref<BiQuerySpec>();
const previewResult = ref<BiQueryResult>();
const sourceRecordID = ref(0);
const chartConfig = ref<CardChartConfig>({ encoding: {}, style: { showLegend: true, showLabel: false } });
const saving = ref(false);
const pendingSave = ref<SaveBiCardRequest>();
const previewing = ref(false);
const loadingDetails = ref(false);
const canEditMetrics = ref(false);
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
const source = computed(() => props.sources.find(item => item.code === form.source_code));
const query = computed<BiQuerySpec>(() => ({
  source_code: form.source_code,
  dimensions: [...form.dimensions],
  measures: form.measures.map(item => ({ ...item })),
  filters: baseQuery.value?.filters ?? [],
  limit: baseQuery.value?.limit || 100,
  time_range: baseQuery.value?.time_range,
  orders: baseQuery.value?.orders ?? []
}));
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
const fields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("bi.card.name"), component: "input", colSpan: 12, props: { maxlength: 128 } },
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
  { prop: "formulas", label: t("bi.sql.formulas"), component: "slot", slotName: "formulas", colSpan: 24 },
  { prop: "chart", label: t("bi.card.chart_type"), component: "slot", slotName: "chart", colSpan: 12 },
  { prop: "preview", label: t("bi.card.preview"), component: "slot", slotName: "preview", colSpan: 12 },
  {
    prop: "refresh_seconds",
    label: t("bi.card.refresh"),
    component: "select",
    colSpan: 12,
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
    saving.value = false;
    previewing.value = false;
    loadingDetails.value = false;
    const original = props.card?.query ?? props.initialQuery;
    const current = original ? normalizeBiQuerySpec(original) : undefined;
    baseQuery.value = current ? (JSON.parse(JSON.stringify(current)) as BiQuerySpec) : undefined;
    Object.assign(form, defaultForm(), {
      name: props.card?.name ?? props.initialTitle ?? "",
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

/** 真实预览精确公共公式候选，并忽略编辑前的迟到响应。 */
async function preview() {
  if (previewing.value || loadingDetails.value || pendingSave.value) return;
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
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (currentSession === session) previewing.value = false;
  }
}

/** 保存精确候选，结果未知时重试冻结请求，避免新预览绕过一次消费。 */
async function submit() {
  if ((!pendingSave.value && !(await dialogRef.value?.validate())) || saving.value || previewing.value || loadingDetails.value) return;
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
    ElMessage.success(saved.sync_summary?.changes?.length ? t("bi.sql.synced", { cards: saved.sync_summary.affected_cards, dashboards: saved.sync_summary.affected_dashboards }) : t("bi.card.saved"));
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
  dialogRef.value?.clearValidate();
}
</script>

<style scoped>
.bi-card-formulas { display: flex; width: 100%; min-width: 0; flex-direction: column; gap: 12px; }
.bi-card-formulas__metric, .bi-card-formulas__diff { display: flex; min-width: 0; flex-direction: column; gap: 6px; }
.bi-card-formulas__diff code { overflow-wrap: anywhere; color: var(--el-text-color-regular); }

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
@media (max-width: 480px) {
  .bi-card-measures__row {
    grid-template-columns: minmax(0, 1fr) auto;
  }
}
</style>
