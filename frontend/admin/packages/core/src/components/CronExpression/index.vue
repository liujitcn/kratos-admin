<template>
  <div class="cron-expression">
    <el-input v-model="currentValue" :placeholder="placeholder" clearable>
      <template #suffix>
        <div class="cron-expression__actions">
          <el-tooltip :content="t('core.cron.edit')" placement="top">
            <el-icon class="cron-expression__icon" @click="handleOpenEditor()">
              <Operation />
            </el-icon>
          </el-tooltip>
        </div>
      </template>
    </el-input>

    <ProDialog v-model="dialogVisible" :title="t('core.cron.title')" width="980px" top="2vh" destroy-on-close>
      <div class="cron-editor">
        <div class="cron-editor__preset">
          <div class="cron-editor__section-title">{{ t("core.cron.presets") }}</div>
          <div class="cron-editor__preset-list">
            <el-tag
              v-for="option in presetOptions"
              :key="option.value"
              class="cron-editor__preset-item"
              effect="plain"
              @click="handleApplyPreset(option.value)"
            >
              {{ option.label }}
            </el-tag>
          </div>
        </div>

        <el-tabs v-model="activeTab">
          <el-tab-pane :label="t('core.cron.segment.second')" name="second">
            <CronSegmentEditor
              unit-key="second"
              :max="59"
              :state="editorState.second"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('second', value)"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('core.cron.segment.minute')" name="minute">
            <CronSegmentEditor
              unit-key="minute"
              :max="59"
              :state="editorState.minute"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('minute', value)"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('core.cron.segment.hour')" name="hour">
            <CronSegmentEditor
              unit-key="hour"
              :max="23"
              :state="editorState.hour"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('hour', value)"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('core.cron.segment.day')" name="day">
            <CronSegmentEditor
              unit-key="day"
              :min="1"
              :max="31"
              :state="editorState.day"
              :supports-unspecified="true"
              :supports-last="true"
              :supports-weekday="true"
              @change="value => handleUpdateSegment('day', value)"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('core.cron.segment.month')" name="month">
            <CronSegmentEditor
              unit-key="month"
              :min="1"
              :max="12"
              :state="editorState.month"
              :supports-unspecified="true"
              @change="value => handleUpdateSegment('month', value)"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('core.cron.segment.year')" name="year">
            <CronSegmentEditor
              unit-key="year"
              :min="currentYear"
              :max="currentYear + 20"
              :state="editorState.year"
              :supports-every="true"
              :supports-unspecified="true"
              :supports-step="false"
              :supports-specific="true"
              @change="value => handleUpdateSegment('year', value)"
            />
          </el-tab-pane>
        </el-tabs>

        <div class="cron-editor__preview">
          <div class="cron-editor__section-title">{{ t("core.cron.expression") }}</div>
          <el-input :model-value="previewExpression" readonly />
          <div class="cron-editor__preview-desc">{{ expressionDescription }}</div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleConfirmEditor">{{ t("common.action.confirm") }}</el-button>
          <el-button @click="dialogVisible = false">{{ t("common.action.cancel") }}</el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="tsx">
import { computed, defineComponent, h, reactive, ref, watch } from "vue";
import type { PropType } from "vue";
import { ElCheckbox, ElCheckboxGroup, ElInputNumber, ElRadio } from "element-plus";
import { Operation } from "@element-plus/icons-vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { useLocaleStore } from "@/locales";

const { t } = useLocaleStore();

/** Cron 单个字段支持的编辑模式。 */
type CronSegmentMode = "every" | "unspecified" | "range" | "step" | "specific" | "last" | "weekday";
/** Cron 表达式字段标识。 */
type CronSegmentKey = "second" | "minute" | "hour" | "day" | "month" | "year";

/** CronExpression 组件属性。 */
interface CronExpressionProps {
  modelValue?: string;
  placeholder?: string;
}

/** Cron 单个字段的编辑状态。 */
interface CronSegmentState {
  mode: CronSegmentMode;
  rangeStart: number;
  rangeEnd: number;
  stepStart: number;
  stepValue: number;
  specific: number[];
  weekday: number;
}

