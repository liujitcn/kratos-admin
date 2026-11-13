<template>
  <div class="bi-chart">
    <div ref="hostRef" class="bi-chart__host" :style="{ visibility: ready ? 'visible' : 'hidden' }" />
    <div v-if="error" class="bi-chart__error" :title="error">!</div>
    <div v-else-if="!rows.length" class="bi-chart__empty">{{ emptyText }}</div>
  </div>
</template>

<script setup lang="ts">
/**
 * 单图表渲染：引擎挂载 + 行数据/主题变化重编译 + ResizeObserver 跟随。
 * 空数据不挂引擎，只显示占位文案。
 */
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { getEngine, transformRows } from "../bi-core";
import type { BiChartEvent, BiChartSchema, BiResolvedTheme, ChartHandle } from "../bi-core";

const props = withDefaults(
  defineProps<{
    chart: BiChartSchema;
    rows: any[];
    theme: BiResolvedTheme;
    loading?: boolean;
    error?: string | null;
    emptyText?: string;
  }>(),
  { loading: false, error: null, emptyText: "" }
);

const emit = defineEmits<{
  "chart-click": [event: BiChartEvent];
}>();

const hostRef = ref<HTMLElement | null>(null);
const ready = ref(false);
let handle: ChartHandle | null = null;
let observer: ResizeObserver | null = null;
let frame = 0;

/** 释放图表及尚未执行的尺寸更新。 */
function teardown() {
  cancelAnimationFrame(frame);
  observer?.disconnect();
  observer = null;
  handle?.dispose();
  handle = null;
  ready.value = false;
}

/** 用最新数据与主题重新编译配置并增量更新已挂载的图表。 */
function compileAndApply() {
  if (!handle) return;
  const engine = getEngine(props.chart.engine);
  handle.update(engine.compile(props.chart, transformRows(props.chart, props.rows), props.theme));
}

/** 等弹窗或折叠区域实际显示后再挂载图表，避免以零尺寸初始化。 */
function setup() {
  teardown();
  if (!hostRef.value || !props.rows.length) return;
  observer = new ResizeObserver(() => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      const host = hostRef.value;
      if (!host?.clientWidth || !host.clientHeight) return;
      if (handle) {
        handle.resize();
        return;
      }
      const engine = getEngine(props.chart.engine);
      handle = engine.mount(host, engine.compile(props.chart, transformRows(props.chart, props.rows), props.theme));
      engine.bindEvents(handle, (e) => emit("chart-click", { ...e, chartId: props.chart.id }));
      ready.value = true;
    });
  });
  observer.observe(hostRef.value);
}

onMounted(setup);

watch(
  () => [props.rows, props.theme, props.chart] as const,
  () => {
    // 数据从空到有 / 引擎未挂时重建；否则原地重编译
    if (!handle || !props.rows.length) {
      setup();
    } else {
      compileAndApply();
    }
  },
  { deep: true }
);

onBeforeUnmount(teardown);
</script>

<style scoped>
.bi-chart {
  position: relative;
  width: 100%;
  height: 100%;
}

.bi-chart__host {
  width: 100%;
  height: 100%;
}

.bi-chart__empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary, #909399);
  font-size: 13px;
  user-select: none;
}

.bi-chart__error {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--el-color-danger, #f56c6c);
  color: #fff;
  font-size: 11px;
  cursor: help;
}
</style>
