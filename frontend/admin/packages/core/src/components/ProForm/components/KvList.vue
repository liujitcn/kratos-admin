<template>
  <div class="kv-list">
    <div v-for="(item, index) in modelValue" :key="index" class="kv-list__item">
      <el-input v-model="item.key" v-bind="keyInputProps" />
      <el-input v-model="item.value" v-bind="valueInputProps" />
      <el-button :icon="Delete" circle type="danger" plain @click="handleRemove(index)" />
    </div>
    <el-button :icon="Plus" type="primary" plain v-bind="addButtonProps" @click="handleAdd">{{ addText }}</el-button>
  </div>
</template>

<script setup lang="ts" name="KvList">
import { computed } from "vue";
import { Delete, Plus } from "@element-plus/icons-vue";

/** 键值对列表单项。 */
interface KvItem {
  key: string;
  value: string;
}

/** 键值对列表组件属性。 */
interface KvListProps {
  modelValue?: KvItem[];
  keyInputProps?: Record<string, any>;
  valueInputProps?: Record<string, any>;
  addText?: string;
  addButtonProps?: Record<string, any>;
}

const props = withDefaults(defineProps<KvListProps>(), {
  modelValue: () => [],
  keyInputProps: () => ({ placeholder: "参数名" }),
  valueInputProps: () => ({ placeholder: "参数值" }),
  addText: "添加",
  addButtonProps: () => ({})
});

const emit = defineEmits<{
  "update:modelValue": [value: KvItem[]];
}>();

/** 统一接管键值对列表值。 */
const modelValue = computed({
  get: () => props.modelValue,
  set: value => emit("update:modelValue", value)
});

/** 新增一行键值对。 */
function handleAdd() {
  modelValue.value = [...modelValue.value, { key: "", value: "" }];
}

/** 删除指定下标的键值对。 */
function handleRemove(index: number) {
  modelValue.value = modelValue.value.filter((_, currentIndex) => currentIndex !== index);
}
</script>

<style scoped lang="scss">
.kv-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}
.kv-list__item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
}
</style>