/** Cron 编辑器整体状态。 */
type CronEditorState = Record<CronSegmentKey, CronSegmentState>;

const props = withDefaults(defineProps<CronExpressionProps>(), {
  modelValue: "",
  placeholder: ""
});
const placeholder = computed(() => props.placeholder || t("core.cron.placeholder"));

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const currentYear = new Date().getFullYear();
const dialogVisible = ref(false);
const activeTab = ref<CronSegmentKey>("second");

const presetOptions = computed(() => [
  { label: t("core.cron.preset.every_minute"), value: "0 * * * * *" },
  { label: t("core.cron.preset.every_five_minutes"), value: "0 */5 * * * *" },
  { label: t("core.cron.preset.every_hour"), value: "0 0 * * * *" },
  { label: t("core.cron.preset.daily_midnight"), value: "0 0 0 * * *" },
  { label: t("core.cron.preset.daily_eight"), value: "0 0 8 * * *" },
  { label: t("core.cron.preset.monthly_first"), value: "0 0 0 1 * *" },
  { label: t("core.cron.preset.next_year_daily"), value: `0 0 0 * * ${currentYear + 1}` }
]);

/** 创建单个 Cron 字段的默认编辑状态。 */
function createSegmentState(min = 0): CronSegmentState {
  return {
    mode: "every",
    rangeStart: min,
    rangeEnd: min,
    stepStart: min,
    stepValue: 1,
    specific: [],
    weekday: 1
  };
}

/** 创建完整 Cron 编辑器的默认状态。 */
function createDefaultEditorState(): CronEditorState {
  return {
    second: createSegmentState(),
    minute: createSegmentState(),
    hour: createSegmentState(),
    day: createSegmentState(1),
    month: createSegmentState(1),
    year: { ...createSegmentState(currentYear), mode: "every", rangeStart: currentYear, rangeEnd: currentYear }
  };
}

const editorState = reactive<CronEditorState>(createDefaultEditorState());

const currentValue = computed({
  get: () => props.modelValue,
  set: value => emit("update:modelValue", value)
});

const previewExpression = computed(() => {
  return [
    buildSegmentValue(editorState.second),
    buildSegmentValue(editorState.minute),
    buildSegmentValue(editorState.hour),
    buildSegmentValue(editorState.day),
    buildSegmentValue(editorState.month),
    buildSegmentValue(editorState.year)
  ].join(" ");
});

const expressionDescription = computed(() => {
  const descriptionList = [
    formatSegmentDescription("second", editorState.second),
    formatSegmentDescription("minute", editorState.minute),
    formatSegmentDescription("hour", editorState.hour),
    formatSegmentDescription("day", editorState.day),
    formatSegmentDescription("month", editorState.month),
    formatSegmentDescription("year", editorState.year)
  ].filter(Boolean);
  return descriptionList.length ? descriptionList.join(t("core.cron.description_separator")) : t("core.cron.not_configured");
});

/** 打开 Cron 编辑弹窗，并按当前表达式回填编辑状态。 */
function handleOpenEditor(tab?: CronSegmentKey | "preset") {
  applyExpressionToState(props.modelValue || "0 * * * * *");
  dialogVisible.value = true;
  activeTab.value = tab && tab !== "preset" ? tab : "second";
}

/** 应用常用 Cron 预设表达式。 */
function handleApplyPreset(value: string) {
  applyExpressionToState(value);
}

/**
 * 同步单个分段状态，保持响应式对象引用稳定，避免子组件因引用替换导致回显失效。
 */
function syncSegmentState(target: CronSegmentState, source: CronSegmentState) {
  target.mode = source.mode;
  target.rangeStart = source.rangeStart;
  target.rangeEnd = source.rangeEnd;
  target.stepStart = source.stepStart;
  target.stepValue = source.stepValue;
  target.specific = [...source.specific];
  target.weekday = source.weekday;
}

/**
 * 处理分段编辑器回传，按字段同步状态，避免多次编辑时丢失响应式关联。
 */
