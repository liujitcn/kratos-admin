<template>
  <ProDialog
    v-model="visible"
    :title="dialogTitle"
    width="min(1400px, calc(100vw - 48px))"
    top="4vh"
    destroy-on-close
    :close-on-click-modal="false"
    @opened="handleOpened"
    @closed="handleClosed"
  >
    <el-steps :active="activeStep" finish-status="success" process-status="process" class="definition-steps">
      <el-step v-for="(step, index) in wizardSteps" :key="step.key" :title="t(step.titleKey)" :status="getStepStatus(index)" />
    </el-steps>
    <div class="definition-step-content">
      <section v-show="activeStep === 0" class="definition-step-panel">
        <div class="definition-basic">
          <el-form ref="formRef" :model="formData" :rules="formRules" label-position="top">
            <el-form-item :label="t('system.workflow.definition.field.name')" prop="name">
              <el-input
                v-model="formData.name"
                :placeholder="t('common.validation.required_input', { field: t('system.workflow.definition.field.name') })"
              />
            </el-form-item>
            <el-row :gutter="24">
              <el-col :xs="24" :sm="16">
                <el-form-item :label="t('system.workflow.definition.field.code')" prop="code">
                  <el-input v-model="formData.code" :disabled="isEdit" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="8">
                <el-form-item :label="t('system.workflow.definition.field.version')" prop="version">
                  <el-input-number
                    v-model="formData.version"
                    :min="1"
                    :precision="0"
                    :step="1"
                    controls-position="right"
                    :disabled="isEdit"
                    class="full-width"
                  />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item :label="t('common.field.remark')" prop="remark">
              <el-input v-model="formData.remark" type="textarea" :rows="4" :placeholder="t('common.placeholder.remark')" />
            </el-form-item>
          </el-form>
        </div>
      </section>
      <section v-show="activeStep === 1" class="definition-step-panel">
        <div class="form-builder">
          <aside class="form-palette">
            <div class="form-builder-heading">
              <h3>{{ t("system.workflow.definition.form_builder.palette") }}</h3>
            </div>
            <p class="definition-hint">{{ t("system.workflow.definition.form_builder.palette_hint") }}</p>
            <div class="form-palette-list">
              <el-button
                v-for="fieldType in formFieldTypes"
                :key="fieldType.type"
                class="form-palette-button"
                plain
                @click="addFormField(fieldType.type)"
              >
                <span class="form-palette-dot" aria-hidden="true" />
                {{ t(fieldType.labelKey) }}
              </el-button>
            </div>
          </aside>
          <section class="form-editor-pane">
            <div class="form-builder-heading form-builder-heading--between">
              <h3>{{ t("system.workflow.definition.form_builder.fields") }}</h3>
              <el-tag size="small" effect="plain">{{ formFields.length }}</el-tag>
            </div>
            <el-alert
              v-if="formSchemaLoadError"
              class="form-schema-alert"
              type="warning"
              :closable="false"
              :title="t('system.workflow.definition.error.form_schema_json')"
              show-icon
            />
            <el-empty
              v-if="!formFields.length"
              :description="t('system.workflow.definition.form_builder.empty')"
              :image-size="72"
            />
            <div v-else class="form-field-list">
              <article v-for="(field, index) in formFields" :key="`${field.key}-${index}`" class="form-field-editor">
                <header class="form-field-editor-header">
                  <div class="form-field-editor-title">
                    <span class="form-field-index">{{ index + 1 }}</span>
                    <strong>{{ field.label || t("system.workflow.definition.form_builder.untitled") }}</strong>
                    <el-tag size="small" effect="plain">{{ t(`system.workflow.definition.form_type.${field.type}`) }}</el-tag>
                  </div>
                  <div class="form-field-actions">
                    <el-tooltip :content="t('system.workflow.definition.form_builder.move_up')">
                      <el-button
                        text
                        size="small"
                        :icon="ArrowUp"
                        :aria-label="t('system.workflow.definition.form_builder.move_up')"
                        :disabled="index === 0"
                        @click="moveFormField(index, -1)"
                      />
                    </el-tooltip>
                    <el-tooltip :content="t('system.workflow.definition.form_builder.move_down')">
                      <el-button
                        text
                        size="small"
                        :icon="ArrowDown"
                        :aria-label="t('system.workflow.definition.form_builder.move_down')"
                        :disabled="index === formFields.length - 1"
                        @click="moveFormField(index, 1)"
                      />
                    </el-tooltip>
                    <el-tooltip :content="t('common.action.delete')">
                      <el-button
                        text
                        type="danger"
                        size="small"
                        :icon="Delete"
                        :aria-label="t('common.action.delete')"
                        @click="removeFormField(index)"
                      />
                    </el-tooltip>
                  </div>
                </header>
                <el-form label-position="top" class="form-field-editor-form">
                  <el-row :gutter="12">
                    <el-col :xs="24" :sm="12">
                      <el-form-item :label="t('system.workflow.definition.form_builder.key')">
                        <el-input
                          v-model="field.key"
                          :placeholder="t('system.workflow.definition.form_builder.key_placeholder')"
                        />
                      </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12">
                      <el-form-item :label="t('system.workflow.definition.form_builder.label')">
                        <el-input
                          v-model="field.label"
                          :placeholder="t('system.workflow.definition.form_builder.label_placeholder')"
                        />
                      </el-form-item>
                    </el-col>
                  </el-row>
                  <el-row :gutter="12">
                    <el-col :xs="24" :sm="12">
                      <el-form-item :label="t('system.workflow.definition.form_builder.type')">
                        <el-select v-model="field.type" class="full-width">
                          <el-option
                            v-for="item in formFieldTypes"
                            :key="item.type"
                            :label="t(item.labelKey)"
                            :value="item.type"
                          />
                        </el-select>
                      </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12">
                      <el-form-item :label="t('system.workflow.definition.form_builder.required')">
                        <el-switch v-model="field.required" />
                      </el-form-item>
                    </el-col>
                  </el-row>
                  <el-form-item v-if="field.type === 'select'" :label="t('system.workflow.definition.form_builder.options')">
                    <el-input
                      v-model="field.options"
                      :placeholder="t('system.workflow.definition.form_builder.options_placeholder')"
                    />
                  </el-form-item>
                </el-form>
              </article>
            </div>
          </section>
          <aside class="form-preview-pane">
            <div class="form-builder-heading">
              <h3>{{ t("system.workflow.definition.form_builder.preview") }}</h3>
            </div>
            <p class="definition-hint">{{ t("system.workflow.definition.form_builder.preview_hint") }}</p>
            <el-form label-position="top" class="form-preview" disabled>
              <template v-if="formFields.length">
                <el-form-item
                  v-for="field in formFields"
                  :key="`preview-${field.key}`"
                  :label="field.label || field.key"
                  :required="field.required"
                >
                  <el-input v-if="field.type === 'text'" />
                  <el-input v-else-if="field.type === 'textarea'" type="textarea" :rows="3" />
                  <el-input-number v-else-if="field.type === 'number'" controls-position="right" class="full-width" />
                  <el-date-picker v-else-if="field.type === 'date'" type="date" class="full-width" />
                  <el-select v-else class="full-width">
                    <el-option
                      v-for="option in getFormSchemaOptions(field.options)"
                      :key="option"
                      :label="option"
                      :value="option"
                    />
                  </el-select>
                </el-form-item>
              </template>
              <el-empty v-else :description="t('system.workflow.definition.form_builder.preview_empty')" :image-size="52" />
            </el-form>
          </aside>
        </div>
        <el-collapse class="form-schema-output">
          <el-collapse-item :title="t('system.workflow.definition.form_builder.generated_json')" name="json">
            <el-input :model-value="formSchemaText" type="textarea" :rows="8" readonly />
          </el-collapse-item>
        </el-collapse>
      </section>
      <section v-show="activeStep === 2" class="definition-step-panel">
        <div class="workflow-design">
          <div class="graph-toolbar">
            <el-button size="small" @click="addNode('SCRIPT')">{{ t("system.workflow.definition.node_type.SCRIPT") }}</el-button>
            <el-button size="small" @click="addNode('TASK')">{{ t("system.workflow.definition.action.add_task") }}</el-button>
            <el-button size="small" @click="addNode('EXCLUSIVE_GATEWAY')">{{
              t("system.workflow.definition.node_type.EXCLUSIVE_GATEWAY")
            }}</el-button>
            <el-button size="small" @click="addNode('PARALLEL_GATEWAY_FORK')">{{
              t("system.workflow.definition.node_type.PARALLEL_GATEWAY_FORK")
            }}</el-button>
            <el-button size="small" @click="addNode('PARALLEL_GATEWAY_JOIN')">{{
              t("system.workflow.definition.node_type.PARALLEL_GATEWAY_JOIN")
            }}</el-button>
            <span class="definition-hint">{{ t("system.workflow.definition.placeholder.select_object") }}</span>
          </div>
          <div class="graph-workspace">
            <WorkflowGraph
              ref="graphRef"
              :draft="draft"
              :selected-node-id="selectedNodeId"
              :selected-edge-index="selectedEdgeIndex"
              :readonly="false"
              @select="selectNode"
              @select-edge="selectEdge"
              @move="moveNode"
              @connect="connectNodes"
            />
            <aside class="graph-properties">
              <h3>
                {{
                  t(
                    selectedNode
                      ? "system.workflow.definition.section.node_config"
                      : selectedEdge
                        ? "system.workflow.definition.section.edges"
                        : "system.workflow.definition.section.properties"
                  )
                }}
              </h3>
              <el-empty
                v-if="!selectedNode && !selectedEdge"
                :description="t('system.workflow.definition.placeholder.select_object')"
                :image-size="64"
              />
              <template v-if="selectedNode">
                <el-form label-position="top">
                  <el-form-item :label="t('system.workflow.definition.field.node_type')">
                    <el-tag effect="plain">{{ t(nodeTypeI18nKey(selectedNode.type)) }}</el-tag>
                  </el-form-item>
                  <el-form-item v-if="selectedNode.type === 'START'" :label="t('system.workflow.script.before_submit')">
                    <ScriptSelect v-model="draft.hooks!.before_submit" />
                  </el-form-item>
                  <template v-if="selectedNode.type === 'END'">
                    <el-form-item v-for="hook in terminalHooks" :key="hook" :label="t(`system.workflow.script.${hook}`)">
                      <ScriptSelect v-model="draft.hooks![hook]" />
                    </el-form-item>
                  </template>
                  <el-form-item v-if="selectedNode.type === 'SCRIPT'" :label="t('system.workflow.script.script')">
                    <ScriptSelect v-model="selectedNode.script_id" />
                  </el-form-item>
                  <template v-if="selectedNode.type === 'TASK'">
                    <el-form-item :label="t('system.workflow.script.approver_mode')">
                      <el-radio-group
                        :model-value="selectedNode.approver_script_id !== undefined ? 'dynamic' : 'static'"
                        @update:model-value="setApproverMode"
                      >
                        <el-radio value="static">{{ t("system.workflow.script.static") }}</el-radio>
                        <el-radio value="dynamic">{{ t("system.workflow.script.dynamic") }}</el-radio>
                      </el-radio-group>
                    </el-form-item>
                    <el-form-item
                      v-if="selectedNode.approver_script_id !== undefined"
                      :label="t('system.workflow.script.approver')"
                    >
                      <ScriptSelect
                        :model-value="selectedNode.approver_script_id"
                        @update:model-value="selectedNode.approver_script_id = $event ?? 0"
                      />
                    </el-form-item>
                    <el-form-item :label="t('system.workflow.definition.field.strategy')">
                      <el-radio-group v-model="selectedNode.strategy" class="strategy-options">
                        <el-radio value="ALL">{{ t("system.workflow.definition.strategy.all") }}</el-radio>
                        <el-radio value="ANY">{{ t("system.workflow.definition.strategy.any") }}</el-radio>
                      </el-radio-group>
                    </el-form-item>
                    <div v-if="selectedNode.approver_script_id === undefined" class="approver-toolbar">
                      <el-select v-model="approverType" :aria-label="t('system.workflow.definition.field.approver_type')">
                        <el-option :label="t('system.workflow.definition.approver_type.USER')" value="USER" />
                        <el-option :label="t('system.workflow.definition.approver_type.ROLE')" value="ROLE" />
                        <el-option :label="t('system.workflow.definition.approver_type.POST')" value="POST" />
                      </el-select>
                      <el-select
                        v-model="approverTarget"
                        filterable
                        :placeholder="t('system.workflow.definition.placeholder.approver_target')"
                        :aria-label="t('system.workflow.definition.field.approver_target')"
                      >
                        <el-option
                          v-for="item in approverTargetOptions"
                          :key="Number(item.value)"
                          :label="item.label"
                          :value="Number(item.value)"
                        />
                      </el-select>
                      <el-button type="primary" plain @click="addApprover">{{
                        t("system.workflow.definition.action.add_approver")
                      }}</el-button>
                    </div>
                    <el-table v-if="selectedNode.approver_script_id === undefined" :data="selectedNode.approvers" size="small">
                      <el-table-column :label="t('system.workflow.definition.field.approver_target')" min-width="130">
                        <template #default="{ row }">
                          <div>{{ approverLabel(row) }}</div>
                          <small class="definition-hint">{{ t(`system.workflow.definition.approver_type.${row.type}`) }}</small>
                        </template>
                      </el-table-column>
                      <el-table-column :label="t('common.field.operation')" width="64" align="center">
                        <template #default="{ $index }">
                          <el-button type="danger" link size="small" @click="selectedNode?.approvers?.splice($index, 1)">{{
                            t("common.action.delete")
                          }}</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                  </template>
                </el-form>
                <el-collapse class="node-details">
                  <el-collapse-item :title="t('system.workflow.definition.section.advanced')" name="id">
                    <span class="definition-hint"
                      >{{ t("system.workflow.definition.field.node_id") }}: {{ selectedNode.id }}</span
                    >
                  </el-collapse-item>
                </el-collapse>
                <el-button v-if="canRemoveNode" type="danger" plain class="full-width" @click="removeSelectedNode">{{
                  t("system.workflow.definition.action.remove_node")
                }}</el-button>
              </template>
              <template v-if="selectedEdge">
                <el-form label-position="top">
                  <el-form-item :label="t('system.workflow.definition.field.edge_from')"
                    ><el-input :model-value="selectedEdge.from" readonly
                  /></el-form-item>
                  <el-form-item :label="t('system.workflow.definition.field.edge_to')"
                    ><el-input :model-value="selectedEdge.to" readonly
                  /></el-form-item>
                  <el-form-item :label="t('system.workflow.definition.field.edge_condition')">
                    <el-input
                      v-model="selectedEdge.condition"
                      type="textarea"
                      :rows="4"
                      :placeholder="t('system.workflow.definition.placeholder.condition')"
                    />
                  </el-form-item>
                </el-form>
                <el-button type="danger" plain class="full-width" @click="removeSelectedEdge">{{
                  t("system.workflow.definition.action.remove_edge")
                }}</el-button>
              </template>
            </aside>
          </div>
        </div>
      </section>
      <section v-show="activeStep === 3" class="definition-step-panel">
        <div class="workflow-review">
          <div class="workflow-review-header">
            <div>
              <h3>{{ t("system.workflow.definition.wizard.review") }}</h3>
              <p class="definition-hint">{{ t("system.workflow.definition.wizard.review_hint") }}</p>
            </div>
            <el-tag v-if="stepStates.review === 'success'" type="success" effect="plain">
              {{ t("system.workflow.definition.wizard.review_passed") }}
            </el-tag>
          </div>
          <div class="workflow-review-list">
            <div v-for="step in wizardSteps" :key="step.key" class="workflow-review-item">
              <div>
                <strong>{{ t(step.titleKey) }}</strong>
                <p v-if="stepErrors[step.key]" class="workflow-review-error">{{ t(stepErrors[step.key]) }}</p>
              </div>
              <el-tag :type="stepTagType(stepStates[step.key])" effect="plain">
                {{ t(`system.workflow.definition.wizard.status.${stepStates[step.key]}`) }}
              </el-tag>
            </div>
          </div>
        </div>
      </section>
    </div>
    <template #footer>
      <div class="definition-footer-actions">
        <el-button @click="visible = false">{{ t("common.action.cancel") }}</el-button>
        <el-button v-if="activeStep > 0" :disabled="saving" @click="handlePrevious">
          {{ t("system.workflow.definition.wizard.previous") }}
        </el-button>
        <el-button v-if="activeStep < lastStepIndex" type="primary" :disabled="saving" @click="handleNext">
          {{ t("system.workflow.definition.wizard.next") }}
        </el-button>
        <el-button v-else type="primary" :loading="saving" @click="handleSave">
          {{ t("system.workflow.definition.wizard.finish") }}
        </el-button>
      </div>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from "vue";
