<template>
  <div v-if="data.metricOnly" class="bi-card-metrics" :class="{ 'is-multiple': data.metrics.length > 1, 'is-compact': compact }">
    <div v-for="metric in data.metrics" :key="metric.code" class="bi-card-metrics__item">
      <span v-if="data.metrics.length > 1">{{ metric.name }}</span
      ><strong>{{ formatCardMetric(metric.value) }}</strong>
    </div>
  </div>
  <div v-else class="bi-card-plot">
    <BiChart :chart="data.chart" :rows="data.rows" :theme="theme" :empty-text="t('common.message.no_data')" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import { buildCardChart, formatCardMetric } from "../card-schema";
import { useBiTheme } from "../env";
import BiChart from "./BiChart.vue";

const props = defineProps<{ card: BiCard; result: BiQueryResult; compact?: boolean }>();
const theme = useBiTheme();
const data = computed(() => {
  const value = buildCardChart(props.card, props.result);
  if (props.compact) {
    value.chart.style = { ...value.chart.style, showLegend: false, showLabel: false, dataZoom: false };
    value.chart.options = { tooltip: { show: false }, visualMap: { show: false } };
    if (["bar", "line", "area", "scatter", "heatmap"].includes(props.card.chart_type)) {
      value.chart.options.grid = { left: 4, right: 4, top: 6, bottom: 4 };
      value.chart.options.xAxis = { type: props.card.chart_type === "scatter" ? "value" : "category", show: false };
      value.chart.options.yAxis = { type: props.card.chart_type === "heatmap" ? "category" : "value", show: false };
    }
  }
  return value;
});
</script>

<style scoped>
.bi-card-metrics {
  display: grid;
  height: 100%;
  align-content: center;
  gap: 16px;
  overflow: auto;
  color: var(--el-text-color-primary);
}
.bi-card-metrics__item {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
  text-align: center;
  overflow-wrap: anywhere;
}
.bi-card-metrics__item strong {
  font-size: 32px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}
.bi-card-metrics__item span,
.bi-card-gauge__details span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.bi-card-metrics:not(.is-multiple) {
  align-content: start;
  padding-top: 4px;
}
.bi-card-metrics:not(.is-multiple) .bi-card-metrics__item {
  text-align: left;
}
.bi-card-metrics.is-multiple {
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(60px, 1fr));
}
.bi-card-metrics.is-multiple strong {
  font-size: 22px;
}
.bi-card-metrics.is-compact .bi-card-metrics__item strong {
  font-size: 20px;
}
.bi-card-plot {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 4px;
}
.bi-card-plot :deep(.bi-chart) {
  flex: 1;
  min-height: 0;
  height: auto;
}
</style>
