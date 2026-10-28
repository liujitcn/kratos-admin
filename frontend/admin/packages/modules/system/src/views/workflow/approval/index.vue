<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="row_key"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestTaskTable"
    />

    <el-drawer v-model="detailVisible" :title="t('system.workflow.approval.title.detail')" size="52%" destroy-on-close>
      <el-descriptions v-if="detail" :column="2" size="small" border>
        <el-descriptions-item :label="t('system.workflow.definition.field.code')">{{
          detail.instance?.definition_code || "-"
        }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.workflow.definition.field.name')">{{
          detail.instance?.definition_name || "-"
        }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.workflow.approval.field.applicant')">
          {{ detail.instance?.applicant_name || detail.instance?.applicant_user_id }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('system.workflow.approval.field.instance_status')">
          <el-tag :type="instanceStatusTag(detail.instance?.status)">{{ instanceStatusLabel(detail.instance?.status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('system.workflow.approval.field.submit_at')" :span="2">
          {{ detail.instance?.submit_at || detail.instance?.created_at || "-" }}
        </el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">{{ t("system.workflow.approval.section.form_data") }}</el-divider>
      <el-descriptions v-if="formDataItems.length" :column="2" size="small" border>
        <el-descriptions-item v-for="item in formDataItems" :key="item.key" :label="item.label">{{
          item.value
        }}</el-descriptions-item>
      </el-descriptions>
      <el-input v-else :model-value="detail?.instance?.form_data ?? '-'" type="textarea" :rows="4" readonly />

      <template v-if="progressNodes.length">
        <el-divider content-position="left">{{ t("system.workflow.approval.section.progress") }}</el-divider>
        <WorkflowGraph
          :draft="{ version: 2, nodes: progressNodes, edges: progressEdges }"
          selected-node-id=""
          readonly
          :node-states="progressNodeStates"
          :edge-states="progressEdgeStates"
          :node-details="progressNodeDetails"
        />
        <div class="progress-legend">
          <el-tag size="small" type="success">{{ t("system.workflow.approval.progress.done") }}</el-tag>
          <el-tag size="small" type="warning">{{ t("system.workflow.approval.progress.active") }}</el-tag>
          <el-tag size="small" type="primary">{{ t("system.workflow.approval.progress.selected") }}</el-tag>
          <el-tag size="small" type="info">{{ t("system.workflow.approval.progress.skipped") }}</el-tag>
          <el-tag size="small" type="danger">{{ t("system.workflow.approval.progress.rejected") }}</el-tag>
        </div>
      </template>

      <el-divider content-position="left">{{ t("system.workflow.script.variables") }}</el-divider>
      <el-input :model-value="detail?.instance?.variables || '{}'" type="textarea" :rows="4" readonly />
      <el-divider content-position="left">{{ t("system.workflow.script.runs") }}</el-divider>
      <el-table :data="detail?.script_runs ?? []" size="small" border>
        <el-table-column :label="t('system.workflow.script.script')" min-width="120">
          <template #default="{ row }">{{
            row.stage === "START" ? t("system.workflow.script.stage.START") : row.script_name
          }}</template>
        </el-table-column>
        <el-table-column prop="version" :label="t('system.workflow.definition.field.version')" width="80" />
        <el-table-column :label="t('system.workflow.script.stage')" width="140"
          ><template #default="{ row }">{{ t(`system.workflow.script.stage.${row.stage}`) }}</template></el-table-column
        >
        <el-table-column prop="node_id" :label="t('system.workflow.definition.field.node_id')" width="110" />
        <el-table-column :label="t('common.field.status')" width="110"
          ><template #default="{ row }">{{ t(`system.workflow.script.status.${row.status}`) }}</template></el-table-column
        >
        <el-table-column prop="attempts" :label="t('system.workflow.script.attempts')" width="90" />
        <el-table-column prop="duration_ms" :label="t('system.workflow.script.duration')" width="90" />
        <el-table-column prop="next_retry_at" :label="t('system.workflow.script.next_retry')" width="170" />
        <el-table-column type="expand"
          ><template #default="{ row }">
            <p>{{ t("system.workflow.script.error") }}: {{ row.last_error || "-" }}</p>
            <p>{{ t("system.workflow.script.input") }}</p>
            <el-input :model-value="row.input_json" type="textarea" readonly />
            <p>{{ t("system.workflow.script.output") }}</p>
            <el-input :model-value="row.output_json" type="textarea" readonly /> </template
        ></el-table-column>
        <el-table-column :label="t('common.field.operation')" width="100"
          ><template #default="{ row }">
            <el-button
              v-if="row.status === 'FAILED'"
              link
              type="primary"
              :loading="retrying === row.id"
              @click="retryScript(row.id)"
              >{{ t("system.workflow.script.retry") }}</el-button
            >
          </template></el-table-column
        >
      </el-table>
      <template v-if="isPendingDetail">
        <el-divider content-position="left">{{ t("system.workflow.approval.section.audit") }}</el-divider>
        <el-input
          v-model="auditComment"
          type="textarea"
          :rows="2"
          :placeholder="t('system.workflow.approval.placeholder.comment')"
        />
        <div class="audit-actions">
          <el-button type="success" :loading="auditing" @click="handleAudit(WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_APPROVE)">
            {{ t("system.workflow.approval.action.approve") }}
          </el-button>
          <el-button type="danger" :loading="auditing" @click="handleAudit(WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_REJECT)">
            {{ t("system.workflow.approval.action.reject") }}
          </el-button>
          <el-button :loading="auditing" @click="openForwardDialog">{{ t("system.workflow.approval.action.forward") }}</el-button>
          <el-button :loading="auditing" @click="openAddApproverDialog">{{
            t("system.workflow.approval.action.add_approver")
          }}</el-button>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="forwardVisible" :title="forwardTitle" width="420px" destroy-on-close>
      <el-select
        v-model="forwardTarget"
        filterable
        style="width: 100%"
        :placeholder="t('system.workflow.approval.placeholder.select_user')"
      >
        <el-option v-for="item in userOptions" :key="Number(item.value)" :label="item.label" :value="item.value" />
      </el-select>
      <template #footer>
        <el-button @click="forwardVisible = false">{{ t("common.action.cancel") }}</el-button>
        <el-button type="primary" @click="handleForwardSubmit">{{ t("common.action.confirm") }}</el-button>
      </template>
    </el-dialog>

    <WorkflowSubmitDrawer ref="submitDrawerRef" @submitted="refreshTable" />
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import { useRoute } from "vue-router";
import { CirclePlus } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import type { ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import { defWorkflowTaskService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_task";
import { defWorkflowInstanceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_instance";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import {
  deriveProgressEdgeStates,
  type GraphEdgeDraft,
  type GraphNodeDraft,
  parseGraph,
  type WorkflowProgressNodeState
} from "../definition/graph-draft";
import { parseFormSchema, type FormSchemaField } from "../definition/form-schema";
import type { WorkflowNodeDetail } from "../definition/workflow-graph.vue";
import type { PageMyWorkflowTaskRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_task";
import type { GetWorkflowTaskResponse } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_task";
import type { GetWorkflowInstanceResponse } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_instance";
import {
  WorkflowAuditAction,
  WorkflowInstanceStatus,
  WorkflowListType,
  WorkflowLogAction,
  WorkflowTaskStatus
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_common";
import WorkflowSubmitDrawer from "./submit-drawer.vue";
import WorkflowGraph from "../definition/workflow-graph.vue";

defineOptions({
  name: "WorkflowApproval",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const { currentTenantId } = useTenantScope();
const route = useRoute();
const proTable = ref<ProTableInstance>();
const submitDrawerRef = ref<InstanceType<typeof WorkflowSubmitDrawer>>();

const detailVisible = ref(false);
const detail = ref<GetWorkflowTaskResponse | GetWorkflowInstanceResponse>();
const detailTaskId = ref(0);
const auditComment = ref("");
const auditing = ref(false);
const retrying = ref(0);
const forwardVisible = ref(false);
const forwardMode = ref<"forward" | "add_approver">("forward");
const forwardTarget = ref<number>();
const userOptions = ref<ProFormOption[]>([]);
const formFields = ref<FormSchemaField[]>([]);

const forwardTitle = computed(() =>
  t(forwardMode.value === "forward" ? "system.workflow.approval.action.forward" : "system.workflow.approval.action.add_approver")
);

/** 实例详情返回的审批任务列表，并补入当前待办详情中的任务。 */
const detailTasks = computed(() => {
  const tasks = (detail.value as GetWorkflowInstanceResponse | undefined)?.tasks ?? [];
  const currentTask = (detail.value as GetWorkflowTaskResponse | undefined)?.task;
  return currentTask ? [...tasks, currentTask] : tasks;
});

/** 进度图的节点与边，来自实例所属流程定义。 */
const progressNodes = ref<GraphNodeDraft[]>([]);
const progressEdges = ref<GraphEdgeDraft[]>([]);

/** 详情是否为待办任务（可审批）。 */
const isPendingDetail = computed(
  () =>
    detailTaskId.value > 0 &&
    (detail.value as GetWorkflowTaskResponse | undefined)?.task?.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING
);

/** 申请表单数据解析为键值对。 */
const formDataItems = computed(() => {
  const raw = detail.value?.instance?.form_data;
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return Object.entries(parsed).map(([key, value]) => ({
      key,
      label: formFields.value.find(field => field.key === key)?.label || key,
      value: typeof value === "object" ? JSON.stringify(value) : String(value)
    }));
  } catch {
    return [];
  }
});

/** 将节点日志和当前待办任务整理为流程图悬浮信息。 */
const progressNodeDetails = computed<Record<string, WorkflowNodeDetail>>(() =>
  Object.fromEntries(
    progressNodes.value.map(node => {
      const activities = (detail.value?.logs ?? [])
        .filter(item => (item.node_id || "start") === node.id)
        .map(item => ({
          label: logActionLabel(item.action),
          operator: item.operator_name,
          timestamp: item.created_at,
          comment: item.comment,
          tagType: logActionTag(item.action)
        }));
      const pendingTasks = detailTasks.value
        .filter(item => item.node_id === node.id && item.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING)
        .map(item => ({
          label: taskStatusLabel(item.status),
          operator: item.assignee_name,
          timestamp: item.created_at,
          comment: item.comment,
          tagType: taskStatusTag(item.status)
        }));
      return [node.id, { activities: [...activities, ...pendingTasks] }];
    })
  )
);

/** 表格列配置，随列表类型切换动作列。 */
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "list_type",
    label: t("system.workflow.approval.field.list_type"),
    minWidth: 110,
    align: "center",
    isSetting: false,
    render: scope => workflowListTypeLabel((scope.row as { list_type?: WorkflowListType }).list_type),
    search: {
      el: "select",
      defaultValue: WorkflowListType.WORKFLOW_LIST_TYPE_UNSPECIFIED,
      enum: [
        { label: t("common.value.all"), value: WorkflowListType.WORKFLOW_LIST_TYPE_UNSPECIFIED },
        { label: t("system.workflow.approval.tab.pending"), value: WorkflowListType.WORKFLOW_LIST_TYPE_PENDING },
        { label: t("system.workflow.approval.tab.done"), value: WorkflowListType.WORKFLOW_LIST_TYPE_DONE },
        { label: t("system.workflow.approval.tab.submitted"), value: WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED }
      ]
    }
  },
  { prop: "definition_code", label: t("system.workflow.definition.field.code"), minWidth: 130 },
  {
    prop: "definition_name",
    label: t("system.workflow.definition.field.name"),
    minWidth: 150,
    search: { el: "input", key: "keyword", label: t("system.workflow.approval.field.keyword") }
  },
  { prop: "applicant_name", label: t("system.workflow.approval.field.applicant"), minWidth: 110 },
  {
    prop: "instance_status",
    label: t("system.workflow.approval.field.instance_status"),
    minWidth: 100,
    align: "center",
    render: scope =>
      h(
        ElTag,
        { type: instanceStatusTag((scope.row as { instance_status?: WorkflowInstanceStatus }).instance_status), size: "small" },
        {
          default: () => instanceStatusLabel((scope.row as { instance_status?: WorkflowInstanceStatus }).instance_status)
        }
      )
  },
  { prop: "submit_at", label: t("system.workflow.approval.field.submit_at"), minWidth: 170, align: "center" },
  { prop: "task_created_at", label: t("system.workflow.approval.field.task_created_at"), minWidth: 170, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      {
        label: t("system.workflow.approval.action.audit"),
        type: "primary",
        link: true,
        hidden: scope =>
          (scope.row as { list_type?: WorkflowListType }).list_type !== WorkflowListType.WORKFLOW_LIST_TYPE_PENDING ||
          !BUTTONS.value["workflow:task:audit"],
        onClick: scope => handleOpenDetail(scope.row as { task_id: number; instance_id: number; list_type?: WorkflowListType })
      },
      {
        label: t("common.action.detail"),
        type: "primary",
        link: true,
        hidden: scope =>
          (scope.row as { list_type?: WorkflowListType }).list_type === WorkflowListType.WORKFLOW_LIST_TYPE_PENDING,
        onClick: scope => handleOpenDetail(scope.row as { task_id: number; instance_id: number; list_type?: WorkflowListType })
      },
      {
        label: t("system.workflow.approval.action.withdraw"),
        type: "danger",
        link: true,
        hidden: scope =>
          (scope.row as { list_type?: WorkflowListType }).list_type !== WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED ||
          !BUTTONS.value["workflow:instance:withdraw"] ||
          false,
        disabled: scope =>
          (scope.row as { instance_status?: WorkflowInstanceStatus }).instance_status !==
          WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_PENDING,
        onClick: scope => handleWithdraw(scope.row as { instance_id: number; definition_name?: string })
      }
    ]
  }
]);

/** 审批类型展示文案。 */
function workflowListTypeLabel(listType?: WorkflowListType): string {
  switch (listType) {
    case WorkflowListType.WORKFLOW_LIST_TYPE_UNSPECIFIED:
      return t("common.value.all");
    case WorkflowListType.WORKFLOW_LIST_TYPE_PENDING:
      return t("system.workflow.approval.tab.pending");
    case WorkflowListType.WORKFLOW_LIST_TYPE_DONE:
      return t("system.workflow.approval.tab.done");
    case WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED:
      return t("system.workflow.approval.tab.submitted");
    default:
      return "-";
  }
}

/** 顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("system.workflow.approval.action.submit"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["workflow:instance:submit"],
    onClick: () => submitDrawerRef.value?.open()
  }
]);

onMounted(() => {
  void loadUserOptions();
});

watch(
  () => [route.query.task_id, route.query.instance_id],
  () => void openNotificationTarget(),
  { immediate: true }
);

/** 从通知查询参数打开审批任务或流程实例详情。 */
async function openNotificationTarget() {
  const taskId = readRouteID(route.query.task_id);
  const instanceId = readRouteID(route.query.instance_id);
  if (!taskId && !instanceId) return;
  try {
    await handleOpenDetail({
      task_id: taskId,
      instance_id: instanceId,
      list_type: taskId ? WorkflowListType.WORKFLOW_LIST_TYPE_PENDING : WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED
    });
  } catch {
    ElMessage.error(t("common.message.request_error"));
  }
}

/** 读取路由中的安全整数编号，拒绝空值和超出 JavaScript 精度的编号。 */
function readRouteID(value: unknown): number {
  if (typeof value !== "string" || !/^\d+$/.test(value)) return 0;
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : 0;
}

/** 预加载用户下拉，转办与加签共用。 */
async function loadUserOptions() {
  const data = await defBaseUserService.OptionBaseUser({ keyword: "", tenant_id: currentTenantId.value });
  userOptions.value = data.list ?? [];
}

/** 按已提交的审批类型和关键词查询分页，并同步列表操作。 */
async function requestTaskTable(params: PageMyWorkflowTaskRequest) {
  const data = await defWorkflowTaskService.PageMyWorkflowTask(buildPageRequest(params));
  const items = (data.items ?? []).map(item => ({
    ...item,
    row_key:
      item.list_type === WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED ? `instance-${item.instance_id}` : `task-${item.task_id}`
  }));
  return { data: { list: items, total: data.total } };
}

/** 刷新表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 实例状态展示文案。 */
function instanceStatusLabel(status?: WorkflowInstanceStatus): string {
  switch (status) {
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_PENDING:
      return t("system.workflow.approval.status.pending");
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_APPROVED:
      return t("system.workflow.approval.status.approved");
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_REJECTED:
      return t("system.workflow.approval.status.rejected");
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_WITHDRAWN:
      return t("system.workflow.approval.status.withdrawn");
    default:
      return "-";
  }
}

/** 实例状态标签类型。 */
function instanceStatusTag(status?: WorkflowInstanceStatus): "success" | "warning" | "danger" | "info" {
  switch (status) {
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_APPROVED:
      return "success";
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_PENDING:
      return "warning";
    case WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_REJECTED:
      return "danger";
    default:
      return "info";
  }
}

/** 任务状态展示文案。 */
function taskStatusLabel(status?: WorkflowTaskStatus): string {
  switch (status) {
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING:
      return t("system.workflow.approval.task_status.pending");
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_APPROVED:
      return t("system.workflow.approval.task_status.approved");
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_REJECTED:
      return t("system.workflow.approval.task_status.rejected");
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_CANCELED:
      return t("system.workflow.approval.task_status.canceled");
    default:
      return "-";
  }
}

/** 任务状态标签类型。 */
function taskStatusTag(status?: WorkflowTaskStatus): "success" | "danger" | "warning" | "info" {
  switch (status) {
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_APPROVED:
      return "success";
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING:
      return "warning";
    case WorkflowTaskStatus.WORKFLOW_TASK_STATUS_REJECTED:
      return "danger";
    default:
      return "info";
  }
}

/** 日志动作展示文案。 */
function logActionLabel(action?: WorkflowLogAction): string {
  switch (action) {
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_SUBMIT:
      return t("system.workflow.approval.log.submit");
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_APPROVE:
      return t("system.workflow.approval.log.approve");
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_REJECT:
      return t("system.workflow.approval.log.reject");
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_FORWARD:
      return t("system.workflow.approval.log.forward");
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_WITHDRAW:
      return t("system.workflow.approval.log.withdraw");
    default:
      return "-";
  }
}

/** 日志动作标签类型。 */
function logActionTag(action?: WorkflowLogAction): "success" | "danger" | "warning" | "info" {
  switch (action) {
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_APPROVE:
      return "success";
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_REJECT:
      return "danger";
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_SUBMIT:
    case WorkflowLogAction.WORKFLOW_LOG_ACTION_WITHDRAW:
      return "warning";
    default:
      return "info";
  }
}

/** 打开详情：待办/已办走任务详情，我发起的走实例详情。 */
async function handleOpenDetail(row: { task_id: number; instance_id: number; list_type?: WorkflowListType }) {
  if (row.list_type === WorkflowListType.WORKFLOW_LIST_TYPE_SUBMITTED) {
    detail.value = await defWorkflowInstanceService.GetWorkflowInstance({ id: row.instance_id });
    detailTaskId.value = 0;
  } else {
    detail.value = await defWorkflowTaskService.GetWorkflowTask({ id: row.task_id });
    detailTaskId.value = row.task_id;
  }
  formFields.value = [];
  await loadFormSchema();
  auditComment.value = "";
  detailVisible.value = true;
  loadProgressGraph();
}

/** 读取流程定义字段标签，详情接口失败时保留字段键名回退。 */
async function loadFormSchema() {
  const definitionId = detail.value?.instance?.definition_id;
  if (!definitionId) return;
  try {
    const definition = await defWorkflowDefinitionService.GetWorkflowDefinition({ id: definitionId });
    formFields.value = parseFormSchema(definition.form_schema);
  } catch {
    // 流程定义可能已被删除，申请内容仍可按原始字段键展示。
  }
}

/** 从实例固定快照加载进度图，不读取后续修改的流程定义。 */
function loadProgressGraph() {
  const parsed = parseGraph(detail.value?.instance?.node_config ?? "");
  progressNodes.value = parsed?.nodes ?? [];
  progressEdges.value = parsed?.edges ?? [];
}

/** 人工重试后重新查询当前详情与列表，展示最新变量和执行记录。 */
async function retryScript(id: number) {
  const instanceId = detail.value?.instance?.id;
  if (!instanceId) return;
  retrying.value = id;
  try {
    await defWorkflowInstanceService.RetryWorkflowScript({ instance_id: instanceId, id });
    await handleOpenDetail({ task_id: detailTaskId.value, instance_id: instanceId });
    refreshTable();
  } finally {
    retrying.value = 0;
  }
}

/** 将办理状态映射给只读 Vue Flow 画布，保持进度高亮。 */
const progressNodeStates = computed(() =>
  Object.fromEntries(progressNodes.value.map(node => [node.id, nodeProgressClass(node)]))
);

/** 将已办理节点映射为连线状态，排他网关只保留实际命中的分支。 */
const progressEdgeStates = computed(() =>
  deriveProgressEdgeStates(progressNodes.value, progressEdges.value, progressNodeStates.value)
);

/** 判定进度图节点状态：已驳回 > 审批中 > 已完成 > 未到达。 */
function nodeProgressClass(node: GraphNodeDraft): WorkflowProgressNodeState {
  if (node.type === "START") return "done";
  if (node.type === "END") {
    return detail.value?.instance?.status === WorkflowInstanceStatus.WORKFLOW_INSTANCE_STATUS_APPROVED ? "done" : "todo";
  }
  if (node.type === "SCRIPT") {
    const runs = detail.value?.script_runs?.filter(item => item.node_id === node.id && item.stage === "NODE") ?? [];
    const run = runs[runs.length - 1];
    if (run?.status === "SUCCEEDED") return "done";
    if (run?.status === "FAILED") return "rejected";
    if (run && run.status !== "CANCELED") return "active";
  }
  const task = (detail.value as GetWorkflowTaskResponse | undefined)?.task;
  if (task?.node_id === node.id) {
    if (task.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING) return "active";
    if (task.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_REJECTED) return "rejected";
    if (task.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_APPROVED) return "done";
  }
  const logs = detail.value?.logs ?? [];
  if (logs.some(item => item.node_id === node.id && item.action === WorkflowLogAction.WORKFLOW_LOG_ACTION_REJECT)) {
    return "rejected";
  }
  if (
    detailTasks.value.some(item => item.node_id === node.id && item.status === WorkflowTaskStatus.WORKFLOW_TASK_STATUS_PENDING)
  ) {
    return "active";
  }
  if (logs.some(item => item.node_id === node.id && item.action === WorkflowLogAction.WORKFLOW_LOG_ACTION_APPROVE)) return "done";
  return "todo";
}

/** 审批（通过/驳回）。 */
async function handleAudit(action: WorkflowAuditAction) {
  if (action === WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_REJECT) {
    try {
      await ElMessageBox.confirm(t("system.workflow.approval.dialog.reject_confirm"), t("common.title.warning"), {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      });
    } catch {
      return;
    }
  }
  auditing.value = true;
  try {
    await defWorkflowTaskService.AuditWorkflowTask({
      task_id: detailTaskId.value,
      action,
      comment: auditComment.value,
      forward_to: 0,
      additional_approver: 0
    });
    ElMessage.success(
      t(
        action === WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_APPROVE
          ? "system.workflow.approval.message.approved"
          : "system.workflow.approval.message.rejected"
      )
    );
    detailVisible.value = false;
    refreshTable();
  } finally {
    auditing.value = false;
  }
}

/** 打开转办对话框。 */
function openForwardDialog() {
  forwardMode.value = "forward";
  forwardTarget.value = undefined;
  forwardVisible.value = true;
}

/** 打开加签对话框。 */
function openAddApproverDialog() {
  forwardMode.value = "add_approver";
  forwardTarget.value = undefined;
  forwardVisible.value = true;
}

/** 提交转办或加签。 */
async function handleForwardSubmit() {
  if (!forwardTarget.value) {
    ElMessage.warning(t("system.workflow.approval.placeholder.select_user"));
    return;
  }
  const action =
    forwardMode.value === "forward"
      ? WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_FORWARD
      : WorkflowAuditAction.WORKFLOW_AUDIT_ACTION_ADD_APPROVER;
  await defWorkflowTaskService.AuditWorkflowTask({
    task_id: detailTaskId.value,
    action,
    comment: "",
    forward_to: forwardMode.value === "forward" ? (forwardTarget.value ?? 0) : 0,
    additional_approver: forwardMode.value === "add_approver" ? (forwardTarget.value ?? 0) : 0
  });
  ElMessage.success(t("common.message.update_success", { resource: t("system.workflow.approval.resource") }));
  forwardVisible.value = false;
  detailVisible.value = false;
  refreshTable();
}

/** 撤回申请。 */
async function handleWithdraw(row: { instance_id: number; definition_name?: string }) {
  try {
    await ElMessageBox.confirm(
      `${t("system.workflow.approval.dialog.withdraw_confirm")}\n${t("common.dialog.resource_field", {
        field: t("system.workflow.definition.field.name"),
        value: row.definition_name || `ID:${row.instance_id}`
      })}`,
      t("common.title.warning"),
      { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
    );
    await defWorkflowInstanceService.WithdrawWorkflowInstance({ id: row.instance_id });
    ElMessage.success(t("system.workflow.approval.message.withdrawn"));
    refreshTable();
  } catch {
    /* 用户取消 */
  }
}
</script>

<style scoped lang="scss">
.audit-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.progress-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
</style>
