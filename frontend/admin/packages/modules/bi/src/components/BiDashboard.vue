<template>
  <div class="bi-dashboard-grid">
    <div
      v-for="item in schema.cards"
      :key="item.id"
      class="bi-dashboard-grid__item"
      :style="{ gridColumn: `${(item.x || 0) + 1} / span ${item.w}`, gridRow: `${(item.y || 0) + 1} / span ${item.h}` }"
    >
      <BiDataCard v-if="results.get(item.id)" :card="dashboardSnapshotCard(item)" :preview-result="results.get(item.id)" :show-category="false" />
      <el-empty v-else :description="t('common.message.no_data')" />
    </div>
    <el-empty v-if="!schema.cards.length" :description="t('bi.dashboard.empty')" />
  </div>
</template>
<script setup lang="ts">
import { t } from "@liujitcn/kratos-admin-core";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import { dashboardSnapshotCard } from "../bi-dashboard-schema";
import type { DashboardLayout } from "../bi-dashboard-schema";
import BiDataCard from "./BiDataCard.vue";
defineProps<{ schema: DashboardLayout; results: Map<string, BiQueryResult | undefined> }>();
</script>
<style scoped>
.bi-dashboard-grid {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  grid-auto-rows: 80px;
  gap: 12px;
}
.bi-dashboard-grid__item {
  min-width: 0;
  min-height: 0;
}
.bi-dashboard-grid__item :deep(.bi-data-card) {
  height: 100%;
}
.bi-dashboard-grid__item :deep(.bi-data-card__chart) {
  flex: 1;
  min-height: 80px;
  height: auto;
}
@media (max-width: 700px) {
  .bi-dashboard-grid {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }
  .bi-dashboard-grid__item {
    grid-column: 1 !important;
    grid-row: auto !important;
    min-height: 360px;
  }
}
</style>