import type { FormInstance, FormRules } from "element-plus";
import { ArrowDown, ArrowUp, Delete } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import { defBaseRoleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_role";
import { defBasePostService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_post";
import type { ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import {
  addGraphEdge,
  type ApproverDraft,
  formatGraphLayout,
  type GraphDraft,
  type GraphNodeDraft,
  makeGraphNode,
  nodeTypeI18nKey,
  parseGraph,
  serializeGraph,
  validateGraphDraft
} from "./graph-draft";
import {
  WORKFLOW_WIZARD_STEP_KEYS,
  canAdvanceWorkflowWizard,
  getWorkflowWizardStepStatus,
  type WorkflowWizardStepKey,
  type WorkflowWizardStepStatus
} from "./workflow-wizard";
import {
  FORM_SCHEMA_FIELD_TYPES,
  createFormSchemaField,
  getFormSchemaOptions,
  parseFormSchema,
  serializeFormSchema,
  validateFormSchema,
  type FormSchemaField,
  type FormSchemaFieldType
} from "./form-schema";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import WorkflowGraph from "./workflow-graph.vue";
import ScriptSelect from "./script-select.vue";

defineOptions({ name: "WorkflowDefinitionDialog" });

const emit = defineEmits<{ saved: [] }>();
const { currentTenantId } = useTenantScope();

const visible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const formRef = ref<FormInstance>();
const graphRef = ref<InstanceType<typeof WorkflowGraph>>();
const activeStep = ref(0);
const terminalHooks = ["approved", "rejected", "withdrawn"] as const;

const wizardSteps = WORKFLOW_WIZARD_STEP_KEYS.map(key => ({
  key,
  titleKey: `system.workflow.definition.wizard.${key}`
}));
const formFieldTypes = FORM_SCHEMA_FIELD_TYPES.map(type => ({
  type,
  labelKey: `system.workflow.definition.form_type.${type}`
}));
const lastStepIndex = wizardSteps.length - 1;
const stepStates = reactive<Record<WorkflowWizardStepKey, WorkflowWizardStepStatus>>({
  basic: "pending",
  form: "pending",
  graph: "pending",
  review: "pending"
});
const stepErrors = reactive<Record<WorkflowWizardStepKey, string>>({ basic: "", form: "", graph: "", review: "" });

const formData = reactive({
  /** 流程定义ID。 */
  id: 0,
  /** 流程名称。 */
  name: "",
  /** 流程编码。 */
  code: "",
  /** 流程版本。 */
  version: 1,
  /** 备注。 */
  remark: ""
});

const draft = reactive<GraphDraft>({ version: 2, nodes: [], edges: [] });
const formSchemaText = ref("");
const formFields = ref<FormSchemaField[]>([]);
const formSchemaLoadError = ref(false);
const selectedNodeId = ref("");
const selectedEdgeIndex = ref(-1);

const approverType = ref<"USER" | "ROLE" | "POST">("USER");
const approverTarget = ref<number>();

/** 用户/角色/岗位下拉选项。 */
const userOptions = ref<ProFormOption[]>([]);
const roleOptions = ref<ProFormOption[]>([]);
const postOptions = ref<ProFormOption[]>([]);

const dialogTitle = computed(() =>
  t(isEdit.value ? "system.workflow.definition.title.edit" : "system.workflow.definition.title.create")
);

const formRules = computed<FormRules>(() => ({
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.workflow.definition.field.name") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.workflow.definition.field.name"), max: 100 }),
      trigger: "blur"
    }
  ],
  code: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.workflow.definition.field.code") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.workflow.definition.field.code"), max: 100 }),
      trigger: "blur"
    }
  ],
  version: [{ required: true, type: "number", min: 1, message: t("common.validation.sort_positive"), trigger: "blur" }],
  remark: [
    { max: 500, message: t("common.validation.max_length", { field: t("common.field.remark"), max: 500 }), trigger: "blur" }
  ]
}));

