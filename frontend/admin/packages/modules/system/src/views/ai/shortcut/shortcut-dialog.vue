<template>
  <ProDialog
    ref="dialog"
    v-model="visible"
    :title="
      t(
        viewOnly
          ? 'system.ai.shortcut.dialog.view'
          : form.id
            ? 'system.ai.shortcut.dialog.edit'
            : 'system.ai.shortcut.dialog.create'
      )
    "
    class="ai-shortcut-dialog"
    body-class="ai-shortcut-dialog__body"
    width="960px"
    align-center
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    @closed="reset"
  >
    <el-steps :active="activeStep" finish-status="success" align-center class="shortcut-steps">
      <el-step
        v-for="(key, index) in stepKeys"
        :key="key"
        :title="t(key)"
        :aria-current="activeStep === index ? 'step' : undefined"
      />
    </el-steps>
    <div class="shortcut-step-content">
      <section v-show="activeStep === 0" class="shortcut-step-panel">
        <ProForm ref="basicForm" :model="form" :fields="basicFields" :rules="rules" :disabled="viewOnly" label-width="8em" />
      </section>

      <section v-show="activeStep === 1" class="shortcut-step-panel">
        <div class="shortcut-graph-design">
          <el-radio-group v-model="useFlow" :disabled="viewOnly">
            <el-radio :value="false">{{ t("system.ai.shortcut.mode.prompt") }}</el-radio>
            <el-radio :value="true" :disabled="form.terminal !== Terminal.TERMINAL_ADMIN">
              {{ t("system.ai.shortcut.mode.flow") }}
            </el-radio>
          </el-radio-group>
          <el-text v-if="form.terminal !== Terminal.TERMINAL_ADMIN" size="small" type="info">
            {{ t("system.ai.shortcut.message.app_prompt") }}
          </el-text>
          <template v-if="useFlow">
            <div v-if="!viewOnly" class="shortcut-graph-toolbar">
              <el-button
                v-for="type in actionTypes"
                :key="type"
                size="small"
                :disabled="nodeGraph.nodes.length >= 24"
                @click="addNode(type)"
              >
                {{ t(nodeTypeI18nKey(type)) }}
              </el-button>
              <el-button size="small" @click="fitLayout">{{ t("system.ai.shortcut.action.layout") }}</el-button>
              <el-text size="small" type="info">{{ t("system.ai.shortcut.message.graph_hint") }}</el-text>
            </div>
            <div class="shortcut-graph-workspace">
              <WorkflowGraph
                v-if="activeStep === 1"
                ref="graphRef"
                :draft="graphView"
                :selected-node-id="selectedNodeId"
                :selected-edge-index="selectedEdgeIndex"
                :readonly="viewOnly"
                @select="selectNode"
                @select-edge="selectEdge"
                @move="moveNode"
                @connect="connectNodes"
              />
              <el-aside class="shortcut-graph-properties" style="--el-aside-width: auto">
                <el-text tag="h3">
                  {{
                    t(
                      selectedNode
                        ? "system.workflow.definition.section.node_config"
                        : selectedEdge
                          ? "system.workflow.definition.section.edges"
                          : "system.workflow.definition.section.properties"
                    )
                  }}
                </el-text>
                <template v-if="selectedNode">
                  <ProForm :model="selectedNode" :fields="nodeFields" :disabled="viewOnly" label-width="auto">
                    <template #type="{ model }">
                      <el-tag effect="plain">{{ t(nodeTypeI18nKey(model.type)) }}</el-tag>
                    </template>
                  </ProForm>
                  <el-button
                    v-if="!viewOnly && selectedNode.id !== 'start' && selectedNode.id !== 'end'"
                    type="danger"
                    plain
                    @click="removeNode"
                  >
                    {{ t("system.workflow.definition.action.remove_node") }}
                  </el-button>
                </template>
                <template v-else-if="selectedEdge">
                  <ProForm :model="selectedEdge" :fields="edgeFields" :disabled="viewOnly" label-width="auto">
                    <template #from="{ model }">
                      <el-text>{{ model.from }} → {{ model.to }}</el-text>
                    </template>
                  </ProForm>
                  <el-button v-if="!viewOnly" type="danger" plain @click="removeEdge">{{
                    t("system.workflow.definition.action.remove_edge")
                  }}</el-button>
                </template>
                <el-empty v-else :image-size="64" :description="t('system.workflow.definition.placeholder.select_object')" />
              </el-aside>
            </div>
          </template>
          <el-alert v-else :title="t('system.ai.shortcut.message.prompt_only')" type="info" show-icon :closable="false" />
          <el-text v-if="requiredTools.length" size="small" class="shortcut-tool-dependencies">
            {{ t("system.ai.shortcut.field.dependencies") }} · {{ requiredTools.join(", ") }}
          </el-text>
        </div>
      </section>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button :disabled="saving" @click="visible = false">{{
          t(viewOnly ? "common.action.close" : "common.action.cancel")
        }}</el-button>
        <el-button v-if="activeStep > 0" :disabled="saving" @click="activeStep--">{{
          t("system.workflow.definition.wizard.previous")
        }}</el-button>
        <el-button v-if="activeStep === 0" type="primary" @click="nextStep">{{
          t("system.workflow.definition.wizard.next")
        }}</el-button>
        <el-button v-else-if="!viewOnly" type="primary" :loading="saving" @click="save">{{ t("common.action.save") }}</el-button>
      </div>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { FormRules } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defAiShortcutService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_shortcut";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import {
  AiShortcutSourceType,
  type AiShortcutForm,
  type AiShortcutGraph,
  type AiShortcutToolOption
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_shortcut";
import WorkflowGraph from "../../workflow/definition/workflow-graph.vue";
import { addGraphEdge, formatGraphLayout, nodeTypeI18nKey, type GraphDraft } from "../../workflow/definition/graph-draft";
import { sources, terminals } from "./shortcut-options";

