import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { join } from "node:path";
import { test } from "node:test";
import {
  addGraphEdge,
  deriveProgressEdgeStates,
  formatGraphLayout,
  makeGraphNode,
  parseGraph,
  serializeGraph,
  validateGraphDraft,
  type GraphDraft
} from "../src/views/workflow/definition/graph-draft.js";
import {
  WORKFLOW_WIZARD_STEP_KEYS,
  canAdvanceWorkflowWizard,
  getWorkflowWizardStepStatus
} from "../src/views/workflow/definition/workflow-wizard.js";
import {
  createFormSchemaField,
  parseFormSchema,
  serializeFormSchema,
  validateFormSchema,
  type FormSchemaField
} from "../src/views/workflow/definition/form-schema.js";

const coreRequire = createRequire(join(process.cwd(), "../../core/package.json"));
const { createI18n } = coreRequire("vue-i18n");

for (const locale of ["zh-CN", "zh-TW", "en-US", "ja-JP"]) {
  test(`流程表单 JSON 示例在 ${locale} 下能够编译并保持原始内容`, async () => {
    const messages = JSON.parse(await readFile(join(process.cwd(), `src/locales/${locale}.json`), "utf8"));
    const key = "system.workflow.definition.placeholder.form_schema";
    const i18n = createI18n({ legacy: false, locale, messages: { [locale]: { schema: messages[key] } } });
    const translated = i18n.global.t("schema");
    const example = translated.match(/\[\{.*\}\]/)?.[0];
    assert.ok(example, "翻译后应包含完整的 JSON 示例");
    const fields = JSON.parse(example);
    assert.equal(fields[0].key, "reason");
    assert.equal(fields[0].type, "text");
    assert.equal(fields[0].required, true);
  });
}

test("完整向导只有当前步骤通过后才能前进，最后一步不能继续", () => {
  assert.deepEqual(WORKFLOW_WIZARD_STEP_KEYS, ["basic", "form", "graph", "review"]);
  assert.equal(canAdvanceWorkflowWizard(0, "pending"), false);
  assert.equal(canAdvanceWorkflowWizard(0, "error"), false);
  assert.equal(canAdvanceWorkflowWizard(0, "success"), true);
  assert.equal(canAdvanceWorkflowWizard(3, "success"), false);
});

test("完整向导步骤映射为 Element Plus 状态", () => {
  assert.equal(getWorkflowWizardStepStatus(0, 1, "success"), "finish");
  assert.equal(getWorkflowWizardStepStatus(1, 1, "pending"), "process");
  assert.equal(getWorkflowWizardStepStatus(2, 1, "error"), "error");
  assert.equal(getWorkflowWizardStepStatus(3, 1, "pending"), "wait");
});

test("流程定义弹窗使用完整向导导航并移除独立校验按钮", async () => {
  const source = await readFile(join(process.cwd(), "src/views/workflow/definition/definition-dialog.vue"), "utf8");
  assert.match(source, /<el-steps/);
  assert.match(source, /WORKFLOW_WIZARD_STEP_KEYS/);
  assert.match(source, /handleNext/);
  assert.match(source, /handlePrevious/);
  assert.doesNotMatch(source, /@click="handleValidate"/);
  assert.match(source, /class="form-builder"/);
  assert.match(source, /serializeFormSchema/);
  assert.doesNotMatch(source, /v-model="formSchemaText"/);
});

test("流程定义新增编辑与查看使用独立组件", async () => {
  const indexSource = await readFile(join(process.cwd(), "src/views/workflow/definition/index.vue"), "utf8");
  const editorSource = await readFile(join(process.cwd(), "src/views/workflow/definition/definition-dialog.vue"), "utf8");
  const viewSource = await readFile(join(process.cwd(), "src/views/workflow/definition/definition-view.vue"), "utf8");
  assert.match(indexSource, /WorkflowDefinitionDialog/);
  assert.match(indexSource, /WorkflowDefinitionView/);
  assert.match(indexSource, /definitionViewRef/);
  assert.match(indexSource, /handleOpenView/);
  assert.match(editorSource, /async function open\(id\?: number\)/);
  assert.doesNotMatch(editorSource, /const readonly/);
  assert.doesNotMatch(editorSource, /viewOnly/);
  assert.doesNotMatch(editorSource, /:disabled="readonly"/);
  assert.match(viewSource, /async function open\(id: number\)/);
  assert.match(viewSource, /defineExpose\(\{ open \}\)/);
  assert.doesNotMatch(viewSource, /handleSave/);
  assert.doesNotMatch(viewSource, /CreateWorkflowDefinition/);
});