const selectedNode = computed<GraphNodeDraft | undefined>(() => draft.nodes.find(node => node.id === selectedNodeId.value));
const selectedEdge = computed(() => draft.edges[selectedEdgeIndex.value]);

watch(activeStep, async step => {
  if (step !== 2) return;
  await nextTick();
  graphRef.value?.fitGraph();
});

watch(formData, () => invalidateStep("basic"), { deep: true });
watch(
  formFields,
  fields => {
    formSchemaText.value = serializeFormSchema(fields);
    invalidateStep("form");
  },
  { deep: true }
);
watch(formSchemaText, () => invalidateStep("form"));
watch(draft, () => invalidateStep("graph"), { deep: true });

/** 弹窗打开后在流程图步骤重新计算画布尺寸。 */
async function handleOpened() {
  await nextTick();
  if (activeStep.value === 2) graphRef.value?.fitGraph();
}

/** 获取步骤头的 Element Plus 展示状态。 */
function getStepStatus(index: number) {
  const key = wizardSteps[index]?.key ?? "review";
  return getWorkflowWizardStepStatus(index, activeStep.value, stepStates[key]);
}

/** 获取发布检查摘要中的标签类型。 */
function stepTagType(status: WorkflowWizardStepStatus): "success" | "danger" | "info" {
  if (status === "success") return "success";
  if (status === "error") return "danger";
  return "info";
}

