<template>
  <div class="core-code-block" :style="{ '--core-code-block-max-height': props.maxHeight }">
    <button
      class="core-code-block__copy"
      type="button"
      :title="t('core.markdown.action.copy_code')"
      :aria-label="t('core.markdown.action.copy_code')"
      @click="copyCode"
    >
      <el-icon>
        <Select v-if="copied" />
        <CopyDocument v-else />
      </el-icon>
    </button>
    <pre class="core-code-block__pre"><code class="hljs" v-html="highlightedHtml"></code></pre>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { CopyDocument, Select } from "@element-plus/icons-vue";
import hljs from "highlight.js";
import { t } from "@/locales";
import { copyText } from "@/utils/clipboard";

defineOptions({ name: "CoreCodeBlock" });

/** 代码块组件属性。 */
interface CodeBlockProps {
  /** 代码原文。 */
  code: string;
  /** 高亮语言，缺省时自动检测。 */
  language?: string;
  /** 代码区最大高度。 */
  maxHeight?: string;
}

const props = withDefaults(defineProps<CodeBlockProps>(), {
  language: "",
  maxHeight: "420px"
});
const copied = ref(false);
let copiedTimer: ReturnType<typeof setTimeout> | undefined;

const highlightedHtml = computed(() => {
  const code = props.code ?? "";
  if (props.language && hljs.getLanguage(props.language)) {
    return hljs.highlight(code, { language: props.language, ignoreIllegals: true }).value;
  }
  return hljs.highlightAuto(code).value;
});

/** 复制代码原文并短暂反馈复制结果。 */
async function copyCode() {
  try {
    await copyText(props.code ?? "");
  } catch {
    ElMessage.error(t("core.markdown.message.copy_failed"));
    return;
  }
  ElMessage.success(t("core.clipboard.success"));
  copied.value = true;
  clearTimeout(copiedTimer);
  copiedTimer = setTimeout(() => {
    copied.value = false;
  }, 2000);
}
</script>

<style scoped lang="scss">
.core-code-block {
  position: relative;
  overflow: hidden;
  background: var(--admin-page-card-bg-muted);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.core-code-block__copy {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
  opacity: 0;
  transition: opacity 0.15s ease;
}

.core-code-block:hover .core-code-block__copy,
.core-code-block__copy:focus-visible {
  opacity: 1;
}

.core-code-block__pre {
  max-height: var(--core-code-block-max-height);
  margin: 0;
  overflow: auto;
}

.core-code-block__pre code.hljs {
  display: block;
  padding: 10px 12px;
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--admin-page-text-primary);
  white-space: pre-wrap;
  word-break: break-word;
  background: transparent;
}

.core-code-block :deep(.hljs-comment),
.core-code-block :deep(.hljs-quote) {
  font-style: italic;
  color: var(--admin-page-text-secondary);
}

.core-code-block :deep(.hljs-keyword),
.core-code-block :deep(.hljs-selector-tag),
.core-code-block :deep(.hljs-built_in),
.core-code-block :deep(.hljs-name) {
  font-weight: 600;
  color: var(--el-color-primary);
}

.core-code-block :deep(.hljs-string),
.core-code-block :deep(.hljs-attr),
.core-code-block :deep(.hljs-symbol),
.core-code-block :deep(.hljs-bullet) {
  color: var(--el-color-success);
}

.core-code-block :deep(.hljs-number),
.core-code-block :deep(.hljs-literal),
.core-code-block :deep(.hljs-variable),
.core-code-block :deep(.hljs-template-variable) {
  color: var(--el-color-warning);
}

.core-code-block :deep(.hljs-title),
.core-code-block :deep(.hljs-section),
.core-code-block :deep(.hljs-type) {
  color: var(--el-color-danger);
}
</style>
