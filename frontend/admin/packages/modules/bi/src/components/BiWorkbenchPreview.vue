<template>
  <div ref="host" class="bi-workbench-preview" :aria-busy="loading">
    <el-skeleton v-if="loading" :rows="2" animated />
    <div v-else-if="error" class="bi-workbench-preview__empty" role="alert">
      <span>{{ error }}</span>
      <el-button link type="primary" @click="load">{{ t("bi.card.retry") }}</el-button>
    </div>
    <template v-else>
      <div v-for="item in layout.cards.slice(0, 2)" :key="item.id" class="bi-workbench-preview__card">
        <h4 :title="item.name">{{ item.name }}</h4>
        <div class="bi-workbench-preview__chart">
          <BiCardVisualization
            v-if="results.get(item.id)"
            :card="dashboardSnapshotCard(item)"
            :result="results.get(item.id)!"
            compact
          />
          <span v-else class="bi-workbench-preview__empty">{{ t("common.message.no_data") }}</span>
        </div>
      </div>
      <span v-if="!layout.cards.length" class="bi-workbench-preview__empty">{{ t("bi.dashboard.empty") }}</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from "vue";
import { useIntersectionObserver } from "@vueuse/core";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiDashboard } from "../rpc/bi/admin/v1/bi_dashboard";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import type { DashboardLayout } from "../bi-dashboard-schema";
import { dashboardResults, dashboardSnapshotCard, parseDashboardLayout } from "../bi-dashboard-schema";
import { defBiDashboardService } from "../api/bi/admin/v1/bi_dashboard";
import BiCardVisualization from "./BiCardVisualization.vue";

const props = defineProps<{ dashboard: BiDashboard }>();
const host = ref<HTMLElement>();
const layout = ref<DashboardLayout>({ schemaVersion: 3, name: "", cards: [] });
const results = ref(new Map<string, BiQueryResult | undefined>());
const loading = ref(true);
const error = ref("");
let active = true;
const { stop } = useIntersectionObserver(host, ([entry]) => {
  if (entry?.isIntersecting) {
    stop();
    void load();
  }
});

/** 入口可见时按发布版本取真实数据，缩略图复用卡片渲染器。 */
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const data = await defBiDashboardService.GetBiDashboardData({
      id: props.dashboard.id,
      version: props.dashboard.published_version
    });
    if (!active) return;
    layout.value = parseDashboardLayout(data.schema_json);
    results.value = dashboardResults(data);
  } catch (cause) {
    if (active) error.value = cause instanceof Error ? cause.message : t("bi.query.error");
  } finally {
    if (active) loading.value = false;
  }
}

onBeforeUnmount(() => {
  active = false;
});
</script>

<style scoped>
.bi-workbench-preview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: stretch;
  gap: 8px;
  box-sizing: border-box;
  height: 144px;
  padding: 16px;
  background: var(--admin-page-card-bg-soft);
  border-bottom: 1px solid var(--el-border-color-lighter);
  pointer-events: none;
}
.bi-workbench-preview__card {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
}
.bi-workbench-preview h4 {
  margin: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  font-weight: 400;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-workbench-preview__chart {
  flex: 1;
  min-height: 0;
}
.bi-workbench-preview__empty,
.bi-workbench-preview > .el-skeleton {
  grid-column: 1 / -1;
}
.bi-workbench-preview__empty {
  display: flex;
  height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  overflow: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow-wrap: anywhere;
  pointer-events: auto;
}
</style>