/** 修改前置配置后使当前步骤和最终检查回到待检查状态。 */
function invalidateStep(step: Exclude<WorkflowWizardStepKey, "review">) {
  stepStates[step] = "pending";
  stepErrors[step] = "";
  stepStates.review = "pending";
  stepErrors.review = "";
}

/** 起止节点不可删除。 */
const canRemoveNode = computed(() => {
  const node = selectedNode.value;
  return !!node && node.type !== "START" && node.type !== "END";
});

/** 审批人目标下拉选项，随类型切换。 */
const approverTargetOptions = computed<ProFormOption[]>(() => {
  if (approverType.value === "USER") return userOptions.value;
  if (approverType.value === "ROLE") return roleOptions.value;
  return postOptions.value;
});

/** 根据审批人类型和目标 ID 显示目标名称。 */
function approverLabel(approver: Record<string, unknown>): string {
  const type = approver.type as ApproverDraft["type"];
  const id = Number(approver.id);
  const options = type === "USER" ? userOptions.value : type === "ROLE" ? roleOptions.value : postOptions.value;
  return options.find(item => Number(item.value) === id)?.label ?? String(id);
}

/** 打开流程定义弹窗并回填草稿，id 为空表示新增。 */
async function open(id?: number) {
  resetState();
  isEdit.value = !!id;
  if (id) {
    const form = await defWorkflowDefinitionService.GetWorkflowDefinition({ id });
    formData.id = form.id;
    formData.name = form.name;
    formData.code = form.code;
    formData.version = form.version;
    formData.remark = form.remark;
    formSchemaText.value = form.form_schema;
    try {
      formFields.value = parseFormSchema(form.form_schema);
      formSchemaLoadError.value = false;
    } catch {
      formFields.value = [];
      formSchemaLoadError.value = true;
    }
    const parsed = parseGraph(form.node_config);
    if (parsed) {
      draft.hooks = parsed.hooks ?? {};
      draft.script_snapshots = parsed.script_snapshots;
      draft.nodes = parsed.nodes;
      draft.edges = parsed.edges;
    }
  }
  await loadOptions();
  visible.value = true;
}

