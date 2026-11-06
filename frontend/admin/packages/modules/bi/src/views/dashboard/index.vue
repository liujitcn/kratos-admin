<template>
  <main class="bi-dashboard-page">
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" /><el-skeleton
      v-else-if="loading"
      :rows="8"
      animated
    /><BiDashboard v-else :schema="layout" :results="results" />
  </main>
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from "vue";
import { useRoute } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import BiDashboard from "../../components/BiDashboard.vue";
import { parseDashboardLayout, dashboardResults } from "../../bi-dashboard-schema";
import type { DashboardLayout } from "../../bi-dashboard-schema";
import type { BiQueryResult } from "../../rpc/bi/admin/v1/bi_query";
import { defBiDashboardService } from "../../api/bi/admin/v1/bi_dashboard";
defineOptions({ name: "BiDashboard", inheritAttrs: false });
const route = useRoute();
const id = computed(() => {
  return Number(route.params.id || route.path.match(/^\/reports\/dashboard\/([1-9]\d*)$/)?.[1] || 0);
});
const layout = ref<DashboardLayout>({ schemaVersion: 3, name: "", cards: [] });
const results = ref(new Map<string, BiQueryResult | undefined>());
const loading = ref(false),
  error = ref("");
let request = 0;
/** 从地址路径读取仪表盘编号，忽略切换页面前的迟到结果。 */
async function load() {
  const current = ++request;
  error.value = "";
  results.value = new Map();
  layout.value = { schemaVersion: 3, name: "", cards: [] };
  if (!Number.isSafeInteger(id.value) || id.value <= 0) {
    error.value = t("bi.dashboard.invalid_id");
    return;
  }
  loading.value = true;
  try {
    const result = await defBiDashboardService.GetBiDashboardData({ id: id.value, version: 0 });
    if (current !== request) return;
    layout.value = parseDashboardLayout(result.schema_json);
    results.value = dashboardResults(result);
  } catch (cause) {
    if (current === request) error.value = cause instanceof Error ? cause.message : t("bi.query.error");
  } finally {
    if (current === request) loading.value = false;
  }
}
watch(id, () => void load(), { immediate: true });
onBeforeUnmount(() => request++);
</script>
<style scoped>
.bi-dashboard-page {
  min-width: 0;
}
</style>
