<template>
  <details class="bi-query-refinement">
    <summary>{{ t('bi.refine.title') }}</summary>
    <div class="bi-query-refinement__fields">
      <el-form-item :label="t('bi.card.dimensions')">
        <el-select size="small" :disabled="busy || disabled" v-model="dimensions" multiple :multiple-limit="2" clearable :aria-label="t('bi.card.dimensions')">
          <el-option v-for="field in source.dimensions" :key="field.code" :label="field.name" :value="field.code" />
        </el-select>
      </el-form-item>
      <el-form-item v-for="field in configurableDimensions" :key="field.code" :label="field.name">
        <el-select size="small" :disabled="busy || disabled" v-model="grouping[field.code]" :aria-label="field.name">
          <el-option :label="t('bi.refine.original')" value="" />
          <el-option v-for="grain in field.time_grains ?? []" :key="grain" :label="t(`bi.refine.grain.${grain}`)" :value="grain" />
          <el-option v-for="level in field.hierarchy_levels ?? []" :key="level" :label="t(`bi.refine.level.${level}`)" :value="level" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="dateFields.length || query.time_range" :label="t('bi.refine.period')">
        <el-select size="small" :disabled="busy || disabled" v-model="period" :aria-label="t('bi.refine.period')">
          <el-option v-if="query.time_range" :label="describeBiTimeRange(query.time_range, t)" value="current" />
          <el-option :label="t('bi.refine.all_dates')" value="all" />
          <template v-if="dateFields.length">
            <el-option :label="t('bi.refine.custom_dates')" value="custom" />
            <el-option :label="t('bi.refine.last_7')" value="7" />
            <el-option :label="t('bi.refine.last_30')" value="30" />
            <el-option :label="t('bi.refine.year_to_date')" value="year_to_date" />
          </template>
        </el-select>
      </el-form-item>
      <el-form-item v-if="period !== 'all' && period !== 'current'" :label="t('bi.refine.date_field')">
        <el-select size="small" :disabled="busy || disabled" v-model="dateField" :aria-label="t('bi.refine.date_field')">
          <el-option v-for="field in dateFields" :key="field.code" :label="field.name" :value="field.code" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="period === 'custom'" :label="t('bi.refine.custom_dates')">
        <el-date-picker size="small" :disabled="busy || disabled" v-model="dates" type="daterange" value-format="YYYY-MM-DD" :start-placeholder="t('bi.refine.start')" :end-placeholder="t('bi.refine.end')" />
      </el-form-item>
      <el-button native-type="button" type="primary" :loading="busy" :disabled="invalid" @click="apply">{{ t('bi.refine.apply') }}</el-button>
    </div>
  </details>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defBiQueryService } from "../api/bi/admin/v1/bi_query";
import { describeBiTimeRange, shiftCalendarDate } from "../query-summary";
import type { AskBiQueryResponse, BiQuerySpec, BiSourceDescriptor, BiTimeRangeSpec } from "../rpc/bi/admin/v1/bi_query";

const props = defineProps<{ source: BiSourceDescriptor; query: BiQuerySpec; recordId: number; disabled?: boolean }>();
const emit = defineEmits<{ applied: [result: AskBiQueryResponse]; busy: [busy: boolean] }>();
const dimensions = ref<string[]>([]);
const grouping = ref<Record<string, string | number>>({});
const period = ref("all");
const dates = ref<[string, string]>();
const dateField = ref("");
const busy = ref(false);
let request = 0;
const configurableDimensions = computed(() => props.source.dimensions.filter(field =>
  dimensions.value.includes(field.code) && (field.hierarchy_levels?.length || field.time_grains?.length)
));
const dateFields = computed(() => props.source.dimensions.filter(field => field.type === "date" && field.filterable && field.time_grains?.length));
const invalid = computed(() => props.disabled || !props.recordId || (period.value === "custom" && !dates.value?.length) ||
  (period.value !== "all" && period.value !== "current" && !dateField.value));

watch(() => [props.query, props.recordId, props.source] as const, ([query]) => {
  request++;
  if (busy.value) emit("busy", false);
  busy.value = false;
  dimensions.value = [...query.dimensions];
  grouping.value = Object.fromEntries(props.source.dimensions.map(field => {
    const option = query.grouping_options.find(item => item.field === field.code);
    return [field.code, option?.time_grain || option?.hierarchy_level || ""];
  }));
  period.value = query.time_range ? "current" : "all";
  dateField.value = dateFields.value.find(field => field.code === query.time_range?.field)?.code ??
    dateFields.value.find(field => query.dimensions.includes(field.code))?.code ?? dateFields.value[0]?.code ?? "";
  dates.value = undefined;
}, { immediate: true });
onBeforeUnmount(() => {
  request++;
  if (busy.value) emit("busy", false);
});

/** 仅提交可调整字段，成功后替换结果及建卡来源。 */
async function apply() {
  if (invalid.value || busy.value) return;
  const current = ++request;
  let timeRange: BiTimeRangeSpec | undefined;
  if (period.value === "current") timeRange = props.query.time_range;
  else if (period.value === "custom" && dates.value) {
    timeRange = { field: dateField.value, start: dates.value[0], end: shiftCalendarDate(dates.value[1], 1), end_exclusive: true };
  } else if (period.value === "7" || period.value === "30") {
    timeRange = { field: dateField.value, start: `now-${period.value}d`, end: "now", end_exclusive: true };
  } else if (period.value === "year_to_date") {
    timeRange = { field: dateField.value, start: "start_of_year", end: "now", end_exclusive: true };
  }
  busy.value = true;
  emit("busy", true);
  try {
    const result = await defBiQueryService.RefineBiQuery({
      query_record_id: props.recordId,
      dimensions: dimensions.value,
      grouping_options: dimensions.value.flatMap(field => {
        const value = grouping.value[field];
        if (!value) return [];
        return [{ field, time_grain: typeof value === "string" ? value : "", hierarchy_level: typeof value === "number" ? value : 0 }];
      }),
      time_range: timeRange
    });
    if (current === request) emit("applied", result);
  } catch (cause) {
    if (current === request) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (current === request) {
      busy.value = false;
      emit("busy", false);
    }
  }
}
</script>

<style scoped>
.bi-query-refinement { width: 100%; margin-block: 10px; }
.bi-query-refinement summary { cursor: pointer; color: var(--el-color-primary); }
.bi-query-refinement__fields { margin-top: 10px; }
.bi-query-refinement :deep(.el-date-editor) { max-width: 100%; }
</style>
