<template>
  <div class="core-rich-text-preview" v-html="sanitizedHtml" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import DOMPurify from "dompurify";
import { formatSrc } from "@/utils/utils";

defineOptions({
  name: "CoreRichTextPreview"
});

/** 富文本预览组件属性。 */
interface RichTextPreviewProps {
  /** 待展示的 HTML 内容。 */
  modelValue: string;
}

const props = defineProps<RichTextPreviewProps>();
const sanitizedHtml = computed(() => {
  const sanitized = DOMPurify.sanitize(props.modelValue, { USE_PROFILES: { html: true } });
  if (typeof DOMParser === "undefined") return sanitized;

  const document = new DOMParser().parseFromString(sanitized, "text/html");
  document.body.querySelectorAll<HTMLElement>("img[src], video[src]").forEach(element => {
    const src = element.getAttribute("src");
    if (src) element.setAttribute("src", formatSrc(src));
  });
  return document.body.innerHTML;
});
</script>

<style scoped lang="scss">
.core-rich-text-preview {
  line-height: 1.7;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.core-rich-text-preview :deep(p) {
  margin: 0 0 8px;
}

.core-rich-text-preview :deep(ul),
.core-rich-text-preview :deep(ol) {
  margin: 0 0 8px;
  padding-left: 24px;
}

.core-rich-text-preview :deep(img),
.core-rich-text-preview :deep(video) {
  max-width: 100%;
  height: auto;
}
</style>