/** 预加载审批人目标下拉。 */
async function loadOptions() {
  const [users, roles, posts] = await Promise.all([
    defBaseUserService.OptionBaseUser({ keyword: "", tenant_id: currentTenantId.value }),
    defBaseRoleService.OptionBaseRole({}),
    defBasePostService.OptionBasePost({})
  ]);
  userOptions.value = users.list ?? [];
  roleOptions.value = roles.list ?? [];
  postOptions.value = posts.list ?? [];
}

defineExpose({ open });

/** 重置弹窗步骤、选中对象和草稿，并补齐最小流程骨架。 */
function resetState() {
  formData.id = 0;
  formData.name = "";
  formData.code = "";
  formData.version = 1;
  formData.remark = "";
  formFields.value = [];
  formSchemaText.value = serializeFormSchema(formFields.value);
  formSchemaLoadError.value = false;
  activeStep.value = 0;
  for (const key of WORKFLOW_WIZARD_STEP_KEYS) {
    stepStates[key] = "pending";
    stepErrors[key] = "";
  }
  selectedNodeId.value = "";
  selectedEdgeIndex.value = -1;
  draft.hooks = {};
  draft.script_snapshots = undefined;
  draft.nodes = [
    { id: "start", type: "START", x: 30, y: 110 },
    { id: "end", type: "END", x: 640, y: 110 }
  ];
  draft.edges = [{ from: "start", to: "end", condition: "" }];
  approverType.value = "USER";
  approverTarget.value = undefined;
}

