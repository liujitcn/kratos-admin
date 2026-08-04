<template>
  <div v-loading="loading" class="app-container ops-monitoring-page">
    <section class="ops-hero">
      <div class="ops-hero__content">
        <span class="ops-eyebrow"
          ><el-icon><Monitor /></el-icon> OPERATIONS CENTER</span
        >
        <h1>项目运维监控</h1>
        <p>基于当前项目构建期采集的数据，查看文档覆盖、模块状态和服务探针。</p>
      </div>
      <div class="ops-hero__actions">
        <el-button :icon="Refresh" :loading="loading" @click="loadOverview">刷新采集</el-button>
        <el-button type="primary" :icon="Reading" @click="openProjectDocuments">查看项目文档</el-button>
      </div>
    </section>

    <section class="ops-status-band" aria-live="polite">
      <div class="ops-status-band__main">
        <span class="status-dot" :class="`is-${collectionStatus}`" />
        <strong>{{ collectionStatusLabel }}</strong>
        <span class="ops-status-band__muted">{{ lastRefreshLabel }}</span>
      </div>
      <div class="ops-status-band__source">数据源 <code>ProjectDocumentService</code></div>
    </section>

    <section class="metric-grid" aria-label="采集概览">
      <article class="metric-card">
        <div class="metric-card__icon is-blue"><FolderOpened /></div>
        <div>
          <span class="metric-card__label">项目数量</span>
          <strong class="metric-card__value">{{ projectCount }}</strong>
          <span class="metric-card__hint">已注册项目文档源</span>
        </div>
      </article>
      <article class="metric-card">
        <div class="metric-card__icon is-green"><Document /></div>
        <div>
          <span class="metric-card__label">采集文档</span>
          <strong class="metric-card__value">{{ documentCount }}</strong>
          <span class="metric-card__hint">Markdown 文档条目</span>
        </div>
      </article>
      <article class="metric-card">
        <div class="metric-card__icon is-purple"><Connection /></div>
        <div>
          <span class="metric-card__label">目录节点</span>
          <strong class="metric-card__value">{{ directoryCount }}</strong>
          <span class="metric-card__hint">可追踪模块目录</span>
        </div>
      </article>
      <article class="metric-card">
        <div class="metric-card__icon is-orange"><Promotion /></div>
        <div>
          <span class="metric-card__label">最近采集</span>
          <strong class="metric-card__value metric-card__value--date">{{ latestDocumentLabel }}</strong>
          <span class="metric-card__hint">{{ latestDocumentPath || "等待文档数据" }}</span>
        </div>
      </article>
    </section>

    <section class="ops-grid ops-grid--primary">
      <el-card class="ops-card health-card" shadow="never">
        <template #header>
          <div class="ops-card__header">
            <div>
              <span class="ops-section-label">SERVICE HEALTH</span>
              <h2>服务探针</h2>
              <p>优先读取当前部署暴露的存活与就绪端点。</p>
            </div>
            <el-tag :type="healthSummaryType" effect="plain">{{ healthSummaryLabel }}</el-tag>
          </div>
        </template>
        <div class="probe-list">
          <div v-for="probe in probes" :key="probe.path" class="probe-row">
            <div class="probe-row__name">
              <span class="status-dot" :class="`is-${probe.status}`" />
              <strong>{{ probe.name }}</strong>
              <code>{{ probe.path }}</code>
            </div>
            <span class="probe-row__message">{{ probe.message }}</span>
          </div>
        </div>
      </el-card>

      <el-card class="ops-card coverage-card" shadow="never">
        <template #header>
          <div class="ops-card__header">
            <div>
              <span class="ops-section-label">PROJECT COVERAGE</span>
              <h2>项目覆盖</h2>
              <p>按当前文档树识别的主要运行模块。</p>
            </div>
            <el-button link type="primary" @click="openProjectDocuments">打开目录</el-button>
          </div>
        </template>
        <div class="module-list">
          <div v-for="module in modules" :key="module.key" class="module-row">
            <div class="module-row__identity">
              <span class="module-row__mark"><component :is="module.icon" /></span>
              <div>
                <strong>{{ module.name }}</strong>
                <span>{{ module.description }}</span>
              </div>
            </div>
            <div class="module-row__meta">
              <strong>{{ module.documentCount }}</strong>
              <span>份文档</span>
            </div>
          </div>
        </div>
      </el-card>
    </section>

    <section class="ops-grid ops-grid--secondary">
      <el-card class="ops-card activity-card" shadow="never">
        <template #header>
          <div class="ops-card__header">
            <div>
              <span class="ops-section-label">RECENT COLLECTIONS</span>
              <h2>最近采集</h2>
              <p>按更新时间排序，帮助定位最近发生变化的文档。</p>
            </div>
            <el-tag type="info" effect="plain">{{ recentDocuments.length }} 条</el-tag>
          </div>
        </template>
        <el-table v-if="recentDocuments.length" :data="recentDocuments" size="small" class="collection-table">
          <el-table-column label="文档" min-width="260">
            <template #default="{ row }">
              <div class="document-cell">
                <Document class="document-cell__icon" />
                <span :title="row.path">{{ row.path }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="projectName" label="项目" width="120" />
          <el-table-column label="更新时间" width="170" align="right">
            <template #default="{ row }">{{ formatDocumentDate(row.updated_at) }}</template>
          </el-table-column>
        </el-table>
        <el-empty v-else :image-size="64" description="暂无已采集文档" />
      </el-card>

      <el-card class="ops-card environment-card" shadow="never">
        <template #header>
          <div class="ops-card__header">
            <div>
              <span class="ops-section-label">RUNTIME CONTEXT</span>
              <h2>运行上下文</h2>
              <p>来自当前项目 README 的环境声明。</p>
            </div>
            <el-button link type="primary" @click="openApiDocuments">API 文档</el-button>
          </div>
        </template>
        <div class="environment-list">
          <div v-for="item in environmentItems" :key="item.label" class="environment-row">
            <span class="environment-row__label">{{ item.label }}</span>
            <div class="environment-row__value">
              <strong>{{ item.value }}</strong>
              <span>{{ item.detail }}</span>
            </div>
            <el-tag size="small" :type="item.available ? 'success' : 'info'" effect="plain">
              {{ item.available ? "已采集" : "未声明" }}
            </el-tag>
          </div>
        </div>
        <div class="environment-note">
          <Warning />
          运行时请求、错误和任务指标请结合系统日志与后端健康端点判断。
          <el-button link type="primary" @click="openSystemLogs">查看系统日志</el-button>
        </div>
      </el-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Connection, Document, FolderOpened, Monitor, Promotion, Reading, Refresh, Warning } from "@element-plus/icons-vue";
