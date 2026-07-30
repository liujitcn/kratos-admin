<template>
  <div class="dynamic-list">
    <div v-for="(item, index) in modelValue" :key="index" class="dynamic-list__item">
      <el-input v-model="modelValue[index]" v-bind="inputProps" />
      <el-button :icon="Delete" circle type="danger" plain @click="handleRemove(index)" />
    </div>
    <el-button :icon="Plus" type="primary" plain @click="handleAdd">添加</el-button>
  </div>
</template>

<script setup lang="ts" name="DynamicList">
import { computed } from "vue";
import { Delete, Plus } from "@element-plus/icons-vue";

/** 动态字符串列表组件属性。 */
interface DynamicListProps {
  modelValue?: string[];
  inputProps?: Record<string, any>;
}

const props = withDefaults(defineProps<DynamicListProps>(), {
  modelValue: () => [],
  inputProps: () => ({ placeholder: "请输入内容" })
});

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
}>();

/** 统一接管动态列表值。 */
const modelValue = computed({
  get: () => props.modelValue,
  set: value => emit("update:modelValue", value)
});

/** 新增一行动态输入。 */
function handleAdd() {
  modelValue.value = [...modelValue.value, ""];
}

/** 删除指定下标的动态输入。 */
function handleRemove(index: number) {
  modelValue.value = modelValue.value.filter((_, currentIndex) => currentIndex !== index);
}
</script>

<style scoped lang="scss">
.dynamic-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}
.dynamic-list__item {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
