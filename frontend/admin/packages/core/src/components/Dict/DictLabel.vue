<template>
  <el-tag v-if="tagType" :type="tagType" :size="size">{{ label }}</el-tag>
  <span v-else>{{ label }}</span>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { useDictStore } from "@/stores/modules/dict";

/** Element Plus 标签支持的展示类型。 */
type TagType = "success" | "warning" | "info" | "primary" | "danger";

/** 字典标签组件属性。 */
interface DictLabelProps {
  code: string;
  modelValue?: string | number;
  size?: "default" | "large" | "small";
}

const props = withDefaults(defineProps<DictLabelProps>(), {
  size: "default"
});

const dictStore = useDictStore();
const matchedItem = computed(() => dictStore.getDictionary(props.code).find(item => item.value == props.modelValue));
const label = computed(() => matchedItem.value?.label ?? "");
const tagType = computed(() => normalizeTagType(matchedItem.value?.tag_type));

/**
 * 过滤后端返回的 tag 类型，确保只传递 Element Plus 支持的枚举值。
 */
function normalizeTagType(rawTagType?: string): TagType | undefined {
  const supportedTagTypes: TagType[] = ["success", "warning", "info", "primary", "danger"];
  if (!rawTagType) return undefined;
  return supportedTagTypes.find(tag => tag === rawTagType);
}

watch(
  () => props.code,
  async code => {
    if (code) await dictStore.ensureDictionary(code);
  },
  { immediate: true }
);
</script>
