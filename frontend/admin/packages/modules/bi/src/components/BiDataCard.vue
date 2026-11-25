<template>
  <article class="bi-data-card" :class="{ 'is-compact': compact }">
    <el-header class="bi-data-card__header" style="--el-header-height: auto; --el-header-padding: 0">
      <div class="bi-data-card__heading">
        <el-text tag="h3" class="bi-data-card__title">
          <el-icon class="bi-data-card__icon"><component :is="icon" /></el-icon>
          <BiCardTitleTooltip :title="card.name">
            <span class="bi-data-card__name">{{ card.name }}</span>
          </BiCardTitleTooltip>
          <el-tag v-if="card.chart_type === 'metric_grid'" size="small" effect="light" type="success">{{
            t("bi.card.ai_generated")
          }}</el-tag>
          <el-tag v-if="showCategory" size="small" effect="plain" type="info">{{
            t(`bi.card.category.${card.category || "mine"}`)
          }}</el-tag>
          <span v-if="sourceName" class="bi-data-card__source">
            <el-icon><component :is="sourceIcon" /></el-icon>{{ sourceName }}
          </span>
        </el-text>
        <div class="bi-data-card__meta">
          <el-select
            v-if="showPeriodFilter"
            :model-value="effectiveRangeDays"
            class="bi-data-card__period"
            size="small"
            :aria-label="t('bi.card.period', { name: card.name })"
            @update:model-value="emit('update:rangeDays', $event)"
          >
            <el-option :value="0" :label="t('bi.card.period.default')" />
            <el-option v-for="days in [7, 30]" :key="days" :value="days" :label="t('bi.card.period.recent_days', { days })" />
          </el-select>
          <span v-if="periodLabel">{{ periodLabel }}</span>
          <el-tag v-if="data?.truncated" size="small" type="warning">{{ t("bi.query.truncated") }}</el-tag>
        </div>
      </div>
      <slot name="actions" />
      <div v-if="!footerActions" class="bi-data-card__native-actions">
        <el-tooltip :content="t('bi.card.export')" placement="top">
          <el-button text circle :icon="Download" :disabled="!data" @click.stop="exportResult" />
        </el-tooltip>
        <el-tooltip v-if="previewable" :content="t('bi.card.enlarge')" placement="top">
          <el-button text circle :icon="ZoomIn" @click.stop="emit('preview')" />
        </el-tooltip>
      </div>
    </el-header>
    <div v-loading="loading" class="bi-data-card__chart" :aria-busy="loading">
      <div v-if="error" class="bi-data-card__error" role="alert">
        <span>{{ error }}</span>
        <el-button link type="primary" @click="loadData">{{ t("bi.card.retry") }}</el-button>
      </div>
      <BiCardVisualization v-else-if="data" :card="card" :result="data" :compact="compact" />
      <el-empty v-else-if="!loading" :image-size="48" :description="t('common.message.no_data')" />
    </div>
    <el-footer
      v-if="$slots.footer || footerActions"
      class="bi-data-card__footer"
      style="--el-footer-height: auto; --el-footer-padding: 0"
    >
      <slot name="footer" />
      <div v-if="footerActions" class="bi-data-card__native-actions">
        <el-tooltip :content="t('bi.card.enlarge')" placement="top">
          <el-button text circle :icon="ZoomIn" :aria-label="t('bi.card.enlarge')" @click.stop="emit('preview')" />
        </el-tooltip>
        <el-tooltip :content="t('bi.card.export')" placement="top">
          <el-button
            text
            circle
            :icon="Download"
            :aria-label="t('bi.card.export')"
            :disabled="!data"
            @click.stop="exportResult"
          />
        </el-tooltip>
      </div>
    </el-footer>
  </article>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { Download, ZoomIn } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import BiCardTitleTooltip from "./BiCardTitleTooltip.vue";
import BiCardVisualization from "./BiCardVisualization.vue";
import { cardRefreshDelay } from "../card-schema";
import { describeBiTimeRange } from "../query-summary";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";

const icon = DataAnalysis;
const sourceIcon = Coin;
const props = withDefaults(
  defineProps<{
    card: BiCard;
    sourceName?: string;
    previewResult?: BiQueryResult;
    refreshKey?: number;
    rangeDays?: number;
    showPeriodFilter?: boolean;
    showCategory?: boolean;
    /** 是否展示内置「放大查看」按钮（仅父级接入放大弹窗时开启）。 */
    previewable?: boolean;
    /** 是否将放大、导出操作放到卡片底部。 */
    footerActions?: boolean;
    /** 是否使用工作台和卡片库的紧凑卡片高度。 */
    compact?: boolean;
  }>(),
  {
    sourceName: "",
    refreshKey: 0,
    rangeDays: 0,
    showPeriodFilter: false,
    showCategory: true,
    previewable: false,
    footerActions: false,
    compact: false
  }
);
const emit = defineEmits<{ "update:rangeDays": [value: number]; preview: [] }>();
const effectiveRangeDays = computed(() => (props.showPeriodFilter ? props.rangeDays : 0));
const periodLabel = computed(() => {
  const range = props.card.query?.time_range;
  return !effectiveRangeDays.value && range ? describeBiTimeRange(range, t) : "";
});
const result = ref<BiQueryResult>();
const loading = ref(false);
const error = ref("");
let timer: ReturnType<typeof setTimeout> | undefined;
let active = true;
let requestID = 0;
const data = computed(() => props.previewResult ?? result.value);

