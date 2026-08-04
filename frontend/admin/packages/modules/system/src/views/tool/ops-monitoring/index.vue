<template>
  <div class="app-container ops-monitoring-page">
    <section class="ops-page-head">
      <div>
        <div class="ops-breadcrumb">开发工具 <span>/</span> 运维监控</div>
        <h1>运维监控</h1>
        <p>生产环境 · API 网关与基础设施运行概览</p>
      </div>
      <div class="ops-page-meta">
        <el-tag type="warning" effect="plain">演示数据 · 待接入指标接口</el-tag>
        <span>当前窗口：近 15 分钟</span>
        <span>最后采样 10:42:18</span>
      </div>
    </section>

    <section class="ops-kpi-grid" aria-label="核心指标">
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
              <h2>请求量与延迟趋势</h2>
              <p>每 1 分钟聚合 · 近 15 分钟</p>
            </div>
            <el-tag size="small" type="info" effect="plain">静态示例</el-tag>
          </div>
        </template>
        <div class="ops-legend">
          <span><i class="ops-legend-mark is-teal" />QPS</span>
          <span><i class="ops-legend-mark is-amber" />P95 延迟</span>
        </div>
        <div class="ops-trend-row">
          <span class="ops-trend-label">QPS</span>
          <div class="ops-trend-bars" aria-label="QPS 趋势，从 92 req/s 上升到 128.4 req/s">
            <i v-for="(point, index) in qpsTrend" :key="`qps-${index}`" class="is-teal" :style="{ height: `${point}%` }" />
          </div>
          <strong>128.4<small>req/s</small></strong>
        </div>
        <div class="ops-trend-row">
          <span class="ops-trend-label">P95</span>
          <div class="ops-trend-bars" aria-label="P95 延迟趋势，从 182 ms 上升到 246 ms">
            <i
              v-for="(point, index) in latencyTrend"
              :key="`latency-${index}`"
              class="is-amber"
              :style="{ height: `${point}%` }"
            />
          </div>
          <strong>246<small>ms</small></strong>
        </div>
        <div class="ops-time-axis"><span>10:28</span><span>10:32</span><span>10:37</span><span>10:42</span></div>
      </el-card>

      <el-card class="ops-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>服务可用性</h2>
              <p>探针与依赖连接状态</p>
            </div>
            <el-tag type="success" effect="plain">3 / 3 在线</el-tag>
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
        <h2>数据存储</h2>
        <span>连接、性能、容量</span>
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
            <el-tag type="success" size="small" effect="plain">正常</el-tag>
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
              <h2>接口与响应</h2>
              <p>按请求量排序 · 近 15 分钟</p>
            </div>
            <span class="ops-muted">4 个接口</span>
          </div>
        </template>
        <el-table :data="endpointItems" size="small" class="ops-table">
          <el-table-column label="接口" min-width="230">
            <template #default="{ row }"
              ><code class="ops-route">{{ row.route }}</code></template
            >
          </el-table-column>
          <el-table-column prop="qps" label="QPS" width="75" />
          <el-table-column prop="latency" label="P95" width="85" />
          <el-table-column prop="errorRate" label="错误率" width="85" />
          <el-table-column label="状态" width="85" align="right">
            <template #default="{ row }"
              ><el-tag :type="row.status === '正常' ? 'success' : 'warning'" size="small" effect="plain">{{
                row.status
              }}</el-tag></template
            >
          </el-table-column>
        </el-table>
      </el-card>

      <el-card class="ops-panel" shadow="never">
        <template #header>
          <div class="ops-section-head">
            <div>
              <h2>实例资源</h2>
              <p>当前值 / 最近采样</p>
            </div>
            <span class="ops-muted">3 个实例</span>
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
            <h2>告警事件</h2>
            <p>需要运维关注的近期事件</p>
          </div>
          <el-tag type="warning" effect="plain">未解决 2</el-tag>
        </div>
      </template>
      <div class="ops-alert-list">
        <div v-for="alert in alertItems" :key="alert.title" class="ops-alert-row">
          <i :class="`is-${alert.tone}`" />
          <div>
            <strong>{{ alert.title }}</strong
            ><span>{{ alert.detail }}</span>
          </div>
          <time>{{ alert.time }}</time>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
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
  status: "正常" | "关注";
}

