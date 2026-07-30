<!-- 💥 这里是一次性加载 LayoutComponents -->
<template>
  <el-watermark id="watermark" :font="font" :content="watermarkContent">
    <component :is="LayoutComponents[layout]" />
    <ThemeDrawer />
  </el-watermark>
</template>

<script setup lang="ts" name="Layout">
import { computed, reactive, watch, type Component } from "vue";
import { LayoutType } from "@/stores/interface";
import { useGlobalStore } from "@/stores/modules/global";
import { useConfigStore } from "@/stores/modules/config";
import ThemeDrawer from "./components/ThemeDrawer/index.vue";
import LayoutVertical from "./LayoutVertical/index.vue";
import LayoutClassic from "./LayoutClassic/index.vue";
import LayoutTransverse from "./LayoutTransverse/index.vue";
import LayoutColumns from "./LayoutColumns/index.vue";

const LayoutComponents: Record<LayoutType, Component> = {
  vertical: LayoutVertical,
  classic: LayoutClassic,
  transverse: LayoutTransverse,
  columns: LayoutColumns
};

const globalStore = useGlobalStore();
const configStore = useConfigStore();

const isDark = computed(() => globalStore.isDark);
const layout = computed(() => globalStore.layout);
const watermark = computed(() => globalStore.watermark);
const watermarkContent = computed(() => {
  if (!watermark.value) return "";

  const watermarkTextList = [configStore.display.sysName, configStore.display.watermark].filter(Boolean);
  return watermarkTextList.length ? watermarkTextList : "";
});

const font = reactive({ color: "rgba(0, 0, 0, .15)" });
watch(isDark, () => (font.color = isDark.value ? "rgba(255, 255, 255, .15)" : "rgba(0, 0, 0, .15)"), {
  immediate: true
});
</script>

<style scoped lang="scss">
.layout {
  min-width: 600px;
}
</style>
