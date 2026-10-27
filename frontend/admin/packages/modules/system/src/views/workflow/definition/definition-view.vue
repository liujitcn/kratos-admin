<template>
  <ProDialog
    v-model="visible"
    :title="t('system.workflow.definition.title.view')"
    width="min(1400px, calc(100vw - 48px))"
    top="4vh"
    destroy-on-close
    :close-on-click-modal="false"
    @opened="handleOpened"
    @closed="resetState"
  >
    <el-skeleton v-if="loading" :rows="8" animated />
    <div v-else class="definition-view-content">
      <section class="definition-view-section">
        <h3>{{ t("system.workflow.definition.section.basic") }}</h3>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="t('system.workflow.definition.field.name')">{{ formData.name }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.workflow.definition.field.code')">{{ formData.code }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.workflow.definition.field.version')">{{ formData.version }}</el-descriptions-item>
          <el-descriptions-item :label="t('common.field.remark')" :span="2">{{ formData.remark || "-" }}</el-descriptions-item>
        </el-descriptions>
      </section>

      <section class="definition-view-section">
        <div class="definition-view-heading">
          <h3>{{ t("system.workflow.definition.form_builder.preview") }}</h3>
          <el-tag size="small" effect="plain">{{ formFields.length }}</el-tag>
        </div>
        <el-empty v-if="!formFields.length" :description="t('system.workflow.definition.form_builder.preview_empty')" :image-size="64" />
        <el-descriptions v-else :column="2" border>
          <el-descriptions-item v-for="field in formFields" :key="field.key" :label="field.label || field.key">
            <span>{{ t(`system.workflow.definition.form_type.${field.type}`) }}</span>
            <el-tag v-if="field.required" class="definition-view-required" size="small" type="warning" effect="plain">
              {{ t("system.workflow.definition.form_builder.required") }}
            </el-tag>
            <span v-if="field.options">{{ `：${field.options}` }}</span>
          </el-descriptions-item>
        </el-descriptions>
      </section>

      <section class="definition-view-section">
        <h3>{{ t("system.workflow.definition.section.graph") }}</h3>
        <div class="definition-view-graph">
          <WorkflowGraph
            ref="graphRef"
            :draft="draft"
            :selected-node-id="selectedNodeId"
            :selected-edge-index="selectedEdgeIndex"
            readonly
            @select="selectNode"
            @select-edge="selectEdge"
          />
          <aside class="definition-view-properties">
            <h4>{{ t("system.workflow.definition.section.properties") }}</h4>
            <el-empty
              v-if="!selectedNode && !selectedEdge"
              :description="t('system.workflow.definition.placeholder.select_object')"
              :image-size="56"
            />
            <template v-if="selectedNode">
              <el-descriptions :column="1" border size="small">
                <el-descriptions-item :label="t('system.workflow.definition.field.node_id')">{{ selectedNode.id }}</el-descriptions-item>
                <el-descriptions-item :label="t('system.workflow.definition.field.node_type')">
                  {{ t(nodeTypeI18nKey(selectedNode.type)) }}
                </el-descriptions-item>
                <el-descriptions-item v-if="selectedNode.type === 'TASK'" :label="t('system.workflow.definition.field.strategy')">
                  {{ t(`system.workflow.definition.strategy.${selectedNode.strategy ?? 'ALL'}`) }}
                </el-descriptions-item>
                <el-descriptions-item v-if="selectedNode.type === 'TASK'" :label="t('system.workflow.definition.field.approver_target')">
                  {{ selectedNode.approvers?.map(item => `${item.type}:${item.id}`).join(', ') || '-' }}
                </el-descriptions-item>
                <el-descriptions-item v-if="selectedNode.type === 'SCRIPT'" :label="t('system.workflow.script.script')">
                  {{ selectedNode.script_id || '-' }}
                </el-descriptions-item>
              </el-descriptions>
            </template>
            <template v-else-if="selectedEdge">
              <el-descriptions :column="1" border size="small">
                <el-descriptions-item :label="t('system.workflow.definition.field.edge_from')">{{ selectedEdge.from }}</el-descriptions-item>
                <el-descriptions-item :label="t('system.workflow.definition.field.edge_to')">{{ selectedEdge.to }}</el-descriptions-item>
                <el-descriptions-item :label="t('system.workflow.definition.field.edge_condition')">
                  {{ selectedEdge.condition || t("system.workflow.definition.field.default_condition") }}
                </el-descriptions-item>
              </el-descriptions>
            </template>
          </aside>
        </div>
      </section>
    </div>
    <template #footer>
      <el-button @click="visible = false">{{ t("common.action.close") }}</el-button>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import WorkflowGraph from "./workflow-graph.vue";
