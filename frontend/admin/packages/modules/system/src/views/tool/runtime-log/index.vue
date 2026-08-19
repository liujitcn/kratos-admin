<template>
  <div class="app-container runtime-log-page">
    <aside class="runtime-log-sources">
      <div class="source-panel__header">
        <h2>{{ t("system.runtime.log.section.sources") }}</h2>
        <el-tooltip :content="t('common.action.refresh')" placement="top">
          <el-button circle text :icon="Refresh" :loading="filesLoading" @click="loadFiles" />
        </el-tooltip>
      </div>

      <button
        type="button"
        class="source-item source-item--live"
        :class="{ 'is-active': activeSource.kind === 'live' }"
        @click="selectLiveConsole"
      >
        <span class="source-item__icon"><el-icon><Operation /></el-icon></span>
        <span class="source-item__body">
          <strong>{{ t("system.runtime.log.value.live_console") }}</strong>
          <small>
            <i class="status-dot" :class="`is-${connectionState}`" />
            {{ connectionStatusText }}
          </small>
        </span>
      </button>

      <div class="source-panel__section">
        <div class="source-panel__label">
          <span>{{ t("system.runtime.log.section.history_files") }}</span>
          <span>{{ filteredFiles.length }}</span>
        </div>
        <el-input
          v-model="fileKeyword"
          clearable
          :prefix-icon="Search"
          :placeholder="t('system.runtime.log.placeholder.search_file')"
        />
      </div>

      <el-scrollbar class="source-panel__list">
        <button
          v-for="file in filteredFiles"
          :key="file.file_id"
          type="button"
          class="source-item source-item--file"
          :class="{ 'is-active': activeFile?.file_id === file.file_id }"
          @click="selectHistoryFile(file)"
        >
          <span class="source-item__icon"><el-icon><Document /></el-icon></span>
          <span class="source-item__body">
            <strong :title="file.name">{{ file.name }}</strong>
            <small>{{ formatFileSize(file.size_bytes) }} · {{ formatDateTime(file.modified_at) }}</small>
          </span>
          <el-tag v-if="file.is_compressed" size="small" effect="plain">GZ</el-tag>
        </button>
        <el-empty
          v-if="!filesLoading && filteredFiles.length === 0"
          :image-size="64"
          :description="t('system.runtime.log.empty.files')"
        />
      </el-scrollbar>
    </aside>

    <main class="runtime-log-workspace">
      <header class="workspace-header">
        <div class="workspace-title">
          <div class="workspace-title__row">
            <h1 :title="activeTitle">{{ activeTitle }}</h1>
            <el-tag v-if="activeSource.kind === 'live'" size="small" :type="connectionTagType" effect="plain">
              {{ connectionStatusText }}
            </el-tag>
            <el-tag v-else-if="activeFile?.is_compressed" size="small" effect="plain">GZIP</el-tag>
          </div>
          <p v-if="activeSource.kind === 'live'">
            {{ t("system.runtime.log.value.instance", { id: instanceId || "-" }) }}
          </p>
          <p v-else-if="activeFile">
            {{ formatFileSize(activeFile.size_bytes) }} · {{ formatDateTime(activeFile.modified_at) }}
          </p>
        </div>

        <div class="workspace-actions">
          <el-tooltip
            v-if="activeSource.kind === 'live'"
            :content="paused ? t('system.runtime.log.action.resume') : t('system.runtime.log.action.pause')"
            placement="top"
          >
            <el-button
              circle
              :icon="paused ? VideoPlay : VideoPause"
              :disabled="connectionState === 'connecting'"
              @click="togglePause"
            />
          </el-tooltip>
          <el-tooltip v-else :content="t('common.action.refresh')" placement="top">
            <el-button circle :icon="Refresh" :loading="contentLoading" @click="refreshHistory" />
          </el-tooltip>
          <el-tooltip v-if="activeFile" :content="t('system.runtime.log.action.download')" placement="top">
            <el-button
              circle
              type="primary"
              :icon="Download"
              :loading="downloadLoading"
              @click="downloadHistoryFile"
            />
          </el-tooltip>
        </div>
      </header>

      <div class="workspace-filters">
        <el-input
          v-model="keyword"
          clearable
          :prefix-icon="Search"
          :placeholder="t('system.runtime.log.placeholder.search_content')"
        />
        <el-select
          v-model="selectedLevels"
          multiple
          collapse-tags
          collapse-tags-tooltip
          :max-collapse-tags="2"
          :placeholder="t('system.runtime.log.placeholder.level')"
        >
          <el-option v-for="level in levelOptions" :key="level" :label="level" :value="level" />
        </el-select>
        <label class="auto-scroll-control">
          <span>{{ t("system.runtime.log.action.auto_scroll") }}</span>
          <el-switch v-model="autoScroll" />
        </label>
        <span class="entry-count">{{ t("system.runtime.log.value.entry_count", { count: displayedEntries.length }) }}</span>
      </div>

      <el-alert
        v-if="gapCount > 0"
        class="runtime-gap-alert"
        type="warning"
        show-icon
        :closable="true"
        :title="t('system.runtime.log.message.gap', { count: gapCount })"
        @close="gapCount = 0"
      />

      <section class="log-viewer" :aria-label="t('system.runtime.log.section.viewer')">
        <el-scrollbar ref="scrollbarRef" class="log-viewer__scroll" always>
          <div class="log-viewer__content">
            <div v-if="activeFile && hasMore" class="load-more-row">
              <el-button text type="primary" :loading="olderLoading" @click="loadOlderHistory">
                {{ t("system.runtime.log.action.load_older") }}
              </el-button>
            </div>
            <div
              v-for="(entry, index) in displayedEntries"
              :key="entryKey(entry, index)"
              class="log-line"
              :class="`is-${entryLevel(entry)}`"
            >
              <span class="log-line__number">{{ index + 1 }}</span>
              <span v-if="entry.level" class="log-line__level">{{ entry.level }}</span>
              <span class="log-line__text">{{ entryLine(entry) }}</span>
            </div>
            <el-empty
              v-if="!contentLoading && displayedEntries.length === 0"
              :image-size="72"
              :description="t('system.runtime.log.empty.entries')"
            />
          </div>
        </el-scrollbar>
        <div v-if="contentLoading" class="log-viewer__loading">
          <el-icon class="is-loading"><Refresh /></el-icon>
          <span>{{ t("system.runtime.log.status.loading") }}</span>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { Document, Download, Operation, Refresh, Search, VideoPause, VideoPlay } from "@element-plus/icons-vue";
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from "vue";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";
import { defRuntimeLogService } from "@liujitcn/kratos-admin-system/api/system/runtime_log";
import { subscribeRuntimeLog } from "@liujitcn/kratos-admin-system/api/system/runtime_log_sse";
import type {
  RuntimeLogEntry,
  RuntimeLogFile
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/runtime_log";

defineOptions({
  name: "RuntimeLog",
  inheritAttrs: false
});

/** 当前日志来源。 */
type RuntimeLogSource = { kind: "live" } | { kind: "file"; file: RuntimeLogFile };

/** 实时控制台连接状态。 */
type RuntimeConnectionState = "connecting" | "live" | "paused" | "disconnected";

/** Element Plus 滚动条公开能力。 */
interface RuntimeLogScrollbar {
  /** 滚动条内容元素。 */
  wrapRef?: HTMLElement;
  /** 设置垂直滚动位置。 */
  setScrollTop: (top: number) => void;
}

const maxLiveEntries = 2000;
const levelOptions = ["DEBUG", "INFO", "WARN", "ERROR"];
const files = ref<RuntimeLogFile[]>([]);
const entries = ref<RuntimeLogEntry[]>([]);
const pendingEntries = ref<RuntimeLogEntry[]>([]);
const activeSource = ref<RuntimeLogSource>({ kind: "live" });
const connectionState = ref<RuntimeConnectionState>("connecting");
const filesLoading = ref(false);
const contentLoading = ref(false);
const olderLoading = ref(false);
const downloadLoading = ref(false);
const paused = ref(false);
const autoScroll = ref(true);
const keyword = ref("");
const fileKeyword = ref("");
const selectedLevels = ref<string[]>([]);
const nextCursor = ref("");
const hasMore = ref(false);
const instanceId = ref("");
const gapCount = ref(0);
const scrollbarRef = ref<RuntimeLogScrollbar>();
let stopRuntimeLog: (() => void) | undefined;
let filterTimer: ReturnType<typeof setTimeout> | undefined;
let activeRequestId = 0;

const activeFile = computed(() => (activeSource.value.kind === "file" ? activeSource.value.file : undefined));
const activeTitle = computed(() => activeFile.value?.name ?? t("system.runtime.log.value.live_console"));
const filteredFiles = computed(() => {
  const search = fileKeyword.value.toLowerCase();
  return search ? files.value.filter(file => file.name.toLowerCase().includes(search)) : files.value;
});
const displayedEntries = computed(() => {
  const search = keyword.value.toLowerCase();
  const levelSet = new Set(selectedLevels.value);
  return entries.value.filter(entry => {
    if (levelSet.size > 0 && !levelSet.has(entry.level)) return false;
    return !search || entryLine(entry).toLowerCase().includes(search);
  });
});
const connectionStatusText = computed(() => t(`system.runtime.log.status.${connectionState.value}`));
const connectionTagType = computed(() => {
  if (connectionState.value === "live") return "success";
  if (connectionState.value === "paused") return "warning";
  return "info";
});

/** 查询左侧历史日志文件列表。 */
async function loadFiles() {
  filesLoading.value = true;
  try {
    const response = await defRuntimeLogService.ListRuntimeLogFiles({});
    files.value = response.files ?? [];
    if (activeFile.value) {
      const refreshed = files.value.find(file => file.name === activeFile.value?.name);
      if (refreshed) activeSource.value = { kind: "file", file: refreshed };
    }
  } finally {
    filesLoading.value = false;
  }
}

/** 切换到实时控制台。 */
function selectLiveConsole() {
  activeSource.value = { kind: "live" };
  entries.value = [];
  pendingEntries.value = [];
  nextCursor.value = "";
  hasMore.value = false;
  paused.value = false;
  const requestId = ++activeRequestId;
  void openRuntimeConsole(requestId);
}

/** 创建并订阅当前用户的实时控制台频道。 */
async function openRuntimeConsole(requestId: number) {
  stopRuntimeSubscription();
  connectionState.value = "connecting";
  contentLoading.value = true;
  try {
    const response = await defRuntimeLogService.OpenRuntimeConsole({ backlog_limit: 300 });
    if (requestId !== activeRequestId || activeSource.value.kind !== "live") return;
    entries.value = (response.entries ?? []).slice(-maxLiveEntries);
    instanceId.value = response.instance_id;
    connectionState.value = "live";
    stopRuntimeLog = subscribeRuntimeLog(
      response.channel_id,
      entry => appendRuntimeEntry(entry),
      gap => {
        gapCount.value += Number(gap.dropped_count || 0);
      }
    );
    scrollToBottom();
  } catch {
    if (requestId === activeRequestId) connectionState.value = "disconnected";
  } finally {
    if (requestId === activeRequestId) contentLoading.value = false;
  }
}

/** 追加一条实时日志，并在暂停时暂存。 */
function appendRuntimeEntry(entry: RuntimeLogEntry) {
  if (activeSource.value.kind !== "live") return;
  if (paused.value) {
    pendingEntries.value.push(entry);
    if (pendingEntries.value.length > maxLiveEntries) pendingEntries.value.splice(0, pendingEntries.value.length - maxLiveEntries);
    return;
  }
  entries.value.push(entry);
  if (entries.value.length > maxLiveEntries) entries.value.splice(0, entries.value.length - maxLiveEntries);
  scrollToBottom();
}

/** 暂停或继续实时日志展示。 */
function togglePause() {
  paused.value = !paused.value;
  connectionState.value = paused.value ? "paused" : "live";
  if (!paused.value && pendingEntries.value.length > 0) {
    entries.value.push(...pendingEntries.value);
    pendingEntries.value = [];
    if (entries.value.length > maxLiveEntries) entries.value.splice(0, entries.value.length - maxLiveEntries);
    scrollToBottom();
  }
}

/** 切换到指定历史日志文件。 */
function selectHistoryFile(file: RuntimeLogFile) {
  stopRuntimeSubscription();
  activeSource.value = { kind: "file", file };
  entries.value = [];
  nextCursor.value = "";
  hasMore.value = false;
  paused.value = false;
  connectionState.value = "disconnected";
  void refreshHistory();
}

/** 从文件尾部重新读取当前历史日志。 */
async function refreshHistory() {
  if (!activeFile.value) return;
  const requestId = ++activeRequestId;
  contentLoading.value = true;
  nextCursor.value = "";
  hasMore.value = false;
  try {
    const response = await readActiveHistory("");
    if (requestId !== activeRequestId) return;
    if (response.file_changed) {
      ElMessage.warning(t("system.runtime.log.message.file_changed"));
      return;
    }
    entries.value = response.entries ?? [];
    nextCursor.value = response.next_cursor;
    hasMore.value = response.has_more;
    scrollToBottom();
  } finally {
    if (requestId === activeRequestId) contentLoading.value = false;
  }
}

/** 读取当前历史日志的更早一页。 */
async function loadOlderHistory() {
  if (!activeFile.value || !hasMore.value || olderLoading.value) return;
  const requestId = activeRequestId;
  olderLoading.value = true;
  try {
    const response = await readActiveHistory(nextCursor.value);
    if (requestId !== activeRequestId) return;
    if (response.file_changed) {
      ElMessage.warning(t("system.runtime.log.message.file_changed"));
      await loadFiles();
      await refreshHistory();
      return;
    }
    entries.value = [...(response.entries ?? []), ...entries.value];
    nextCursor.value = response.next_cursor;
    hasMore.value = response.has_more;
  } finally {
    olderLoading.value = false;
  }
}

/** 按当前筛选条件读取历史日志。 */
function readActiveHistory(cursor: string) {
  const file = activeFile.value;
  if (!file) throw new Error("runtime log file is not selected");
  return defRuntimeLogService.ReadRuntimeLogFile({
    file_id: file.file_id,
    cursor,
    limit: 300,
    keyword: keyword.value,
    levels: selectedLevels.value
  });
}

/** 下载当前选中的历史日志原文件。 */
async function downloadHistoryFile() {
  const file = activeFile.value;
  if (!file) return;
  downloadLoading.value = true;
  try {
    await defRuntimeLogService.DownloadRuntimeLogFile({ file_id: file.file_id }, file.name);
    ElMessage.success(t("system.runtime.log.message.download_started"));
  } finally {
    downloadLoading.value = false;
  }
}

/** 关闭当前实时日志 SSE 订阅。 */
function stopRuntimeSubscription() {
  stopRuntimeLog?.();
  stopRuntimeLog = undefined;
}

/** 在启用自动滚动时移动到日志底部。 */
function scrollToBottom() {
  if (!autoScroll.value) return;
  void nextTick(() => {
    const wrap = scrollbarRef.value?.wrapRef;
    if (wrap) scrollbarRef.value?.setScrollTop(wrap.scrollHeight);
  });
}

/** 返回稳定的日志行渲染键。 */
function entryKey(entry: RuntimeLogEntry, index: number) {
  return entry.sequence || `${entry.timestamp ?? ""}:${index}:${entryLine(entry)}`;
}

/** 返回日志级别对应的样式名称，兼容 Proto JSON 省略空字段。 */
function entryLevel(entry: RuntimeLogEntry) {
  return String(entry.level ?? "").toLowerCase() || "unknown";
}

/** 返回可展示的日志正文，兼容 Proto JSON 省略空字段。 */
function entryLine(entry: RuntimeLogEntry) {
  return String(entry.line ?? entry.message ?? "");
}

/** 格式化日志文件大小。 */
function formatFileSize(rawBytes: number) {
  const bytes = Number(rawBytes || 0);
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

/** 格式化日志文件时间。 */
function formatDateTime(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}

watch([keyword, selectedLevels], () => {
  if (!activeFile.value) return;
  if (filterTimer) clearTimeout(filterTimer);
  filterTimer = setTimeout(() => void refreshHistory(), 300);
});

onMounted(() => {
  void loadFiles();
  selectLiveConsole();
});

onActivated(() => {
  if (activeSource.value.kind === "live" && !stopRuntimeLog && connectionState.value !== "connecting") {
    const requestId = ++activeRequestId;
    void openRuntimeConsole(requestId);
  }
});

onDeactivated(stopRuntimeSubscription);

onBeforeUnmount(() => {
  if (filterTimer) clearTimeout(filterTimer);
  stopRuntimeSubscription();
});
</script>

<style scoped lang="scss">
.runtime-log-page {
  display: grid;
  grid-template-columns: minmax(240px, 286px) minmax(0, 1fr);
  height: calc(100vh - 142px);
  min-height: 560px;
  padding: 0;
  overflow: hidden;
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 6px;
}

.runtime-log-sources {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  padding: 14px 12px;
  background: var(--admin-page-card-bg-soft);
  border-right: 1px solid var(--admin-page-divider);
}

.source-panel__header,
.source-panel__label,
.workspace-header,
.workspace-filters,
.workspace-title__row,
.workspace-actions,
.auto-scroll-control {
  display: flex;
  align-items: center;
}

.source-panel__header {
  justify-content: space-between;
  padding: 0 4px 10px;
}

.source-panel__header h2,
.workspace-title h1 {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: 15px;
  font-weight: 650;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-item {
  display: flex;
  width: 100%;
  min-height: 54px;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 5px;
}

.source-item:hover {
  background: var(--el-fill-color);
}

.source-item.is-active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
}

.source-item__icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 5px;
}

.source-item__body {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;
}

.source-item__body strong,
.source-item__body small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-item__body strong {
  font-size: 13px;
  font-weight: 600;
}

.source-item__body small {
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
}

.status-dot {
  display: inline-block;
  width: 7px;
  height: 7px;
  margin-right: 4px;
  background: var(--el-text-color-placeholder);
  border-radius: 50%;
}

.status-dot.is-live {
  background: var(--el-color-success);
}

.status-dot.is-paused {
  background: var(--el-color-warning);
}

.status-dot.is-connecting {
  background: var(--el-color-primary);
}

.status-dot.is-disconnected {
  background: var(--el-color-danger);
}

.source-panel__section {
  padding: 18px 4px 8px;
}

.source-panel__label {
  justify-content: space-between;
  margin-bottom: 8px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.source-panel__list {
  min-height: 0;
  flex: 1;
  margin-top: 4px;
}

.source-item--file + .source-item--file {
  margin-top: 2px;
}

.runtime-log-workspace {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.workspace-header {
  min-height: 72px;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.workspace-title {
  min-width: 0;
}

.workspace-title__row {
  min-width: 0;
  gap: 8px;
}

.workspace-title p {
  margin: 5px 0 0;
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
}

.workspace-actions {
  flex: 0 0 auto;
  gap: 6px;
}

.workspace-filters {
  min-height: 54px;
  gap: 10px;
  padding: 9px 18px;
  background: var(--admin-page-card-bg-soft);
  border-bottom: 1px solid var(--admin-page-divider);
}

.workspace-filters :deep(.el-input) {
  width: min(320px, 32vw);
}

.workspace-filters :deep(.el-select) {
  width: 210px;
}

.auto-scroll-control {
  flex: 0 0 auto;
  gap: 8px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.entry-count {
  margin-left: auto;
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
  white-space: nowrap;
}

.runtime-gap-alert {
  border-radius: 0;
}

.log-viewer {
  position: relative;
  min-height: 0;
  flex: 1;
  overflow: hidden;
  background: var(--el-bg-color-page);
}

.log-viewer__scroll {
  height: 100%;
}

.log-viewer__content {
  min-width: 760px;
  min-height: 100%;
  padding: 10px 0 18px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
}

.load-more-row {
  display: flex;
  justify-content: center;
  padding: 2px 0 8px;
}

.log-line {
  display: grid;
  grid-template-columns: 54px 58px minmax(0, 1fr);
  min-height: 25px;
  align-items: start;
  padding: 3px 16px 3px 0;
  border-left: 2px solid transparent;
}

.log-line:hover {
  background: var(--el-fill-color-light);
}

.log-line.is-error {
  background: var(--el-color-danger-light-9);
  border-left-color: var(--el-color-danger);
}

.log-line.is-warn {
  background: var(--el-color-warning-light-9);
  border-left-color: var(--el-color-warning);
}

.log-line__number {
  padding-right: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 11px;
  line-height: 19px;
  text-align: right;
  user-select: none;
}

.log-line__level {
  width: fit-content;
  min-width: 42px;
  padding: 1px 5px;
  color: var(--el-text-color-regular);
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
  background: var(--el-fill-color);
  border-radius: 3px;
}

.log-line.is-debug .log-line__level {
  color: var(--el-color-info);
}

.log-line.is-info .log-line__level {
  color: var(--el-color-success);
}

.log-line.is-warn .log-line__level {
  color: var(--el-color-warning-dark-2);
}

.log-line.is-error .log-line__level {
  color: var(--el-color-danger);
}

.log-line__text {
  min-width: 0;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  line-height: 19px;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.log-viewer__loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--admin-page-text-secondary);
  background: color-mix(in srgb, var(--el-bg-color-page) 82%, transparent);
}

@media (max-width: 900px) {
  .runtime-log-page {
    grid-template-columns: 220px minmax(0, 1fr);
  }

  .workspace-filters {
    flex-wrap: wrap;
  }

  .workspace-filters :deep(.el-input) {
    width: min(100%, 320px);
  }

  .entry-count {
    display: none;
  }
}

@media (max-width: 680px) {
  .runtime-log-page {
    display: flex;
    height: auto;
    min-height: calc(100vh - 120px);
    flex-direction: column;
    overflow: visible;
  }

  .runtime-log-sources {
    max-height: 310px;
    border-right: 0;
    border-bottom: 1px solid var(--admin-page-divider);
  }

  .source-panel__list {
    min-height: 120px;
  }

  .runtime-log-workspace {
    min-height: 560px;
  }

  .workspace-header,
  .workspace-filters {
    padding-right: 12px;
    padding-left: 12px;
  }

  .workspace-filters :deep(.el-input),
  .workspace-filters :deep(.el-select) {
    width: 100%;
  }

  .auto-scroll-control {
    margin-left: auto;
  }
}
</style>