const emit = defineEmits<{ saved: [] }>();
const dialog = ref<InstanceType<typeof ProDialog>>();
const graphRef = ref<InstanceType<typeof WorkflowGraph>>();
const basicForm = ref<ProFormInstance>();
const visible = ref(false);
const viewOnly = ref(false);
const saving = ref(false);
const activeStep = ref(0);
const form = ref(emptyForm());
const useFlow = ref(false);
const nodeGraph = ref(freshGraph());
const selectedNodeId = ref("");
const selectedEdgeIndex = ref(-1);
const loadingTools = ref(false);
const tools = ref<AiShortcutToolOption[]>([]);
const stepKeys = ["system.workflow.definition.wizard.basic", "system.workflow.definition.wizard.graph"];
const actionTypes = ["PROMPT", "TOOL", "BRANCH", "CONFIRM"] as const;
const textExample = JSON.stringify({ text: "…" });
// 提示词和流程参数保留原始内容，标识在校验时拒绝空白。
const untrimmedInput = { modelModifiers: { trim: false } };
const isApi = computed(() => form.value.source_type === AiShortcutSourceType.AI_SHORTCUT_SOURCE_TYPE_API);
const selectedNode = computed(() => nodeGraph.value.nodes.find(node => node.id === selectedNodeId.value));
const selectedEdge = computed(() => nodeGraph.value.edges[selectedEdgeIndex.value]);
const requiredTools = computed(() => [
  ...new Set([form.value.tool_name, ...(useFlow.value ? nodeGraph.value.nodes.map(node => node.tool_name) : [])].filter(Boolean))
]);
// 画布复用位置和连线协议，AI 节点独立维护业务属性。
const graphView = computed(() => ({ version: 1, ...nodeGraph.value }) as unknown as GraphDraft);
const toolOptions = computed(() => {
  const available = tools.value.map(tool => ({
    label: tool.title ? `${tool.title} (${tool.tool_name})` : tool.tool_name,
    value: tool.tool_name,
    disabled: false
  }));
  for (const name of requiredTools.value) {
    if (!available.some(tool => tool.value === name)) available.push({ label: name, value: name, disabled: true });
  }
  return available;
});
const rules = computed<FormRules<AiShortcutForm>>(() => ({
  title: [
    {
      required: true,
      whitespace: true,
      message: t("common.validation.required_input", { field: t("system.ai.shortcut.field.title") }),
      trigger: "blur"
    }
  ],
  shortcut_key: [
    { required: true, pattern: /^[a-z][a-z0-9_]*$/, message: t("system.ai.shortcut.validation.key"), trigger: "blur" }
  ],
  tool_name: [{ required: true, message: t("system.ai.shortcut.validation.tool"), trigger: "change" }],
  prompt: [
    {
      required: true,
      whitespace: true,
      message: t("common.validation.required_input", { field: t("system.ai.shortcut.field.prompt") }),
      trigger: "blur"
    }
  ]
}));
const toolField = computed<ProFormField>(() => ({
  prop: "tool_name",
  label: t("system.ai.shortcut.field.api"),
  component: viewOnly.value ? "input" : "select",
  options: toolOptions.value,
  props: viewOnly.value
    ? undefined
    : { filterable: true, loading: loadingTools.value, placeholder: t("common.placeholder.select") }
}));
const basicFields = computed<ProFormField[]>(() => [
  {
    prop: "title",
    label: t("system.ai.shortcut.field.title"),
    component: "input",
    props: { ...untrimmedInput, maxlength: 200, placeholder: t("system.ai.shortcut.placeholder.title") }
  },
  {
    prop: "shortcut_key",
    label: t("system.ai.shortcut.field.key"),
    component: "input",
    colSpan: 12,
    props: {
      ...untrimmedInput,
      disabled: Boolean(form.value.id),
      maxlength: 100,
      placeholder: t("system.ai.shortcut.placeholder.key")
    }
  },
  {
    prop: "terminal",
    label: t("system.ai.shortcut.field.terminal"),
    component: "radio-group",
    colSpan: 12,
    options: terminals(),
    props: { onChange: handleTerminalChange }
  },
  {
    prop: "source_type",
    label: t("system.ai.shortcut.field.source"),
    component: "radio-group",
    options: sources(),
    props: { onChange: handleSourceChange }
  },
  { ...toolField.value, visible: () => isApi.value },
  {
    prop: "prompt",
    label: t("system.ai.shortcut.field.prompt"),
    component: "textarea",
    props: {
      ...untrimmedInput,
      rows: 3,
      maxlength: 2000,
      showWordLimit: true,
      resize: "none",
      placeholder: t("system.ai.shortcut.placeholder.prompt")
    }
  },
  {
    prop: "sort",
    label: t("common.field.sort"),
    component: "input-number",
    colSpan: 12,
    props: { min: 0, max: 99999, precision: 0, controlsPosition: "right", style: { maxWidth: "100%" } }
  },
  {
    prop: "status",
    label: t("common.field.status"),
    component: "switch",
    colSpan: 12,
    props: {
      disabled: Boolean(form.value.id),
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      inlinePrompt: true,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled")
    }
  }
]);
const nodeFields = computed<ProFormField[]>(() => [
  { prop: "type", label: t("system.workflow.definition.field.node_type"), component: "slot" },
  {
    prop: "step",
    label: t("system.ai.shortcut.field.step"),
    component: "input",
    props: { ...untrimmedInput, maxlength: 100 }
  },
  { ...toolField.value, visible: node => node.type === "TOOL" },
  {
    prop: "payload_json",
    label: t("system.ai.shortcut.field.payload"),
    component: "textarea",
    visible: node => ["PROMPT", "TOOL", "CONFIRM"].includes(node.type),
    props: node => ({
      ...untrimmedInput,
      rows: 4,
      maxlength: 8192,
      placeholder: t(node.type === "TOOL" ? "system.ai.shortcut.placeholder.arguments" : "system.ai.shortcut.placeholder.text", {
        example: textExample
      })
    })
  }
]);
const edgeFields = computed<ProFormField[]>(() => [
  { prop: "from", label: t("system.workflow.definition.section.edges"), component: "slot" },
  {
    prop: "condition",
    label: t("system.workflow.definition.field.edge_condition"),
    component: "input",
    visible: edge => nodeGraph.value.nodes.some(node => node.id === edge.from && node.type === "BRANCH"),
    props: { ...untrimmedInput, maxlength: 256, placeholder: t("system.ai.shortcut.placeholder.condition") }
  }
]);

