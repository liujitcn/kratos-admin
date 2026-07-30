<template>
  <div class="cron-expression">
    <el-input v-model="currentValue" :placeholder="placeholder" clearable>
      <template #suffix>
        <div class="cron-expression__actions">
          <el-tooltip content="编辑表达式" placement="top">
            <el-icon class="cron-expression__icon" @click="handleOpenEditor()">
              <Operation />
            </el-icon>
          </el-tooltip>
        </div>
      </template>
    </el-input>

    <el-dialog v-model="dialogVisible" title="编辑定时表达式" width="980px" top="2vh" destroy-on-close>
      <div class="cron-editor">
        <div class="cron-editor__preset">
          <div class="cron-editor__section-title">常用表达式</div>
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
          <el-tab-pane label="秒" name="second">
            <CronSegmentEditor
              unit="秒"
              :max="59"
              :state="editorState.second"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('second', value)"
            />
          </el-tab-pane>
          <el-tab-pane label="分" name="minute">
            <CronSegmentEditor
              unit="分"
              :max="59"
              :state="editorState.minute"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('minute', value)"
            />
          </el-tab-pane>
          <el-tab-pane label="时" name="hour">
            <CronSegmentEditor
              unit="时"
              :max="23"
              :state="editorState.hour"
              :supports-unspecified="false"
              @change="value => handleUpdateSegment('hour', value)"
            />
          </el-tab-pane>
          <el-tab-pane label="日" name="day">
            <CronSegmentEditor
              unit="日"
              :min="1"
              :max="31"
              :state="editorState.day"
              :supports-unspecified="true"
              :supports-last="true"
              :supports-weekday="true"
              @change="value => handleUpdateSegment('day', value)"
            />
          </el-tab-pane>
          <el-tab-pane label="月" name="month">
            <CronSegmentEditor
              unit="月"
              :min="1"
              :max="12"
              :state="editorState.month"
              :supports-unspecified="true"
              @change="value => handleUpdateSegment('month', value)"
            />
          </el-tab-pane>
          <el-tab-pane label="年" name="year">
            <CronSegmentEditor
              unit="年"
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
          <div class="cron-editor__section-title">Cron 表达式</div>
          <el-input :model-value="previewExpression" readonly />
          <div class="cron-editor__preview-desc">{{ expressionDescription }}</div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleConfirmEditor">确 定</el-button>
          <el-button @click="dialogVisible = false">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="tsx">
import { computed, defineComponent, h, reactive, ref, watch } from "vue";
import type { PropType } from "vue";
import { ElCheckbox, ElCheckboxGroup, ElInputNumber, ElRadio } from "element-plus";
import { Operation } from "@element-plus/icons-vue";

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
  placeholder: "请输入 cron 表达式"
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const currentYear = new Date().getFullYear();
const dialogVisible = ref(false);
const activeTab = ref<CronSegmentKey>("second");

const presetOptions = [
  { label: "每分钟执行", value: "0 * * * * *" },
  { label: "每 5 分钟执行", value: "0 */5 * * * *" },
  { label: "每小时执行", value: "0 0 * * * *" },
  { label: "每天零点执行", value: "0 0 0 * * *" },
  { label: "每天早上 8 点执行", value: "0 0 8 * * *" },
  { label: "每月 1 号零点执行", value: "0 0 0 1 * *" },
  { label: "明年每天零点执行", value: `0 0 0 * * ${currentYear + 1}` }
];

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
    formatSegmentDescription("秒", editorState.second),
    formatSegmentDescription("分钟", editorState.minute),
    formatSegmentDescription("小时", editorState.hour),
    formatSegmentDescription("日期", editorState.day),
    formatSegmentDescription("月份", editorState.month),
    formatSegmentDescription("年份", editorState.year)
  ].filter(Boolean);
  return descriptionList.length ? descriptionList.join("，") : "未配置执行规则";
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

