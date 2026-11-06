<template>
  <article class="bi-data-card">
    <header class="bi-data-card__header">
      <div class="bi-data-card__heading">
        <h3 class="bi-data-card__title">
          <el-icon class="bi-data-card__icon"><component :is="icon" /></el-icon>{{ card.name }}
        </h3>
        <div class="bi-data-card__meta">
          <el-tag v-if="showCategory" size="small" effect="plain" type="info">{{ t(`bi.card.category.${card.category || "mine"}`) }}</el-tag>
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
    </header>
    <div v-loading="loading" class="bi-data-card__chart" :aria-busy="loading">
      <div v-if="error" class="bi-data-card__error" role="alert">
        <span>{{ error }}</span>
        <el-button link type="primary" @click="loadData">{{ t("bi.card.retry") }}</el-button>
      </div>
      <BiCardVisualization v-else-if="data" :card="card" :result="data" />
      <el-empty v-else-if="!loading" :image-size="48" :description="t('common.message.no_data')" />
    </div>
    <div v-if="sourceName" class="bi-data-card__source">
      <el-icon><component :is="sourceIcon" /></el-icon>{{ sourceName }}
    </div>
    <footer v-if="$slots.footer" class="bi-data-card__footer"><slot name="footer" /></footer>
  </article>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import BiCardVisualization from "./BiCardVisualization.vue";
import { cardRefreshDelay } from "../card-schema";
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
  }>(),
  { sourceName: "", refreshKey: 0, rangeDays: 0, showPeriodFilter: false, showCategory: true }
);
const emit = defineEmits<{ "update:rangeDays": [value: number] }>();
const effectiveRangeDays = computed(() => (props.showPeriodFilter ? props.rangeDays : 0));
const periodLabel = computed(() => {
  const range = props.card.query?.time_range;
  return !effectiveRangeDays.value && range ? `${range.start.slice(0, 10)} — ${range.end.slice(0, 10)}` : "";
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
  min-width: 0;
  flex-direction: column;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
  background: var(--el-bg-color);
}
.bi-data-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 16px 16px 0;
}
.bi-data-card__heading {
  min-width: 0;
}
.bi-data-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
}
.bi-data-card__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.bi-data-card__chart {
  box-sizing: border-box;
  height: 280px;
  padding: 12px 16px;
}
.bi-data-card__period {
  width: 130px;
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.bi-data-card__source {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 16px 12px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
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