/** 将拖拽结束后的画布坐标保存到节点草稿。 */
function moveNode(id: string, x: number, y: number) {
  const node = draft.nodes.find(item => item.id === id);
  if (node) Object.assign(node, { x, y });
}

/** 拖动端口建立连线并打开对应的条件配置。 */
function connectNodes(from: string, to: string) {
  const errorKey = addGraphEdge(draft, from, to);
  if (errorKey) {
    ElMessage.warning(t(errorKey));
    return;
  }
  selectEdge(draft.edges.length - 1);
}

/** 选中节点或点击空白画布时清除连线选择。 */
function selectNode(id: string) {
  selectedNodeId.value = id;
  selectedEdgeIndex.value = -1;
  approverTarget.value = undefined;
}

/** 选中连线以在右侧配置条件。 */
function selectEdge(index: number) {
  selectedNodeId.value = "";
  selectedEdgeIndex.value = index;
}

/** 新增节点，坐标在画布空闲区顺延。 */
function addNode(type: GraphNodeDraft["type"]) {
  const node = makeGraphNode(
    type,
    160 + (draft.nodes.length % 3) * 180,
    30 + Math.floor(draft.nodes.length / 3) * 90,
    draft.nodes
  );
  draft.nodes.push(node);
  selectNode(node.id);
}

/** 删除选中节点及其关联边。 */
function removeSelectedNode() {
  const node = selectedNode.value;
  if (!node || !canRemoveNode.value) return;
  draft.nodes = draft.nodes.filter(item => item.id !== node.id);
  draft.edges = draft.edges.filter(edge => edge.from !== node.id && edge.to !== node.id);
  selectNode("");
}

