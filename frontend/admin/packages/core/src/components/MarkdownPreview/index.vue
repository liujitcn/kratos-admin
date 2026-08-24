<template>
  <MdPreview
    v-bind="$attrs"
    :id="resolvedId"
    class="core-markdown-preview"
    :model-value="props.modelValue"
    :theme="props.isDark ? 'dark' : 'light'"
    preview-theme="github"
    code-theme="github"
    :language="locale"
    :show-code-row-number="false"
    :code-foldable="false"
    :no-mermaid="true"
    :no-katex="true"
    :no-echarts="true"
    :no-img-zoom-in="true"
    :sanitize="sanitizeMarkdownHtml"
    :md-heading-id="createMarkdownHeadingId"
    :style="{ '--core-markdown-code-max-height': props.maxCodeHeight }"
  />
</template>

<script setup lang="ts">
import { computed, useId } from "vue";
import DOMPurify from "dompurify";
import hljs from "highlight.js";
import { config as configureMarkdownPreview, MdPreview } from "md-editor-v3";
import type { MdHeadingId } from "md-editor-v3";
import { getLocaleText, useLocaleStore } from "@/locales";
import "md-editor-v3/lib/preview.css";

defineOptions({
  name: "CoreMarkdownPreview",
  inheritAttrs: false
});

/** Markdown 预览组件属性。 */
interface MarkdownPreviewProps {
  /** Markdown 原文。 */
  modelValue: string;
  /** 是否使用暗色主题。 */
  isDark?: boolean;
  /** 预览根节点 ID。 */
  id?: string;
  /** 代码块最大高度。 */
  maxCodeHeight?: string;
}

const props = withDefaults(defineProps<MarkdownPreviewProps>(), {
  isDark: false,
  id: "",
  maxCodeHeight: "480px"
});
const generatedId = useId();
const resolvedId = computed(() => props.id || `core-markdown-preview-${generatedId.replaceAll(":", "")}`);
const markdownHeadingIdCounts = new Map<string, number>();
const { locale } = useLocaleStore();

configureMarkdownPreview({
  editorExtensions: {
    highlight: {
      instance: hljs
    }
  },
  editorConfig: {
    languageUserDefined: {
      "zh-TW": {
        copyCode: {
          text: getLocaleText("zh-TW", "core.markdown.action.copy_code"),
          successTips: getLocaleText("zh-TW", "core.markdown.message.copy_success"),
          failTips: getLocaleText("zh-TW", "core.markdown.message.copy_failed")
        }
      },
      "ja-JP": {
        copyCode: {
          text: getLocaleText("ja-JP", "core.markdown.action.copy_code"),
          successTips: getLocaleText("ja-JP", "core.markdown.message.copy_success"),
          failTips: getLocaleText("ja-JP", "core.markdown.message.copy_failed")
        }
      }
    }
  }
});

/** 清洗 Markdown 中允许渲染的原生 HTML。 */
function sanitizeMarkdownHtml(html: string) {
  return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
}

/** 生成兼容中文和重复标题的稳定 Markdown 锚点。 */
function createMarkdownHeadingId({ text, index }: Parameters<MdHeadingId>[0]) {
  if (index === 1) markdownHeadingIdCounts.clear();
  const slug =
    text
      .trim()
      .toLocaleLowerCase()
      .replace(/\s+/g, "-")
      .replace(/[^\p{Letter}\p{Number}_-]/gu, "") || `heading-${index}`;
  const duplicateIndex = markdownHeadingIdCounts.get(slug) ?? 0;
  markdownHeadingIdCounts.set(slug, duplicateIndex + 1);
  return duplicateIndex ? `${slug}-${duplicateIndex}` : slug;
}
</script>