test("工作流用户选项按当前租户查询，避免空关键字返回空列表", async () => {
  const sources = await Promise.all(
    [
      "src/views/workflow/definition/definition-dialog.vue",
      "src/views/workflow/approval/index.vue",
      "src/views/workflow/delegation/index.vue"
    ].map(path => readFile(join(process.cwd(), path), "utf8"))
  );
  for (const source of sources) {
    assert.match(source, /useTenantScope/);
    assert.match(source, /OptionBaseUser\(\{ keyword: "", tenant_id: currentTenantId\.value \}\)/);
  }
});

test("审批中心默认显示全部审批而不是待我审批", async () => {
  const source = await readFile(join(process.cwd(), "src/views/workflow/approval/index.vue"), "utf8");
  assert.match(source, /defaultValue: WorkflowListType\.WORKFLOW_LIST_TYPE_UNSPECIFIED/);
  assert.match(source, /WORKFLOW_LIST_TYPE_UNSPECIFIED/);
  assert.match(source, /prop: "list_type"[\s\S]*?render: scope => workflowListTypeLabel/);
  assert.doesNotMatch(source, /defaultValue: WorkflowListType\.WORKFLOW_LIST_TYPE_PENDING/);
});

test("审批详情按表单标签展示申请内容，并将审批活动交给节点悬浮卡", async () => {
  const approvalSource = await readFile(join(process.cwd(), "src/views/workflow/approval/index.vue"), "utf8");
  const graphSource = await readFile(join(process.cwd(), "src/views/workflow/definition/workflow-graph.vue"), "utf8");
  assert.match(approvalSource, /parseFormSchema/);
  assert.match(approvalSource, /label: formFields\.value\.find\(field => field\.key === key\)\?\.label \|\| key/);
  assert.match(approvalSource, /:node-details="progressNodeDetails"/);
  assert.doesNotMatch(approvalSource, /system\.workflow\.approval\.section\.logs/);
  assert.doesNotMatch(approvalSource, /system\.workflow\.approval\.section\.tasks/);
  assert.match(graphSource, /<el-popover v-if="nodeDetails" placement="top" trigger="hover"/);
  assert.match(graphSource, /activities/);
});

test("审批详情先展示申请内容，再展示流程进度", async () => {
  const source = await readFile(join(process.cwd(), "src/views/workflow/approval/index.vue"), "utf8");
  assert.ok(source.indexOf("section.form_data") < source.indexOf("section.progress"));
});

test("流程定义向导文案在四种语言中均已注册", async () => {
  const keys = [
    "system.workflow.definition.wizard.basic",
    "system.workflow.definition.wizard.form",
    "system.workflow.definition.wizard.graph",
    "system.workflow.definition.wizard.review",
    "system.workflow.definition.wizard.previous",
    "system.workflow.definition.wizard.next",
    "system.workflow.definition.wizard.finish",
    "system.workflow.definition.wizard.review_hint",
    "system.workflow.definition.wizard.review_passed",
    "system.workflow.definition.wizard.status.pending",
    "system.workflow.definition.wizard.status.success",
    "system.workflow.definition.wizard.status.error",
    "system.workflow.definition.form_builder.palette",
    "system.workflow.definition.form_builder.fields",
    "system.workflow.definition.form_builder.preview",
    "system.workflow.definition.form_builder.generated_json",
    "system.workflow.definition.form_type.text",
    "system.workflow.definition.form_type.textarea",
    "system.workflow.definition.form_type.number",
    "system.workflow.definition.form_type.date",
    "system.workflow.definition.form_type.select",
    "system.workflow.definition.error.form_field_key_required",
    "system.workflow.definition.error.form_field_key_invalid",
    "system.workflow.definition.error.form_field_key_duplicate",
    "system.workflow.definition.error.form_field_label_required",
    "system.workflow.definition.error.form_field_options_required"
  ];
  for (const locale of ["zh-CN", "zh-TW", "en-US", "ja-JP"]) {
    const messages = JSON.parse(await readFile(join(process.cwd(), `src/locales/${locale}.json`), "utf8"));
    for (const key of keys) assert.ok(messages[key], `${locale} 缺少 ${key}`);
  }
});

