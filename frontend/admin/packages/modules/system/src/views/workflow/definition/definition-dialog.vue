<template>
  <ProDialog
    v-model="visible"
    :title="dialogTitle"
    class="workflow-definition-dialog"
    body-class="workflow-definition-dialog__body"
    width="1280px"
    align-center
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @opened="handleOpened"
    @closed="handleClosed"
  >
    <el-steps :active="activeStep" finish-status="success" simple class="definition-steps">
      <el-step
        v-for="(step, index) in wizardSteps"
        :key="step.key"
        :title="t(step.titleKey)"
        :status="getWorkflowWizardStepStatus(index, activeStep, stepStates[step.key])"
        :aria-current="activeStep === index ? 'step' : undefined"
      />
    </el-steps>
    <div class="definition-step-content">
      <section v-show="activeStep === 0" class="definition-step-panel">
        <ProForm ref="formRef" :model="formData" :fields="basicFields" :rules="formRules" label-width="8em" />
      </section>
      <section v-show="activeStep === 1" class="definition-step-panel">
        <div class="form-builder">
          <el-aside class="form-palette" style="--el-aside-width: auto; overflow: visible">
            <div class="form-builder-heading">
              <el-text tag="h3">{{ t("system.workflow.definition.form_builder.palette") }}</el-text>
            </div>
            <el-text tag="p" class="definition-hint">{{ t("system.workflow.definition.form_builder.palette_hint") }}</el-text>
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
          </el-aside>
          <section class="form-editor-pane">
            <div class="form-builder-heading form-builder-heading--between">
              <el-text tag="h3">{{ t("system.workflow.definition.form_builder.fields") }}</el-text>
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
              <article v-for="(field, index) in formFields" :key="index" class="form-field-editor">
                <el-header class="form-field-editor-header" style="--el-header-height: auto; --el-header-padding: 0">
                  <div class="form-field-editor-title">
                    <span class="form-field-index">{{ index + 1 }}</span>
                    <el-text tag="strong">{{ field.label || t("system.workflow.definition.form_builder.untitled") }}</el-text>
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
                </el-header>
                <ProForm :model="field" :fields="fieldEditorFields" label-width="auto" class="form-field-editor-form" />
              </article>
            </div>
          </section>
          <el-aside class="form-preview-pane" style="--el-aside-width: auto; overflow: visible">
            <div class="form-builder-heading">
              <el-text tag="h3">{{ t("system.workflow.definition.form_builder.preview") }}</el-text>
            </div>
            <el-text tag="p" class="definition-hint">{{ t("system.workflow.definition.form_builder.preview_hint") }}</el-text>
            <el-form class="form-preview" disabled>
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
          </el-aside>
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
            <el-aside class="graph-properties" style="--el-aside-width: auto">
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
              <el-empty
                v-if="!selectedNode && !selectedEdge"
                :description="t('system.workflow.definition.placeholder.select_object')"
                :image-size="64"
              />
              <template v-if="selectedNode">
                <ProForm :model="selectedNode" :fields="nodeFields" label-width="auto">
                  <template #type="{ model }">
                    <el-tag effect="plain">{{ t(nodeTypeI18nKey(model.type)) }}</el-tag>
                  </template>
                  <template #hook="{ field }">
                    <ScriptSelect v-model="draft.hooks![field.prop as keyof GraphHooks]" />
                  </template>
                  <template #script_id="{ model }">
                    <ScriptSelect v-model="model.script_id" />
                  </template>
                  <template #approver_mode="{ model }">
                    <el-radio-group
                      :model-value="model.approver_script_id !== undefined ? 'dynamic' : 'static'"
                      @update:model-value="setApproverMode"
                    >
                      <el-radio value="static">{{ t("system.workflow.script.static") }}</el-radio>
                      <el-radio value="dynamic">{{ t("system.workflow.script.dynamic") }}</el-radio>
                    </el-radio-group>
                  </template>
                  <template #approver_script_id="{ model }">
                    <ScriptSelect
                      :model-value="model.approver_script_id"
                      @update:model-value="model.approver_script_id = $event ?? 0"
                    />
                  </template>
                </ProForm>
                <template v-if="selectedNode.type === 'TASK' && selectedNode.approver_script_id === undefined">
                  <div class="approver-toolbar">
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
                  <el-table :data="selectedNode.approvers" size="small">
                    <el-table-column :label="t('system.workflow.definition.field.approver_target')" min-width="130">
                      <template #default="{ row }">
                        <div>{{ approverLabel(row) }}</div>
                        <el-text tag="small" class="definition-hint">{{
                          t(`system.workflow.definition.approver_type.${row.type}`)
                        }}</el-text>
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
                <ProForm :model="selectedEdge" :fields="edgeFields" label-width="auto" />
                <el-button type="danger" plain class="full-width" @click="removeSelectedEdge">{{
                  t("system.workflow.definition.action.remove_edge")
                }}</el-button>
              </template>
            </el-aside>
          </div>
        </div>
      </section>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button :disabled="saving" @click="visible = false">{{ t("common.action.cancel") }}</el-button>
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
import type { FormRules } from "element-plus";
import { ArrowDown, ArrowUp, Delete } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import { defBaseRoleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_role";
import { defBasePostService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_post";
import type { ProFormField, ProFormInstance, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import {
  addGraphEdge,
  type ApproverDraft,
  formatGraphLayout,
  type GraphDraft,
  type GraphHooks,
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
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import WorkflowGraph from "./workflow-graph.vue";
import ScriptSelect from "./script-select.vue";

defineOptions({ name: "WorkflowDefinitionDialog" });

const emit = defineEmits<{ saved: [] }>();
const { currentTenantId } = useTenantScope();

const visible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const formRef = ref<ProFormInstance>();
const graphRef = ref<InstanceType<typeof WorkflowGraph>>();
const activeStep = ref(0);
const terminalHooks = ["approved", "rejected", "withdrawn"] as const;
// 流程编码、表单标识和表达式沿用原始输入交给现有校验。
const untrimmedInput = { modelModifiers: { trim: false } };

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
  graph: "pending"
});

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
const formFields = ref<FormSchemaField[]>([]);
const formSchemaText = computed(() => serializeFormSchema(formFields.value));
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
const basicFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: t("system.workflow.definition.field.name"),
    component: "input",
    props: {
      ...untrimmedInput,
      placeholder: t("common.validation.required_input", { field: t("system.workflow.definition.field.name") })
    }
  },
  {
    prop: "code",
    label: t("system.workflow.definition.field.code"),
    component: "input",
    colSpan: 16,
    props: { ...untrimmedInput, disabled: isEdit.value }
  },
  {
    prop: "version",
    label: t("system.workflow.definition.field.version"),
    component: "input-number",
    colSpan: 8,
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", disabled: isEdit.value, style: { width: "100%" } }
  },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { ...untrimmedInput, rows: 4, placeholder: t("common.placeholder.remark") }
  }
]);
const fieldEditorFields = computed<ProFormField[]>(() => [
  {
    prop: "key",
    label: t("system.workflow.definition.form_builder.key"),
    component: "input",
    props: { ...untrimmedInput, placeholder: t("system.workflow.definition.form_builder.key_placeholder") }
  },
  {
    prop: "label",
    label: t("system.workflow.definition.form_builder.label"),
    component: "input",
    props: { ...untrimmedInput, placeholder: t("system.workflow.definition.form_builder.label_placeholder") }
  },
  {
    prop: "type",
    label: t("system.workflow.definition.form_builder.type"),
    component: "select",
    options: formFieldTypes.map(item => ({ value: item.type, label: t(item.labelKey) }))
  },
  { prop: "required", label: t("system.workflow.definition.form_builder.required"), component: "switch" },
  {
    prop: "options",
    label: t("system.workflow.definition.form_builder.options"),
    component: "input",
    visible: field => field.type === "select",
    props: { ...untrimmedInput, placeholder: t("system.workflow.definition.form_builder.options_placeholder") }
  }
]);
const nodeFields = computed<ProFormField[]>(() => [
  { prop: "type", label: t("system.workflow.definition.field.node_type"), component: "slot" },
  ...["before_submit", ...terminalHooks].map<ProFormField>(hook => ({
    prop: hook,
    label: t("system.workflow.script." + hook),
    component: "slot",
    slotName: "hook",
    visible: node => node.type === (hook === "before_submit" ? "START" : "END")
  })),
  {
    prop: "script_id",
    label: t("system.workflow.script.script"),
    component: "slot",
    visible: node => node.type === "SCRIPT"
  },
  {
    prop: "approver_mode",
    label: t("system.workflow.script.approver_mode"),
    component: "slot",
    visible: node => node.type === "TASK"
  },
  {
    prop: "approver_script_id",
    label: t("system.workflow.script.approver"),
    component: "slot",
    visible: node => node.type === "TASK" && node.approver_script_id !== undefined
  },
  {
    prop: "strategy",
    label: t("system.workflow.definition.field.strategy"),
    component: "radio-group",
    options: [
      { value: "ALL", label: t("system.workflow.definition.strategy.all") },
      { value: "ANY", label: t("system.workflow.definition.strategy.any") }
    ],
    props: { class: "strategy-options" },
    visible: node => node.type === "TASK"
  }
]);
const edgeFields = computed<ProFormField[]>(() => [
  { prop: "from", label: t("system.workflow.definition.field.edge_from"), component: "input", props: { readonly: true } },
  { prop: "to", label: t("system.workflow.definition.field.edge_to"), component: "input", props: { readonly: true } },
  {
    prop: "condition",
    label: t("system.workflow.definition.field.edge_condition"),
    component: "textarea",
    props: { ...untrimmedInput, rows: 4, placeholder: t("system.workflow.definition.placeholder.condition") }
  }
]);