<style scoped lang="scss">
.core-markdown-preview {
  color: var(--admin-page-text-primary);
  background: transparent;

  --md-color: var(--admin-page-text-primary);
  --md-hover-color: var(--admin-page-text-primary);
  --md-bk-color: transparent;
  --md-bk-color-outstand: var(--admin-page-card-bg-muted);
  --md-bk-hover-color: var(--admin-page-card-bg-soft);
  --md-border-color: var(--admin-page-card-border-soft);
  --md-border-hover-color: var(--admin-page-divider-strong);
  --md-border-active-color: var(--admin-page-text-secondary);
}
.core-markdown-preview :deep(.md-editor-preview) {
  font-size: 15px;
  line-height: 1.75;
  color: var(--admin-page-text-primary);
  word-break: normal;

  --md-theme-color: var(--admin-page-text-primary);
  --md-theme-heading-color: var(--admin-page-text-primary);
  --md-theme-heading-6-color: var(--admin-page-text-secondary);
  --md-theme-heading-bg-color: transparent;
  --md-theme-heading-1-border: 1px solid var(--admin-page-divider-strong);
  --md-theme-heading-2-border: 1px solid var(--admin-page-divider);
  --md-theme-link-color: var(--el-color-primary);
  --md-theme-link-hover-color: var(--el-color-primary-light-3);
  --md-theme-border-color: var(--admin-page-card-border-soft);
  --md-theme-quote-color: var(--admin-page-text-secondary);
  --md-theme-quote-border: 4px solid var(--el-color-primary-light-5);
  --md-theme-table-stripe-color: var(--el-fill-color-lighter);
  --md-theme-table-tr-bg-color: transparent;
  --md-theme-table-td-border-color: var(--admin-page-card-border-soft);
  --md-theme-code-inline-color: var(--admin-page-accent-soft-text);
  --md-theme-code-inline-bg-color: var(--admin-page-card-bg-muted);
  --md-theme-code-inline-radius: 4px;
  --md-theme-code-block-color: var(--admin-page-text-primary);
  --md-theme-code-block-bg-color: var(--admin-page-card-bg-muted);
  --md-theme-code-before-bg-color: var(--admin-page-card-bg-muted);
  --md-theme-code-block-radius: var(--admin-page-radius);
}
.core-markdown-preview :deep(.md-editor-preview > :first-child) {
  margin-top: 0;
}
.core-markdown-preview :deep(.md-editor-preview > :last-child) {
  margin-bottom: 0;
}
.core-markdown-preview :deep(h1),
.core-markdown-preview :deep(h2),
.core-markdown-preview :deep(h3),
.core-markdown-preview :deep(h4),
.core-markdown-preview :deep(h5),
.core-markdown-preview :deep(h6) {
  line-height: 1.35;
  color: var(--admin-page-text-primary);
  letter-spacing: 0;
  scroll-margin-top: 16px;
}
.core-markdown-preview :deep(a:hover) {
  text-underline-offset: 3px;
}
.core-markdown-preview :deep(blockquote) {
  padding: 10px 16px;
  margin: 20px 0;
  background: var(--admin-page-card-bg-soft);
}
.core-markdown-preview :deep(blockquote > :last-child) {
  margin-bottom: 0;
}
.core-markdown-preview :deep(table) {
  display: block;
  width: max-content;
  min-width: 100%;
  max-width: 100%;
  margin: 18px 0 24px;
  overflow-x: auto;
  border-spacing: 0;
  border-collapse: collapse;
}
.core-markdown-preview :deep(th),
.core-markdown-preview :deep(td) {
  min-width: 120px;
  padding: 9px 12px;
  line-height: 1.55;
  vertical-align: top;
  text-align: left;
  border: 1px solid var(--admin-page-card-border-soft);
}
.core-markdown-preview :deep(th) {
  font-weight: 600;
  background: var(--admin-page-card-bg-soft);
}
.core-markdown-preview :deep(tbody tr:nth-child(even)) {
  background: var(--el-fill-color-lighter);
}
.core-markdown-preview :deep(.md-editor-code) {
  margin: 16px 0 20px;
  overflow: hidden;
  background: var(--admin-page-card-bg-muted);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.core-markdown-preview :deep(.md-editor-preview .md-editor-code .md-editor-code-head) {
  position: absolute;
  inset: 6px 6px auto auto;
  z-index: 2;
  display: block;
  width: auto;
  height: auto;
  margin: 0;
  pointer-events: none;
  background: transparent;
  opacity: 0;
  transition: opacity 0.15s ease;
}
.core-markdown-preview :deep(.md-editor-code:hover .md-editor-code-head) {
  pointer-events: auto;
  opacity: 1;
}
.core-markdown-preview :deep(.md-editor-code-flag),
.core-markdown-preview :deep(.md-editor-code-lang),
.core-markdown-preview :deep(.md-editor-collapse-tips) {
  display: none;
}
.core-markdown-preview :deep(.md-editor-code .md-editor-code-action > *) {
  margin: 0;
}
.core-markdown-preview :deep(.md-editor-copy-button) {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 8px;
  font-size: 12px;
  line-height: 1;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 4px;
  box-shadow: var(--admin-page-shadow);
}
.core-markdown-preview :deep(.md-editor-code pre) {
  max-height: var(--core-markdown-code-max-height);
  margin: 0;
  overflow: auto;
}
.core-markdown-preview :deep(.md-editor-code pre code) {
  min-height: 0;
  padding: 10px 12px;
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--admin-page-text-primary);
  letter-spacing: 0;
  white-space: pre;
  background: transparent;
  border-radius: 0;
}
.core-markdown-preview :deep(.hljs-comment),
.core-markdown-preview :deep(.hljs-quote) {
  font-style: italic;
  color: var(--admin-page-text-secondary);
}
.core-markdown-preview :deep(.hljs-keyword),
.core-markdown-preview :deep(.hljs-selector-tag),
.core-markdown-preview :deep(.hljs-built_in),
.core-markdown-preview :deep(.hljs-name) {
  color: var(--el-color-primary);
}
.core-markdown-preview :deep(.hljs-string),
.core-markdown-preview :deep(.hljs-attr),
.core-markdown-preview :deep(.hljs-symbol),
.core-markdown-preview :deep(.hljs-bullet) {
  color: var(--el-color-success);
}
.core-markdown-preview :deep(.hljs-number),
.core-markdown-preview :deep(.hljs-literal),
.core-markdown-preview :deep(.hljs-variable),
.core-markdown-preview :deep(.hljs-template-variable) {
  color: var(--el-color-warning);
}
.core-markdown-preview :deep(.hljs-title),
.core-markdown-preview :deep(.hljs-section),
.core-markdown-preview :deep(.hljs-type) {
  color: var(--el-color-danger);
}
.core-markdown-preview :deep(hr) {
  height: 1px;
  margin: 28px 0;
  background: var(--admin-page-divider-strong);
  border: 0;
}
.core-markdown-preview :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: var(--admin-page-radius);
}

@media (width <= 720px) {
  .core-markdown-preview :deep(h1) {
    font-size: 26px;
  }
  .core-markdown-preview :deep(h2) {
    font-size: 21px;
  }
}
</style>