watch(
  () => (visible.value && !viewOnly.value && (isApi.value || useFlow.value) ? form.value.terminal : undefined),
  async (terminal, _, onCleanup) => {
    let current = true;
    onCleanup(() => {
      current = false;
    });
    tools.value = [];
    loadingTools.value = terminal !== undefined;
    if (terminal === undefined) return;
    try {
      const result = await defAiShortcutService.OptionAiShortcutTool({ terminal });
      if (current) tools.value = result.tools ?? [];
    } catch {
      // 请求层展示错误，失效引用仍可查看并重新选择。
    } finally {
      if (current) loadingTools.value = false;
    }
  }
);

/** 打开创建、编辑或只读查看弹窗。 */
async function open(id?: number, readonly = false) {
  await dialog.value?.open({
    load: () => (id ? defAiShortcutService.GetAiShortcut({ id }) : emptyForm()),
    commit: data => {
      form.value = {
        ...emptyForm(),
        ...data,
        tool_name: data.tool_name ?? "",
        group_name: data.group_name ?? "",
        sort: data.sort ?? 0
      };
      useFlow.value = Boolean(data.graph);
      nodeGraph.value = data.graph
        ? {
            nodes: (data.graph.nodes ?? []).map(node => ({
              ...node,
              tool_name: node.tool_name ?? "",
              payload_json: node.payload_json ?? ""
            })),
            edges: (data.graph.edges ?? []).map(edge => ({ ...edge, condition: edge.condition ?? "" }))
          }
        : freshGraph();
      viewOnly.value = readonly;
      activeStep.value = 0;
      selectNode("");
    }
  });
}

