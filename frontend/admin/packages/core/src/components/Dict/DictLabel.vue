<template>
  <el-tag v-if="tagType" :type="tagType" :size="size">{{ label }}</el-tag>
  <span v-else>{{ label }}</span>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
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
const label = ref("");
const tagType = ref<TagType | undefined>();

/**
 * 过滤后端返回的 tag 类型，确保只传递 Element Plus 支持的枚举值。
 */
function normalizeTagType(rawTagType?: string): TagType | undefined {
  const supportedTagTypes: TagType[] = ["success", "warning", "info", "primary", "danger"];
  if (!rawTagType) return undefined;
  return supportedTagTypes.find(tag => tag === rawTagType);
}

/**
 * 根据字典值刷新标签文本和标签类型。
 */
async function refreshLabelAndTag() {
  if (!props.code) {
    label.value = "";
    tagType.value = undefined;
    return;
  }

  // 字典缓存可能来自持久化旧数据，按当前编码兜底刷新一次。
  const dictList = await dictStore.ensureDictionary(props.code);

  const matchedItem = dictList.find(dictItem => dictItem.value == props.modelValue);
  label.value = matchedItem?.label ?? "";
  tagType.value = normalizeTagType(matchedItem?.tag_type);
}

watch(
  () => [props.code, props.modelValue],
  async () => {
    await refreshLabelAndTag();
  },
  { immediate: true }
);
</script>
