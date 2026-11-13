<template>
  <div v-if="active" class="component-editor-example">
    <template v-if="componentId === 'CodeEditor'">
      <CodeEditor v-model="code" language="typescript" height="280px" />
      <p class="example-tip">修改内容会通过 v-model 回传。</p>
    </template>
    <YamlEditor v-else-if="componentId === 'YamlEditor'" v-model="yaml" min-height="240px" max-height="360px" />
    <template v-else-if="componentId === 'MarkdownEditor'">
      <MarkdownEditor v-model="markdown" :is-dark="globalStore.isDark" height="340px" placeholder="输入 Markdown 内容" />
    </template>
    <template v-else-if="componentId === 'WangEditor'">
      <div @drop.capture.prevent>
        <WangEditor v-model:value="html" height="320px" :toolbar-config="richToolbar" :editor-config="richConfig" />
      </div>
      <p class="example-tip">本例提供文本格式编辑；图片和视频上传请接入文件服务。</p>
    </template>
    <MarkdownPreview v-else-if="componentId === 'MarkdownPreview'" :model-value="markdown" :is-dark="globalStore.isDark" />
    <RichTextPreview v-else-if="componentId === 'RichTextPreview'" :model-value="html" />
    <CodeBlock v-else :code="code" language="typescript" max-height="280px" />
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref } from "vue";
import CodeBlock from "@/components/CodeBlock/index.vue";
import RichTextPreview from "@/components/RichTextPreview/index.vue";
import { useGlobalStore } from "@/stores/modules/global";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const CodeEditor = defineAsyncComponent(() => import("@/components/CodeEditor/index.vue"));
const YamlEditor = defineAsyncComponent(() => import("@/components/YamlEditor/index.vue"));
const MarkdownEditor = defineAsyncComponent(() => import("@/components/MarkdownEditor/index.vue"));
const MarkdownPreview = defineAsyncComponent(() => import("@/components/MarkdownPreview/index.vue"));
const WangEditor = defineAsyncComponent(() => import("@/components/WangEditor/index.vue"));
const globalStore = useGlobalStore();
const code = ref('const project = {\n  name: "智慧校园平台",\n  enabled: true\n};\n\nconsole.log(project.name);');
const yaml = ref("app:\n  name: 智慧校园平台\n  enabled: true\nserver:\n  port: 8000\n  timeout: 30s");
const markdown = ref(
  "# 项目说明\n\n统一校园服务入口，支持 **项目管理** 与通知。\n\n- 校园支付\n- 访客管理\n\n```typescript\nconst enabled = true;\n```"
);
const html = ref(
  "<h2>项目公告</h2><p>欢迎使用 <strong>智慧校园平台</strong>。</p><ul><li>统一应用入口</li><li>实时消息通知</li></ul>"
);
const richToolbar = {
  toolbarKeys: ["headerSelect", "bold", "italic", "underline", "color", "bulletedList", "numberedList", "undo", "redo"]
};
const richConfig = {
  placeholder: "输入项目公告",
  // 阻止粘贴图片触发组件内置的业务上传。
  pasteIgnoreImg: true,
  customPaste: (_editor: unknown, event: ClipboardEvent) => {
    if (!event.clipboardData?.files.length) return true;
    event.preventDefault();
    return false;
  }
};
</script>

<style scoped lang="scss">
.component-editor-example {
  min-width: 0;
}
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
</style>