import { defProjectDocumentService } from "@liujitcn/kratos-admin-system/api/system/project_document";
import type {
  ProjectDocumentDirectory,
  ProjectDocumentListItem,
  ProjectDocumentProject
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/project_document";

defineOptions({
  name: "OpsMonitoring",
  inheritAttrs: false
});

/** 服务探针状态。 */
type ProbeStatus = "checking" | "healthy" | "unavailable";

/** 项目服务探针。 */
interface ProbeState {
  /** 探针名称。 */
  name: string;
  /** 探针路径。 */
  path: string;
  /** 当前探针状态。 */
  status: ProbeStatus;
  /** 状态说明。 */
  message: string;
}

/** 页面展示的项目模块摘要。 */
interface ModuleSummary {
  /** 模块稳定标识。 */
  key: string;
  /** 模块名称。 */
  name: string;
  /** 模块说明。 */
  description: string;
  /** 模块图标。 */
  icon: typeof Connection;
  /** 模块文档数量。 */
  documentCount: number;
}

/** 运行上下文展示项。 */
interface EnvironmentItem {
  /** 展示名称。 */
  label: string;
  /** 采集到的值。 */
  value: string;
  /** 补充说明。 */
  detail: string;
  /** 是否从项目资料中识别到。 */
  available: boolean;
}

/** 展示用的文档行。 */
interface RecentDocument extends ProjectDocumentListItem {
  /** 所属项目名称。 */
  projectName: string;
}

const router = useRouter();
const loading = ref(false);
const collectionReady = ref(false);
const lastRefreshAt = ref("");
const projects = ref<ProjectDocumentProject[]>([]);
const rootReadme = ref("");
const probes = ref<ProbeState[]>([
  { name: "存活检查", path: "/healthz", status: "checking", message: "检测中" },
  { name: "就绪检查", path: "/readyz", status: "checking", message: "检测中" }
]);

const allDocuments = computed<RecentDocument[]>(() => {
  const documents: RecentDocument[] = [];
  projects.value.forEach(project => {
    collectProjectDocuments(project.documents, project.name, documents);
    collectDirectories(project.directories, project.name, documents);
  });
  return documents.sort((left, right) => right.updated_at.localeCompare(left.updated_at));
});
const recentDocuments = computed(() => allDocuments.value.slice(0, 8));
const projectCount = computed(() => projects.value.length);
const documentCount = computed(() => allDocuments.value.length);
const directoryCount = computed(() =>
  projects.value.reduce((count, project) => count + countDirectories(project.directories), 0)
);
const latestDocument = computed(() => allDocuments.value[0]);
const latestDocumentPath = computed(() => latestDocument.value?.path ?? "");
const latestDocumentLabel = computed(() => (latestDocument.value ? formatDocumentDate(latestDocument.value.updated_at) : "--"));
const collectionStatus = computed(() => (collectionReady.value ? "healthy" : "unavailable"));
const collectionStatusLabel = computed(() => (collectionReady.value ? "采集服务正常" : "采集服务不可用"));
const lastRefreshLabel = computed(() =>
  lastRefreshAt.value ? `最近刷新 ${formatDocumentDate(lastRefreshAt.value)}` : "等待刷新"
);
const healthSummary = computed(() => {
  const healthyCount = probes.value.filter(probe => probe.status === "healthy").length;
  const checkingCount = probes.value.filter(probe => probe.status === "checking").length;
  return { healthyCount, checkingCount };
});
const healthSummaryType = computed(() => {
  if (healthSummary.value.checkingCount) return "info";
  return healthSummary.value.healthyCount === probes.value.length ? "success" : "warning";
});
const healthSummaryLabel = computed(() => {
  if (healthSummary.value.checkingCount) return "检测中";
  if (healthSummary.value.healthyCount === probes.value.length) return "探针正常";
  if (healthSummary.value.healthyCount) return "部分可用";
  return "待接入";
});
const modules = computed<ModuleSummary[]>(() => {
  const definitions = [
    { key: "backend", name: "Backend", description: "Go + Kratos 服务、Proto、迁移", icon: Connection },
    { key: "admin", name: "Admin", description: "Vue 管理后台与 System 模块", icon: Monitor },
    { key: "uni-app", name: "uni-app", description: "应用端 H5 与微信小程序", icon: Promotion },
    { key: "taro-app", name: "Taro", description: "React / Taro 应用底座", icon: Reading },
    { key: "docs", name: "Docs", description: "架构、流程与专题说明", icon: Document }
  ] as const;

  return definitions.map(definition => ({
    ...definition,
    documentCount: allDocuments.value.filter(
      document => document.path === `${definition.key}/README.md` || document.path.startsWith(`${definition.key}/`)
    ).length
  }));
});
const environmentItems = computed<EnvironmentItem[]>(() => {
  const content = rootReadme.value;
  const backendAddress = readEnvironmentValue(content, "后端 HTTP") || "http://localhost:7001";
  const grpcAddress = readEnvironmentValue(content, "后端 gRPC") || "localhost:6001";
  return [
    { label: "Backend HTTP", value: backendAddress, detail: "服务端默认 HTTP 地址", available: content.includes("后端 HTTP") },
    { label: "Backend gRPC", value: grpcAddress, detail: "服务端默认 gRPC 地址", available: content.includes("后端 gRPC") },
    { label: "MySQL", value: "已声明", detail: "默认运行依赖", available: content.includes("MySQL") },
    { label: "Redis", value: "已声明", detail: "默认运行依赖", available: content.includes("Redis") }
  ];
});

/** 加载项目文档目录和根 README。 */
async function loadOverview() {
  loading.value = true;
  probes.value = probes.value.map(probe => ({ ...probe, status: "checking", message: "检测中" }));
  try {
    const response = await defProjectDocumentService.TreeProjectDocument({});
    projects.value = response.projects ?? [];
    collectionReady.value = true;
    lastRefreshAt.value = new Date().toISOString();
    const readme = allDocuments.value.find(document => document.path === "README.md");
    if (readme) {
      const detail = await defProjectDocumentService.GetProjectDocument({ id: readme.id });
      rootReadme.value = detail.content;
    }
    await Promise.all(probes.value.map(checkProbe));
  } catch {
    collectionReady.value = false;
    ElMessage.error("项目采集信息加载失败");
  } finally {
    loading.value = false;
  }
}

/** 检查单个后端健康探针。 */
async function checkProbe(probe: ProbeState) {
  try {
    const response = await fetch(probe.path, { credentials: "include" });
    probe.status = response.status === 204 || response.ok ? "healthy" : "unavailable";
    probe.message = probe.status === "healthy" ? "响应正常" : `HTTP ${response.status}`;
  } catch {
    probe.status = "unavailable";
    probe.message = "当前页面未接入后端探针";
  }
}

/** 递归收集项目根目录下的文档。 */
function collectProjectDocuments(items: ProjectDocumentListItem[], projectName: string, target: RecentDocument[]) {
  items.forEach(item => target.push({ ...item, projectName }));
}

/** 递归收集目录下的文档。 */
function collectDirectories(directories: ProjectDocumentDirectory[], projectName: string, target: RecentDocument[]) {
  directories.forEach(directory => {
    collectProjectDocuments(directory.documents, projectName, target);
    collectDirectories(directory.directories, projectName, target);
  });
}

/** 计算项目文档目录数量。 */
function countDirectories(directories: ProjectDocumentDirectory[]): number {
  return directories.reduce((count, directory) => count + 1 + countDirectories(directory.directories), 0);
}

/** 从 README 的环境表中读取地址值。 */
function readEnvironmentValue(content: string, label: string) {
  const line = content.split("\n").find(item => item.includes(`| ${label} |`));
  return line?.split("|")[2]?.trim() ?? "";
}

/** 格式化文档时间。 */
function formatDocumentDate(value: string) {
  if (!value) return "--";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}

/** 打开项目文档页面。 */
function openProjectDocuments() {
  void router.push("/project-doc");
}

/** 打开 API 文档页面。 */
function openApiDocuments() {
  void router.push("/api-doc");
}

/** 打开系统日志页面。 */
function openSystemLogs() {
  void router.push("/base/log");
}

onMounted(() => {
  void loadOverview();
});
</script>

<style scoped lang="scss">
.ops-monitoring-page {
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.ops-hero,
.ops-status-band,
.metric-card,
.ops-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.ops-hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  padding: 28px 30px;
}

.ops-hero__content {
  min-width: 0;
}

.ops-eyebrow,
.ops-section-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.ops-hero h1,
.ops-card h2 {
  margin: 8px 0 0;
  color: var(--admin-page-text-primary);
  font-weight: 650;
}

.ops-hero h1 {
  font-size: 28px;
  line-height: 1.25;
}

.ops-hero p,
.ops-card__header p {
  margin: 8px 0 0;
  color: var(--admin-page-text-secondary);
  font-size: 13px;
}

.ops-hero__actions {
  display: flex;
  flex-shrink: 0;
  gap: 10px;
}

.ops-status-band {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 14px;
  padding: 12px 16px;
  box-shadow: none;
}

.ops-status-band__main,
.ops-status-band__source {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.ops-status-band__main strong {
  color: var(--admin-page-text-primary);
}

.ops-status-band__muted {
  color: var(--admin-page-text-placeholder);
}

code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--admin-page-card-bg-muted);
  color: var(--admin-page-text-secondary);
  font-family: var(--el-font-family-monospace);
  font-size: 11px;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  border-radius: 50%;
  background: var(--el-color-info);
}

