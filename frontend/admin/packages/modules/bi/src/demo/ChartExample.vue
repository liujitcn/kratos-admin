<template>
  <div v-if="active" class="chart-example">
    <template v-if="componentId === 'BiCardChartEditor'">
      <div class="chart-example__editor">
        <BiCardChartEditor v-model="config" v-model:chart-type="chartType" :query="demoQuery" :result="demoResult" />
        <div class="chart-example__plot"><BiCardVisualization :card="card" :result="demoResult" /></div>
      </div>
      <p>切换图型或字段映射可立即预览；图型可用性由本地结果的字段类型决定。</p>
    </template>
    <template v-else>
      <div class="chart-example__toolbar">
        <el-radio-group v-model="chartType" aria-label="演示图表类型">
          <el-radio-button value="bar">柱状图</el-radio-button>
          <el-radio-button value="line">折线图</el-radio-button>
          <el-radio-button value="pie">饼图</el-radio-button>
        </el-radio-group>
        <el-checkbox v-if="componentId === 'BiChart'" v-model="empty">空数据</el-checkbox>
        <el-checkbox v-if="componentId === 'BiCardVisualization'" v-model="compact">紧凑展示</el-checkbox>
      </div>
      <BiDataCard v-if="componentId === 'BiDataCard'" :card="card" :preview-result="demoResult" source-name="本地服务数据">
        <template #actions><el-tag type="info" effect="plain">本地预览</el-tag></template>
        <template #footer><span>6 条演示记录</span><span>自动刷新已关闭</span></template>
      </BiDataCard>
      <div v-else class="chart-example__plot">
        <BiChart
          v-if="componentId === 'BiChart'"
          :chart="chartData.chart"
          :rows="empty ? [] : chartData.rows"
          :theme="theme"
          empty-text="暂无演示数据"
          @chart-click="handleChartClick"
        />
        <BiCardVisualization v-else :card="card" :result="demoResult" :compact="compact" />
      </div>
      <div v-if="componentId === 'BiCardVisualization'" class="chart-example__metric">
        <BiCardVisualization :card="demoMetricCard" :result="demoMetricResult" />
      </div>
      <p aria-live="polite">{{ feedback }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref } from "vue";
import BiChart from "../components/BiChart.vue";
import BiCardVisualization from "../components/BiCardVisualization.vue";
import BiDataCard from "../components/BiDataCard.vue";
import BiCardChartEditor from "../components/BiCardChartEditor.vue";
import { buildCardChart } from "../card-schema";
import type { CardChartConfig } from "../card-schema";
import type { BiChartEvent } from "../bi-core";
import { useBiTheme } from "../env";
import { demoCard, demoChartConfig, demoMetricCard, demoMetricResult, demoQuery, demoResult } from "./fixtures";

defineProps<{ componentId: string }>();
const active = ref(true);
const chartType = ref("bar");
const empty = ref(false);
const compact = ref(false);
const feedback = ref("当前展示本地演示数据。");
const config = ref<CardChartConfig>({ encoding: { ...demoChartConfig.encoding }, style: { ...demoChartConfig.style } });
const theme = useBiTheme();
const card = computed(() => ({ ...demoCard, chart_type: chartType.value, chart_json: JSON.stringify(config.value) }));
const chartData = computed(() => buildCardChart(card.value, demoResult));

/** 显示图表事件，保持交互结果在当前页面。 */
function handleChartClick(event: BiChartEvent) {
  feedback.value = `点击图表：${JSON.stringify(event)}`;
}

onActivated(() => {
  active.value = true;
});
// 释放缓存页面内的图表引擎及其观察器。
onDeactivated(() => {
  active.value = false;
});
</script>

<style scoped>
.chart-example {
  display: grid;
  gap: 16px;
  min-width: 0;
}
.chart-example__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 20px;
  align-items: center;
}
.chart-example__plot {
  min-width: 0;
  height: 300px;
}
.chart-example__editor {
  display: grid;
  grid-template-columns: minmax(250px, 1fr) minmax(0, 1fr);
  gap: 24px;
}
.chart-example__metric {
  min-height: 100px;
  padding: 16px;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}
.chart-example p,
.chart-example :deep(.bi-data-card__footer) {
  margin: 0;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}
@media (max-width: 900px) {
  .chart-example__editor {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
