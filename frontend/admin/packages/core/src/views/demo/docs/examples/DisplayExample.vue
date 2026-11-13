<template>
  <div v-if="active">
    <template v-if="componentId === 'ECharts'">
      <el-radio-group v-model="chartType" size="small">
        <el-radio-button value="bar">柱状图</el-radio-button>
        <el-radio-button value="line">折线图</el-radio-button>
      </el-radio-group>
      <div class="chart-example">
        <ECharts
          :key="globalStore.isDark ? 'dark' : 'light'"
          :theme="globalStore.isDark ? 'dark' : undefined"
          :option="option"
          :on-click="event => (selected = `${event.name}：${event.value}`)"
        />
      </div>
      <p class="example-tip" role="status">{{ selected || "点击图表数据查看回调结果。" }}</p>
    </template>
    <DataPanelCard v-else-if="componentId === 'DataPanelCard'" title="本月访问" description="最近四周的本地示例数据" primary>
      <template #extra><el-button size="small" @click="visits += 120">模拟新增</el-button></template>
      <div class="data-number">{{ visits.toLocaleString() }}<span>次访问</span></div>
      <div class="card-chart">
        <ECharts :key="globalStore.isDark ? 'dark' : 'light'" :theme="globalStore.isDark ? 'dark' : undefined" :option="option" />
      </div>
    </DataPanelCard>
    <template v-else>
      <el-switch v-model="collapsed" active-text="折叠为一行" />
      <Grid class="grid-example" :cols="{ xs: 1, sm: 2, md: 2, lg: 3, xl: 3 }" :gap="16" :collapsed="collapsed">
        <GridItem v-for="(item, index) in gridItems" :key="item.title" :index="index" :span="item.span" :xs="{ span: 1 }">
          <div class="grid-cell">
            <strong>{{ item.title }}</strong
            ><span>{{ item.description }}</span>
          </div>
        </GridItem>
      </Grid>
      <p class="example-tip">调整窗口宽度可观察响应式列数，最后一项跨两列。</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from "vue";
import DataPanelCard from "@/components/Card/DataPanelCard.vue";
import Grid from "@/components/Grid/index.vue";
import GridItem from "@/components/Grid/components/GridItem.vue";
import type { ECOption } from "@/components/ECharts/config";
import { useGlobalStore } from "@/stores/modules/global";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const ECharts = defineAsyncComponent(() => import("@/components/ECharts/index.vue"));
const globalStore = useGlobalStore();
const chartType = ref<"bar" | "line">("bar");
const selected = ref("");
const visits = ref(12860);
const collapsed = ref(false);
const gridItems = [
  { title: "项目概览", description: "占一列", span: 1 },
  { title: "待办事项", description: "占一列", span: 1 },
  { title: "运行状态", description: "占一列", span: 1 },
  { title: "最近活动", description: "占两列，窄屏时占一列", span: 2 }
];
const option = computed<ECOption>(() => ({
  backgroundColor: "transparent",
  tooltip: { trigger: "axis" },
  grid: { top: 25, right: 20, bottom: 30, left: 45 },
  xAxis: { type: "category", data: ["第一周", "第二周", "第三周", "第四周"] },
  yAxis: { type: "value" },
  series: [{ type: chartType.value, data: [1860, 2450, 3200, 5350], barMaxWidth: 42 }]
}));
</script>

<style scoped lang="scss">
.chart-example {
  height: 280px;
  margin-top: 16px;
}
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.data-number {
  display: flex;
  gap: 12px;
  align-items: baseline;
  padding: 12px 0 0;
  font-size: 34px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.data-number span {
  font-size: 13px;
  font-weight: 400;
  color: var(--admin-page-text-secondary);
}
.card-chart {
  height: 280px;
}
.grid-example {
  margin-top: 20px;
}
.grid-cell {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 96px;
  padding: 20px;
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg-muted);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.grid-cell span {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
</style>
