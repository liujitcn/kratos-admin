<template>
  <el-select
    v-if="type === 'select'"
    v-model="selectedSingleValue"
    :placeholder="placeholder"
    :disabled="disabled"
    clearable
    :style="style"
    @change="handleSingleChange"
  >
    <el-option v-for="option in options" :key="option.value" :label="option.label" :value="option.value" />
  </el-select>

  <el-radio-group
    v-else-if="type === 'radio'"
    v-model="selectedSingleValue"
    :disabled="disabled"
    :style="style"
    @change="handleSingleChange"
  >
    <el-radio v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </el-radio>
  </el-radio-group>

  <el-checkbox-group
    v-else-if="type === 'checkbox'"
    v-model="selectedMultiValue"
    :disabled="disabled"
    :style="style"
    @change="handleMultiChange"
  >
    <el-checkbox v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </el-checkbox>
  </el-checkbox-group>
</template>

<script setup lang="ts">
import type { CSSProperties } from "vue";
import { ref, watch } from "vue";
import { useDictStore } from "@/stores/modules/dict";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";

/** 字典组件支持的控件类型。 */
type DictType = "select" | "radio" | "checkbox";
/** 字典值转换后的目标类型。 */
type DictCodeType = "string" | "number";
/** 字典组件对外绑定值类型。 */
type DictValue = string | number;

/** 字典组件属性。 */
interface DictProps {
  code: string;
  codeType?: DictCodeType;
  modelValue?: DictValue | DictValue[];
  type?: DictType;
  placeholder?: string;
  disabled?: boolean;
  style?: CSSProperties;
}

const props = withDefaults(defineProps<DictProps>(), {
  codeType: "number",
  type: "select",
  placeholder: "请选择",
  disabled: false,
  style: () =>
    ({
      width: "300px"
    }) as CSSProperties
});

const emit = defineEmits<{
  (e: "update:modelValue", value: DictValue | DictValue[] | undefined): void;
}>();

const dictStore = useDictStore();
const options = ref<Array<{ label: string; value: DictValue }>>([]);
const selectedSingleValue = ref<DictValue | undefined>();
const selectedMultiValue = ref<DictValue[]>([]);

/**
 * 同步外部传入值到组件内部值。
 */
function syncSelectedValue(modelValue: DictValue | DictValue[] | undefined) {
  if (props.type === "checkbox") {
    selectedMultiValue.value = Array.isArray(modelValue) ? modelValue : [];
    return;
  }
  selectedSingleValue.value = Array.isArray(modelValue) ? undefined : modelValue;
}

/**
 * 按配置将字典值转换为字符串或数字。
 */
function convertDictValue(dictItem: OptionBaseDictResponse_BaseDictItem): DictValue {
  if (props.codeType === "number") return Number(dictItem.value);
  return dictItem.value;
}

/**
 * 加载当前字典选项，若本地无缓存则自动拉取。
 */
async function loadOptions() {
  // 按当前字典编码兜底刷新，避免持久化旧缓存里缺少该字典导致下拉无数据。
  const dictList = await dictStore.ensureDictionary(props.code);

  options.value = dictList.map(dictItem => ({
    label: dictItem.label,
    value: convertDictValue(dictItem)
  }));
}

/**
 * 校验当前值是否仍在可选项内，避免字典切换后残留无效值。
 */
function ensureSelectedValueValid() {
  if (props.type === "checkbox") return;
  if (selectedSingleValue.value === undefined || selectedSingleValue.value === null || selectedSingleValue.value === "") return;
  const matched = options.value.some(option => option.value === selectedSingleValue.value);
  if (matched) return;
  selectedSingleValue.value = undefined;
  emit("update:modelValue", undefined);
}

/**
 * 对外派发单选值变化。
 */
function handleSingleChange(value: string | number | boolean | undefined) {
  emit("update:modelValue", value as DictValue | undefined);
}

/**
 * 对外派发多选值变化。
 */
function handleMultiChange(value: Array<string | number | boolean>) {
  emit("update:modelValue", value as DictValue[]);
}

watch(
  () => props.modelValue,
  modelValue => {
    syncSelectedValue(modelValue);
  },
  { immediate: true }
);

watch(
  () => [props.code, props.codeType],
  async () => {
    await loadOptions();
    ensureSelectedValueValid();
  },
  { immediate: true }
);
</script>
