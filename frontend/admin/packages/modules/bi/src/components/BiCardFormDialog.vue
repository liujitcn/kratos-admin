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
    :confirm-loading="saving"
    :close-on-click-modal="!saving"
    :form-props="{ disabled: saving }"
    @update:model-value="emit('update:modelValue', $event)"
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
          <el-select disabled v-model="measure.aggregation" :aria-label="t('bi.card.aggregation')">
            <el-option
              v-for="aggregation in source?.measures.find(item => item.code === measure.code)?.aggregations ?? []"
              :key="aggregation"
              :label="t(`bi.card.aggregation.${aggregation}`)"
              :value="aggregation"
            />
          </el-select>
        </div>
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
      <el-button :disabled="saving" @click="emit('update:modelValue', false)">{{ t("common.action.cancel") }}</el-button>
      <el-button :loading="previewing" :disabled="saving" @click="preview">{{ t("bi.card.preview") }}</el-button>
      <el-button type="primary" :loading="saving" @click="submit">{{ t("bi.card.save") }}</el-button>
    </template>
  </FormDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiMeasureSpec, BiQueryResult, BiQuerySpec, BiSourceDescriptor } from "../rpc/bi/admin/v1/bi_query";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import { defBiQueryService } from "../api/bi/admin/v1/bi_query";
import BiDataCard from "./BiDataCard.vue";
import BiCardChartEditor from "./BiCardChartEditor.vue";
import { parseCardChart } from "../card-schema";
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
const previewing = ref(false);
let previewRequest = 0;
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
    if (!visible) return;
    const current = props.card?.query ?? props.initialQuery;
    baseQuery.value = current ? (JSON.parse(JSON.stringify(current)) as BiQuerySpec) : undefined;
    Object.assign(form, defaultForm(), {
      name: props.card?.name ?? props.initialTitle ?? "",
      source_code: current?.source_code || props.sources[0]?.code || "",
      dimensions: [...(current?.dimensions ?? [])],
      measures: (current?.measures ?? []).map(item => ({ ...item })),
      chart_type: props.card?.chart_type || (current && !current.dimensions.length ? "metric" : props.initialChartType || "bar"),
      refresh_seconds: props.card?.refresh_seconds ?? 0,
      is_shared: props.card?.is_shared ?? false
    });
    previewResult.value = props.initialResult;
    sourceRecordID.value = props.card?.query_record_id || props.initialResult?.query_record_id || 0;
    chartConfig.value = parseCardChart(props.card ?? { chart_json: "" });
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

/** 预览当前语义配置，忽略配置变化前的迟到响应。 */
async function preview() {
  if (!(await dialogRef.value?.validate()) || previewing.value) return;
  if (!sourceRecordID.value) {
    ElMessage.warning(t("bi.card.ask_required"));
    return;
  }
  const request = ++previewRequest;
  previewing.value = true;
  try {
    const result = await defBiQueryService.PreviewBiQuery({ query: query.value });
    if (request === previewRequest) {
      previewResult.value = result;
    }
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    previewing.value = false;
  }
}

/** 仅保存与原始智能问数关联的卡片，图表编辑不改变查询定义。 */
async function submit() {
  if (!(await dialogRef.value?.validate()) || saving.value || previewing.value) return;
  if (!sourceRecordID.value) {
    ElMessage.warning(t("bi.card.ask_required"));
    return;
  }
  if (!previewResult.value) await preview();
  if (!previewResult.value || !sourceRecordID.value) return;
  saving.value = true;
  try {
    const saved = await defBiCardService.SaveBiCard({
      id: props.card?.id ?? 0,
      card: {
        name: form.name,
        query: query.value,
        chart_type: form.chart_type,
        chart_json: JSON.stringify(chartConfig.value),
        query_record_id: sourceRecordID.value,
        refresh_seconds: form.refresh_seconds,
        is_shared: form.is_shared
      }
    });
    emit("saved", saved);
    emit("update:modelValue", false);
    ElMessage.success(t("bi.card.saved"));
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.card.save_failed"));
  } finally {
    saving.value = false;
  }
}

/** 关闭弹窗后清空表单、预览与校验状态。 */
function reset() {
  previewRequest++;
  baseQuery.value = undefined;
  Object.assign(form, defaultForm());
  previewResult.value = undefined;
  sourceRecordID.value = 0;
  chartConfig.value = { encoding: {}, style: { showLegend: true, showLabel: false } };
  dialogRef.value?.clearValidate();
}
</script>

<style scoped>
.bi-card-measures {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 8px;
}
.bi-card-measures__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(90px, 0.65fr) auto;
  gap: 8px;
}
@media (max-width: 480px) {
  .bi-card-measures__row {
    grid-template-columns: minmax(0, 1fr) 90px auto;
  }
}
</style>
