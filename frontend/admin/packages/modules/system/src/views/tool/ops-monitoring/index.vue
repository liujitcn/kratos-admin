<template>
  <div class="app-container ops-monitoring-page">
    <section class="ops-page-head">
      <div>
        <div class="ops-breadcrumb">{{ t("system.ops_monitoring.breadcrumb") }}</div>
        <h1>{{ t("system.ops_monitoring.title") }}</h1>
        <p>{{ runtimeDescription }}</p>
      </div>
      <div class="ops-page-meta">
        <el-tag :type="loading ? 'info' : 'success'" effect="plain">
          {{ loading ? t("system.ops_monitoring.collecting") : t("system.ops_monitoring.realtime") }}
        </el-tag>
        <span>{{ t("system.ops_monitoring.window", { minutes: windowMinutes }) }}</span>
        <span>{{ t("system.ops_monitoring.last_sample", { time: formatDateTime(lastCollectedAt) }) }}</span>
      </div>
    </section>

    <section class="ops-kpi-grid" :aria-label="t('system.ops_monitoring.core_metrics')">
      <el-card v-for="item in kpiItems" :key="item.label" class="ops-kpi-card" shadow="never">
        <span class="ops-kpi-label">{{ item.label }}</span>
        <strong class="ops-kpi-value">{{ item.value }}</strong>
        <span class="ops-kpi-foot" :class="item.tone"
          >{{ item.change }} <em>{{ item.context }}</em></span
        >
        <i class="ops-kpi-rule" :style="{ backgroundColor: item.color }" />
      </el-card>
    </section>

    <section class="ops-primary-grid">
      <el-card class="ops-panel ops-trend-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>{{ t("system.ops_monitoring.request_latency_trend") }}</h2>
              <p>{{ t("system.ops_monitoring.aggregation", { minutes: windowMinutes }) }}</p>
            </div>
            <el-tag size="small" type="success" effect="plain">{{ t("system.ops_monitoring.realtime") }}</el-tag>
          </div>
        </template>
        <div class="ops-legend">
          <span><i class="ops-legend-mark is-teal" />QPS</span>
          <span><i class="ops-legend-mark is-amber" />{{ t("system.ops_monitoring.p95_latency") }}</span>
        </div>
        <div class="ops-trend-row">
          <span class="ops-trend-label">QPS</span>
          <div
            class="ops-trend-bars"
            :aria-label="
              t('system.ops_monitoring.qps_trend', { value: `${formatNumber(trafficSummary?.qps)} req/s` })
            "
          >
            <i v-for="(point, index) in qpsTrend" :key="`qps-${index}`" class="is-teal" :style="{ height: `${point}%` }" />
          </div>
          <strong>{{ formatNumber(trafficSummary?.qps) }}<small>req/s</small></strong>
        </div>
        <div class="ops-trend-row">
          <span class="ops-trend-label">P95</span>
          <div
            class="ops-trend-bars"
            :aria-label="
              t('system.ops_monitoring.p95_trend', { value: `${formatNumber(trafficSummary?.p95_latency_ms)} ms` })
            "
          >
            <i
              v-for="(point, index) in latencyTrend"
              :key="`latency-${index}`"
              class="is-amber"
              :style="{ height: `${point}%` }"
            />
          </div>
          <strong>{{ formatNumber(trafficSummary?.p95_latency_ms) }}<small>ms</small></strong>
        </div>
        <div class="ops-time-axis"><span v-for="point in trendAxis" :key="point">{{ point }}</span></div>
      </el-card>

      <el-card class="ops-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>{{ t("system.ops_monitoring.service_availability") }}</h2>
              <p>{{ t("system.ops_monitoring.service_dependency") }}</p>
            </div>
            <el-tag :type="onlineCount === serviceCount ? 'success' : 'warning'" effect="plain">
              {{ t("system.ops_monitoring.online", { online: onlineCount, total: serviceCount }) }}
            </el-tag>
          </div>
        </template>
        <div class="ops-summary-list">
          <div v-for="item in availabilityItems" :key="item.label" class="ops-summary-row">
            <span>{{ item.label }}</span>
            <strong :class="`is-${item.tone}`"><i />{{ item.value }}</strong>
          </div>
        </div>
      </el-card>
    </section>

    <section class="ops-storage-section">
      <div class="ops-section-title">
        <h2>{{ t("system.ops_monitoring.data_storage") }}</h2>
        <span>{{ t("system.ops_monitoring.storage_summary") }}</span>
      </div>
      <div class="ops-storage-grid">
        <el-card v-for="item in storageItems" :key="item.name" class="ops-storage-card" shadow="never">
          <div class="ops-storage-head">
            <div class="ops-storage-name">
              <span class="ops-storage-mark" :style="{ backgroundColor: item.color }">{{ item.shortName }}</span>
              <div>
                <h3>{{ item.name }}</h3>
                <code>{{ item.address }}</code>
              </div>
            </div>
            <el-tag :type="item.status === '正常' ? 'success' : 'warning'" size="small" effect="plain">{{ item.statusLabel }}</el-tag>
          </div>
          <div class="ops-storage-metrics">
            <div v-for="metric in item.metrics" :key="metric.label">
              <strong>{{ metric.value }}</strong
              ><span>{{ metric.label }}</span>
            </div>
          </div>
          <div class="ops-capacity">
            <span>{{ item.capacityLabel }}</span
            ><el-progress :percentage="item.capacity" :show-text="false" :color="item.color" /><span>{{ item.capacity }}%</span>
          </div>
        </el-card>
      </div>
    </section>

    <section class="ops-secondary-grid">
      <el-card class="ops-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>{{ t("system.ops_monitoring.interface_response") }}</h2>
              <p>{{ t("system.ops_monitoring.sort_recent", { minutes: windowMinutes }) }}</p>
            </div>
            <span class="ops-muted">{{ t("system.ops_monitoring.endpoint_count", { count: endpointItems.length }) }}</span>
          </div>
        </template>
        <el-table :data="endpointItems" size="small" class="ops-table">
          <el-table-column :label="t('system.ops_monitoring.endpoint')" min-width="230">
            <template #default="{ row }"
              ><code class="ops-route">{{ row.route }}</code></template
            >
          </el-table-column>
          <el-table-column prop="qps" label="QPS" width="75" />
          <el-table-column prop="latency" label="P95" width="85" />
          <el-table-column prop="errorRate" :label="t('system.ops_monitoring.error_rate')" width="85" />
          <el-table-column :label="t('system.ops_monitoring.status.label')" width="85" align="right">
            <template #default="{ row }"
              ><el-tag :type="row.healthy ? 'success' : 'warning'" size="small" effect="plain">{{ row.status }}</el-tag></template
            >
          </el-table-column>
        </el-table>
      </el-card>

      <el-card class="ops-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>{{ t("system.ops_monitoring.instance_resources") }}</h2>
              <p>{{ t("system.ops_monitoring.current_recent_sample") }}</p>
            </div>
            <span class="ops-muted">{{ t("system.ops_monitoring.instance_count", { count: nodeItems.length }) }}</span>
          </div>
        </template>
        <div class="ops-node-list">
          <div v-for="node in nodeItems" :key="node.name" class="ops-node-row">
            <code>{{ node.name }}</code>
            <div class="ops-node-meters">
              <div v-for="metric in node.metrics" :key="metric.label" class="ops-meter">
                <span>{{ metric.label }}</span
                ><el-progress :percentage="metric.value" :show-text="false" :color="metric.color" /><strong
                  >{{ metric.value }}%</strong
                >
              </div>
            </div>
          </div>
        </div>
      </el-card>
    </section>

    <el-card class="ops-panel ops-alert-panel" shadow="never">
      <template #header>
        <div class="ops-section-head">
          <div>
            <h2>{{ t("system.ops_monitoring.alert_events") }}</h2>
            <p>{{ t("system.ops_monitoring.alert_attention") }}</p>
          </div>
          <el-tag type="warning" effect="plain">{{ t("system.ops_monitoring.unresolved", { count: alertItems.length }) }}</el-tag>
        </div>
      </template>
      <div class="ops-alert-list">
        <div v-for="alert in alertItems" :key="alert.title" class="ops-alert-row">
          <i :class="`is-${alert.tone}`" />
          <div>
            <strong>{{ alert.title }}</strong
            ><span>{{ alert.detail }}</span>
          </div>
          <time>{{ formatRelativeTime(alert.at) }}</time>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { defOpsMonitoringService } from "@liujitcn/kratos-admin-system/api/system/ops_monitoring";
