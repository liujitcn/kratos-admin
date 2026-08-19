<template>
  <div class="app-container ops-monitoring-page">
    <section class="ops-page-head">
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
        <div class="ops-trend-plot-row">
          <div class="ops-trend-chart-shell">
            <div class="ops-trend-scale is-left" aria-hidden="true">
              <span>{{ formatNumber(trendChart.qpsMax) }}<small>req/s</small></span>
              <span>0</span>
            </div>
            <svg
              class="ops-trend-chart"
              viewBox="0 0 1000 220"
              preserveAspectRatio="none"
              role="img"
              :aria-label="trendAriaLabel"
            >
              <line
                v-for="line in trendGridLines"
                :key="`trend-grid-${line}`"
                class="ops-trend-grid-line"
                x1="0"
                :y1="line"
                x2="1000"
                :y2="line"
              />
              <polygon v-if="qpsArea" :points="qpsArea" class="ops-trend-area is-teal" />
              <polygon v-if="latencyArea" :points="latencyArea" class="ops-trend-area is-amber" />
              <polyline v-if="qpsLine" :points="qpsLine" class="ops-trend-line is-teal" />
              <polyline v-if="latencyLine" :points="latencyLine" class="ops-trend-line is-amber" />
              <g v-for="point in trendPoints" :key="`trend-point-${point.index}`">
                <circle :cx="point.x" :cy="point.qpsY" r="4" class="ops-trend-point is-teal">
                  <title>{{ formatTrendPoint(point) }}</title>
                </circle>
                <circle :cx="point.x" :cy="point.latencyY" r="4" class="ops-trend-point is-amber">
                  <title>{{ formatTrendPoint(point) }}</title>
                </circle>
              </g>
            </svg>
            <div class="ops-trend-scale is-right" aria-hidden="true">
              <span>{{ formatNumber(trendChart.latencyMax) }}<small>ms</small></span>
              <span>0</span>
            </div>
          </div>
          <div class="ops-trend-current" aria-hidden="true">
            <div>
              <span class="is-teal">QPS</span>
              <strong>{{ formatNumber(trafficSummary?.qps) }}<small>req/s</small></strong>
            </div>
            <div>
              <span class="is-amber">P95</span>
              <strong>{{ formatNumber(trafficSummary?.p95_latency_ms) }}<small>ms</small></strong>
            </div>
          </div>
        </div>
        <div class="ops-time-axis">
          <span v-for="point in trendAxis" :key="point">{{ point }}</span>
        </div>
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
            <el-tag :type="item.status === '正常' ? 'success' : 'warning'" size="small" effect="plain">{{
              item.statusLabel
            }}</el-tag>
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
                <div class="ops-meter-head">
                  <span class="ops-meter-label">{{ metric.label }}</span>
                  <span class="ops-meter-stats">
                    <span>
                      {{ t("system.ops_monitoring.node_metric.current") }} {{ metric.usedLabel }} /
                      {{ t("system.ops_monitoring.node_metric.total") }} {{ metric.totalLabel }}
                    </span>
                    <strong>{{ metric.percentageLabel }}</strong>
                  </span>
                </div>
                <el-progress
                  :percentage="metric.percentage"
                  :show-text="false"
                  :stroke-width="10"
                  :color="metric.color"
                />
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
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from "vue";
import { defOpsMonitoringService } from "@liujitcn/kratos-admin-system/api/system/ops_monitoring";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";
import type {
  OpsAlert,
  OpsEndpoint,
  OpsNode,
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

/** 实例资源单项指标。 */
interface NodeMetricItem {
  /** 指标名称。 */
  label: string;
  /** 指标使用百分比。 */
  percentage: number;
  /** 两位小数百分比。 */
  percentageLabel: string;
  /** 当前已用容量。 */
  usedLabel: string;
  /** 总容量。 */
  totalLabel: string;
  /** 进度条颜色。 */
  color: string;
}

/** 实例资源信息。 */
interface NodeItem {
  /** 实例名称。 */
  name: string;
  /** 实例资源指标。 */
  metrics: NodeMetricItem[];
}

/** 折线图中的流量趋势点。 */
interface TrendPoint {
  /** 点在图表中的序号。 */
  index: number;
  /** 时间标签。 */
  time: string;
  /** 每秒请求数。 */
  qps: number;
  /** P95 响应延迟。 */
  latency: number;
  /** 横坐标。 */
  x: number;
  /** QPS 折线纵坐标。 */
  qpsY: number;
  /** P95 折线纵坐标。 */
  latencyY: number;
}

/** 折线图数据和双 Y 轴刻度。 */
interface TrendChart {
  /** 折线图数据点。 */
  points: TrendPoint[];
  /** QPS 轴最大值。 */
  qpsMax: number;
  /** P95 轴最大值。 */
  latencyMax: number;
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
const traffic = ref<OpsTrafficResponse>();
const services = ref<OpsServicesResponse>();
const storage = ref<OpsStorageResponse>();
const endpoints = ref<OpsEndpointsResponse>();
const nodes = ref<OpsNodesResponse>();
const alerts = ref<OpsAlertsResponse>();
const windowMinutes = ref(15);
const lastCollectedAt = ref("");
const trendGridLines = [16, 62, 108, 154, 198];
const nodeMetricLabelKeys: Record<string, string> = {
  堆内存: "system.ops_monitoring.node_metric.heap_memory",
  内存: "system.ops_monitoring.node_metric.memory",
  硬盘: "system.ops_monitoring.node_metric.disk"
};
const byteUnits = ["B", "KB", "MB", "GB", "TB", "PB"];
const trafficSummary = computed(() => traffic.value?.traffic);

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

const trendChart = computed<TrendChart>(() => {
  const source = traffic.value?.points ?? [];
  const rawPoints = source.length
    ? source.map(point => ({
        at: point.at,
        qps: Math.max(0, Number.isFinite(point.qps) ? point.qps : 0),
        latency: Math.max(0, Number.isFinite(point.p95_latency_ms) ? point.p95_latency_ms : 0)
      }))
    : Array.from({ length: 4 }, () => ({ at: "", qps: 0, latency: 0 }));
  const qpsMax = Math.max(...rawPoints.map(point => point.qps), 0);
  const latencyMax = Math.max(...rawPoints.map(point => point.latency), 0);
  const plotHeight = 198 - 16;

  return {
    qpsMax,
    latencyMax,
    points: rawPoints.map((point, index) => ({
      index,
      time: formatClock(point.at),
      qps: point.qps,
      latency: point.latency,
      x: rawPoints.length === 1 ? 500 : (index / (rawPoints.length - 1)) * 1000,
      qpsY: 198 - (qpsMax ? (point.qps / qpsMax) * plotHeight : 0),
      latencyY: 198 - (latencyMax ? (point.latency / latencyMax) * plotHeight : 0)
    }))
  };
});
const trendPoints = computed(() => trendChart.value.points);
const qpsLine = computed(() => trendPoints.value.map(point => `${point.x},${point.qpsY}`).join(" "));
const latencyLine = computed(() => trendPoints.value.map(point => `${point.x},${point.latencyY}`).join(" "));
const qpsArea = computed(() => (qpsLine.value ? `${qpsLine.value} 1000,198 0,198` : ""));
const latencyArea = computed(() => (latencyLine.value ? `${latencyLine.value} 1000,198 0,198` : ""));
const trendAriaLabel = computed(
  () =>
    `${t("system.ops_monitoring.qps_trend", { value: `${formatNumber(trafficSummary.value?.qps)} req/s` })}; ${t(
      "system.ops_monitoring.p95_trend",
      { value: `${formatNumber(trafficSummary.value?.p95_latency_ms)} ms` }
    )}`
);
const trendAxis = computed(() => {
  if (!traffic.value?.points.length) return [];
  const lastIndex = trendPoints.value.length - 1;
  return Array.from(new Set([0, Math.floor(lastIndex / 2), lastIndex])).map(index => trendPoints.value[index].time);
});

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

let monitoringTimer: number | undefined;
let monitoringActive = false;
let monitoringRun = 0;

async function loadMonitoring() {
  loading.value = true;
  const results = await Promise.allSettled([
    defOpsMonitoringService.GetOpsTraffic({ window_minutes: windowMinutes.value }),
    defOpsMonitoringService.GetOpsServices({}),
    defOpsMonitoringService.GetOpsStorage({}),
    defOpsMonitoringService.GetOpsEndpoints({ window_minutes: windowMinutes.value }),
    defOpsMonitoringService.GetOpsNodes({}),
    defOpsMonitoringService.GetOpsAlerts({ window_minutes: windowMinutes.value })
  ]);
  const [trafficResult, servicesResult, storageResult, endpointsResult, nodesResult, alertsResult] = results;
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

async function startMonitoring() {
  if (monitoringActive) return;
  monitoringActive = true;
  const run = ++monitoringRun;
  await loadMonitoring();
  if (!monitoringActive || run !== monitoringRun) return;
  monitoringTimer = window.setInterval(() => {
    if (!loading.value) void loadMonitoring();
  }, 5000);
}

function stopMonitoring() {
  monitoringActive = false;
  monitoringRun += 1;
  if (monitoringTimer === undefined) return;
  window.clearInterval(monitoringTimer);
  monitoringTimer = undefined;
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

/** 格式化折线图悬浮提示内容。 */
function formatTrendPoint(point: TrendPoint) {
  return `${point.time || "--:--"} / QPS ${formatNumber(point.qps)} req/s / P95 ${formatNumber(point.latency)} ms`;
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
    metrics: (node.metrics ?? []).map(metric => {
      const percentage = Math.max(0, Math.min(metric.value, 100));
      return {
        label: translateNodeMetricLabel(metric.label),
        percentage,
        percentageLabel: `${formatFixedNumber(percentage)}%`,
        usedLabel: formatBytes(metric.used_bytes),
        totalLabel: formatBytes(metric.total_bytes),
        color: percentage >= 70 ? "var(--el-color-warning)" : "var(--el-color-primary)"
      };
    })
  };
}

/** 将字节数格式化为保留两位小数的可读容量。 */
function formatBytes(value: number) {
  if (!Number.isFinite(value) || value < 0) return "--";
  let unitIndex = 0;
  let displayValue = value;
  while (displayValue >= 1024 && unitIndex < byteUnits.length - 1) {
    displayValue /= 1024;
    unitIndex += 1;
  }
  return `${formatFixedNumber(displayValue)} ${byteUnits[unitIndex]}`;
}

/** 将数值格式化为固定两位小数。 */
function formatFixedNumber(value: number) {
  return new Intl.NumberFormat(getCurrentLocale(), {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value);
}

/** 翻译实例资源指标名称，未知指标保留后端原值。 */
function translateNodeMetricLabel(label: string) {
  const localeKey = nodeMetricLabelKeys[label];
  return localeKey ? t(localeKey) : label;
}

function mapAlert(alert: OpsAlert) {
  return {
    title: alert.title,
    detail: alert.detail,
    at: alert.at,
    tone: alert.status === "正常" ? "ok" : "warn"
  };
}

onMounted(startMonitoring);
onActivated(startMonitoring);
onDeactivated(stopMonitoring);
onBeforeUnmount(stopMonitoring);
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
  padding: 0;
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
.ops-page-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
  margin-left: auto;
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
  min-width: 0;
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
.ops-legend-mark.is-teal {
  background: var(--el-color-primary);
}
.ops-legend-mark.is-amber {
  background: var(--el-color-warning);
}
.ops-trend-plot-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 72px;
  gap: 10px;
  align-items: stretch;
}
.ops-trend-chart-shell {
  display: grid;
  grid-template-columns: 54px minmax(0, 1fr) 48px;
  gap: 8px;
  min-width: 0;
  min-height: 220px;
}
.ops-trend-scale {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 0;
  padding: 4px 0 18px;
  font-size: 10px;
  line-height: 1.2;
  color: var(--ops-text-placeholder);
}
.ops-trend-scale span {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ops-trend-scale small {
  font-size: 9px;
  color: var(--ops-text-placeholder);
}
.ops-trend-scale.is-right {
  text-align: left;
}
.ops-trend-chart {
  display: block;
  width: 100%;
  height: 220px;
  overflow: visible;
}
.ops-trend-grid-line {
  stroke: var(--el-border-color-lighter);
  stroke-dasharray: 4 5;
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
.ops-trend-area {
  opacity: 0.08;
}
.ops-trend-area.is-teal {
  fill: var(--el-color-primary);
}
.ops-trend-area.is-amber {
  fill: var(--el-color-warning);
}
.ops-trend-line {
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 3;
  vector-effect: non-scaling-stroke;
}
.ops-trend-line.is-teal,
.ops-trend-point.is-teal {
  stroke: var(--el-color-primary);
}
.ops-trend-line.is-amber,
.ops-trend-point.is-amber {
  stroke: var(--el-color-warning);
}
.ops-trend-point {
  fill: var(--ops-card-bg);
  stroke-width: 2;
  vector-effect: non-scaling-stroke;
}
.ops-trend-current {
  display: grid;
  align-content: center;
  gap: 22px;
  min-width: 0;
}
.ops-trend-current > div {
  min-width: 0;
}
.ops-trend-current span {
  display: block;
  margin-bottom: 4px;
  font-size: 10px;
  font-weight: 550;
}
.ops-trend-current span.is-teal {
  color: var(--el-color-primary);
}
.ops-trend-current span.is-amber {
  color: var(--el-color-warning);
}
.ops-trend-current strong {
  display: block;
  overflow: hidden;
  font-size: 14px;
  font-weight: 650;
  color: var(--ops-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ops-trend-current small {
  margin-left: 3px;
  font-size: 9px;
  font-weight: 400;
  color: var(--ops-text-placeholder);
}
.ops-time-axis {
  display: flex;
  justify-content: space-between;
  margin: -1px 80px 0 62px;
  font-size: 10px;
  color: var(--ops-text-placeholder);
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
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.9fr);
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
  min-width: 0;
  padding-top: 3px;
}
.ops-node-row {
  display: grid;
  grid-template-columns: minmax(110px, 0.24fr) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
  min-width: 0;
}
.ops-node-row > code {
  overflow: hidden;
  font-family: var(--el-font-family-monospace);
  font-size: 11px;
  color: var(--ops-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ops-node-meters {
  display: grid;
  gap: 13px;
  min-width: 0;
}
.ops-meter {
  display: grid;
  gap: 7px;
  min-width: 0;
  font-size: 11px;
  color: var(--ops-text-placeholder);
}
.ops-meter-head,
.ops-meter-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
}
.ops-meter-label {
  flex: none;
  font-weight: 600;
  color: var(--ops-text-secondary);
}
.ops-meter-stats {
  flex: 1;
  justify-content: flex-end;
  text-align: right;
}
.ops-meter-stats > span {
  overflow-wrap: anywhere;
}
.ops-meter-stats strong {
  flex: none;
  font-size: 11px;
  font-weight: 650;
  color: var(--ops-text-secondary);
  font-variant-numeric: tabular-nums;
}
.ops-meter :deep(.el-progress),
.ops-meter :deep(.el-progress-bar) {
  min-width: 0;
  width: 100%;
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
  .ops-storage-grid,
  .ops-secondary-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 680px) {
  .ops-page-head {
    flex-direction: column;
    gap: 13px;
    align-items: flex-start;
  }
  .ops-page-meta {
    justify-content: flex-start;
    margin-left: 0;
  }
  .ops-kpi-grid,
  .ops-storage-grid {
    grid-template-columns: 1fr;
  }
  .ops-trend-plot-row {
    grid-template-columns: 1fr;
    gap: 14px;
  }
  .ops-trend-current {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
    padding-top: 12px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .ops-time-axis {
    margin-right: 56px;
    margin-left: 54px;
  }
  .ops-node-row {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .ops-meter-stats {
    flex-basis: 100%;
    justify-content: flex-start;
    text-align: left;
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