function handleUpdateSegment(segment: CronSegmentKey, value: CronSegmentState) {
  syncSegmentState(editorState[segment], value);
}

/** 确认编辑结果并回写外部 v-model。 */
function handleConfirmEditor() {
  emit("update:modelValue", previewExpression.value);
  dialogVisible.value = false;
}

/** 将单个字段编辑状态转换为 Cron 片段。 */
function buildSegmentValue(segment: CronSegmentState) {
  switch (segment.mode) {
    case "every":
      return "*";
    case "unspecified":
      return "?";
    case "range":
      return `${segment.rangeStart}-${segment.rangeEnd}`;
    case "step":
      return `${segment.stepStart}/${segment.stepValue}`;
    case "specific":
      return segment.specific.length ? [...segment.specific].sort((a, b) => a - b).join(",") : "*";
    case "last":
      return "L";
    case "weekday":
      return `${segment.weekday}W`;
    default:
      return "*";
  }
}

/** 将单个字段编辑状态转换为当前语言的预览描述。 */
function formatSegmentDescription(segmentKey: CronSegmentKey, segment: CronSegmentState) {
  const segmentName = t(`core.cron.segment.${segmentKey}`);
  const unit = t(`core.cron.unit.${segmentKey}`);
  switch (segment.mode) {
    case "every":
      return t("core.cron.description.every", { unit });
    case "unspecified":
      return t("core.cron.description.unspecified", { segment: segmentName });
    case "range":
      return t("core.cron.description.range", { segment: segmentName, start: segment.rangeStart, end: segment.rangeEnd });
    case "step":
      return t("core.cron.description.step", {
        start: segment.stepStart,
        unit,
        step: segment.stepValue,
        cycleUnit: t(`core.cron.cycle_unit.${segmentKey}`)
      });
    case "specific":
      return segment.specific.length
        ? t("core.cron.description.specific", {
            segment: segmentName,
            values: segment.specific.map(item => formatSpecificLabel(segmentKey, item)).join(t("core.cron.value_separator"))
          })
        : t("core.cron.description.unspecified", { segment: segmentName });
    case "last":
      return t("core.cron.last_day");
    case "weekday":
      return t("core.cron.description.weekday", { day: segment.weekday });
    default:
      return "";
  }
}

/** 格式化指定值模式下的单个展示值。 */
function formatSpecificLabel(segmentKey: CronSegmentKey, value: number) {
  return t("core.cron.value_with_unit", { value, unit: t(`core.cron.unit.${segmentKey}`) });
}

/**
 * 将外部 Cron 表达式解析后回填到编辑器状态，确保表单回显和再次编辑保持一致。
 */
function applyExpressionToState(expression: string) {
  const parts = expression.trim().split(/\s+/);
  const normalizedParts = parts.length === 6 ? parts : ["0", "*", "*", "*", "*", "*"];

  syncSegmentState(editorState.second, parseSegmentValue(normalizedParts[0], 0));
  syncSegmentState(editorState.minute, parseSegmentValue(normalizedParts[1], 0));
  syncSegmentState(editorState.hour, parseSegmentValue(normalizedParts[2], 0));
  syncSegmentState(editorState.day, parseSegmentValue(normalizedParts[3], 1));
  syncSegmentState(editorState.month, parseSegmentValue(normalizedParts[4], 1));
  syncSegmentState(editorState.year, parseSegmentValue(normalizedParts[5], currentYear));
}

