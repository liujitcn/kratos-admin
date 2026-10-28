<template>
  <div class="workflow-graph">
    <VueFlow
      :id="flowId"
      :nodes="nodes"
      :edges="edges"
      :nodes-draggable="!readonly"
      :nodes-connectable="!readonly"
      :edges-updatable="false"
      :delete-key-code="null"
      :min-zoom="0.3"
      :max-zoom="2"
      @node-click="selectNode"
      @edge-click="selectEdge"
      @node-drag-stop="moveNode"
      @connect="connectNodes"
      @pane-click="emit('select', '')"
      @init="fitGraph"
      @nodes-initialized="fitGraph"
    >
      <Background :gap="20" :size="1" pattern-color="var(--el-border-color-darker)" />
      <Controls :show-interactive="false" />
      <MiniMap
        :width="140"
        :height="85"
        pannable
        zoomable
        node-color="var(--el-color-primary-light-5)"
        node-stroke-color="var(--el-border-color)"
        mask-color="var(--el-fill-color-light)"
      />
      <template #node-workflow="{ id, data }">
        <el-popover v-if="nodeDetails" placement="top" trigger="hover" :width="320">
          <template #reference>
            <div
              class="workflow-node"
              :class="[
                `node-type-${data.type.toLowerCase()}`,
                `state-${nodeStates?.[id] ?? ''}`,
                { selected: selectedNodeId === id }
              ]"
            >
              <Handle v-if="data.type !== 'START'" id="in" type="target" :position="Position.Left" />
              <strong>{{ t(nodeTypeI18nKey(data.type)) }}</strong>
              <small>{{ id }}</small>
              <Handle v-if="data.type !== 'END'" id="out" type="source" :position="Position.Right" />
            </div>
          </template>
          <div class="workflow-node-detail">
            <div class="workflow-node-detail-title">
              <strong>{{ t(nodeTypeI18nKey(data.type)) }}</strong>
              <small>{{ id }}</small>
            </div>
            <div v-if="nodeDetails[id]?.activities.length" class="workflow-node-activities">
              <div
                v-for="(activity, index) in nodeDetails[id].activities"
                :key="`${activity.timestamp}-${index}`"
                class="workflow-node-activity"
              >
                <div class="workflow-node-activity-heading">
                  <el-tag size="small" :type="activity.tagType">{{ activity.label }}</el-tag>
                  <span>{{ activity.operator || "-" }}</span>
                </div>
                <small>{{ activity.timestamp || "-" }}</small>
                <p v-if="activity.comment">{{ activity.comment }}</p>
              </div>
            </div>
            <span v-else class="workflow-node-empty">-</span>
          </div>
        </el-popover>
        <div
          v-else
          class="workflow-node"
          :class="[
            `node-type-${data.type.toLowerCase()}`,
            `state-${nodeStates?.[id] ?? ''}`,
            { selected: selectedNodeId === id }
          ]"
        >
          <Handle v-if="data.type !== 'START'" id="in" type="target" :position="Position.Left" />
          <strong>{{ t(nodeTypeI18nKey(data.type)) }}</strong>
          <small>{{ id }}</small>
          <Handle v-if="data.type !== 'END'" id="out" type="source" :position="Position.Right" />
        </div>
      </template>
    </VueFlow>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from "vue";
import { VueFlow, Handle, MarkerType, Position, useVueFlow } from "@vue-flow/core";
import type { Connection, Edge, EdgeMouseEvent, Node, NodeDragEvent, NodeMouseEvent } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import { t } from "@liujitcn/kratos-admin-core";
import {
  CONDITION_DEFAULT,
  nodeTypeI18nKey,
  type GraphDraft,
  type GraphNodeDraft,
  type WorkflowProgressEdgeState
} from "./graph-draft";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/controls/dist/style.css";
import "@vue-flow/minimap/dist/style.css";