/** 返回空白入口并默认停用。 */
function emptyForm(): AiShortcutForm {
  return {
    id: 0,
    shortcut_key: "",
    title: "",
    source_type: AiShortcutSourceType.AI_SHORTCUT_SOURCE_TYPE_CUSTOM,
    terminal: Terminal.TERMINAL_ADMIN,
    tool_name: "",
    prompt: "",
    group_name: "",
    sort: 0,
    status: Status.STATUS_DISABLE,
    graph: undefined,
    revision: 0,
    updated_at: ""
  };
}

/** 生成开始直连结束的画布草稿。 */
function freshGraph(): AiShortcutGraph {
  return {
    nodes: [
      { id: "start", step: "start", type: "START", x: 40, y: 110, tool_name: "", payload_json: "" },
      { id: "end", step: "end", type: "END", x: 240, y: 110, tool_name: "", payload_json: "" }
    ],
    edges: [{ from: "start", to: "end", condition: "" }]
  };
}

/** 清理弹窗关闭后的草稿和校验状态。 */
function reset() {
  if (visible.value) return;
  basicForm.value?.clearValidate();
  form.value = emptyForm();
  nodeGraph.value = freshGraph();
  useFlow.value = false;
  viewOnly.value = false;
  activeStep.value = 0;
  selectNode("");
}

/** 更换终端后清除主工具并使用该终端支持的入口形式。 */
function handleTerminalChange() {
  form.value.tool_name = "";
  if (form.value.terminal !== Terminal.TERMINAL_ADMIN) useFlow.value = false;
}

/** 自定义入口的工具依赖由图节点决定。 */
function handleSourceChange() {
  if (!isApi.value) form.value.tool_name = "";
}

/** 校验基本信息后进入流程设计。 */
async function nextStep() {
  if (!viewOnly.value) {
    if (!(await basicForm.value?.validate())) return;
    if (isApi.value && !tools.value.some(tool => tool.tool_name === form.value.tool_name)) {
      ElMessage.warning(t("system.ai.shortcut.validation.tool"));
      return;
    }
  }
  activeStep.value = 1;
}

/** 校验流程并保存当前入口。 */
async function save() {
  if (saving.value) return;
  if (useFlow.value) {
    const steps = new Set(nodeGraph.value.nodes.map(node => node.step));
    if (steps.size !== nodeGraph.value.nodes.length || [...steps].some(step => !/^[a-z][a-z0-9_]*$/.test(step))) {
      ElMessage.warning(t("system.ai.shortcut.validation.step"));
      return;
    }
    if (
      nodeGraph.value.nodes.some(node => node.type === "TOOL" && !tools.value.some(tool => tool.tool_name === node.tool_name))
    ) {
      ElMessage.warning(t("system.ai.shortcut.validation.tool"));
      return;
    }
  }
  const shortcut = {
    ...form.value,
    sort: form.value.sort ?? 0,
    graph: useFlow.value
      ? { nodes: nodeGraph.value.nodes, edges: nodeGraph.value.edges.map(edge => ({ ...edge, condition: edge.condition ?? "" })) }
      : undefined
  };
  saving.value = true;
  try {
    if (shortcut.id) await defAiShortcutService.UpdateAiShortcut({ shortcut });
    else await defAiShortcutService.CreateAiShortcut({ shortcut });
    ElMessage.success(t("common.message.save_success"));
    visible.value = false;
    emit("saved");
  } finally {
    saving.value = false;
  }
}