const selectedNode = computed<GraphNodeDraft | undefined>(() => draft.nodes.find(node => node.id === selectedNodeId.value));
const selectedEdge = computed(() => draft.edges[selectedEdgeIndex.value]);

watch(activeStep, async step => {
  if (step !== 2) return;
  await nextTick();
  graphRef.value?.fitGraph();
});

watch(
  formData,
  () => {
    stepStates.basic = "pending";
  },
  { deep: true }
);
watch(
  formFields,
  () => {
    stepStates.form = "pending";
  },
  { deep: true }
);
watch(
  draft,
  () => {
    stepStates.graph = "pending";
  },
  { deep: true }
);

/** 弹窗打开后在流程图步骤重新计算画布尺寸。 */
async function handleOpened() {
  await nextTick();
  if (activeStep.value === 2) graphRef.value?.fitGraph();
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
  formSchemaLoadError.value = false;
  activeStep.value = 0;
  for (const key of WORKFLOW_WIZARD_STEP_KEYS) {
    stepStates[key] = "pending";
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
async function validateStep(step: WorkflowWizardStepKey): Promise<boolean> {
  let valid = true;
  let errorKey = "";
  if (step === "basic") {
    valid = Boolean(await formRef.value?.validate());
  } else if (step === "form") {
    errorKey = formSchemaLoadError.value
      ? "system.workflow.definition.error.form_schema_json"
      : validateFormSchema(formFields.value);
    valid = !errorKey;
  } else {
    errorKey = validateGraphDraft(draft);
    valid = !errorKey;
  }
  stepStates[step] = valid ? "success" : "error";
  if (!valid && errorKey) ElMessage.error(t(errorKey));
  return valid;
}

/** 保存前重新校验各步骤，失败时回到首个错误步骤。 */
async function validateAllSteps(): Promise<boolean> {
  for (const step of WORKFLOW_WIZARD_STEP_KEYS) {
    if (!(await validateStep(step))) {
      activeStep.value = WORKFLOW_WIZARD_STEP_KEYS.indexOf(step);
      return false;
    }
  }
  return true;
}

/** 当前步骤校验通过后进入下一步。 */
async function handleNext() {
  const currentStep = activeStep.value;
  if (currentStep >= lastStepIndex) return;
  const step = WORKFLOW_WIZARD_STEP_KEYS[currentStep];
  await validateStep(step);
  if (activeStep.value === currentStep && canAdvanceWorkflowWizard(currentStep, stepStates[step])) {
    activeStep.value = currentStep + 1;
  }
}

/** 向导返回上一步，不清除已填写草稿。 */
function handlePrevious() {
  if (activeStep.value > 0) activeStep.value -= 1;
}

/** 校验并保存所有步骤共用的流程草稿。 */
async function handleSave() {
  if (saving.value) return;
  saving.value = true;
  try {
    if (!(await validateAllSteps())) return;
    formatGraphLayout(draft);
    const request = {
      definition: {
        ...formData,
        node_config: serializeGraph(draft),
        form_schema: formSchemaText.value
      }
    };
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
:global(.workflow-definition-dialog.el-dialog) {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - 32px);
}

:global(.workflow-definition-dialog__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.definition-steps {
  flex: none;
  margin-bottom: 16px;
}

.definition-step-content {
  flex: 1;
  min-height: 0;
  margin-inline: -10px;
}

.definition-step-panel {
  box-sizing: border-box;
  height: 100%;
  padding-inline: 10px;
  overflow: auto;
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
  grid-template-rows: minmax(0, 1fr);
  gap: 16px;
  min-height: 300px;

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
.graph-properties :deep(.strategy-options) {
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
@media (width <= 900px) {
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