import {
  subscribeOpsMonitoringNodes,
  subscribeOpsMonitoringServices,
  subscribeOpsMonitoringStorage,
  subscribeOpsMonitoringTraffic,
  type OpsMonitoringSseStop
} from "@liujitcn/kratos-admin-system/api/system/ops_monitoring_sse";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";
import type {
  OpsAlert,
  OpsEndpoint,
  OpsNode,
  OpsRuntime,
  OpsAlertsResponse,
  OpsEndpointsResponse,
  OpsServicesResponse,
  OpsStorage,
  OpsStorageResponse,
  OpsTrafficResponse,
  OpsNodesResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ops_monitoring";

defineOptions({
  name: "OpsMonitoring",
  inheritAttrs: false
});

/** 核心运维指标。 */
interface KpiItem {
  /** 指标名称。 */
  label: string;
  /** 当前指标值。 */
  value: string;
  /** 环比变化。 */
  change: string;
  /** 变化上下文。 */
  context: string;
  /** 变化颜色语义。 */
  tone: "is-up" | "is-down";
  /** 指标强调色。 */
  color: string;
}

/** 接口响应摘要。 */
interface EndpointItem {
  /** HTTP 路径。 */
  route: string;
  /** 每秒请求数。 */
  qps: string;
  /** P95 响应延迟。 */
  latency: string;
  /** 请求错误率。 */
  errorRate: string;
  /** 当前健康状态。 */
  status: string;
  /** 是否为正常状态。 */
  healthy: boolean;
}

/** 实例资源指标。 */
interface NodeItem {
  /** 实例名称。 */
  name: string;
  /** CPU 与内存指标。 */
  metrics: Array<{ label: string; value: number; color: string }>;
}

/** 服务依赖摘要。 */
interface AvailabilityItem {
  /** 服务名称。 */
  label: string;
  /** 当前状态。 */
  value: string;
  /** 状态颜色。 */
  tone: "ok" | "warn";
}

const loading = ref(false);
const runtime = ref<OpsRuntime>();
const traffic = ref<OpsTrafficResponse>();
const services = ref<OpsServicesResponse>();
const storage = ref<OpsStorageResponse>();
const endpoints = ref<OpsEndpointsResponse>();
const nodes = ref<OpsNodesResponse>();
const alerts = ref<OpsAlertsResponse>();
const windowMinutes = ref(15);
const lastCollectedAt = ref("");
const trafficSummary = computed(() => traffic.value?.traffic);
const runtimeDescription = computed(() => {
  if (!runtime.value) return t("system.ops_monitoring.runtime_unavailable");
  const environment = runtime.value.environment || t("system.ops_monitoring.unknown");
  const service = runtime.value.service_name || t("system.ops_monitoring.unknown");
  return t("system.ops_monitoring.runtime_description", { environment, service });
});

const kpiItems = computed<KpiItem[]>(() => [
  {
    label: t("system.ops_monitoring.kpi.throughput"),
    value: `${formatNumber(trafficSummary.value?.qps)} req/s`,
    change: t("system.ops_monitoring.realtime"),
    context: t("system.ops_monitoring.kpi.current_window"),
    tone: "is-up",
    color: "var(--el-color-primary)"
  },
  {
    label: t("system.ops_monitoring.kpi.p95_latency"),
    value: `${formatNumber(trafficSummary.value?.p95_latency_ms)} ms`,
    change: t("system.ops_monitoring.realtime"),
    context: t("system.ops_monitoring.kpi.current_window"),
    tone: "is-down",
    color: "var(--el-color-warning)"
  },
  {
    label: t("system.ops_monitoring.kpi.error_rate"),
    value: `${formatNumber(trafficSummary.value?.error_rate)}%`,
    change: t("system.ops_monitoring.realtime"),
    context: t("system.ops_monitoring.kpi.current_window"),
    tone: "is-up",
    color: "var(--el-color-danger)"
  },
  {
    label: t("system.ops_monitoring.kpi.availability"),
    value: `${formatNumber(trafficSummary.value?.availability)}%`,
    change: t("system.ops_monitoring.realtime"),
    context: t("system.ops_monitoring.kpi.current_window"),
    tone: "is-up",
    color: "var(--el-color-success)"
  }
]);

const qpsTrend = computed(() => normalizeTrend(traffic.value?.points.map(point => point.qps_percent)));
const latencyTrend = computed(() => normalizeTrend(traffic.value?.points.map(point => point.latency_percent)));
const trendAxis = computed(() => (traffic.value?.points ?? []).filter((_, index, list) => index === 0 || index === list.length - 1).map(point => formatClock(point.at)));

const availabilityItems = computed<AvailabilityItem[]>(() =>
  (services.value?.services ?? []).map(service => ({
    label: service.name,
    value: translateStatus(service.status),
    tone: service.status === "正常" ? "ok" : "warn"
  }))
);

const storageItems = computed(() => (storage.value?.storage ?? []).map(item => mapStorage(item)));
const endpointItems = computed<EndpointItem[]>(() => (endpoints.value?.endpoints ?? []).map(mapEndpoint));
const nodeItems = computed<NodeItem[]>(() => (nodes.value?.nodes ?? []).map(mapNode));
const alertItems = computed(() => (alerts.value?.alerts ?? []).map(mapAlert));
const serviceCount = computed(() => availabilityItems.value.length);
const onlineCount = computed(() => availabilityItems.value.filter(item => item.tone === "ok").length);

let sseStops: OpsMonitoringSseStop[] = [];

async function loadMonitoring() {
  loading.value = true;
  const results = await Promise.allSettled([
    defOpsMonitoringService.GetOpsRuntime({}),
    defOpsMonitoringService.GetOpsTraffic({ window_minutes: windowMinutes.value }),
    defOpsMonitoringService.GetOpsServices({}),
    defOpsMonitoringService.GetOpsStorage({}),
    defOpsMonitoringService.GetOpsEndpoints({ window_minutes: windowMinutes.value }),
    defOpsMonitoringService.GetOpsNodes({}),
    defOpsMonitoringService.GetOpsAlerts({ window_minutes: windowMinutes.value })
  ]);
  const [runtimeResult, trafficResult, servicesResult, storageResult, endpointsResult, nodesResult, alertsResult] = results;
  if (runtimeResult.status === "fulfilled") runtime.value = runtimeResult.value;
  if (trafficResult.status === "fulfilled") setTraffic(trafficResult.value);
  if (servicesResult.status === "fulfilled") setServices(servicesResult.value);
  if (storageResult.status === "fulfilled") setStorage(storageResult.value);
  if (endpointsResult.status === "fulfilled") setEndpoints(endpointsResult.value);
  if (nodesResult.status === "fulfilled") setNodes(nodesResult.value);
  if (alertsResult.status === "fulfilled") setAlerts(alertsResult.value);
  loading.value = false;
}

function setTraffic(value: OpsTrafficResponse) {
  traffic.value = value;
  windowMinutes.value = value.window_minutes || windowMinutes.value;
  updateCollectedAt(value.collected_at);
}

function setServices(value: OpsServicesResponse) {
  services.value = value;
  updateCollectedAt(value.collected_at);
}

function setStorage(value: OpsStorageResponse) {
  storage.value = value;
  updateCollectedAt(value.collected_at);
}

function setEndpoints(value: OpsEndpointsResponse) {
  endpoints.value = value;
  updateCollectedAt(value.collected_at);
}

function setNodes(value: OpsNodesResponse) {
  nodes.value = value;
  updateCollectedAt(value.collected_at);
}

function setAlerts(value: OpsAlertsResponse) {
  alerts.value = value;
  updateCollectedAt(value.collected_at);
}

function updateCollectedAt(value?: string) {
  if (value) lastCollectedAt.value = value;
}

function subscribeRealtime() {
  sseStops = [
    subscribeOpsMonitoringTraffic(setTraffic),
    subscribeOpsMonitoringServices(setServices),
    subscribeOpsMonitoringStorage(setStorage),
    subscribeOpsMonitoringNodes(setNodes)
  ];
}

function stopRealtime() {
  sseStops.forEach(stop => stop());
  sseStops = [];
}

function formatNumber(value?: number) {
  if (value === undefined || Number.isNaN(value)) return "--";
  return new Intl.NumberFormat(getCurrentLocale(), { maximumFractionDigits: 2 }).format(value);
}

function formatDateTime(value?: string) {
  if (!value) return "--";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--";
  return new Intl.DateTimeFormat(getCurrentLocale(), { dateStyle: "short", timeStyle: "medium" }).format(date);
}

function formatClock(value?: string) {
  if (!value) return "--:--";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--:--";
  return new Intl.DateTimeFormat(getCurrentLocale(), { hour: "2-digit", minute: "2-digit" }).format(date);
}

function formatRelativeTime(value?: string) {
  if (!value) return "--";
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return "--";
  const minutes = Math.max(0, Math.round((Date.now() - timestamp) / 60000));
  if (minutes < 1) return t("system.ops_monitoring.just_now");
  if (minutes < 60) return t("system.ops_monitoring.minutes_ago", { count: minutes });
  const hours = Math.round(minutes / 60);
  if (hours < 24) return t("system.ops_monitoring.hours_ago", { count: hours });
  return t("system.ops_monitoring.days_ago", { count: Math.round(hours / 24) });
}

function normalizeTrend(values?: number[]) {
  return values?.length ? values.map(value => Math.max(0, Math.min(value, 100))) : [0, 0, 0, 0];
}

function translateStatus(status?: string) {
  if (status === "正常") return t("system.ops_monitoring.status.normal");
  if (status === "未配置") return t("system.ops_monitoring.status.unconfigured");
  if (status === "异常") return t("system.ops_monitoring.status.error");
  if (status === "关注") return t("system.ops_monitoring.status.attention");
  return status || t("system.ops_monitoring.unknown");
}

function mapStorage(item: OpsStorage) {
  return {
    ...item,
    color: item.status === "正常" ? "var(--el-color-success)" : "var(--el-color-warning)",
    statusLabel: translateStatus(item.status),
    shortName: item.short_name || item.name,
    capacityLabel: item.capacity_label || t("system.ops_monitoring.capacity"),
    capacity: Math.max(0, Math.min(item.capacity || 0, 100)),
    metrics: item.metrics ?? []
  };
}

function mapEndpoint(endpoint: OpsEndpoint): EndpointItem {
  return {
    route: endpoint.route,
    qps: formatNumber(endpoint.qps),
    latency: `${formatNumber(endpoint.p95_latency_ms)} ms`,
    errorRate: `${formatNumber(endpoint.error_rate)}%`,
    status: translateStatus(endpoint.status),
    healthy: endpoint.status === "正常"
  };
}

function mapNode(node: OpsNode): NodeItem {
  return {
    name: node.name,
    metrics: (node.metrics ?? []).map(metric => ({
      label: metric.label,
      value: Math.max(0, Math.min(metric.value, 100)),
      color: metric.value >= 70 ? "var(--el-color-warning)" : "var(--el-color-primary)"
    }))
  };
}

function mapAlert(alert: OpsAlert) {
  return {
    title: alert.title,
    detail: alert.detail,
    at: alert.at,
    tone: alert.status === "正常" ? "ok" : "warn"
  };
}

onMounted(async () => {
  await loadMonitoring();
  subscribeRealtime();
});

onBeforeUnmount(stopRealtime);
</script>

<style scoped lang="scss">
.ops-monitoring-page {
  --ops-card-bg: var(--admin-page-card-bg);
  --ops-card-border: var(--admin-page-card-border);
  --ops-text-primary: var(--admin-page-text-primary);
  --ops-text-secondary: var(--admin-page-text-secondary);
  --ops-text-placeholder: var(--admin-page-text-placeholder);

  box-sizing: border-box;
  min-height: 100%;
  padding: 24px;
  color: var(--ops-text-primary);
  background: var(--el-bg-color-page);
}
.ops-page-head,
.ops-primary-grid,
.ops-secondary-grid,
.ops-storage-grid {
  min-width: 0;
}
.ops-page-head {
  display: flex;
  gap: 24px;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 22px;
}
.ops-breadcrumb {
  display: flex;
  gap: 7px;
  margin-bottom: 9px;
  font-size: 12px;
  color: var(--ops-text-placeholder);
}
.ops-breadcrumb span {
  color: var(--el-border-color);
}
.ops-page-head h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 650;
  line-height: 1.2;
  color: var(--ops-text-primary);
}
.ops-page-head p {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--ops-text-secondary);
}
.ops-page-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
  font-size: 12px;
  color: var(--ops-text-placeholder);
}
.ops-kpi-grid,
.ops-primary-grid,
.ops-storage-grid,
.ops-secondary-grid {
  display: grid;
  gap: 14px;
}
.ops-kpi-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 24px;
}
.ops-kpi-card,
.ops-panel,
.ops-storage-card,
.ops-alert-panel {
  background: var(--ops-card-bg);
  border: 1px solid var(--ops-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.ops-kpi-card {
  position: relative;
  min-width: 0;
  padding: 15px 16px 18px;
}
.ops-kpi-card :deep(.el-card__body),
.ops-panel :deep(.el-card__body),
.ops-storage-card :deep(.el-card__body),
.ops-alert-panel :deep(.el-card__body) {
  padding: 0;
}
.ops-kpi-label {
  display: block;
  font-size: 12px;
  color: var(--ops-text-secondary);
}
.ops-kpi-value {
  display: block;
  margin: 5px 0 4px;
  font-size: 22px;
  font-weight: 650;
  line-height: 1.2;
  color: var(--ops-text-primary);
}
.ops-kpi-foot {
  display: flex;
  gap: 7px;
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-kpi-foot em {
  font-style: normal;
}
.ops-kpi-foot.is-up {
  color: var(--el-color-success);
}
.ops-kpi-foot.is-down {
  color: var(--el-color-danger);
}
.ops-kpi-foot.is-up em,
.ops-kpi-foot.is-down em {
  color: var(--ops-text-placeholder);
}
.ops-kpi-rule {
  position: absolute;
  right: 16px;
  bottom: 10px;
  left: 16px;
  height: 2px;
  border-radius: 2px;
  opacity: 0.65;
}
.ops-primary-grid {
  grid-template-columns: minmax(0, 1.65fr) minmax(280px, 0.85fr);
  margin-bottom: 24px;
}
.ops-panel :deep(.el-card__header),
.ops-storage-card :deep(.el-card__header),
.ops-alert-panel :deep(.el-card__header) {
  padding: 16px 18px 13px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ops-panel :deep(.el-card__body),
.ops-storage-card :deep(.el-card__body),
.ops-alert-panel :deep(.el-card__body) {
  padding: 0 18px 18px;
}
.ops-section-head,
.ops-section-title {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}
.ops-section-head h2,
.ops-section-title h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 650;
  color: var(--ops-text-primary);
}
.ops-section-head p {
  margin: 4px 0 0;
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-muted,
.ops-section-title > span {
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-legend {
  display: flex;
  gap: 16px;
  margin: 17px 0 10px;
  font-size: 11px;
  color: var(--ops-text-secondary);
}
.ops-legend span {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}
.ops-legend-mark {
  width: 15px;
  height: 3px;
  border-radius: 3px;
}
.ops-legend-mark.is-teal,
.ops-trend-bars i.is-teal {
  background: var(--el-color-primary);
}
.ops-legend-mark.is-amber,
.ops-trend-bars i.is-amber {
  background: var(--el-color-warning);
}
.ops-trend-row {
  display: grid;
  grid-template-columns: 55px minmax(0, 1fr) 67px;
  gap: 10px;
  align-items: end;
  margin-bottom: 13px;
}
.ops-trend-label {
  font-size: 12px;
  color: var(--ops-text-secondary);
}
.ops-trend-bars {
  display: grid;
  grid-template-columns: repeat(12, minmax(4px, 1fr));
  gap: 5px;
  align-items: end;
  height: 105px;
  padding: 0 2px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ops-trend-bars i {
  display: block;
  min-height: 4px;
  border-radius: 3px 3px 0 0;
  opacity: 0.8;
}
.ops-trend-bars i:last-child {
  opacity: 1;
}
.ops-trend-row > strong {
  font-size: 14px;
  font-weight: 650;
  color: var(--ops-text-primary);
  text-align: right;
}
.ops-trend-row > strong small {
  display: block;
  font-size: 10px;
  font-weight: 400;
  color: var(--ops-text-placeholder);
}
.ops-time-axis {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  margin: -5px 77px 0 65px;
  font-size: 10px;
  color: var(--ops-text-placeholder);
}
.ops-time-axis span:nth-child(2),
.ops-time-axis span:nth-child(3) {
  text-align: center;
}
.ops-time-axis span:last-child {
  text-align: right;
}
.ops-summary-list {
  margin-top: 5px;
}
.ops-summary-row {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  min-height: 40px;
  font-size: 12px;
  color: var(--ops-text-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ops-summary-row:last-child {
  border-bottom: 0;
}
.ops-summary-row strong {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  font-size: 12px;
  font-weight: 500;
}
.ops-summary-row strong i {
  width: 7px;
  height: 7px;
  background: currentColor;
  border-radius: 50%;
}
.ops-summary-row strong.is-ok {
  color: var(--el-color-success);
}
.ops-summary-row strong.is-warn {
  color: var(--el-color-warning);
}
.ops-storage-section {
  margin-bottom: 24px;
}
.ops-section-title {
  margin-bottom: 12px;
}
.ops-storage-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.ops-storage-card {
  min-width: 0;
  padding: 17px 18px;
}
.ops-storage-head {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 15px;
}
.ops-storage-name {
  display: flex;
  gap: 9px;
  align-items: center;
  min-width: 0;
}
.ops-storage-mark {
  display: grid;
  flex: 0 0 28px;
  place-items: center;
  width: 28px;
  height: 28px;
  font-size: 9px;
  font-weight: 600;
  color: #ffffff;
  border-radius: 6px;
}
.ops-storage-name h3 {
  margin: 0 0 2px;
  font-size: 14px;
  font-weight: 650;
  color: var(--ops-text-primary);
}
.ops-storage-name code,
.ops-route {
  font-family: var(--el-font-family-monospace);
  font-size: 11px;
  color: var(--ops-text-secondary);
}
.ops-storage-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.ops-storage-metrics > div {
  min-width: 0;
  padding-right: 10px;
  border-right: 1px solid var(--el-border-color-lighter);
}
.ops-storage-metrics > div:last-child {
  padding-right: 0;
  border-right: 0;
}
.ops-storage-metrics strong {
  display: block;
  margin-bottom: 2px;
  font-size: 14px;
  font-weight: 650;
  color: var(--ops-text-primary);
}
.ops-storage-metrics span {
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-capacity {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  margin-top: 16px;
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-capacity :deep(.el-progress-bar__outer),
.ops-meter :deep(.el-progress-bar__outer) {
  background: var(--el-fill-color-light);
}
.ops-secondary-grid {
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.9fr);
  margin-bottom: 24px;
}
.ops-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}
.ops-table :deep(.el-table__header-wrapper th.el-table__cell),
.ops-table :deep(.el-table__body-wrapper td.el-table__cell) {
  background: transparent;
}
.ops-table :deep(.el-table__header-wrapper th.el-table__cell) {
  font-size: 11px;
  font-weight: 400;
  color: var(--ops-text-placeholder);
}
.ops-table :deep(.el-table__body-wrapper td.el-table__cell) {
  font-size: 12px;
  color: var(--ops-text-secondary);
}
.ops-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: var(--el-fill-color-light);
}
.ops-node-list {
  display: grid;
  gap: 16px;
  padding-top: 3px;
}
.ops-node-row {
  display: grid;
  grid-template-columns: 95px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
}
.ops-node-row > code {
  font-family: var(--el-font-family-monospace);
  font-size: 11px;
  color: var(--ops-text-secondary);
}
.ops-node-meters {
  display: grid;
  gap: 7px;
}
.ops-meter {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 35px;
  gap: 7px;
  align-items: center;
  font-size: 10px;
  color: var(--ops-text-placeholder);
}
.ops-meter strong {
  font-size: 11px;
  font-weight: 500;
  color: var(--ops-text-secondary);
  text-align: right;
}
.ops-alert-panel {
  margin-bottom: 0;
}
.ops-alert-list {
  display: grid;
}
.ops-alert-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  gap: 9px;
  align-items: start;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ops-alert-row:last-child {
  border-bottom: 0;
}
.ops-alert-row > i {
  width: 7px;
  height: 7px;
  margin-top: 5px;
  background: currentColor;
  border-radius: 50%;
}
.ops-alert-row > i.is-warn {
  color: var(--el-color-warning);
}
.ops-alert-row > i.is-ok {
  color: var(--el-color-success);
}
.ops-alert-row strong,
.ops-alert-row span {
  display: block;
}
.ops-alert-row strong {
  font-size: 12px;
  font-weight: 500;
  color: var(--ops-text-primary);
}
.ops-alert-row span,
.ops-alert-row time {
  margin-top: 2px;
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-alert-row time {
  white-space: nowrap;
}

@media (width <= 1100px) {
  .ops-kpi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .ops-primary-grid,
  .ops-secondary-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 680px) {
  .ops-monitoring-page {
    padding: 16px;
  }
  .ops-page-head {
    flex-direction: column;
    gap: 13px;
    align-items: flex-start;
  }
  .ops-page-meta {
    justify-content: flex-start;
  }
  .ops-kpi-grid,
  .ops-storage-grid {
    grid-template-columns: 1fr;
  }
  .ops-trend-row {
    grid-template-columns: 45px minmax(0, 1fr) 61px;
    gap: 7px;
  }
  .ops-trend-bars {
    gap: 3px;
  }
  .ops-time-axis {
    margin-right: 69px;
    margin-left: 52px;
  }
  .ops-node-row {
    grid-template-columns: 82px minmax(0, 1fr);
    gap: 7px;
  }
  .ops-alert-row {
    grid-template-columns: 8px minmax(0, 1fr);
  }
  .ops-alert-row time {
    grid-column: 2;
    margin-top: -5px;
  }
}
</style>
