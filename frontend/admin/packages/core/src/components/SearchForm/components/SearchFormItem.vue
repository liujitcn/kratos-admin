<template>
  <Dict
    v-if="shouldUseDictComponent"
    v-model="_searchParam[column.search?.key ?? handleProp(column.prop!)]"
    :code="column.search?.dictCode ?? column.dictCode!"
    :code-type="column.search?.dictValueType ?? column.dictValueType ?? 'number'"
    :placeholder="placeholder.placeholder"
    v-bind="handleSearchProps"
    :style="handleSearchStyle"
  />
  <ElSelect
    v-else-if="column.search?.el === 'select' && !column.search?.render"
    v-bind="{ ...handleSearchProps, ...placeholder, searchParam: _searchParam, clearable }"
    v-model="_searchParam[column.search?.key ?? handleProp(column.prop!)]"
    :style="handleSearchStyle"
  >
    <el-option v-for="(col, index) in columnEnum" :key="index" :label="col[fieldNames.label]" :value="col[fieldNames.value]" />
  </ElSelect>
  <component
    v-else
    :is="searchComponent"
    v-bind="{ ...handleSearchProps, ...placeholder, searchParam: _searchParam, clearable }"
    v-model.trim="_searchParam[column.search?.key ?? handleProp(column.prop!)]"
    :data="column.search?.el === 'tree-select' ? columnEnum : []"
    :options="['cascader', 'select-v2'].includes(column.search?.el!) ? columnEnum : []"
  >
    <template v-if="column.search?.el === 'cascader'" #default="{ data }">
      <span>{{ data[fieldNames.label] }}</span>
    </template>
    <slot v-if="column.search?.el !== 'cascader'"></slot>
  </component>
</template>

<script setup lang="ts" name="SearchFormItem">
import { computed, inject, ref, unref, type Component } from "vue";
import {
  ElCascader,
  ElDatePicker,
  ElInput,
  ElInputNumber,
  ElSelect,
  ElSelectV2,
  ElSlider,
  ElSwitch,
  ElTimePicker,
  ElTimeSelect,
  ElTreeSelect
} from "element-plus";
import Dict from "@/components/Dict/index.vue";
import { handleProp } from "@/utils";
import { ColumnProps, SearchType } from "@/components/ProTable/interface";

/** 单个搜索表单项组件属性。 */
interface SearchFormItem {
  column: ColumnProps;
  searchParam: { [key: string]: any };
}
const props = defineProps<SearchFormItem>();

// Re receive SearchParam
const _searchParam = computed(() => props.searchParam);

const searchComponentMap: Record<SearchType, Component> = {
  input: ElInput,
  "input-number": ElInputNumber,
  select: ElSelect,
  "select-v2": ElSelectV2,
  "tree-select": ElTreeSelect,
  cascader: ElCascader,
  "date-picker": ElDatePicker,
  "time-picker": ElTimePicker,
  "time-select": ElTimeSelect,
  switch: ElSwitch,
  slider: ElSlider
};

/**
 * 获取搜索项实际渲染组件，避免按需加载后动态字符串组件无法解析。
 */
const searchComponent = computed(() => {
  const render = props.column.search?.render;
  if (render) return render;
  const searchEl = props.column.search?.el;
  return searchEl ? searchComponentMap[searchEl] : ElInput;
});

/**
 * 判断当前搜索项是否应直接使用字典组件渲染。
 */
const shouldUseDictComponent = computed(() => {
  return (
    !!(props.column.search?.dictCode ?? props.column.dictCode) &&
    props.column.search?.el === "select" &&
    !props.column.search?.render
  );
});

// 判断 fieldNames 设置 label && value && children 的 key 值
const fieldNames = computed(() => {
  return {
    label: props.column.fieldNames?.label ?? "label",
    value: props.column.fieldNames?.value ?? "value",
    children: props.column.fieldNames?.children ?? "children"
  };
});

// 接收 enumMap (el 为 select-v2 需单独处理 enumData)
const enumMap = inject("enumMap", ref(new Map()));

const columnEnum = computed(() => {
  const searchEnum = unref(props.column.search?.enum);
  if (Array.isArray(searchEnum)) return searchEnum;
  const staticEnum = typeof props.column.enum !== "function" ? unref(props.column.enum) : undefined;
  if (Array.isArray(staticEnum) && staticEnum.length) return staticEnum;

  let enumData = enumMap.value.get(props.column.prop);
  if (!enumData) return [];
  if (props.column.search?.el === "select-v2" && props.column.fieldNames) {
    enumData = enumData.map((item: { [key: string]: any }) => {
      return { ...item, label: item[fieldNames.value.label], value: item[fieldNames.value.value] };
    });
  }
  return enumData;
});

/**
 * 统一处理 Dict 搜索组件的宽度透传，避免覆盖组件默认宽度。
 */
const handleSearchStyle = computed(() => {
  return props.column.search?.props?.style ?? { width: "100%" };
});

// 处理透传的 searchProps (el 为 tree-select、cascader 的时候需要给下默认 label && value && children)
const handleSearchProps = computed(() => {
  const label = fieldNames.value.label;
  const value = fieldNames.value.value;
  const children = fieldNames.value.children;
  const searchEl = props.column.search?.el;
  let searchProps = props.column.search?.props ?? {};
  if (searchEl === "tree-select") {
    searchProps = { ...searchProps, props: { ...searchProps, label, children }, nodeKey: value };
  }
  if (searchEl === "cascader") {
    searchProps = { ...searchProps, props: { ...searchProps, label, value, children } };
  }
  return searchProps;
});

// 处理默认 placeholder
const placeholder = computed(() => {
  const search = props.column.search;
  if (["datetimerange", "daterange", "monthrange"].includes(search?.props?.type) || search?.props?.isRange) {
    return {
      rangeSeparator: search?.props?.rangeSeparator ?? "至",
      startPlaceholder: search?.props?.startPlaceholder ?? "开始时间",
      endPlaceholder: search?.props?.endPlaceholder ?? "结束时间"
    };
  }
  const placeholder = search?.props?.placeholder ?? (search?.el?.includes("input") ? "请输入" : "请选择");
  return { placeholder };
});

// 是否有清除按钮 (当搜索项有默认值时，清除按钮不显示)
const clearable = computed(() => {
  const search = props.column.search;
  return search?.props?.clearable ?? (search?.defaultValue == null || search?.defaultValue == undefined);
});
</script>