/** 选择节点并清除连线选择。 */
function selectNode(id: string) {
  selectedNodeId.value = id;
  selectedEdgeIndex.value = -1;
}

/** 选择连线并清除节点选择。 */
function selectEdge(index: number) {
  selectedNodeId.value = "";
  selectedEdgeIndex.value = index;
}

/** 将拖拽完成的坐标写回表单草稿。 */
function moveNode(id: string, x: number, y: number) {
  const node = nodeGraph.value.nodes.find(item => item.id === id);
  if (node) Object.assign(node, { x, y });
}

/** 建立连线并切换到对应属性面板。 */
function connectNodes(from: string, to: string) {
  if (nodeGraph.value.edges.length >= 48) return;
  const error = addGraphEdge(graphView.value, from, to);
  if (error) ElMessage.warning(t(error));
  else selectEdge(nodeGraph.value.edges.length - 1);
}

/** 添加节点，在线性流程里自动插在结束节点前。 */
function addNode(type: (typeof actionTypes)[number]) {
  const end = nodeGraph.value.nodes.find(node => node.id === "end");
  if (!end) return;
  let sequence = nodeGraph.value.nodes.length - 1;
  let id: string;
  do {
    id = `${type.toLowerCase()}_${sequence++}`;
  } while (nodeGraph.value.nodes.some(node => node.id === id));
  nodeGraph.value.nodes.push({
    id,
    step: id,
    type,
    x: end.x,
    y: end.y + 100,
    tool_name: type === "TOOL" ? form.value.tool_name : "",
    payload_json: ""
  });
  const closingEdges = nodeGraph.value.edges.filter(edge => edge.to === "end");
  if (closingEdges.length === 1 && nodeGraph.value.edges.length < 48) {
    closingEdges[0].to = id;
    nodeGraph.value.edges.push({ from: id, to: "end", condition: "" });
    void fitLayout();
  }
  selectNode(id);
}

/** 删除已选节点及其关联连线。 */
function removeNode() {
  const id = selectedNodeId.value;
  nodeGraph.value.nodes = nodeGraph.value.nodes.filter(node => node.id !== id);
  nodeGraph.value.edges = nodeGraph.value.edges.filter(edge => edge.from !== id && edge.to !== id);
  selectNode("");
}

/** 删除当前选中的连线。 */
function removeEdge() {
  nodeGraph.value.edges.splice(selectedEdgeIndex.value, 1);
  selectNode("");
}

/** 沿用现有画布的自动布局与缩放。 */
async function fitLayout() {
  formatGraphLayout(graphView.value);
  await nextTick();
  graphRef.value?.fitGraph();
}

defineExpose({ open });
</script>

<style scoped lang="scss">
:global(.ai-shortcut-dialog.el-dialog) {
  display: flex;
  flex-direction: column;
  height: min(640px, calc(100dvh - 32px));
}
:global(.ai-shortcut-dialog__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.shortcut-steps {
  flex: none;
  margin-bottom: 16px;
}
.shortcut-step-content {
  flex: 1;
  min-height: 0;
  margin-inline: -10px;
}
.shortcut-step-panel {
  box-sizing: border-box;
  height: 100%;
  padding-inline: 10px;
  overflow: auto;
}
.shortcut-graph-design {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}
.shortcut-graph-design > :not(.shortcut-graph-workspace) {
  flex: none;
}
.shortcut-tool-dependencies {
  align-self: flex-start;
  overflow-wrap: anywhere;
}
.shortcut-graph-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.shortcut-graph-toolbar .el-button + .el-button {
  margin-left: 0;
}
.shortcut-graph-workspace {
  display: grid;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) 320px;
  grid-template-rows: minmax(0, 1fr);
  gap: 16px;
  min-height: 300px;
}
.shortcut-graph-workspace > .workflow-graph {
  height: 100%;
  box-sizing: border-box;
}
.shortcut-graph-properties {
  padding: 16px;
  overflow: auto;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
  h3 {
    margin: 0 0 20px;
    font-size: 14px;
    font-weight: 500;
  }
}
@media (width <= 900px) {
  .shortcut-graph-design {
    height: auto;
  }
  .shortcut-graph-workspace {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: 360px auto;
  }
}
</style>