/** 实例资源指标。 */
interface NodeItem {
  /** 实例名称。 */
  name: string;
  /** CPU 与内存指标。 */
  metrics: Array<{ label: string; value: number; color: string }>;
}

const kpiItems: KpiItem[] = [
  {
    label: "请求吞吐 QPS",
    value: "128.4 req/s",
    change: "↗ 12.6%",
    context: "较上一周期",
    tone: "is-up",
    color: "var(--el-color-primary)"
  },
  {
    label: "P95 延迟",
    value: "246 ms",
    change: "↗ 8 ms",
    context: "较上一周期",
    tone: "is-down",
    color: "var(--el-color-warning)"
  },
  {
    label: "错误率",
    value: "0.18%",
    change: "↓ 0.04%",
    context: "5xx 占比 0.12%",
    tone: "is-up",
    color: "var(--el-color-danger)"
  },
  { label: "可用性", value: "99.98%", change: "稳定", context: "过去 30 天", tone: "is-up", color: "var(--el-color-success)" }
];

const qpsTrend = [43, 51, 47, 58, 55, 67, 63, 77, 71, 87, 81, 91];
const latencyTrend = [51, 46, 56, 48, 62, 55, 66, 61, 73, 69, 78, 82];

const availabilityItems = [
  { label: "HTTP / HTTPS", value: "正常", tone: "ok" },
  { label: "实例在线", value: "3 / 3", tone: "ok" },
  { label: "MySQL 连接", value: "正常", tone: "ok" },
  { label: "Redis 连接", value: "正常", tone: "ok" },
  { label: "后台任务", value: "1 项延迟", tone: "warn" }
];

const storageItems = [
  {
    name: "MySQL · primary",
    shortName: "SQL",
    address: "mysql-prod-01:3306",
    color: "var(--el-color-primary)",
    capacityLabel: "连接池",
    capacity: 24,
    metrics: [
      { value: "48 / 200", label: "活跃连接" },
      { value: "18 ms", label: "查询 P95" },
      { value: "3", label: "慢查询 / 15m" }
    ]
  },
  {
    name: "Redis · cache",
    shortName: "RED",
    address: "redis-prod-01:6379",
    color: "var(--el-color-success)",
    capacityLabel: "内存",
    capacity: 26,
    metrics: [
      { value: "98.7%", label: "缓存命中率" },
      { value: "1,245/s", label: "命令 OPS" },
      { value: "2.1 / 8 GB", label: "内存使用" }
    ]
  }
];

const endpointItems: EndpointItem[] = [
  { route: "/api/auth/profile", qps: "64.2", latency: "182 ms", errorRate: "0.02%", status: "正常" },
  { route: "/api/project-doc/tree", qps: "31.4", latency: "284 ms", errorRate: "0.42%", status: "关注" },
  { route: "/api/ai/chat", qps: "18.7", latency: "612 ms", errorRate: "1.80%", status: "关注" },
  { route: "/api/base/menu/list", qps: "13.9", latency: "144 ms", errorRate: "0.00%", status: "正常" }
];

const nodeItems: NodeItem[] = [
  {
    name: "backend-01",
    metrics: [
      { label: "CPU", value: 42, color: "var(--el-color-primary)" },
      { label: "内存", value: 63, color: "var(--el-color-success)" }
    ]
  },
  {
    name: "backend-02",
    metrics: [
      { label: "CPU", value: 68, color: "var(--el-color-warning)" },
      { label: "内存", value: 71, color: "var(--el-color-warning)" }
    ]
  },
  {
    name: "worker-01",
    metrics: [
      { label: "CPU", value: 29, color: "var(--el-color-primary)" },
      { label: "内存", value: 48, color: "var(--el-color-success)" }
    ]
  }
];

const alertItems = [
  { title: "AI Chat P95 延迟超过 500 ms", detail: "/api/ai/chat · 当前 612 ms", time: "2 分钟前", tone: "warn" },
  { title: "定时任务 queue-retry 延迟执行", detail: "worker-01 · 延迟 3 分钟", time: "8 分钟前", tone: "warn" },
  { title: "MySQL 慢查询告警已恢复", detail: "持续 4 分钟 · 峰值 1.2 s", time: "18 分钟前", tone: "ok" }
];
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
