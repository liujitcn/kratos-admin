<template>
  <el-tooltip :content="title" placement="right" :show-after="0" :disabled="!isOverflow">
    <span ref="titleRef" class="sle" :title="isOverflow ? title : undefined">{{ title }}</span>
  </el-tooltip>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = defineProps<{
  /** 菜单标题。 */
  title: string;
}>();

const titleRef = ref<HTMLElement>();
const isOverflow = ref(false);
let resizeObserver: ResizeObserver | undefined;

/** 检测菜单标题是否被当前菜单宽度截断。 */
function updateOverflow() {
  const element = titleRef.value;
  isOverflow.value = Boolean(element && element.clientWidth > 0 && element.scrollWidth > element.clientWidth);
}

onMounted(() => {
  void nextTick(updateOverflow);
  if (typeof ResizeObserver === "undefined") return;

  resizeObserver = new ResizeObserver(updateOverflow);
  if (titleRef.value) resizeObserver.observe(titleRef.value);
});

watch(
  () => props.title,
  () => {
    void nextTick(updateOverflow);
  }
);

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
});
</script>
