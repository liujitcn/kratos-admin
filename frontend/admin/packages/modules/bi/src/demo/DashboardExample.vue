<template>
  <div v-if="active" class="dashboard-example">
    <BiDashboard v-if="componentId === 'BiDashboard'" :schema="demoLayout" :results="demoResults" />
    <template v-else>
      <p>点击“添加”后可调整尺寸、移除并重新添加。草稿卡片尚未保存，图表显示组件空态。</p>
      <BiDesigner v-model="layout" :cards="[demoCard]" @create-card="feedback = '创建卡片事件已发出，实际业务由父页面接入。'" />
      <p aria-live="polite">{{ feedback }} · 当前布局：{{ JSON.stringify(layout.cards) }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onActivated, onDeactivated, ref } from "vue";
import BiDashboard from "../components/BiDashboard.vue";
import BiDesigner from "../components/BiDesigner.vue";
import type { DashboardLayout } from "../bi-dashboard-schema";
import { demoCard, demoLayout, demoResults } from "./fixtures";

defineProps<{ componentId: string }>();
const active = ref(true);
const layout = ref<DashboardLayout>({ schemaVersion: 3, name: "本地布局草稿", cards: [] });
const feedback = ref("布局修改仅保存在当前示例中");

onActivated(() => {
  active.value = true;
});
onDeactivated(() => {
  active.value = false;
});
</script>

<style scoped>
.dashboard-example {
  display: grid;
  min-width: 0;
  gap: 16px;
}
.dashboard-example p {
  margin: 0;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}
.dashboard-example :deep(.bi-designer__canvas) {
  min-height: 420px;
}
.dashboard-example :deep(.bi-designer__library) {
  min-height: 320px;
  height: 420px;
}
</style>
