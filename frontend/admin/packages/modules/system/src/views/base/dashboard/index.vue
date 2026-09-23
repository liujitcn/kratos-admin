<template>
  <main class="dashboard-page">
    <el-skeleton v-if="loading && !overview" :rows="3" animated />
    <template v-else>
      <section class="metric-grid" :aria-label="t('system.dashboard.overview')">
        <el-card
          v-for="metric in metrics"
          :key="metric.key"
          :class="['metric-card', `metric-card--${metric.tone}`, 'admin-page-card']"
        >
          <div class="metric-top">
            <span class="metric-label">{{ metric.label }}</span
            ><span class="metric-icon"
              ><el-icon><component :is="metric.icon" /></el-icon
            ></span>
          </div>
          <strong class="metric-value">{{ metric.value }}</strong>
        </el-card>
      </section>
      <section class="chart-grid">
        <el-card class="chart-card chart-card-wide admin-page-card">
          <template #header
            ><div class="chart-heading">
              <span>{{ t("system.dashboard.login_trend") }}</span
              ><small>{{ t("system.dashboard.last_7_days") }}</small>
            </div></template
          >
          <div class="chart-wrap"><ECharts :option="loginTrendOption" height="285" /></div>
        </el-card>
        <el-card class="chart-card admin-page-card">
          <template #header
            ><div class="chart-heading">
              <span>{{ t("system.dashboard.login_distribution") }}</span
              ><small>{{ t("system.dashboard.total", { count: loginDistributionTotal }) }}</small>
            </div></template
          >
          <div class="chart-wrap"><ECharts :option="loginDistributionOption" height="265" /></div>
        </el-card>
        <el-card class="chart-card admin-page-card">
          <template #header
            ><div class="chart-heading">
              <span>{{ t("system.dashboard.operation_distribution") }}</span
              ><small>{{ t("system.dashboard.total", { count: operationDistributionTotal }) }}</small>
            </div></template
          >
          <div class="chart-wrap"><ECharts :option="operationDistributionOption" height="265" /></div>
        </el-card>
      </section>
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import ECharts from "@liujitcn/kratos-admin-core/components/ECharts/index.vue";
import { defBaseDashboardService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_dashboard";
import type {
  BaseDashboardDistributionItem,
  BaseDashboardOverview,
  BaseDashboardTrendPoint
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dashboard";
import { t } from "@liujitcn/kratos-admin-core";

const loading = ref(true);
const overview = ref<BaseDashboardOverview>();
const trend = ref<BaseDashboardTrendPoint[]>([]);
const loginDistribution = ref<BaseDashboardDistributionItem[]>([]);
const operationDistribution = ref<BaseDashboardDistributionItem[]>([]);
const chartTheme = ref({
  primary: "#409eff",
  success: "#67c23a",
  warning: "#e6a23c",
  info: "#909399",
  danger: "#f56c6c",
  text: "#606266",
  divider: "#e4e7ed",
  background: "#ffffff"
});
const metrics = computed(() => [
  { key: "users", label: t("system.dashboard.users"), value: overview.value?.user_count ?? 0, icon: User, tone: "primary" },
  { key: "roles", label: t("system.dashboard.roles"), value: overview.value?.role_count ?? 0, icon: UserFilled, tone: "success" },
  {
    key: "logins",
    label: t("system.dashboard.today_logins"),
    value: overview.value?.today_login_count ?? 0,
    icon: Timer,
    tone: "warning"
  },
  {
    key: "operations",
    label: t("system.dashboard.today_operations"),
    value: overview.value?.today_operation_count ?? 0,
    icon: Operation,
    tone: "danger"
  }
]);
const loginDistributionTotal = computed(() => loginDistribution.value.reduce((total, item) => total + item.count, 0));
const operationDistributionTotal = computed(() => operationDistribution.value.reduce((total, item) => total + item.count, 0));

function readThemeColor(name: string, fallback: string) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}
function refreshChartTheme() {
  chartTheme.value = {
    primary: readThemeColor("--el-color-primary", "#409eff"),
    success: readThemeColor("--el-color-success", "#67c23a"),
    warning: readThemeColor("--el-color-warning", "#e6a23c"),
    info: readThemeColor("--el-color-info", "#909399"),
    danger: readThemeColor("--el-color-danger", "#f56c6c"),
    text: readThemeColor("--el-text-color-secondary", "#606266"),
    divider: readThemeColor("--el-border-color-lighter", "#e4e7ed"),
    background: readThemeColor("--el-bg-color", "#ffffff")
  };
}
const loginTrendOption = computed(() => ({
  color: [chartTheme.value.primary],
  tooltip: { trigger: "axis" as const },
  grid: { left: 42, right: 20, top: 20, bottom: 28 },
  xAxis: {
    type: "category" as const,
    boundaryGap: false,
    data: trend.value.map(item => item.date),
    axisLine: { lineStyle: { color: chartTheme.value.divider } },
    axisLabel: { color: chartTheme.value.text }
  },
  yAxis: {
    type: "value" as const,
    minInterval: 1,
    splitLine: { lineStyle: { color: chartTheme.value.divider } },
    axisLabel: { color: chartTheme.value.text }
  },
  series: [
    {
      type: "line" as const,
      smooth: true,
      data: trend.value.map(item => item.count),
      symbol: "circle",
      symbolSize: 7,
      lineStyle: { width: 3, color: chartTheme.value.primary },
      itemStyle: { color: chartTheme.value.primary, borderWidth: 2, borderColor: chartTheme.value.background },
      areaStyle: { color: chartTheme.value.primary, opacity: 0.12 }
    }
  ]
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
  } catch {
    /* 请求层已展示具体错误。 */
  } finally {
    loading.value = false;
  }
}
/** 将审计技术枚举名转换为当前语言的展示文案。 */
function distributionItemLabel(label: string) {
  const mappings = [
    ["BASE_OPERATION_ACTION_", "system.base.log.operation_action."],
    ["BASE_LOG_RESULT_", "system.base.log.result."]
  ];
  const mapping = mappings.find(([prefix]) => label.startsWith(prefix));
  if (!mapping) return label;
  const value = label.slice(mapping[0].length).toLowerCase();
  return value === "unspecified" ? t("common.message.unknown") : t(`${mapping[1]}${value}`);
}
/** 创建使用本地化图例的分布图配置。 */
function distributionOption(items: BaseDashboardDistributionItem[]) {
  return {
    color: [
      chartTheme.value.primary,
      chartTheme.value.success,
      chartTheme.value.info,
      chartTheme.value.warning,
      chartTheme.value.danger
    ],
    tooltip: { trigger: "item" as const },
    legend: { bottom: 0, type: "scroll" as const, textStyle: { color: chartTheme.value.text } },
    series: [
      {
        type: "pie" as const,
        radius: ["55%", "76%"] as [string, string],
        center: ["50%", "43%"] as [string, string],
        label: { color: chartTheme.value.text },
        itemStyle: { borderColor: chartTheme.value.background, borderWidth: 3 },
        data: items.map(item => ({ name: distributionItemLabel(item.label), value: item.count }))
      }
    ]
  };
}
onMounted(() => {
  refreshChartTheme();
  loadDashboard();
  window.addEventListener("theme-change", refreshChartTheme);
});
onBeforeUnmount(() => window.removeEventListener("theme-change", refreshChartTheme));
</script>

<style scoped>
.dashboard-page {
  min-width: 0;
}
.dashboard-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 22px;
}
.dashboard-eyebrow {
  display: block;
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
  letter-spacing: 0.12em;
}
.dashboard-heading h1 {
  margin: 6px 0 0;
  color: var(--admin-page-text-primary);
  font-size: 26px;
  line-height: 1.2;
}
.dashboard-updated {
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
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
.metric-card {
  position: relative;
  overflow: hidden;
}
.metric-card :deep(.el-card__body) {
  min-height: 124px;
  padding: 18px 20px 16px;
}
.metric-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.metric-label {
  color: var(--admin-page-text-secondary);
  font-size: 13px;
}
.metric-value {
  display: block;
  margin-top: 18px;
  color: var(--admin-page-text-primary);
  font-size: 32px;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.metric-icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 10px;
  color: var(--metric-color);
  background: var(--metric-bg);
}
.metric-icon .el-icon {
  font-size: 17px;
}
.metric-card--primary {
  --metric-color: var(--el-color-primary);
  --metric-bg: var(--el-color-primary-light-9);
}
.metric-card--success {
  --metric-color: var(--el-color-success);
  --metric-bg: var(--el-color-success-light-9);
}
.metric-card--warning {
  --metric-color: var(--el-color-warning);
  --metric-bg: var(--el-color-warning-light-9);
}
.metric-card--danger {
  --metric-color: var(--el-color-danger);
  --metric-bg: var(--el-color-danger-light-9);
}
.chart-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.chart-card-wide {
  grid-column: 1 / -1;
}
.chart-card :deep(.el-card__header) {
  padding: 17px 22px 14px;
  border-bottom-color: var(--admin-page-divider);
}
.chart-card :deep(.el-card__body) {
  padding: 0 20px 16px;
}
.chart-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chart-heading span {
  color: var(--admin-page-text-primary);
  font-size: 16px;
  font-weight: 600;
}
.chart-heading small {
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
  font-weight: 400;
}
.chart-wrap {
  height: 285px;
}
.chart-card:not(.chart-card-wide) .chart-wrap {
  height: 265px;
}
@media (width <= 960px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (width <= 640px) {
  .dashboard-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }
  .metric-grid,
  .chart-grid {
    grid-template-columns: 1fr;
  }
  .chart-card-wide {
    grid-column: auto;
  }
}
</style>