/** 将单个字段编辑状态转换为中文预览描述。 */
function formatSegmentDescription(label: string, segment: CronSegmentState) {
  switch (segment.mode) {
    case "every":
      return `每${label === "分钟" ? "分钟" : label === "小时" ? "小时" : label}`;
    case "unspecified":
      return `${label}不指定`;
    case "range":
      return `${label}${segment.rangeStart}到${segment.rangeEnd}`;
    case "step":
      return `从${segment.stepStart}${formatUnitSuffix(label)}开始，每${segment.stepValue}${formatCycleSuffix(label)}执行一次`;
    case "specific":
      return segment.specific.length
        ? `${label}指定为 ${segment.specific.map(item => formatSpecificLabel(label, item)).join("、")}`
        : `${label}未指定`;
    case "last":
      return "本月最后一天";
    case "weekday":
      return `本月 ${segment.weekday} 号最近的工作日`;
    default:
      return "";
  }
}

/** 获取 Cron 字段在范围描述中的单位后缀。 */
function formatUnitSuffix(label: string) {
  if (label === "日期") return "日";
  if (label === "月份") return "月";
  if (label === "年份") return "年";
  return label;
}

/** 获取 Cron 字段在周期描述中的单位后缀。 */
function formatCycleSuffix(label: string) {
  if (label === "日期") return "天";
  if (label === "月份") return "个月";
  if (label === "年份") return "年";
  return label;
}

/** 格式化指定值模式下的单个展示值。 */
function formatSpecificLabel(label: string, value: number) {
  if (label === "年份") return `${value}年`;
  return String(value);
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
    unit: {
      type: String,
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
      return numberOptions.value.map(item => ({
        value: item,
        label: segmentProps.unit === "年" ? `${item}年` : `${item}${segmentProps.unit}`
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
            <span>每{segmentProps.unit}</span>
          </label>
        )}

        {segmentProps.supportsUnspecified && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "unspecified", onChange: () => handleModeChange("unspecified") })}
            <span>不指定</span>
          </label>
        )}

        <label class="segment-editor__row">
          {h(ElRadio, { modelValue: localState.mode, value: "range", onChange: () => handleModeChange("range") })}
          <span>周期</span>
          <span>从</span>
          {h(ElInputNumber, {
            modelValue: localState.rangeStart,
            min: segmentProps.min,
            max: segmentProps.max,
            controlsPosition: "right",
            "onUpdate:modelValue": value => handleNumberChange("rangeStart", Number(value))
          })}
          <span>至</span>
          {h(ElInputNumber, {
            modelValue: localState.rangeEnd,
            min: segmentProps.min,
            max: segmentProps.max,
            controlsPosition: "right",
            "onUpdate:modelValue": value => handleNumberChange("rangeEnd", Number(value))
          })}
          <span>{segmentProps.unit}</span>
        </label>

        {segmentProps.supportsStep && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "step", onChange: () => handleModeChange("step") })}
            <span>循环</span>
            <span>从</span>
            {h(ElInputNumber, {
              modelValue: localState.stepStart,
              min: segmentProps.min,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("stepStart", Number(value))
            })}
            <span>{segmentProps.unit}开始，每</span>
            {h(ElInputNumber, {
              modelValue: localState.stepValue,
              min: 1,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("stepValue", Number(value))
            })}
            <span>{segmentProps.unit}执行一次</span>
          </label>
        )}

        {segmentProps.supportsSpecific && (
          <div class="segment-editor__row segment-editor__row--top">
            {h(ElRadio, { modelValue: localState.mode, value: "specific", onChange: () => handleModeChange("specific") })}
            <span>指定</span>
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
            <span>本月最后一天</span>
          </label>
        )}

        {segmentProps.supportsWeekday && (
          <label class="segment-editor__row">
            {h(ElRadio, { modelValue: localState.mode, value: "weekday", onChange: () => handleModeChange("weekday") })}
            <span>工作日</span>
            <span>本月</span>
            {h(ElInputNumber, {
              modelValue: localState.weekday,
              min: segmentProps.min,
              max: segmentProps.max,
              controlsPosition: "right",
              "onUpdate:modelValue": value => handleNumberChange("weekday", Number(value))
            })}
            <span>号最近的工作日</span>
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
  border-radius: 8px;
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
