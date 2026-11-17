<template>
  <div class="bi-card-chart-editor">
    <div class="bi-card-chart-editor__types">
      <el-button
        v-for="type in types"
        :key="type"
        :type="type === chartType ? 'primary' : 'default'"
        :plain="type !== chartType"
        :disabled="Boolean(reason(type))"
        :title="reason(type)"
        @click="changeType(type)"
        >{{ ["metric", "metric_grid"].includes(type) ? t(`bi.card.chart.${type}`) : messages(`types.${type}`) }}</el-button
      >
    </div>
    <p class="bi-card-chart-editor__hint">{{ t("bi.card.chart_hint") }}</p>
    <el-form label-position="top">
      <el-form-item v-if="!['metric', 'metric_grid', 'gauge'].includes(chartType)" :label="t('bi.card.dimension_field')">
        <el-select :model-value="modelValue.encoding.dimension || dimensionFields[0]?.code" @change="patchEncoding({ dimension: $event })"
          ><el-option v-for="field in dimensionFields" :key="field.code" :value="field.code" :label="field.name"
        /></el-select>
      </el-form-item>
      <el-form-item v-if="chartType !== 'metric_grid'" :label="t('bi.card.value_field')"
        ><el-select :model-value="modelValue.encoding.measure || numbers[chartType === 'scatter' ? 1 : 0]?.code" @change="patchEncoding({ measure: $event })"
          ><el-option v-for="field in numbers" :key="field.code" :value="field.code" :label="field.name" /></el-select
      ></el-form-item>
      <div v-if="chartType === 'metric_grid'" class="bi-card-chart-editor__metric-grid">
        <div v-for="(item, index) in metricGridItems" :key="index" class="bi-card-chart-editor__metric-grid-row">
          <el-select
            :model-value="item.value"
            :aria-label="t('bi.card.metric_grid.current')"
            @change="patchMetricGridItem(index, { value: $event })"
          >
            <el-option v-for="field in numbers" :key="field.code" :value="field.code" :label="field.name" />
          </el-select>
          <el-select
            :model-value="item.compare"
            clearable
            :placeholder="t('bi.card.metric_grid.compare')"
            :aria-label="t('bi.card.metric_grid.compare')"
            @change="patchMetricGridItem(index, { compare: $event || undefined })"
          >
            <el-option v-for="field in numbers" :key="field.code" :value="field.code" :label="field.name" :disabled="field.code === item.value" />
          </el-select>
          <el-input
            :model-value="item.compareLabel"
            :placeholder="t('bi.card.metric_grid.compare_default')"
            :aria-label="t('bi.card.metric_grid.compare_label')"
            @update:model-value="patchMetricGridItem(index, { compareLabel: $event || undefined })"
          />
        </div>
      </div>
      <el-form-item v-if="['sankey', 'heatmap', 'sunburst'].includes(chartType)" :label="t('bi.card.series_field')"
        ><el-select :model-value="modelValue.encoding.series || categories[1]?.code" @change="patchEncoding({ series: $event })"
          ><el-option v-for="field in categories" :key="field.code" :value="field.code" :label="field.name" /></el-select
      ></el-form-item>
      <el-form-item v-if="['bar', 'line', 'area'].includes(chartType)" :label="t('bi.card.measure')"
        ><el-checkbox-group
          :model-value="modelValue.encoding.measures || []"
          @change="patchEncoding({ measures: $event as string[] })"
          ><el-checkbox v-for="field in numbers" :key="field.code" :value="field.code">{{
            field.name
          }}</el-checkbox></el-checkbox-group
        ></el-form-item
      >
      <div v-if="chartType !== 'metric_grid'" class="bi-card-chart-editor__style">
        <el-checkbox :model-value="modelValue.style.showLegend" @change="patchStyle({ showLegend: Boolean($event) })">{{
          t("bi.card.show_legend")
        }}</el-checkbox
        ><el-checkbox :model-value="modelValue.style.showLabel" @change="patchStyle({ showLabel: Boolean($event) })">{{
          t("bi.card.show_label")
        }}</el-checkbox
        ><el-checkbox :model-value="modelValue.style.stack" @change="patchStyle({ stack: Boolean($event) })">{{
          t("bi.card.stack")
        }}</el-checkbox
        ><el-checkbox :model-value="modelValue.style.dataZoom" @change="patchStyle({ dataZoom: Boolean($event) })">{{
          t("bi.card.data_zoom")
        }}</el-checkbox>
      </div>
    </el-form>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiQueryResult, BiQuerySpec } from "../rpc/bi/admin/v1/bi_query";