/** 将 Cron 片段解析为单个字段编辑状态。 */
function parseSegmentValue(value: string, min: number) {
  const nextState = createSegmentState(min);
  if (!value || value === "*") {
    nextState.mode = "every";
    return nextState;
  }
  if (value === "?") {
    nextState.mode = "unspecified";
    return nextState;
  }
  if (value === "L") {
    nextState.mode = "last";
    return nextState;
  }
  if (value.endsWith("W")) {
    nextState.mode = "weekday";
    nextState.weekday = Number(value.replace("W", "")) || min;
    return nextState;
  }
  if (value.includes("-")) {
    const [start, end] = value.split("-").map(Number);
    nextState.mode = "range";
    nextState.rangeStart = start;
    nextState.rangeEnd = end;
    return nextState;
  }
  if (value.includes("/")) {
    const [start, step] = value.split("/").map(Number);
    nextState.mode = "step";
    nextState.stepStart = Number.isNaN(start) ? min : start;
    nextState.stepValue = Number.isNaN(step) ? 1 : step;
    return nextState;
  }
  if (value.includes(",")) {
    nextState.mode = "specific";
    nextState.specific = value
      .split(",")
      .map(Number)
      .filter(item => !Number.isNaN(item));
    return nextState;
  }

  const singleValue = Number(value);
  if (!Number.isNaN(singleValue)) {
    nextState.mode = "specific";
    nextState.specific = [singleValue];
  }
  return nextState;
}

watch(
  () => props.modelValue,
  value => {
    // 外部表单重置、编辑弹窗重新赋值时，需要同步回内部编辑态，保证内容可回显。
    applyExpressionToState(value || "0 * * * * *");
  },
  { immediate: true }
);

