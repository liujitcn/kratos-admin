<template>
  <MarkdownPreview
    class="agent-markdown"
    :model-value="content"
    :is-dark="globalStore.isDark"
    max-code-height="360px"
  />
</template>

<script setup lang="ts" name="AiMarkdown">
import MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";
import { useGlobalStore } from "@liujitcn/kratos-admin-core/stores/runtime";

/** AI 回复 Markdown 渲染组件入参。 */
type AiMarkdownProps = {
  /** 需要渲染的 Markdown 正文。 */
  content: string;
  /** 是否处于流式输出中。 */
  streaming?: boolean;
};

withDefaults(defineProps<AiMarkdownProps>(), {
  streaming: false
});

const globalStore = useGlobalStore();
</script>

<style scoped lang="scss">
.agent-markdown {
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  padding: 0;
  margin: 0;
  line-height: 28px;
  color: inherit;
  overflow-wrap: anywhere;
  :deep(.md-editor-preview) {
    box-sizing: border-box;
    width: 100%;
    max-width: 100%;
    padding: 0;
    margin: 0;
    overflow-wrap: anywhere;
  }
  :deep(.md-editor-preview > *) {
    padding-left: 0;
    margin-left: 0;
  }
  :deep(.md-editor-preview > :first-child) {
    margin-top: 0;
  }
  :deep(.md-editor-preview > :last-child) {
    margin-bottom: 0;
  }
  :deep(p),
  :deep(ul),
  :deep(ol),
  :deep(blockquote),
  :deep(table),
  :deep(pre) {
    margin: 6px 0;
  }
  :deep(h1),
  :deep(h2),
  :deep(h3),
  :deep(h4),
  :deep(h5),
  :deep(h6) {
    margin: 8px 0 4px;
    font-weight: 700;
    line-height: 1.4;
    color: var(--admin-page-text-primary);
  }
  :deep(h1) {
    font-size: 20px;
  }
  :deep(h2) {
    font-size: 18px;
  }
  :deep(h3) {
    font-size: 16px;
  }
  :deep(h4),
  :deep(h5),
  :deep(h6) {
    font-size: 14px;
  }
  :deep(ul),
  :deep(ol) {
    padding-left: 16px;
  }
  :deep(li + li) {
    margin-top: 4px;
  }
  :deep(a) {
    color: var(--el-color-primary);
    text-decoration: none;
    &:hover {
      text-decoration: underline;
    }
  }
  :deep(blockquote) {
    padding: 2px 0 2px 12px;
    color: var(--admin-page-text-secondary);
    border-left: 3px solid var(--el-border-color);
  }
  :deep(table) {
    display: block;
    width: 100%;
    max-width: 100%;
    overflow: auto;
    border-collapse: collapse;
  }
  :deep(th),
  :deep(td) {
    padding: 6px 10px;
    border: 1px solid var(--el-border-color-light);
  }
  :deep(th) {
    font-weight: 700;
    background: var(--el-fill-color-light);
  }
  :deep(.md-editor-code) {
    max-width: 100%;
    border-radius: var(--admin-page-radius);
  }
  :deep(.md-editor-code-head) {
    color: var(--admin-page-text-primary);
  }
  :deep(.md-editor-code pre code) {
    max-width: 100%;
    font-size: 13px;
  }
  :deep(code:not(pre code)) {
    color: var(--el-color-primary);
    border-radius: 6px;
  }
}
</style>