test("申请表单可以在可视化字段和 JSON 结构之间双向转换", () => {
  const fields = parseFormSchema(
    JSON.stringify([
      { key: "reason", label: "请假事由", type: "textarea", required: true },
      { key: "category", label: "类别", type: "select", required: false, options: ["事假", "年假"] }
    ])
  );
  assert.deepEqual(fields, [
    { key: "reason", label: "请假事由", type: "textarea", required: true },
    { key: "category", label: "类别", type: "select", required: false, options: "事假,年假" }
  ]);
  assert.equal(validateFormSchema(fields), "");
  assert.deepEqual(parseFormSchema(serializeFormSchema(fields)), fields);
});

test("可视化申请表单校验字段标识、标签、唯一性和下拉选项", () => {
  const field = createFormSchemaField("select", 0);
  const completeField: FormSchemaField = { ...field, label: "类别", options: "事假,年假" };
  assert.equal(validateFormSchema([]), "");
  assert.equal(validateFormSchema([{ ...field, key: "" }]), "system.workflow.definition.error.form_field_key_required");
  assert.equal(validateFormSchema([{ ...field, key: "1_reason" }]), "system.workflow.definition.error.form_field_key_invalid");
  assert.equal(
    validateFormSchema([field, { ...field, key: "another" }]),
    "system.workflow.definition.error.form_field_label_required"
  );
  assert.equal(
    validateFormSchema([completeField, { ...completeField, key: field.key }]),
    "system.workflow.definition.error.form_field_key_duplicate"
  );
  assert.equal(
    validateFormSchema([{ ...completeField, options: "" }]),
    "system.workflow.definition.error.form_field_options_required"
  );
});

test("流程节点拖拽坐标保存再打开后保持不变，审批规则不丢失", () => {
  const draft: GraphDraft = {
    version: 2,
    nodes: [
      { id: "start", type: "START", x: -30, y: 20 },
      { id: "task", type: "TASK", x: 240, y: 150, strategy: "ANY", approvers: [{ type: "USER", id: 7 }] },
      { id: "end", type: "END", x: 540, y: 20 }
    ],
    edges: [
      { from: "start", to: "task" },
      { from: "task", to: "end", condition: "default" }
    ]
  };
  assert.deepEqual(parseGraph(serializeGraph(draft)), draft);
});

test("无坐标的流程仍能生成可显示的节点布局", () => {
  const parsed = parseGraph(JSON.stringify({ version: 2, nodes: [{ id: "start", type: "START" }], edges: [] }));
  assert.equal(typeof parsed?.nodes[0].x, "number");
  assert.equal(typeof parsed?.nodes[0].y, "number");
});

test("保存前自动按拓扑整理节点坐标，线性节点对齐且分支节点分行", () => {
  const draft: GraphDraft = {
    version: 2,
    nodes: [
      { id: "start", type: "START" },
      { id: "gateway", type: "PARALLEL_GATEWAY_FORK" },
      { id: "left", type: "TASK" },
      { id: "right", type: "TASK" },
      { id: "join", type: "PARALLEL_GATEWAY_JOIN" },
      { id: "end", type: "END" }
    ],
    edges: [
      { from: "start", to: "gateway" },
      { from: "gateway", to: "left" },
      { from: "gateway", to: "right" },
      { from: "left", to: "join" },
      { from: "right", to: "join" },
      { from: "join", to: "end" }
    ]
  };
  formatGraphLayout(draft);
  assert.deepEqual(
    draft.nodes.map(node => [node.id, node.x, node.y]),
    [
      ["start", 40, 100],
      ["gateway", 240, 100],
      ["left", 440, 40],
      ["right", 440, 160],
      ["join", 640, 100],
      ["end", 840, 100]
    ]
  );
});

test("流程定义保存前执行节点自动布局", async () => {
  const source = await readFile(join(process.cwd(), "src/views/workflow/definition/definition-dialog.vue"), "utf8");
  assert.match(source, /formatGraphLayout\(draft\)/);
});

test("后端将缺失坐标补为零时仍能展开默认流程节点", () => {
  const parsed = parseGraph(
    JSON.stringify({
      version: 2,
      nodes: [
        { id: "start", type: "START", x: 0, y: 0 },
        { id: "admin_approval", type: "TASK", x: 0, y: 0, strategy: "ANY", approvers: [{ type: "ROLE", id: 1 }] },
        { id: "end", type: "END", x: 0, y: 0 }
      ],
      edges: [
        { from: "start", to: "admin_approval" },
        { from: "admin_approval", to: "end" }
      ]
    })
  );
  assert.deepEqual(
    parsed?.nodes.map(node => [node.id, node.x, node.y]),
    [
      ["start", 40, 40],
      ["admin_approval", 240, 40],
      ["end", 440, 40]
    ]
  );
});

