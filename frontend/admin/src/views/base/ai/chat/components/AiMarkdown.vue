<template>
  <MarkdownRenderer
    class="agent-markdown"
    :markdown="content"
    :allow-html="false"
    :sanitize="true"
    :enable-breaks="true"
    :enable-latex="false"
    :enable-animate="streaming"
    :enable-shiki="false"
    :enable-mermaid="false"
    :is-dark="globalStore.isDark"
    :show-code-block-header="true"
    :sticky-code-block-header="false"
    :enable-code-line-number="false"
    code-max-height="360px"
    :style="markdownRootStyle"
  />
</template>

<script setup lang="ts" name="AiMarkdown">
import { MarkdownRenderer } from "x-markdown-vue";
import "x-markdown-vue/style";
import { useGlobalStore } from "@/stores/modules/global";

/** AI 回复 Markdown 渲染组件入参。 */
type AiMarkdownProps = {
  /** 需要渲染的 Markdown 正文。 */
  content: string;
  /** 是否处于流式输出中，开启后按词显示动画。 */
  streaming?: boolean;
};

withDefaults(defineProps<AiMarkdownProps>(), {
  streaming: false
});

const globalStore = useGlobalStore();
/** 覆盖 MarkdownRenderer 默认根样式，避免内置 16px padding 破坏消息列对齐。 */
const markdownRootStyle = {
  padding: "0",
  color: "inherit",
  backgroundColor: "transparent"
};
</script>

<style scoped lang="scss">
.agent-markdown {
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  line-height: 28px;
  color: inherit;
  overflow-wrap: anywhere;
  margin: 0;
  padding: 0;
  :deep(.x-md-core) {
    box-sizing: border-box;
    width: 100%;
    max-width: 100%;
    margin: 0;
    padding: 0;
    overflow-wrap: anywhere;
  }
  :deep(.x-md-core > *) {
    margin-left: 0;
    padding-left: 0;
  }
  :deep(.x-md-core > :first-child) {
    margin-top: 0;
  }
  :deep(.x-md-core > :last-child) {
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
  :deep(.x-md-code-block),
  :deep(.markdown-mermaid) {
    max-width: 100%;
    border-radius: var(--admin-page-radius);
  }
  :deep(.x-md-code-header) {
    color: var(--admin-page-text-primary);
  }
  :deep(.x-md-plain-pre),
  :deep(.x-md-syntax-code-block pre) {
    max-width: 100%;
    font-size: 13px;
  }
  :deep(.x-md-inline-code) {
    color: var(--el-color-primary);
    border-radius: 6px;
  }
}
</style>
