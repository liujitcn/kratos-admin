<template>
  <div class="core-rich-text-preview" v-html="renderedHtml" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import DOMPurify from "dompurify";
import { loadFileAssetSource } from "@/utils/fileAsset";

defineOptions({
  name: "CoreRichTextPreview"
});

/** 富文本预览组件属性。 */
interface RichTextPreviewProps {
  /** 待展示的 HTML 内容。 */
  modelValue: string;
}

const props = defineProps<RichTextPreviewProps>();
const renderedHtml = ref("");
let objectUrls: string[] = [];

/** 释放富文本资源创建的 Blob URL。 */
function revokeObjectUrls() {
  objectUrls.forEach(url => URL.revokeObjectURL(url));
  objectUrls = [];
}

/** 加载富文本中的图片和视频资源，并保持 HTML 内容经过净化。 */
async function renderRichText() {
  revokeObjectUrls();
  const sanitized = DOMPurify.sanitize(props.modelValue, { USE_PROFILES: { html: true } });
  if (typeof DOMParser === "undefined") {
    renderedHtml.value = sanitized;
    return;
  }

  const document = new DOMParser().parseFromString(sanitized, "text/html");
  const mediaElements = [...document.body.querySelectorAll<HTMLElement>("img[src], video[src]")];
  await Promise.all(mediaElements.map(async element => {
    const src = element.getAttribute("src");
    if (!src) return;
    try {
      const resolvedSrc = await loadFileAssetSource(src);
      if (resolvedSrc.startsWith("blob:")) objectUrls.push(resolvedSrc);
      element.setAttribute("src", resolvedSrc);
    } catch {
      element.removeAttribute("src");
    }
  }));
  renderedHtml.value = document.body.innerHTML;
}

watch(() => props.modelValue, () => void renderRichText(), { immediate: true });
onBeforeUnmount(revokeObjectUrls);
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