/** 流程图节点上展示的一条审批活动。 */
export interface WorkflowNodeActivity {
  /** 活动名称。 */
  label: string;
  /** 操作人或办理人。 */
  operator: string;
  /** 活动时间。 */
  timestamp: string;
  /** 审批意见或任务备注。 */
  comment: string;
  /** 活动标签颜色。 */
  tagType: "success" | "warning" | "danger" | "info";
}

/** 流程图节点悬浮展示的信息。 */
export interface WorkflowNodeDetail {
  /** 节点审批活动。 */
  activities: WorkflowNodeActivity[];
}

/** 流程画布仅处理图交互，审批配置和提交由设计器维护。 */
interface WorkflowGraphProps {
  draft: GraphDraft;
  selectedNodeId: string;
  /** 设计器中当前配置的连线序号，审批详情不传入。 */
  selectedEdgeIndex?: number;
  readonly: boolean;
  /** 审批详情中的节点办理状态，设计器不传入。 */
  nodeStates?: Record<string, "done" | "active" | "rejected" | "todo">;
  /** 审批详情中的连线执行状态，设计器不传入。 */
  edgeStates?: Record<number, WorkflowProgressEdgeState>;
  /** 审批详情中的节点活动，设计器和定义查看页不传入。 */
  nodeDetails?: Record<string, WorkflowNodeDetail>;
}

const props = defineProps<WorkflowGraphProps>();
const emit = defineEmits<{
  select: [id: string];
  selectEdge: [index: number];
  move: [id: string, x: number, y: number];
  connect: [from: string, to: string];
}>();
const flowId = `workflow-${useId()}`;
const { fitView, viewportHelper } = useVueFlow(flowId);

const nodes = computed<Node<GraphNodeDraft>[]>(() =>
  props.draft.nodes.map(node => ({
    id: node.id,
    type: "workflow",
    position: { x: node.x ?? 0, y: node.y ?? 0 },
    data: node,
    selected: props.selectedNodeId === node.id
  }))
);

const edges = computed<Edge[]>(() =>
  props.draft.edges.map((edge, index) => ({
    id: `edge-${index}`,
    source: edge.from,
    target: edge.to,
    sourceHandle: "out",
    targetHandle: "in",
    type: "smoothstep",
    pathOptions: { borderRadius: 10, offset: 20 },
    selected: props.selectedEdgeIndex === index,
    label: edge.condition === CONDITION_DEFAULT ? t("system.workflow.definition.field.default_condition") : edge.condition,
    class: [
      edge.condition ? "conditional" : "",
      props.nodeStates?.[edge.to] === "done" ? "done" : "",
      props.edgeStates?.[index] ? `progress-${props.edgeStates[index]}` : ""
    ].join(" "),
    markerEnd: MarkerType.ArrowClosed
  }))
);

/** 画布初始化或容器显示后适配节点，避免动画期间尺寸不完整。 */
function fitGraph() {
  if (!viewportHelper.value.viewportInitialized) return;
  void fitView({ padding: 0.2, maxZoom: 1 });
}

/** 选择节点以展示其审批配置。 */
function selectNode({ node }: NodeMouseEvent) {
  emit("select", node.id);
}

/** 点击连线后通知设计器打开对应的条件配置。 */
function selectEdge({ edge }: EdgeMouseEvent) {
  emit(
    "selectEdge",
    props.draft.edges.findIndex(item => item.from === edge.source && item.to === edge.target)
  );
}

/** 将缩放换算后的画布坐标回写业务草稿。 */
function moveNode({ node }: NodeDragEvent) {
  if (props.readonly) return;
  emit("move", node.id, node.position.x, node.position.y);
}

/** 拖动端口连线时交由抽屉统一校验。 */
function connectNodes(connection: Connection) {
  if (props.readonly) return;
  emit("connect", connection.source, connection.target);
}

defineExpose({ fitGraph });
</script>