const CronSegmentEditor = defineComponent({
  name: "CronSegmentEditor",
  props: {
    unitKey: {
      type: String as PropType<CronSegmentKey>,
      required: true
    },
    min: {
      type: Number,
      default: 0
    },
    max: {
      type: Number,
      required: true
    },
    state: {
      type: Object as PropType<CronSegmentState>,
      required: true
    },
    supportsEvery: {
      type: Boolean,
      default: true
    },
    supportsUnspecified: {
      type: Boolean,
      default: false
    },
    supportsStep: {
      type: Boolean,
      default: true
    },
    supportsSpecific: {
      type: Boolean,
      default: true
    },
    supportsLast: {
      type: Boolean,
      default: false
    },
    supportsWeekday: {
      type: Boolean,
      default: false
    }
  },
  emits: ["change"],
  setup(segmentProps, { emit: segmentEmit }) {
    const localState = reactive<CronSegmentState>({ ...segmentProps.state });

    watch(
      () => segmentProps.state,
      value => {
        localState.mode = value.mode;
        localState.rangeStart = value.rangeStart;
        localState.rangeEnd = value.rangeEnd;
        localState.stepStart = value.stepStart;
        localState.stepValue = value.stepValue;
        localState.specific = [...value.specific];
        localState.weekday = value.weekday;
      },
      { deep: true, immediate: true }
    );

    const numberOptions = computed(() => {
      return Array.from({ length: segmentProps.max - segmentProps.min + 1 }, (_, index) => segmentProps.min + index);
    });

    const specificOptions = computed(() => {
      const unit = t(`core.cron.unit.${segmentProps.unitKey}`);
      return numberOptions.value.map(item => ({
        value: item,
        label: t("core.cron.value_with_unit", { value: item, unit })
      }));
    });

    /** 向父级 Cron 编辑器同步当前字段状态。 */
    function emitChange() {
      segmentEmit("change", { ...localState, specific: [...localState.specific] });
    }

    /** 切换当前字段的编辑模式。 */
    function handleModeChange(mode: CronSegmentMode) {
      localState.mode = mode;
      emitChange();
    }

    function handleNumberChange<K extends keyof CronSegmentState>(key: K, value: CronSegmentState[K]) {
      localState[key] = value;
      emitChange();
    }

    /** 同步多选指定值到当前字段状态。 */
    function handleSpecificChange(value: Array<string | number>) {
      localState.specific = value.map(Number).filter(item => !Number.isNaN(item));
      emitChange();
    }

    return () => (
      <div class="segment-editor">
        {segmentProps.supportsEvery && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "every", onChange: () => handleModeChange("every") })}
            <span>{t("core.cron.every_unit", { unit: t(`core.cron.unit.${segmentProps.unitKey}`) })}</span>
          </label>
        )}

        {segmentProps.supportsUnspecified && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "unspecified", onChange: () => handleModeChange("unspecified") })}
            <span>{t("core.cron.unspecified")}</span>
          </label>
        )}

        <label class="segment-editor__row">
          {h(ElRadio, { modelValue: localState.mode, value: "range", onChange: () => handleModeChange("range") })}
          <span>{t("core.cron.range")}</span>
          <span>{t("core.cron.from")}</span>
          {h(ElInputNumber, {
            modelValue: localState.rangeStart,
            min: segmentProps.min,
            max: segmentProps.max,
            controlsPosition: "right",
            "onUpdate:modelValue": value => handleNumberChange("rangeStart", Number(value))
          })}
          <span>{t("core.cron.to")}</span>
          {h(ElInputNumber, {
            modelValue: localState.rangeEnd,
            min: segmentProps.min,
            max: segmentProps.max,
            controlsPosition: "right",
            "onUpdate:modelValue": value => handleNumberChange("rangeEnd", Number(value))
          })}
          <span>{t(`core.cron.unit.${segmentProps.unitKey}`)}</span>
        </label>

        {segmentProps.supportsStep && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "step", onChange: () => handleModeChange("step") })}
            <span>{t("core.cron.repeat")}</span>
            <span>{t("core.cron.from")}</span>
            {h(ElInputNumber, {
              modelValue: localState.stepStart,
              min: segmentProps.min,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("stepStart", Number(value))
            })}
            <span>{t("core.cron.start_every", { unit: t(`core.cron.unit.${segmentProps.unitKey}`) })}</span>
            {h(ElInputNumber, {
              modelValue: localState.stepValue,
              min: 1,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("stepValue", Number(value))
            })}
            <span>{t("core.cron.run_once", { unit: t(`core.cron.cycle_unit.${segmentProps.unitKey}`) })}</span>
          </label>
        )}

        {segmentProps.supportsSpecific && (
          <div class="segment-editor__row segment-editor__row--top">
            {h(ElRadio, { modelValue: localState.mode, value: "specific", onChange: () => handleModeChange("specific") })}
            <span>{t("core.cron.specific")}</span>
            {h(
              ElCheckboxGroup,
              {
                modelValue: localState.specific,
                class: "segment-editor__checkboxes",
                "onUpdate:modelValue": handleSpecificChange
              },
              {
                default: () =>
                  specificOptions.value.map(option =>
                    h(ElCheckbox, { key: option.value, label: option.value }, { default: () => option.label })
                  )
              }
            )}
          </div>
        )}

        {segmentProps.supportsLast && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "last", onChange: () => handleModeChange("last") })}
            <span>{t("core.cron.last_day")}</span>
          </label>
        )}

        {segmentProps.supportsWeekday && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "weekday", onChange: () => handleModeChange("weekday") })}
            <span>{t("core.cron.workday")}</span>
            <span>{t("core.cron.this_month")}</span>
            {h(ElInputNumber, {
              modelValue: localState.weekday,
              min: segmentProps.min,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("weekday", Number(value))
            })}
            <span>{t("core.cron.nearest_workday")}</span>
          </label>
        )}
      </div>
    );
  }
});
</script>

<style scoped lang="scss">
.cron-expression {
  width: 100%;
}
.cron-expression__actions {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding-right: 4px;
}
.cron-expression__icon {
  font-size: 16px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: color 0.2s ease;
}
.cron-expression__icon:hover {
  color: var(--el-color-primary);
}
.cron-expression__panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cron-expression__title,
.cron-editor__section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.cron-expression__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cron-expression__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 4px 0;
  margin-left: 0;
}
.cron-expression__item code {
  color: var(--el-color-primary);
}
.cron-expression__tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cron-editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cron-editor__preset,
.cron-editor__preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: var(--admin-page-radius);
}
.cron-editor__preset-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.cron-editor__preset-item {
  cursor: pointer;
}
.cron-editor__preview-desc {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.segment-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 2px 0 6px;
}
.segment-editor__row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.segment-editor__row :deep(.el-input-number) {
  width: 112px;
}
.segment-editor__row--top {
  align-items: flex-start;
}
.segment-editor__checkboxes {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  gap: 4px 10px;
  min-width: 320px;
}
</style>
