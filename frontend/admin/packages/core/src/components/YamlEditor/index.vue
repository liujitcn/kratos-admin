<template>
  <div class="yaml-editor">
    <div ref="editorHost" class="yaml-editor__host" :class="{ 'is-invalid': invalid }" />
    <div class="yaml-editor__footer">
      <span>{{ statusText }}</span>
      <span>{{ lineCount }} {{ lineLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { basicSetup, EditorView } from "codemirror";
import { yaml } from "@codemirror/lang-yaml";
import { placeholder } from "@codemirror/view";

/** YAML 编辑器组件属性。 */
interface YamlEditorProps {
  modelValue: string;
  placeholderText?: string;
  minHeight?: string;
  invalid?: boolean;
  lineLabel?: string;
  /** 根据光标行列生成状态栏文案。 */
  formatPosition?: (line: number, column: number) => string;
}

const props = withDefaults(defineProps<YamlEditorProps>(), {
  placeholderText: "",
  minHeight: "280px",
  invalid: false,
  lineLabel: "lines",
  formatPosition: (line: number, column: number) => `Ln ${line}, Col ${column}`
});

const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const editorHost = ref<HTMLElement>();
const lineCount = ref(1);
const statusText = ref("");
let editor: EditorView | undefined;

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
      EditorView.lineWrapping,
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

watch(() => props.modelValue, value => {
  if (!editor || value === editor.state.doc.toString()) return;
  editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: value } });
  lineCount.value = editor.state.doc.lines;
});

onBeforeUnmount(() => editor?.destroy());
</script>

<style scoped>
.yaml-editor { width: 100%; overflow: hidden; border: 1px solid var(--el-border-color); border-radius: 4px; }
.yaml-editor:focus-within { border-color: var(--el-color-primary); }
.yaml-editor__host :deep(.cm-editor) { min-height: v-bind(minHeight); }
.yaml-editor__host :deep(.cm-editor.cm-focused) { outline: none; }
.yaml-editor__host.is-invalid { border-color: var(--el-color-danger); }
.yaml-editor__footer { display: flex; justify-content: space-between; min-height: 24px; padding: 3px 10px; color: var(--el-text-color-secondary); background: var(--el-fill-color-lighter); border-top: 1px solid var(--el-border-color-lighter); font-size: 12px; }
</style>