<style scoped lang="scss">
.workflow-graph {
  height: 360px;
  overflow: hidden;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.workflow-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 116px;
  min-height: 56px;
  padding: 8px 12px;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius, 6px);

  strong {
    font-size: 13px;
    font-weight: 500;
  }
  small {
    color: var(--el-text-color-secondary);
    font-size: 11px;
  }

  &.node-type-start {
    background: var(--el-color-success-light-9);
    border-color: var(--el-color-success);
    border-radius: 32px;
  }
  &.node-type-end {
    background: var(--el-color-info-light-9);
    border-color: var(--el-color-info);
    border-radius: 32px;
  }
  &.node-type-task {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }
  &.node-type-exclusive_gateway {
    background: var(--el-color-warning-light-9);
    border-color: var(--el-color-warning);
  }
  &.node-type-parallel_gateway_fork,
  &.node-type-parallel_gateway_join {
    background: var(--el-color-danger-light-9);
    border-color: var(--el-color-danger);
  }
  &.selected {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-5);
  }
}

.workflow-node-detail {
  color: var(--el-text-color-primary);
}

.workflow-node-detail-title,
.workflow-node-activity-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.workflow-node-detail-title {
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  small {
    color: var(--el-text-color-secondary);
  }
}

.workflow-node-activities {
  max-height: 260px;
  margin-top: 8px;
  overflow-y: auto;
}

.workflow-node-activity {
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: 0;
  }

  small {
    display: block;
    margin-top: 4px;
    color: var(--el-text-color-secondary);
  }

  p {
    margin: 4px 0 0;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
}

.workflow-node-empty {
  display: block;
  padding-top: 8px;
  color: var(--el-text-color-secondary);
}

.workflow-graph :deep(.vue-flow__edge-path) {
  stroke: var(--el-border-color-darker);
  stroke-width: 1.5;
}

.workflow-node.state-done {
  background: var(--el-color-success-light-9);
  border-color: var(--el-color-success);
}
.workflow-node.state-active {
  background: var(--el-color-warning-light-9);
  border-color: var(--el-color-warning);
  box-shadow: 0 0 0 2px var(--el-color-warning-light-5);
}
.workflow-node.state-rejected {
  background: var(--el-color-danger-light-9);
  border-color: var(--el-color-danger);
}
.workflow-graph :deep(.done .vue-flow__edge-path) {
  stroke: var(--el-color-success);
}
.workflow-graph :deep(.conditional .vue-flow__edge-path) {
  stroke: var(--el-color-warning);
  stroke-dasharray: 5 3;
}
.workflow-graph :deep(.progress-active .vue-flow__edge-path) {
  stroke: var(--el-color-warning);
  stroke-width: 2.5;
  stroke-dasharray: none;
}
.workflow-graph :deep(.progress-done .vue-flow__edge-path) {
  stroke: var(--el-color-success);
  stroke-width: 2.5;
  stroke-dasharray: none;
}
.workflow-graph :deep(.progress-selected .vue-flow__edge-path) {
  stroke: var(--el-color-primary);
  stroke-width: 2.5;
  stroke-dasharray: none;
}
.workflow-graph :deep(.progress-skipped) {
  opacity: 0.4;
}
.workflow-graph :deep(.progress-skipped .vue-flow__edge-path) {
  stroke: var(--el-text-color-placeholder);
  stroke-dasharray: 3 4;
}
.workflow-graph :deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: var(--el-color-primary);
  stroke-width: 2.5;
}
.workflow-graph :deep(.vue-flow__edge-text) {
  fill: var(--el-text-color-primary);
}
.workflow-graph :deep(.vue-flow__edge-textbg) {
  fill: var(--admin-page-card-bg);
}
.workflow-graph :deep(.vue-flow__handle) {
  width: 8px;
  height: 8px;
  background: var(--el-color-primary);
  border-color: var(--admin-page-card-bg);
}
.workflow-graph :deep(.vue-flow__controls),
.workflow-graph :deep(.vue-flow__minimap) {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius, 6px);
}
.workflow-graph :deep(.vue-flow__controls-button) {
  color: var(--el-text-color-primary);
  fill: currentColor;
  background: var(--admin-page-card-bg);
  border-color: var(--el-border-color);
}
</style>