/** 删除当前连线并清空属性面板。 */
function removeSelectedEdge() {
  if (!selectedEdge.value) return;
  draft.edges.splice(selectedEdgeIndex.value, 1);
  selectedEdgeIndex.value = -1;
}

/** 切换审批人来源时清除互斥配置。 */
function setApproverMode(mode: string | number | boolean | undefined) {
  const node = selectedNode.value;
  if (!node) return;
  node.approvers = [];
  node.approver_script_id = mode === "dynamic" ? 0 : undefined;
}

/** 新增审批人。 */
function addApprover() {
  const node = selectedNode.value;
  if (!node || node.type !== "TASK") return;
  if (!approverTarget.value) {
    ElMessage.warning(t("system.workflow.definition.placeholder.approver_target"));
    return;
  }
  node.approvers = [
    ...(node.approvers ?? []),
    { type: approverType.value, id: approverTarget.value } as ApproverDraft & { label?: string }
  ];
  approverTarget.value = undefined;
}

/** 添加一个申请表单字段。 */
function addFormField(type: FormSchemaFieldType) {
  formSchemaLoadError.value = false;
  let index = formFields.value.length;
  while (formFields.value.some(field => field.key === `field_${index + 1}`)) index += 1;
  formFields.value.push(createFormSchemaField(type, index));
}

/** 删除申请表单字段。 */
function removeFormField(index: number) {
  formFields.value.splice(index, 1);
}

/** 调整申请表单字段顺序。 */
function moveFormField(index: number, offset: -1 | 1) {
  const targetIndex = index + offset;
  if (targetIndex < 0 || targetIndex >= formFields.value.length) return;
  const fields = formFields.value;
  const field = fields[index];
  if (!field) return;
  fields.splice(index, 1);
  fields.splice(targetIndex, 0, field);
}

/** 校验单个向导步骤并记录状态。 */
async function validateStep(step: Exclude<WorkflowWizardStepKey, "review">): Promise<boolean> {
  let valid = true;
  let errorKey = "";
  if (step === "basic") {
    valid = formRef.value ? await formRef.value.validate().catch(() => false) : false;
  } else if (step === "form") {
    errorKey = formSchemaLoadError.value
      ? "system.workflow.definition.error.form_schema_json"
      : validateFormSchema(formFields.value);
    valid = !errorKey;
    formSchemaText.value = serializeFormSchema(formFields.value);
  } else {
    errorKey = validateGraphDraft(draft);
    valid = !errorKey;
  }
  stepStates[step] = valid ? "success" : "error";
  stepErrors[step] = errorKey;
  stepStates.review = "pending";
  stepErrors.review = "";
  if (!valid && errorKey) ElMessage.error(t(errorKey));
  return valid;
}

/** 重新校验前三个步骤，失败时回到首个错误步骤。 */
async function validateAllSteps(): Promise<boolean> {
  for (const step of ["basic", "form", "graph"] as const) {
    if (!(await validateStep(step))) {
      activeStep.value = wizardSteps.findIndex(item => item.key === step);
      return false;
    }
  }
  stepStates.review = "success";
  stepErrors.review = "";
  return true;
}

/** 向导前进，编辑态必须先通过当前步骤校验。 */
async function handleNext() {
  if (activeStep.value >= lastStepIndex) return;
  const step = wizardSteps[activeStep.value]?.key;
  if (!step || step === "review") return;
  if (!(await validateStep(step)) || !canAdvanceWorkflowWizard(activeStep.value, stepStates[step])) return;
  activeStep.value += 1;
  if (activeStep.value === lastStepIndex) await validateAllSteps();
}

/** 向导返回上一步，不清除已填写草稿。 */
function handlePrevious() {
  if (activeStep.value > 0) activeStep.value -= 1;
}

/** 校验并保存所有步骤共用的流程草稿。 */
async function handleSave() {
  if (!(await validateAllSteps())) return;
  saving.value = true;
  formatGraphLayout(draft);
  const request = {
    definition: {
      id: formData.id,
      name: formData.name,
      code: formData.code,
      version: formData.version,
      remark: formData.remark,
      node_config: serializeGraph(draft),
      form_schema: serializeFormSchema(formFields.value)
    }
  };
  try {
    if (formData.id) await defWorkflowDefinitionService.UpdateWorkflowDefinition(request);
    else await defWorkflowDefinitionService.CreateWorkflowDefinition(request);
    ElMessage.success(
      t(formData.id ? "common.message.update_success" : "common.message.create_success", {
        resource: t("system.workflow.definition.resource")
      })
    );
    visible.value = false;
    emit("saved");
  } finally {
    saving.value = false;
  }
}

