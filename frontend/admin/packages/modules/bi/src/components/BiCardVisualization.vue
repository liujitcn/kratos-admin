<template>
  <div v-if="data.metricOnly && card.chart_type === 'metric_grid'" class="bi-card-metric-grid" :class="{ 'is-compact': compact }">
    <div v-for="metric in data.metrics" :key="metric.code" class="bi-card-metric-grid__item">
      <span class="bi-card-metric-grid__name">{{ metric.name }}</span>
      <div class="bi-card-metric-grid__value">
        <el-text tag="strong">{{ formatCardMetric(metric.value) }}</el-text>
        <el-text tag="small" v-if="metric.unit">{{ metric.unit }}</el-text>
      </div>
      <div v-if="metric.delta && metric.trend" class="bi-card-metric-grid__trend" :class="`is-${metric.trend}`">
        <el-icon><component :is="trendIcons[metric.trend]" /></el-icon>
        <span>{{ metric.delta }}% {{ metric.compareLabel === "较昨日" ? t("bi.card.metric_grid.compare_default") : metric.compareLabel }}</span>
      </div>
    </div>
  </div>
  <div v-else-if="data.metricOnly" class="bi-card-metrics" :class="{ 'is-multiple': data.metrics.length > 1, 'is-compact': compact }">
    <div v-for="metric in data.metrics" :key="metric.code" class="bi-card-metrics__item">
      <span v-if="data.metrics.length > 1">{{ metric.name }}</span
      ><el-text tag="strong">{{ formatCardMetric(metric.value) }}</el-text>
    </div>
  </div>
  <div v-else class="bi-card-plot">
    <BiChart :chart="data.chart" :rows="data.rows" :theme="theme" :empty-text="t('common.message.no_data')" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { ArrowDown, ArrowUp, Minus } from "@element-plus/icons-vue";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import { buildCardChart, formatCardMetric } from "../card-schema";
import { useBiTheme } from "../env";
import BiChart from "./BiChart.vue";

const props = defineProps<{ card: BiCard; result: BiQueryResult; compact?: boolean }>();
const theme = useBiTheme();
const trendIcons = { up: ArrowUp, down: ArrowDown, flat: Minus };
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
.bi-card-metric-grid {
  display: grid;
  height: 100%;
  min-height: 0;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-content: stretch;
  color: var(--el-text-color-primary);
}
.bi-card-metric-grid__item {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  padding: 12px 28px;
  overflow-wrap: anywhere;
}
.bi-card-metric-grid__item:nth-child(odd):not(:last-child) {
  border-right: 1px solid var(--el-border-color-lighter);
}
.bi-card-metric-grid__item:nth-child(n + 3) {
  border-top: 1px solid var(--el-border-color-lighter);
}
.bi-card-metric-grid__name {
  color: var(--el-text-color-secondary);
  font-size: 16px;
  line-height: 1.3;
}
.bi-card-metric-grid__value {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 10px;
}
.bi-card-metric-grid__value strong {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 36px;
  font-weight: 600;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}
.bi-card-metric-grid__value small {
  color: var(--el-text-color-secondary);
  font-size: 16px;
}
.bi-card-metric-grid__trend {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 16px;
  line-height: 1.3;
}
.bi-card-metric-grid__trend.is-up {
  color: var(--el-color-danger);
}
.bi-card-metric-grid__trend.is-down {
  color: var(--el-color-success);
}
.bi-card-metric-grid__trend.is-flat {
  color: var(--el-text-color-secondary);
}
.bi-card-metric-grid.is-compact .bi-card-metric-grid__item {
  gap: 4px;
  padding: 6px 10px;
}
.bi-card-metric-grid.is-compact .bi-card-metric-grid__name,
.bi-card-metric-grid.is-compact .bi-card-metric-grid__trend {
  font-size: 11px;
}
.bi-card-metric-grid.is-compact .bi-card-metric-grid__value {
  gap: 4px;
}
.bi-card-metric-grid.is-compact .bi-card-metric-grid__value strong {
  font-size: 20px;
}
.bi-card-metric-grid.is-compact .bi-card-metric-grid__value small {
  font-size: 11px;
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