import type { CardChartConfig, MetricGridItem } from "../card-schema";
import type { BiChartEncoding } from "../schema/types";
import { getChartTypes } from "../engine/chart-types";
import { useBiI18n } from "../env";
const props = defineProps<{ modelValue: CardChartConfig; chartType: string; query: BiQuerySpec; result?: BiQueryResult }>();
const emit = defineEmits<{ "update:modelValue": [value: CardChartConfig]; "update:chartType": [value: string] }>();
const { t: messages } = useBiI18n();
const types = [...getChartTypes(), "metric", "metric_grid"];
const numbers = computed(
  () =>
    props.result?.columns.filter(field => field.type === "number") ??
    props.query.measures.map(field => ({ code: field.code, name: field.code, type: "number" }))
);
const categories = computed(() =>
  props.query.dimensions.map(
    code => props.result?.columns.find(field => field.code === code) ?? { code, name: code, type: "string" }
  )
);
const dimensionFields = computed(() => (props.chartType === "scatter" ? numbers.value : categories.value));
const metricGridItems = computed<MetricGridItem[]>(() =>
  props.modelValue.metricGrid?.items?.length
    ? props.modelValue.metricGrid.items.slice(0, 4)
    : numbers.value.slice(0, 4).map(field => ({ value: field.code }))
);
/** 根据真实结果结构启用图型，避免产生不能渲染的卡片。 */
function reason(type: string): string {
  if (["metric", "metric_grid", "gauge"].includes(type)) return props.query.dimensions.length ? t("bi.card.need_ungrouped") : "";
  if (type === "scatter") return numbers.value.length < 2 ? t("bi.card.need_two_numbers") : "";
  if (["sankey", "heatmap", "sunburst"].includes(type))
    return categories.value.length < 2 ? t("bi.card.need_two_dimensions") : "";
  return categories.value.length ? "" : t("bi.card.need_dimension");
}
/** 切换图型时复用当前查询结果，并补齐该图型的默认字段映射。 */
function changeType(type: string) {
  emit("update:chartType", type);
  const metricGrid = type === "metric_grid" ? { items: numbers.value.slice(0, 4).map(field => ({ value: field.code })) } : undefined;
  emit("update:modelValue", {
    encoding: {
      dimension: type === "scatter" ? numbers.value[0]?.code : categories.value[0]?.code,
      measure: type === "scatter" ? numbers.value[1]?.code : numbers.value[0]?.code,
      series: type === "scatter" ? undefined : categories.value[1]?.code,
      measures:
        ["bar", "line", "area"].includes(type) && categories.value.length < 2 ? numbers.value.map(field => field.code) : undefined
    },
    style: { ...props.modelValue.style },
    metricGrid
  });
}
/** 同步字段映射，不重新执行已完成的查询。 */
function patchEncoding(value: Partial<BiChartEncoding>) {
  emit("update:modelValue", { ...props.modelValue, encoding: { ...props.modelValue.encoding, ...value } });
}
/** 同步指标网格的当前列、比较列和比较文案。 */
function patchMetricGridItem(index: number, value: Partial<NonNullable<CardChartConfig["metricGrid"]>["items"][number]>) {
  const items = [...metricGridItems.value];
  if (!items[index]) return;
  items[index] = { ...items[index], ...value };
  emit("update:modelValue", { ...props.modelValue, metricGrid: { items } });
}
/** 同步受控样式选项。 */
function patchStyle(value: Partial<CardChartConfig["style"]>) {
  emit("update:modelValue", { ...props.modelValue, style: { ...props.modelValue.style, ...value } });
}
</script>
<style scoped>
.bi-card-chart-editor__types {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.bi-card-chart-editor__types :deep(.el-button) {
  margin: 0;
  padding: 8px;
  min-width: 0;
}
.bi-card-chart-editor__hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin: 10px 0 16px;
}
.bi-card-chart-editor__style {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
}
.bi-card-chart-editor__metric-grid {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 8px;
}
.bi-card-chart-editor__metric-grid-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bi-card-chart-editor :deep(.el-select) {
  width: 100%;
}
.bi-card-chart-editor__metric-grid-row :deep(.el-input) {
  min-width: 0;
}
@media (max-width: 600px) {
  .bi-card-chart-editor__types {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .bi-card-chart-editor__metric-grid-row {
    grid-template-columns: 1fr;
  }
}
</style>
