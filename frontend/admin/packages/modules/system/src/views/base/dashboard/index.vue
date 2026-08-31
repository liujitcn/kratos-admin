<template>
  <main class="dashboard-page">
    <header class="dashboard-header">
      <div>
        <h1>{{ t("system.dashboard.title") }}</h1>
        <p>{{ t("system.dashboard.updated_at", { time: updatedAt }) }}</p>
      </div>
      <el-button :loading="loading" @click="loadDashboard">
        <el-icon><Refresh /></el-icon>
        {{ t("core.table.refresh") }}
      </el-button>
    </header>

    <el-skeleton v-if="loading && !overview" :rows="3" animated />
    <template v-else>
      <section class="metric-grid" :aria-label="t('system.dashboard.overview')">
        <el-card v-for="metric in metrics" :key="metric.key" shadow="never" class="metric-card">
          <span class="metric-label">{{ metric.label }}</span>
          <strong class="metric-value">{{ metric.value }}</strong>
        </el-card>
      </section>

      <section class="chart-grid">
        <el-card shadow="never" class="chart-card chart-card-wide">
          <template #header>{{ t("system.dashboard.login_trend") }}</template>
          <div class="chart-wrap"><ECharts :option="loginTrendOption" height="320" /></div>
        </el-card>
        <el-card shadow="never" class="chart-card">
          <template #header>{{ t("system.dashboard.login_distribution") }}</template>
          <div class="chart-wrap"><ECharts :option="loginDistributionOption" height="320" /></div>
        </el-card>
        <el-card shadow="never" class="chart-card">
          <template #header>{{ t("system.dashboard.operation_distribution") }}</template>
          <div class="chart-wrap"><ECharts :option="operationDistributionOption" height="320" /></div>
        </el-card>
      </section>
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import ECharts from "@liujitcn/kratos-admin-core/components/ECharts/index.vue";
import { defBaseDashboardService } from "@liujitcn/kratos-admin-system/api/system/base_dashboard";
import type {
  BaseDashboardDistributionItem,
  BaseDashboardOverview,
  BaseDashboardTrendPoint
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dashboard";
import { t } from "@liujitcn/kratos-admin-core";

const loading = ref(false);
const overview = ref<BaseDashboardOverview>();
const trend = ref<BaseDashboardTrendPoint[]>([]);
const loginDistribution = ref<BaseDashboardDistributionItem[]>([]);
const operationDistribution = ref<BaseDashboardDistributionItem[]>([]);
const updatedAt = ref("");

const metrics = computed(() => [
  { key: "users", label: t("system.dashboard.users"), value: overview.value?.user_count ?? 0 },
  { key: "roles", label: t("system.dashboard.roles"), value: overview.value?.role_count ?? 0 },
  { key: "logins", label: t("system.dashboard.today_logins"), value: overview.value?.today_login_count ?? 0 },
  { key: "operations", label: t("system.dashboard.today_operations"), value: overview.value?.today_operation_count ?? 0 }
]);

const loginTrendOption = computed(() => ({
  tooltip: { trigger: "axis" as const },
  grid: { left: 40, right: 20, top: 24, bottom: 32 },
  xAxis: { type: "category" as const, boundaryGap: false, data: trend.value.map((item) => item.date) },
  yAxis: { type: "value" as const, minInterval: 1 },
  series: [{ type: "line" as const, smooth: true, data: trend.value.map((item) => item.count), areaStyle: {} }]
}));

const loginDistributionOption = computed(() => distributionOption(loginDistribution.value));
const operationDistributionOption = computed(() => distributionOption(operationDistribution.value));

async function loadDashboard() {
  loading.value = true;
  try {
    const [overviewResponse, trendResponse, loginResponse, operationResponse] = await Promise.all([
      defBaseDashboardService.GetBaseDashboardOverview({}),
      defBaseDashboardService.GetBaseDashboardLoginTrend({ days: 7 }),
      defBaseDashboardService.GetBaseDashboardLoginDistribution({}),
      defBaseDashboardService.GetBaseDashboardOperationDistribution({})
    ]);
    overview.value = overviewResponse;
    trend.value = trendResponse.points ?? [];
    loginDistribution.value = loginResponse.items ?? [];
    operationDistribution.value = operationResponse.items ?? [];
    updatedAt.value = new Date().toLocaleString();
  } catch {
    ElMessage.error(t("system.dashboard.load_failed"));
  } finally {
    loading.value = false;
  }
}

/** 将审计技术枚举名转换为当前语言的展示文案。 */
function distributionItemLabel(label: string) {
  const mappings = [
    ["BASE_OPERATION_ACTION_", "system.base.audit.operation_action."],
    ["BASE_AUDIT_RESULT_", "system.base.audit.result."]
  ];
  const mapping = mappings.find(([prefix]) => label.startsWith(prefix));
  if (!mapping) return label;
  const value = label.slice(mapping[0].length).toLowerCase();
  return value === "unspecified" ? t("common.message.unknown") : t(`${mapping[1]}${value}`);
}

/** 创建使用本地化图例的分布图配置。 */
function distributionOption(items: BaseDashboardDistributionItem[]) {
  return {
    tooltip: { trigger: "item" as const },
    legend: { bottom: 0, type: "scroll" as const },
    series: [{ type: "pie" as const, radius: ["38%", "68%"] as [string, string], center: ["50%", "45%"] as [string, string], data: items.map((item) => ({ name: distributionItemLabel(item.label), value: item.count })) }]
  };
}

onMounted(loadDashboard);
</script>

<style scoped>
.dashboard-page {
  padding: 20px;
}

.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.dashboard-header h1 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 22px;
  font-weight: 600;
}

.dashboard-header p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.metric-grid,
.chart-grid {
  display: grid;
  gap: 16px;
}

.metric-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 16px;
}

.metric-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 92px;
  justify-content: center;
}

.metric-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.metric-value {
  color: var(--el-text-color-primary);
  font-size: 28px;
  line-height: 1;
}

.chart-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.chart-card-wide {
  grid-column: 1 / -1;
}

.chart-wrap {
  height: 320px;
}

@media (max-width: 960px) {
  .metric-grid,
  .chart-grid {
    grid-template-columns: 1fr;
  }

  .chart-card-wide {
    grid-column: auto;
  }
}
</style>