import { nodeTypeI18nKey, parseGraph, type GraphDraft, type GraphNodeDraft } from "./graph-draft";
import { parseFormSchema, type FormSchemaField } from "./form-schema";

defineOptions({ name: "WorkflowDefinitionView" });

const visible = ref(false);
const loading = ref(false);
const graphRef = ref<InstanceType<typeof WorkflowGraph>>();
const selectedNodeId = ref("");
const selectedEdgeIndex = ref(-1);
const formData = reactive({ id: 0, name: "", code: "", version: 1, remark: "" });
const draft = reactive<GraphDraft>({ version: 2, nodes: [], edges: [] });
const formFields = ref<FormSchemaField[]>([]);

const selectedNode = computed<GraphNodeDraft | undefined>(() => draft.nodes.find(node => node.id === selectedNodeId.value));
const selectedEdge = computed(() => draft.edges[selectedEdgeIndex.value]);
/** 打开流程定义查看器并加载完整配置。 */
async function open(id: number) {
  resetState();
  loading.value = true;
  try {
    const form = await defWorkflowDefinitionService.GetWorkflowDefinition({ id });
    formData.id = form.id;
    formData.name = form.name;
    formData.code = form.code;
    formData.version = form.version;
    formData.remark = form.remark;
    formFields.value = parseFormSchema(form.form_schema);
    const parsed = parseGraph(form.node_config);
    if (!parsed) throw new Error(t("system.workflow.definition.error.form_schema_json"));
    draft.hooks = parsed.hooks ?? {};
    draft.script_snapshots = parsed.script_snapshots;
    draft.nodes = parsed.nodes;
    draft.edges = parsed.edges;
    visible.value = true;
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t("common.message.request_error"));
  } finally {
    loading.value = false;
  }
}

defineExpose({ open });

/** 查看器打开后适配流程图尺寸。 */
async function handleOpened() {
  await nextTick();
  graphRef.value?.fitGraph();
}

/** 选中节点并显示属性。 */
function selectNode(id: string) {
  selectedNodeId.value = id;
  selectedEdgeIndex.value = -1;
}

/** 选中连线并显示属性。 */
function selectEdge(index: number) {
  selectedNodeId.value = "";
  selectedEdgeIndex.value = index;
}

/** 清理查看器状态，避免下次打开残留配置。 */
function resetState() {
  formData.id = 0;
  formData.name = "";
  formData.code = "";
  formData.version = 1;
  formData.remark = "";
  formFields.value = [];
  draft.hooks = {};
  draft.script_snapshots = undefined;
  draft.nodes = [];
  draft.edges = [];
  selectedNodeId.value = "";
  selectedEdgeIndex.value = -1;
}
</script>

<style scoped lang="scss">
.definition-view-content {
  height: min(700px, calc(100dvh - 220px));
  overflow: auto;
}

.definition-view-section {
  padding: 0 8px 24px;
}

.definition-view-section h3,
.definition-view-properties h4 {
  margin: 0 0 12px;
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 600;
}

.definition-view-heading {
  display: flex;
  align-items: center;
  gap: 8px;
}

.definition-view-required {
  margin-left: 8px;
}

.definition-view-graph {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 320px);
  gap: 16px;
}

.definition-view-properties {
  min-height: 360px;
  padding: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

@media (max-width: 900px) {
  .definition-view-graph {
    grid-template-columns: 1fr;
  }
}
</style>
