<!-- 💥 这里是异步加载 LayoutComponents -->
<template>
  <el-watermark id="watermark" :font="font" :content="watermarkContent">
    <suspense>
      <template #default>
        <component :is="LayoutComponents[layout]" />
      </template>
      <template #fallback>
        <Loading />
      </template>
    </suspense>
    <ThemeDrawer />
  </el-watermark>
</template>

<script setup lang="ts" name="layoutAsync">
import { computed, defineAsyncComponent, reactive, watch, type Component } from "vue";
import { LayoutType } from "@/stores/interface";
import { useGlobalStore } from "@/stores/modules/global";
import { useConfigStore } from "@/stores/modules/config";
import Loading from "@/components/Loading/index.vue";
import ThemeDrawer from "./components/ThemeDrawer/index.vue";

const LayoutComponents: Record<LayoutType, Component> = {
  vertical: defineAsyncComponent(() => import("./LayoutVertical/index.vue")),
  classic: defineAsyncComponent(() => import("./LayoutClassic/index.vue")),
  transverse: defineAsyncComponent(() => import("./LayoutTransverse/index.vue")),
  columns: defineAsyncComponent(() => import("./LayoutColumns/index.vue"))
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