test("打开现有流程后新增节点的编号与已有节点不重复", () => {
  const node = makeGraphNode("TASK", 0, 0, [{ id: "task_1", type: "TASK" }]);
  assert.notEqual(node.id, "task_1");
});

test("端口连线和表单连线都保留条件，并阻止无效或重复连线", () => {
  const draft: GraphDraft = {
    version: 2,
    nodes: [
      { id: "start", type: "START" },
      { id: "end", type: "END" }
    ],
    edges: []
  };
  assert.equal(addGraphEdge(draft, "start", "end", "default"), "");
  assert.deepEqual(draft.edges, [{ from: "start", to: "end", condition: "default" }]);
  assert.equal(addGraphEdge(draft, "start", "end"), "system.workflow.definition.error.edge_duplicate");
  assert.equal(addGraphEdge(draft, "start", "start"), "system.workflow.definition.error.edge_self");
  assert.equal(addGraphEdge(draft, "start", "missing"), "system.workflow.definition.error.unknown_edge_node");
  assert.equal(addGraphEdge(draft, "", "end"), "system.workflow.definition.error.edge_required");
  assert.equal(draft.edges.length, 1);
});

test("审批进度能标记排他网关实际分支和未走分支", () => {
  const nodes: GraphDraft["nodes"] = [
    { id: "start", type: "START" },
    { id: "gateway", type: "EXCLUSIVE_GATEWAY" },
    { id: "left", type: "TASK" },
    { id: "right", type: "TASK" },
    { id: "end", type: "END" }
  ];
  const edges: GraphDraft["edges"] = [
    { from: "start", to: "gateway" },
    { from: "gateway", to: "left", condition: "amount > 0" },
    { from: "gateway", to: "right", condition: "default" },
    { from: "left", to: "end" },
    { from: "right", to: "end" }
  ];
  assert.deepEqual(
    deriveProgressEdgeStates(nodes, edges, { start: "done", gateway: "todo", left: "active", right: "todo", end: "todo" }),
    { 0: "selected", 1: "active", 2: "skipped", 3: "todo", 4: "todo" }
  );
});

test("脚本节点、动态审批人、生命周期钩子和发布快照保存后完整回填", () => {
  const draft: GraphDraft = {
    version: 2,
    hooks: { before_submit: 11, approved: 12, rejected: 13, withdrawn: 14 },
    script_snapshots: { "11": { id: 11, name: "提交检查", language: 1, version: 3, source: "return {allowed:true}" } },
    nodes: [
      { id: "start", type: "START", x: 0, y: 0 },
      { id: "script", type: "SCRIPT", script_id: 15, x: 200, y: 0 },
      { id: "task", type: "TASK", strategy: "ALL", approvers: [], approver_script_id: 16, x: 400, y: 0 },
      { id: "end", type: "END", x: 600, y: 0 }
    ],
    edges: [
      { from: "start", to: "script" },
      { from: "script", to: "task" },
      { from: "task", to: "end" }
    ]
  };
  assert.equal(validateGraphDraft(draft), "");
  assert.deepEqual(parseGraph(serializeGraph(draft)), draft);
  const cleaned = { ...draft, script_snapshots: { "11": { id: 11, name: "提交检查", language: 1, version: 3 } } };
  assert.deepEqual(parseGraph(serializeGraph(cleaned)), cleaned);
});

test("脚本节点必须绑定脚本，动态审批人与静态审批人互斥", () => {
  const draft: GraphDraft = {
    version: 2,
    nodes: [
      { id: "start", type: "START" },
      { id: "script", type: "SCRIPT", script_id: 9 },
      { id: "task", type: "TASK", approver_script_id: 10, approvers: [] },
      { id: "end", type: "END" }
    ],
    edges: [
      { from: "start", to: "script" },
      { from: "script", to: "task" },
      { from: "task", to: "end" }
    ]
  };
  assert.equal(validateGraphDraft(draft), "");
  draft.nodes[1].script_id = undefined;
  assert.equal(validateGraphDraft(draft), "system.workflow.script.required");
  draft.nodes[1].script_id = 9;
  draft.nodes[2].approvers = [{ type: "USER", id: 4 }];
  assert.equal(validateGraphDraft(draft), "system.workflow.definition.error.task_approvers");
});
