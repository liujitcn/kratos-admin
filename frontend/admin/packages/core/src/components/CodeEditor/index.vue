<template>
  <div ref="editorHost" class="code-editor" :style="{ height }" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import * as monaco from "monaco-editor";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

/** 代码编辑器属性。 */
interface CodeEditorProps {
  /** 编辑器源码。 */
  modelValue: string;
  /** Monaco 语言标识。 */
  language?: string;
  /** 编辑器高度。 */
  height?: string;
  /** 是否只读。 */
  readonly?: boolean;
}

const props = withDefaults(defineProps<CodeEditorProps>(), {
  language: "javascript",
  height: "360px",
  readonly: false
});

const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const editorHost = ref<HTMLElement>();
let editor: monaco.editor.IStandaloneCodeEditor | undefined;
let model: monaco.editor.ITextModel | undefined;

/** 创建 Monaco 编辑器实例：初始化源码模型、配置 Vite worker，并把内容变更同步给父组件。 */
function createEditor() {
  if (!editorHost.value) return;
  // Monaco 默认从 CDN 加载 worker，Vite 打包场景必须指向本地 worker 文件。
  self.MonacoEnvironment = {
    getWorker() {
      return new editorWorker();
    }
  };
  model = monaco.editor.createModel(props.modelValue, props.language);
  editor = monaco.editor.create(editorHost.value, {
    model,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 13,
    tabSize: 2,
    scrollBeyondLastLine: false,
    wordWrap: "on",
    readOnly: props.readonly,
    padding: { top: 12, bottom: 12 }
  });
  editor.onDidChangeModelContent(() => {
    emit("update:modelValue", editor?.getValue() ?? "");
  });
}

/** 组件卸载时销毁编辑器与模型实例，避免 Monaco 资源泄漏。 */
function disposeEditor() {
  editor?.dispose();
  model?.dispose();
  editor = undefined;
  model = undefined;
}

onMounted(createEditor);
onBeforeUnmount(disposeEditor);

watch(
  () => props.modelValue,
  value => {
    if (!editor || value === editor.getValue()) return;
    editor.setValue(value);
  }
);

watch(
  () => props.language,
  value => {
    if (!model) return;
    monaco.editor.setModelLanguage(model, value);
  }
);
</script>

<style scoped>
.code-editor {
  width: 100%;
  min-height: 240px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
}
</style>
