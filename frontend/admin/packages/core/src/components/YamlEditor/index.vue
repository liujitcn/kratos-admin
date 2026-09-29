<template>
  <div class="yaml-editor" :class="{ 'is-invalid': invalid, 'is-bounded': !!maxHeight }">
    <div class="yaml-editor__toolbar">
      <span class="yaml-editor__lang">
        <el-icon><Document /></el-icon>
        <span>YAML</span>
      </span>
      <div class="yaml-editor__actions">
        <button
          type="button"
          class="yaml-editor__action"
          :class="{ 'is-active': wrapEnabled }"
          :title="t('core.yaml_editor.action.wrap')"
          :aria-label="t('core.yaml_editor.action.wrap')"
          @click="toggleWrap"
        >
          <el-icon :size="15"><WrapText /></el-icon>
        </button>
        <button
          type="button"
          class="yaml-editor__action"
          :title="t('core.markdown.action.copy_code')"
          :aria-label="t('core.markdown.action.copy_code')"
          @click="copyCode"
        >
          <el-icon :size="15">
            <Select v-if="copied" />
            <CopyDocument v-else />
          </el-icon>
        </button>
      </div>
    </div>
    <div ref="editorHost" class="yaml-editor__host" />
    <div class="yaml-editor__footer">
      <span>{{ statusText }}</span>
      <span>{{ lineCount }} {{ lineLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { basicSetup, EditorView } from "codemirror";
import { Compartment } from "@codemirror/state";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";
import { yaml } from "@codemirror/lang-yaml";
import { placeholder } from "@codemirror/view";
import { CopyDocument, Document, Select } from "@element-plus/icons-vue";
import { WrapText } from "@lucide/vue";
import { t } from "@/locales";
import { copyText } from "@/utils/clipboard";

/** YAML 编辑器组件属性。 */
interface YamlEditorProps {
  modelValue: string;
  placeholderText?: string;
  minHeight?: string;
  /** 编辑器最大高度；内容超出后编辑器不再撑高页面，改为内部滚动。 */
  maxHeight?: string;
  /** 是否默认开启自动换行，可通过工具栏切换。 */
  wrap?: boolean;
  invalid?: boolean;
  lineLabel?: string;
  /** 根据光标行列生成状态栏文案。 */
  formatPosition?: (line: number, column: number) => string;
}

const props = withDefaults(defineProps<YamlEditorProps>(), {
  placeholderText: "",
  minHeight: "280px",
  maxHeight: undefined,
  wrap: true,
  invalid: false,
  lineLabel: "lines",
  formatPosition: (line: number, column: number) => `Ln ${line}, Col ${column}`
});

const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const editorHost = ref<HTMLElement>();
const lineCount = ref(1);
const statusText = ref("");
const wrapEnabled = ref(props.wrap);
const copied = ref(false);
let copiedTimer: ReturnType<typeof setTimeout> | undefined;
let editor: EditorView | undefined;

const wrapCompartment = new Compartment();

/** 与管理端配色一致的 YAML 高亮样式（键绿、串蓝、值橙、注释灰斜体）。 */
const yamlHighlightStyle = HighlightStyle.define([
  { tag: [tags.propertyName, tags.attributeName], color: "var(--el-color-success)" },
  { tag: [tags.string, tags.attributeValue], color: "var(--el-color-primary)" },
  { tag: [tags.number, tags.bool, tags.keyword], color: "var(--el-color-warning)" },
  { tag: [tags.lineComment, tags.blockComment], color: "var(--el-text-color-secondary)", fontStyle: "italic" },
  { tag: [tags.typeName, tags.labelName, tags.definition(tags.variableName)], color: "var(--el-color-danger)" },
  { tag: [tags.meta, tags.separator, tags.punctuation], color: "var(--el-text-color-placeholder)" }
]);

const editorTheme = EditorView.theme({
  "&": { color: "var(--el-text-color-primary)", backgroundColor: "var(--el-fill-color-blank)", fontSize: "13px" },
  ".cm-content": { minHeight: props.minHeight, padding: "12px 0", caretColor: "var(--el-color-primary)" },
  ".cm-gutters": { color: "var(--el-text-color-placeholder)", backgroundColor: "var(--el-fill-color-light)", border: "none", borderRight: "1px solid var(--el-border-color-lighter)" },
  ".cm-activeLineGutter, .cm-activeLine": { backgroundColor: "var(--el-fill-color-lighter)" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--el-color-primary)" },
  ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": { backgroundColor: "var(--el-color-primary-light-8)" },
  ".cm-scroller": { overflow: "auto", fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', monospace" },
  ".cm-placeholder": { color: "var(--el-text-color-placeholder)" }
}, { dark: document.documentElement.classList.contains("dark") });

onMounted(() => {
  if (!editorHost.value) return;
  editor = new EditorView({
    doc: props.modelValue,
    parent: editorHost.value,
    extensions: [
      basicSetup,
      yaml(),
      editorTheme,
      syntaxHighlighting(yamlHighlightStyle),
      wrapCompartment.of(wrapEnabled.value ? EditorView.lineWrapping : []),
      EditorView.contentAttributes.of({ "aria-label": "YAML editor", spellcheck: "false" }),
      EditorView.updateListener.of(update => {
        if (!update.docChanged) return;
        const value = update.state.doc.toString();
        lineCount.value = update.state.doc.lines;
        statusText.value = "";
        emit("update:modelValue", value);
      }),
      EditorView.domEventHandlers({
        keyup: (_event, view) => {
          const cursor = view.state.selection.main.head;
          const line = view.state.doc.lineAt(cursor);
          statusText.value = props.formatPosition(line.number, cursor - line.from + 1);
          return false;
        },
        click: (_event, view) => {
          const cursor = view.state.selection.main.head;
          const line = view.state.doc.lineAt(cursor);
          statusText.value = props.formatPosition(line.number, cursor - line.from + 1);
          return false;
        }
      }),
      ...(props.placeholderText ? [placeholder(props.placeholderText)] : [])
    ]
  });
  lineCount.value = editor.state.doc.lines;
});

/** 切换自动换行，通过 Compartment 热更新编辑器扩展。 */
function toggleWrap() {
  wrapEnabled.value = !wrapEnabled.value;
  editor?.dispatch({
    effects: wrapCompartment.reconfigure(wrapEnabled.value ? EditorView.lineWrapping : [])
  });
}

/** 复制编辑器全文并短暂反馈复制结果。 */
async function copyCode() {
  const code = editor?.state.doc.toString() ?? "";
  try {
    await copyText(code);
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

watch(() => props.modelValue, value => {
  if (!editor || value === editor.state.doc.toString()) return;
  editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: value } });
  lineCount.value = editor.state.doc.lines;
});

onBeforeUnmount(() => {
  clearTimeout(copiedTimer);
  editor?.destroy();
});
</script>

<style scoped>
.yaml-editor { display: flex; flex-direction: column; width: 100%; overflow: hidden; background: var(--el-fill-color-blank); border: 1px solid var(--el-border-color); border-radius: 4px; }
.yaml-editor.is-bounded { max-height: v-bind(maxHeight); }
.yaml-editor:focus-within { border-color: var(--el-color-primary); }
.yaml-editor.is-invalid { border-color: var(--el-color-danger); }
.yaml-editor__toolbar { display: flex; align-items: center; justify-content: space-between; min-height: 32px; padding: 2px 8px; background: var(--el-fill-color-light); border-bottom: 1px solid var(--el-border-color-lighter); }
.yaml-editor__lang { display: inline-flex; gap: 5px; align-items: center; color: var(--el-text-color-secondary); font-size: 12px; font-weight: 600; letter-spacing: 0.5px; }
.yaml-editor__actions { display: inline-flex; gap: 2px; }
.yaml-editor__action { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; padding: 0; color: var(--el-text-color-secondary); cursor: pointer; background: transparent; border: none; border-radius: 4px; transition: color 0.15s ease, background-color 0.15s ease; }
.yaml-editor__action:hover, .yaml-editor__action:focus-visible { color: var(--el-color-primary); background: var(--el-fill-color); outline: none; }
.yaml-editor__action.is-active { color: var(--el-color-primary); }
.yaml-editor__host { flex: 1 1 auto; min-height: 0; }
.yaml-editor__host :deep(.cm-editor) { display: flex; flex-direction: column; min-height: v-bind(minHeight); height: 100%; }
.yaml-editor__host :deep(.cm-editor.cm-focused) { outline: none; }
.yaml-editor__host :deep(.cm-scroller) { flex: 1 1 auto; min-height: 0; }
.yaml-editor__footer { display: flex; justify-content: space-between; min-height: 24px; padding: 3px 10px; color: var(--el-text-color-secondary); background: var(--el-fill-color-lighter); border-top: 1px solid var(--el-border-color-lighter); font-size: 12px; }
</style>
