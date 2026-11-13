<template>
  <div id="echarts" ref="chartRef" :style="echartsStyle" />
</template>

<script setup lang="ts" name="ECharts">
import { ref, onMounted, onBeforeUnmount, watch, computed, markRaw, nextTick, onActivated } from "vue";
import { EChartsType, ECElementEvent } from "echarts/core";
import echarts, { ECOption } from "./config";
import { useDebounceFn } from "@vueuse/core";
import { useGlobalStore } from "@/stores/modules/global";
import { storeToRefs } from "pinia";

// ECharts 组件入参，控制图表配置、渲染方式、尺寸和事件回调。
interface Props {
  option: ECOption;
  renderer?: "canvas" | "svg";
  resize?: boolean;
  theme?: object | string;
  width?: number | string;
  height?: number | string;
  onClick?: (event: ECElementEvent) => any;
  onMouseup?: (event: ECElementEvent) => any;
}

const props = withDefaults(defineProps<Props>(), {
  renderer: "canvas",
  resize: true
});

const echartsStyle = computed(() => {
  return props.width || props.height
    ? { height: props.height + "px", width: props.width + "px" }
    : { height: "100%", width: "100%" };
});

const chartRef = ref<HTMLDivElement | HTMLCanvasElement>();
const chartInstance = ref<EChartsType>();

/** 按最新 option 全量重绘图表（notMerge 替换，避免残留旧系列配置）。 */
const draw = () => {
  if (chartInstance.value) {
    chartInstance.value.setOption(props.option, { notMerge: true });
  }
};

watch(props, () => {
  draw();
});

/** 图表元素点击事件透传给父组件。 */
const handleClick = (event: ECElementEvent) => props.onClick && props.onClick(event);
/** 图表元素松开事件透传给父组件。 */
const handleMouseup = (event: ECElementEvent) => props.onMouseup && props.onMouseup(event);

/** 初始化图表：同一 DOM 已有实例则复用（keep-alive 场景），否则创建实例并绑定事件。 */
const init = () => {
  if (!chartRef.value) return;
  chartInstance.value = echarts.getInstanceByDom(chartRef.value);

  if (!chartInstance.value) {
    chartInstance.value = markRaw(
      echarts.init(chartRef.value, props.theme, {
        renderer: props.renderer
      })
    );
    chartInstance.value.on("click", handleClick);
    chartInstance.value.on("mouseup", handleMouseup);
    draw();
  }
};

/** 重设图表尺寸，附带 300ms 过渡动画；可通过 resize 入参关闭。 */
const resize = () => {
  if (chartInstance.value && props.resize) {
    chartInstance.value.resize({ animation: { duration: 300 } });
  }
};

const debouncedResize = useDebounceFn(resize, 300, { maxWait: 800 });

const globalStore = useGlobalStore();
const { maximize, isCollapse, tabs, footer } = storeToRefs(globalStore);

watch(
  () => [maximize, isCollapse, tabs, footer],
  () => {
    debouncedResize();
  },
  { deep: true }
);

onMounted(() => {
  nextTick(() => init());
  window.addEventListener("resize", debouncedResize);
});

onActivated(() => {
  if (chartInstance.value) {
    chartInstance.value.resize();
  }
});

onBeforeUnmount(() => {
  chartInstance.value?.dispose();
  window.removeEventListener("resize", debouncedResize);
});

defineExpose({
  getInstance: () => chartInstance.value,
  resize,
  draw
});
</script>