/** 弹窗关闭后重置表单、校验和页面状态。 */
function handleClosed() {
  formRef.value?.resetFields();
  resetState();
}
</script>

<style scoped lang="scss">
.definition-steps {
  flex-shrink: 0;
  margin: 0 8px 18px;
}

.definition-step-content {
  height: min(640px, calc(100dvh - 220px));
  min-height: 0;
  overflow: hidden;
}

.definition-step-panel {
  height: 100%;
  overflow: auto;
}

.workflow-review {
  max-width: 820px;
  padding: 24px;
  margin: 0 auto;
}

.workflow-review-header,
.workflow-review-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.workflow-review-header {
  margin-bottom: 20px;

  h3 {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
  }
}

.workflow-review-list {
  display: grid;
  gap: 10px;
}

.workflow-review-item {
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.workflow-review-error {
  margin: 6px 0 0;
  color: var(--el-color-danger);
  font-size: 12px;
}

.definition-basic {
  max-width: 760px;
  padding: 20px 24px;
  margin: 0 auto;
}

.definition-schema {
  padding: 0 4px 16px;

  :deep(textarea) {
    font-family: var(--el-font-family-monospace, monospace);
    line-height: 1.7;
  }
}

.definition-hint {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}
.definition-schema > .definition-hint {
  margin-bottom: 12px;
}

.form-builder {
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr) 300px;
  gap: 16px;
  min-height: 100%;
}

.form-palette,
.form-editor-pane,
.form-preview-pane {
  min-width: 0;
}

.form-palette {
  padding: 0 12px 16px 0;
  border-right: 1px solid var(--el-border-color-lighter);
}

.form-preview-pane {
  padding: 0 0 16px 16px;
  border-left: 1px solid var(--el-border-color-lighter);
}

.form-builder-heading {
  display: flex;
  align-items: center;
  min-height: 32px;

  h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
  }
}

.form-builder-heading--between {
  justify-content: space-between;
  margin-bottom: 12px;
}

.form-palette-list {
  display: grid;
  gap: 8px;
  margin-top: 16px;
}

.form-palette-button {
  justify-content: flex-start;
  width: 100%;
  margin: 0;
}

.form-palette-dot {
  width: 6px;
  height: 6px;
  margin-right: 8px;
  background: var(--el-color-primary);
  border-radius: 50%;
}

.form-editor-pane {
  min-height: 0;
  overflow: auto;
}

.form-schema-alert {
  margin-bottom: 12px;
}

.form-field-list {
  display: grid;
  gap: 12px;
}

.form-field-editor {
  padding: 14px 16px 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.form-field-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.form-field-editor-title,
.form-field-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.form-field-editor-title {
  min-width: 0;

  strong {
    overflow: hidden;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.form-field-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 22px;
  width: 22px;
  height: 22px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 50%;
  font-size: 12px;
}

.form-field-editor-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.form-preview {
  margin-top: 16px;
}

.form-schema-output {
  margin-top: 16px;
}

.form-schema-output :deep(textarea) {
  font-family: var(--el-font-family-monospace, monospace);
  line-height: 1.7;
}

.workflow-design {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}
.graph-toolbar {
  display: flex;
  flex-shrink: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;

  .el-button + .el-button {
    margin-left: 0;
  }
}
.graph-workspace {
  display: grid;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
  min-height: 0;

  > .workflow-graph {
    height: 100%;
    min-height: 0;
    box-sizing: border-box;
  }
}
.graph-properties {
  padding: 16px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);

  h3 {
    margin: 0 0 20px;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 500;
  }
}
.strategy-options {
  display: flex;
  flex-direction: column;
  align-items: flex-start;

  .el-radio {
    margin-right: 0;
  }
}
.approver-toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}
.node-details {
  margin: 20px 0;
}
.full-width {
  width: 100%;
}
.definition-footer-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}
.definition-footer-actions .el-button + .el-button {
  margin-left: 0;
}

@media (max-width: 900px) {
  .definition-step-content {
    height: auto;
    min-height: 480px;
    overflow: visible;
  }
  .definition-step-panel {
    height: auto;
    overflow: visible;
  }
  .definition-basic {
    padding: 12px 4px;
  }
  .form-builder {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .form-palette,
  .form-preview-pane {
    padding: 0 0 16px;
    border: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .form-palette-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .form-preview-pane {
    order: 3;
  }
  .workflow-design {
    height: auto;
    min-height: 100%;
  }
  .graph-workspace {
    display: flex;
    flex-direction: column;

    > .workflow-graph {
      flex-shrink: 0;
      height: 340px;
    }
  }
  .graph-properties {
    overflow: visible;
  }
}
</style>
