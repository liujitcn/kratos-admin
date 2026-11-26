<template>
  <MarkdownPreview
    class="agent-markdown"
    :class="{ 'is-streaming': streaming }"
    :model-value="content"
    :is-dark="globalStore.isDark"
    max-code-height="360px"
  />
</template>

<script setup lang="ts" name="AiMarkdown">
import MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";
import { useGlobalStore } from "@liujitcn/kratos-admin-core/stores/runtime";

defineProps<{
  /** 助手回复正文。 */
  content: string;
  /** 本轮仍在接收增量。 */
  streaming?: boolean;
}>();

const globalStore = useGlobalStore();
</script>

<style scoped lang="scss">
.agent-markdown {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;

  :deep(.md-editor-preview) {
    padding: 0;
    font-family: inherit;
    font-size: 15px;
    line-height: 26px;
    letter-spacing: normal;
  }

  :deep(.md-editor-preview > *) {
    margin-block: 0 14px;
  }

  :deep(.md-editor-preview > :last-child) {
    margin-bottom: 0;
  }

  :deep(:is(h1, h2, h3, h4, h5, h6)) {
    padding: 0;
    border-bottom: 0;
    font-weight: 650;
    line-height: 1.42;
  }

  :deep(h1) { font-size: 23px; }
  :deep(h2) { font-size: 20px; }
  :deep(h3) { font-size: 17px; }
  :deep(:is(h4, h5, h6)) { font-size: 15px; }

  :deep(:is(ul, ol)) {
    padding-inline-start: 24px;
  }

  :deep(li + li) {
    margin-top: 3px;
  }

  :deep(blockquote) {
    padding: 4px 0 4px 12px;
    border-inline-start: 2px solid var(--admin-page-divider-strong);
    background: transparent;
  }

  :deep(table) {
    min-width: 0;
    max-width: 100%;
    overflow-x: auto;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
  }

  :deep(:is(th, td)) {
    min-width: auto;
    padding: 8px 12px;
    word-break: normal;
    overflow-wrap: normal;
    white-space: nowrap;
  }

  :deep(.md-editor-preview .md-editor-code .md-editor-code-head),
  :deep(.md-editor-code:focus-within .md-editor-code-head) {
    top: 3px;
    opacity: 1;
    pointer-events: auto;
  }

  :deep(.md-editor-code pre code) {
    padding-top: 34px;
    font-size: 13px;
    line-height: 1.65;
  }

  :deep(img) {
    max-width: 100%;
    max-height: 460px;
    object-fit: contain;
  }
}

.agent-markdown.is-streaming :deep(.md-editor-preview > p:last-child)::after {
  display: inline-block;
  width: 0.42em;
  height: 0.42em;
  margin-inline-start: 0.25em;
  border-radius: 50%;
  background: var(--el-color-primary);
  content: "";
}
</style>
