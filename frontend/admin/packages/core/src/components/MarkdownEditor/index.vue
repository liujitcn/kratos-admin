<template>
  <MdEditor
    v-bind="$attrs"
    class="core-markdown-editor"
    :model-value="props.modelValue"
    :theme="props.isDark ? 'dark' : 'light'"
    :language="locale"
    preview-theme="github"
    code-theme="github"
    :style="{ height: props.height }"
    :placeholder="props.placeholder"
    :max-length="props.maxLength"
    :no-upload-img="true"
    :no-mermaid="true"
    :no-katex="true"
    :no-echarts="true"
    @update:model-value="value => emit('update:modelValue', value)"
  />
</template>

<script setup lang="ts">
import { MdEditor } from "md-editor-v3";
import { useLocaleStore } from "@/locales";
import "md-editor-v3/lib/style.css";

defineOptions({
  name: "CoreMarkdownEditor",
  inheritAttrs: false
});

/** Markdown 编辑器属性。 */
interface MarkdownEditorProps {
  /** Markdown 原文。 */
  modelValue: string;
  /** 是否使用暗色主题。 */
  isDark?: boolean;
  /** 编辑器高度。 */
  height?: string;
  /** 输入占位文案。 */
  placeholder?: string;
  /** 最大字符数。 */
  maxLength?: number;
}

const props = withDefaults(defineProps<MarkdownEditorProps>(), {
  isDark: false,
  height: "360px",
  placeholder: "",
  maxLength: undefined
});
const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const { locale } = useLocaleStore();
</script>

<style scoped lang="scss">
.core-markdown-editor {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
}
</style>