.status-dot.is-healthy {
  background: var(--el-color-success);
}

.status-dot.is-checking {
  background: var(--el-color-warning);
}

.status-dot.is-unavailable {
  background: var(--el-color-danger);
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-top: 14px;
}

.metric-card {
  display: flex;
  align-items: flex-start;
  gap: 13px;
  min-width: 0;
  padding: 18px;
  box-shadow: none;
}

.metric-card__icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  border-radius: 8px;
}

.metric-card__icon.is-blue {
  background: var(--admin-page-accent-soft-bg);
  color: var(--el-color-primary);
}

.metric-card__icon.is-green {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.metric-card__icon.is-purple {
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
}

.metric-card__icon.is-orange {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}

.metric-card__label,
.metric-card__hint {
  display: block;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.metric-card__value {
  display: block;
  margin: 4px 0;
  color: var(--admin-page-text-primary);
  font-size: 24px;
  line-height: 1.15;
}

.metric-card__value--date {
  font-size: 19px;
}

.metric-card__hint {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-grid {
  display: grid;
  gap: 14px;
  margin-top: 14px;
}

.ops-grid--primary {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.ops-grid--secondary {
  grid-template-columns: minmax(0, 1.15fr) minmax(360px, 0.85fr);
}

.ops-card {
  min-width: 0;
  overflow: hidden;
  box-shadow: none;
}

.ops-card :deep(.el-card__header) {
  padding: 18px 20px 14px;
  border-bottom-color: var(--admin-page-divider);
}

.ops-card :deep(.el-card__body) {
  padding: 0 20px 20px;
}

.ops-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.ops-card h2 {
  font-size: 17px;
}

.probe-list,
.module-list,
.environment-list {
  display: flex;
  flex-direction: column;
}

.probe-row,
.module-row,
.environment-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 58px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.probe-row:last-child,
.module-row:last-child,
.environment-row:last-child {
  border-bottom: 0;
}

.probe-row__name,
.module-row__identity {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.probe-row__name strong,
.module-row__identity strong,
.environment-row__value strong {
  color: var(--admin-page-text-primary);
  font-size: 13px;
}

.probe-row__name code {
  margin-left: 3px;
}

.probe-row__message,
.module-row__identity span,
.module-row__meta span,
.environment-row__value span {
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.probe-row__message {
  flex-shrink: 0;
}

.module-row__mark {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  place-items: center;
  border: 1px solid var(--admin-page-accent-soft-border);
  border-radius: 7px;
  background: var(--admin-page-accent-soft-bg);
  color: var(--admin-page-accent-soft-text);
}

.module-row__identity > div,
.environment-row__value {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.module-row__identity span,
.environment-row__value span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.module-row__meta {
  display: flex;
  align-items: baseline;
  flex-shrink: 0;
  gap: 5px;
}

.module-row__meta strong {
  color: var(--admin-page-text-primary);
  font-size: 16px;
}

.collection-table {
  width: 100%;
}

.collection-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.document-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.document-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.document-cell__icon {
  flex: 0 0 auto;
  color: var(--el-color-primary);
}

.environment-row {
  min-height: 62px;
}

.environment-row__label {
  width: 94px;
  flex: 0 0 94px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.environment-row__value {
  flex: 1;
}

.environment-row__value strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.environment-note {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-top: 14px;
  padding: 10px 12px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.environment-note > svg {
  flex: 0 0 auto;
  margin-top: 2px;
  color: var(--el-color-warning);
}

.environment-note .el-button {
  flex: 0 0 auto;
  margin: -2px 0 0 auto;
}

@media (max-width: 1100px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ops-grid--secondary {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 760px) {
  .ops-monitoring-page {
    padding: 14px;
  }

  .ops-hero {
    align-items: flex-start;
    flex-direction: column;
    padding: 22px 20px;
  }

  .ops-hero__actions {
    width: 100%;
  }

  .ops-hero__actions .el-button {
    flex: 1;
  }

  .ops-status-band {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .ops-grid--primary {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 520px) {
  .metric-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-card :deep(.el-card__header),
  .ops-card :deep(.el-card__body) {
    padding-right: 14px;
    padding-left: 14px;
  }

  .ops-card__header {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .probe-row,
  .module-row,
  .environment-row {
    align-items: flex-start;
    flex-wrap: wrap;
    padding: 12px 0;
  }

  .probe-row__message {
    width: 100%;
    padding-left: 17px;
  }

  .environment-note {
    flex-wrap: wrap;
  }

  .environment-note .el-button {
    margin-left: 20px;
  }
}
</style>