/** 按当前卡片周期加载数据，关闭或切换卡片后丢弃迟到响应。 */
async function loadData() {
  clearTimeout(timer);
  if (!active || props.card.id === 0 || props.previewResult) return;
  const current = ++requestID;
  loading.value = true;
  error.value = "";
  try {
    const loadedResult = await defBiCardService.GetBiCardData({ id: props.card.id, range_days: effectiveRangeDays.value });
    if (current === requestID) result.value = loadedResult;
  } catch (cause) {
    if (current === requestID) {
      error.value = cause instanceof Error ? cause.message : t("bi.query.error");
      result.value = undefined;
    }
  } finally {
    if (current === requestID) {
      loading.value = false;
      scheduleRefresh();
    }
  }
}

/** 隐藏标签页时延后自动取数，避免后台持续查询。 */
function scheduleRefresh() {
  clearTimeout(timer);
  if (active && props.card.refresh_seconds > 0)
    timer = setTimeout(() => {
      if (document.hidden) scheduleRefresh();
      else void loadData();
    }, cardRefreshDelay(props.card.refresh_seconds));
}

/** 标签页重新可见后补一次刷新，避免错过每日刷新时间。 */
function handleVisibility() {
  if (active && !document.hidden && props.card.refresh_seconds > 0 && !loading.value) void loadData();
}

/** 停止非活动页面的定时器与响应回填。 */
function stop() {
  active = false;
  requestID++;
  loading.value = false;
  clearTimeout(timer);
}

/** 将当前卡片数据（含预览快照）导出为 CSV。 */
function exportResult() {
  const current = data.value;
  if (!current) return;
  const csv = [
    (current.columns || []).map(item => '"' + String(item.name || item.code || "").replaceAll('"', '""') + '"'),
    ...(current.rows || []).map(row => (row.values || []).map(value => '"' + String(value ?? "").replaceAll('"', '""') + '"'))
  ]
    .map(row => row.join(","))
    .join("\n");
  const url = URL.createObjectURL(new Blob(["\ufeff" + csv], { type: "text/csv;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = props.card.name + ".csv";
  link.click();
  URL.revokeObjectURL(url);
  ElMessage.success(t("bi.card.export_success", { name: props.card.name }));
}
watch(
  () => [props.card.id, props.card.updated_at, props.refreshKey, effectiveRangeDays.value],
  () => {
    result.value = undefined;
    void loadData();
  },
  { immediate: true }
);
onActivated(() => {
  if (!active) {
    active = true;
    void loadData();
  }
});
onDeactivated(stop);
onMounted(() => document.addEventListener("visibilitychange", handleVisibility));
onBeforeUnmount(() => {
  stop();
  document.removeEventListener("visibilitychange", handleVisibility);
});
</script>

<style scoped>
.bi-data-card {
  display: flex;
  flex-direction: column;

  min-width: 0;

  --bi-card-header-height: 56px;
  --bi-card-footer-height: 46px;
  --bi-card-action-size: 32px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
  background: var(--el-bg-color);
}
.bi-data-card__header {
  display: flex;
  box-sizing: border-box;
  height: var(--bi-card-header-height);
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.bi-data-card__heading {
  min-width: 0;
}
.bi-data-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
}
.bi-data-card__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-data-card__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}
.bi-data-card__chart {
  box-sizing: border-box;
  height: 280px;
  padding: 12px 16px;
}
.bi-data-card.is-compact .bi-data-card__chart {
  height: 160px;
}
.bi-data-card__period {
  width: 130px;
}

.bi-data-card__native-actions {
  display: flex;
  flex: none;
  align-items: center;
  gap: 0;
}
.bi-data-card__header :deep(.bi-data-card__slot-actions) {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0;
  min-width: max-content;
  margin-left: auto;
  white-space: nowrap;
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-tooltip__trigger) {
  display: inline-flex;
  flex: 0 0 auto;
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button),
.bi-data-card__native-actions :deep(.el-button) {
  box-sizing: border-box;
  width: var(--bi-card-action-size);
  height: var(--bi-card-action-size);
  padding: 8px;
  margin: 0;
  color: var(--el-text-color-secondary);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button--warning) {
  color: var(--el-color-warning);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button--danger) {
  color: var(--el-color-danger);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button.is-disabled),
.bi-data-card__native-actions :deep(.el-button.is-disabled) {
  color: var(--el-text-color-disabled);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button:not(.is-disabled):hover),
.bi-data-card__native-actions :deep(.el-button:not(.is-disabled):hover) {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button--warning:not(.is-disabled):hover) {
  color: var(--el-color-warning);
}
.bi-data-card__header :deep(.bi-data-card__slot-actions .el-button--danger:not(.is-disabled):hover) {
  color: var(--el-color-danger);
}
.bi-data-card__error {
  display: flex;
  height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow-wrap: anywhere;
}
.bi-data-card__footer {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 8px;
  min-height: var(--bi-card-footer-height);
  padding: 6px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.bi-data-card__footer > .bi-data-card__native-actions {
  margin-left: auto;
}

.bi-data-card__source {
  display: flex;
  flex: 0 1 auto;
  align-items: center;
  gap: 5px;
  min-width: 0;
  padding: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-data-card__icon {
  flex: none;
  width: 27px;
  height: 27px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: var(--admin-page-radius);
}
</style>
