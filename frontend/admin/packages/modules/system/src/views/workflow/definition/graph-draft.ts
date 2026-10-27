/**
 * 流程定义设计器的纯逻辑模块：节点图草稿的序列化、装配与校验。
 * 与组件状态解耦，便于单测复用；校验规则与后端 engine 包语义一致。
 */

/** 审批人草稿。 */
export interface ApproverDraft {
  /** 审批人类型：USER 指定用户、ROLE 指定角色、POST 指定岗位。 */
  type: "USER" | "ROLE" | "POST";
  /** 类型目标 ID：用户/角色/岗位编号。 */
  id: number;
}

/** 图节点草稿，TASK 节点携带审批人配置。 */
export interface GraphNodeDraft {
  /** 节点 ID，图内唯一。 */
  id: string;
  /** 节点类型。 */
  type: "START" | "END" | "TASK" | "SCRIPT" | "EXCLUSIVE_GATEWAY" | "PARALLEL_GATEWAY_FORK" | "PARALLEL_GATEWAY_JOIN";
  /** 审批策略：ALL 会签、ANY 或签，仅 TASK 节点有效。 */
  strategy?: "ALL" | "ANY";
  /** 审批人列表，仅 TASK 节点有效。 */
  approvers?: ApproverDraft[];
  /** 执行脚本编号。 */
  script_id?: number;
  /** 动态审批人脚本编号，与静态审批人互斥。 */
  approver_script_id?: number;
  /** 画布横坐标。 */
  x?: number;
  /** 画布纵坐标。 */
  y?: number;
}

/** 图边草稿，排他网关出边的 condition 为表达式或 "default"。 */
export interface GraphEdgeDraft {
  /** 起点 ID。 */
  from: string;
  /** 终点 ID。 */
  to: string;
  /** 出边条件表达式，排他网关必填。 */
  condition?: string;
}

/** 工作流生命周期脚本钩子。 */
export interface GraphHooks {
  before_submit?: number;
  approved?: number;
  rejected?: number;
  withdrawn?: number;
}

/** 发布时由服务端固定的脚本快照。 */
export interface ScriptSnapshot {
  id: number;
  name: string;
  language: number | string;
  version: number;
  source?: string;
}

/** 节点图草稿。 */
export interface GraphDraft {
  /** 图格式版本，当前固定 2。 */
  version: number;
  /** 生命周期脚本配置。 */
  hooks?: GraphHooks;
  /** 已发布脚本快照，编辑保存时原样保留。 */
  script_snapshots?: Record<string, ScriptSnapshot>;
  /** 节点列表。 */
  nodes: GraphNodeDraft[];
  /** 边列表。 */
  edges: GraphEdgeDraft[];
}

/** 审批详情中的节点办理状态。 */
export type WorkflowProgressNodeState = "done" | "active" | "rejected" | "todo";

/** 审批详情中的连线执行状态。 */
export type WorkflowProgressEdgeState = "done" | "active" | "selected" | "skipped" | "todo";

/** 默认条件标识。 */
export const CONDITION_DEFAULT = "default";

const WORKFLOW_GATEWAY_TYPES = new Set<GraphNodeDraft["type"]>([
  "EXCLUSIVE_GATEWAY",
  "PARALLEL_GATEWAY_FORK",
  "PARALLEL_GATEWAY_JOIN"
]);

/** 根据已办理节点推导流程图连线状态，并标记排他网关未命中的分支。 */
export function deriveProgressEdgeStates(
  nodes: GraphNodeDraft[],
  edges: GraphEdgeDraft[],
  nodeStates: Record<string, WorkflowProgressNodeState>
): Record<number, WorkflowProgressEdgeState> {
  const nodeMap = new Map(nodes.map(node => [node.id, node]));
  const outgoing = new Map<string, GraphEdgeDraft[]>();
  for (const edge of edges) {
    const list = outgoing.get(edge.from) ?? [];
    list.push(edge);
    outgoing.set(edge.from, list);
  }

  function reachedNode(nodeID: string, visited = new Set<string>()): boolean {
    if (nodeStates[nodeID] && nodeStates[nodeID] !== "todo") return true;
    const node = nodeMap.get(nodeID);
    if (!node || !WORKFLOW_GATEWAY_TYPES.has(node.type) || visited.has(nodeID)) return false;
    const nextVisited = new Set(visited);
    nextVisited.add(nodeID);
    return (outgoing.get(nodeID) ?? []).some(edge => reachedNode(edge.to, nextVisited));
  }

  const selected = edges.map(edge => {
    const targetState = nodeStates[edge.to];
    if (targetState && targetState !== "todo") return true;
    const sourceState = nodeStates[edge.from];
    const targetNode = nodeMap.get(edge.to);
    if (sourceState === "done" && targetNode && WORKFLOW_GATEWAY_TYPES.has(targetNode.type)) return true;
    const sourceNode = nodeMap.get(edge.from);
    return Boolean(sourceNode && WORKFLOW_GATEWAY_TYPES.has(sourceNode.type) && reachedNode(edge.to));
  });

  const selectedByExclusive = new Map<string, boolean>();
  edges.forEach((edge, index) => {
    if (nodeMap.get(edge.from)?.type !== "EXCLUSIVE_GATEWAY" || !selected[index]) return;
    selectedByExclusive.set(edge.from, true);
  });

  return Object.fromEntries(
    edges.map((edge, index) => {
      const targetState = nodeStates[edge.to];
      if (targetState === "done") return [index, "done"];
      if (targetState === "active") return [index, "active"];
      if (selected[index]) return [index, "selected"];
      if (nodeMap.get(edge.from)?.type === "EXCLUSIVE_GATEWAY" && selectedByExclusive.get(edge.from)) {
        return [index, "skipped"];
      }
      return [index, "todo"];
    })
  ) as Record<number, WorkflowProgressEdgeState>;
}

/** 节点类型展示 key 前缀。 */
const NODE_TYPE_I18N_PREFIX = "system.workflow.definition.node_type.";

/** 节点类型的展示文案 key。 */
export function nodeTypeI18nKey(type: string): string {
  return `${NODE_TYPE_I18N_PREFIX}${type}`;
}

/** 生成节点自动编号游标。 */
let nodeCounter = 0;

/** 新建节点草稿，生成不与现有流程节点冲突的编号。 */
export function makeGraphNode(
  type: GraphNodeDraft["type"],
  x: number,
  y: number,
  existingNodes: GraphNodeDraft[] = []
): GraphNodeDraft {
  const prefix =
    type === "START"
      ? "start"
      : type === "END"
        ? "end"
        : type === "TASK"
          ? "task"
          : type === "SCRIPT"
            ? "script"
            : type === "EXCLUSIVE_GATEWAY"
              ? "gw"
              : type === "PARALLEL_GATEWAY_FORK"
                ? "fork"
                : "join";
  do {
    nodeCounter += 1;
  } while (existingNodes.some(node => node.id === `${prefix}_${nodeCounter}`));
  const node: GraphNodeDraft = { id: `${prefix}_${nodeCounter}`, type, x, y };
  if (type === "TASK") {
    node.strategy = "ALL";
    node.approvers = [];
  }
  return node;
}

/** 序列化节点图草稿为后端 node_config JSON 文本。 */
export function serializeGraph(draft: GraphDraft): string {
  return JSON.stringify({
    version: draft.version,
    ...(draft.hooks ? { hooks: draft.hooks } : {}),
    ...(draft.script_snapshots ? { script_snapshots: draft.script_snapshots } : {}),
    nodes: draft.nodes.map(node => ({
      id: node.id,
      type: node.type,
      x: node.x,
      y: node.y,
      ...(node.type === "SCRIPT" ? { script_id: node.script_id } : {}),
      ...(node.type === "TASK"
        ? {
            strategy: node.strategy ?? "ALL",
            approvers: node.approvers ?? [],
            ...(node.approver_script_id ? { approver_script_id: node.approver_script_id } : {})
          }
        : {})
    })),
    edges: draft.edges.map(edge => ({
      from: edge.from,
      to: edge.to,
      ...(edge.condition ? { condition: edge.condition } : {})
    }))
  });
}

/** 解析流程草稿并保留布局，无坐标时按节点序号铺开。 */
export function parseGraph(nodeConfig: string): GraphDraft | null {
  try {
    const parsed = JSON.parse(nodeConfig) as GraphDraft;
    if (parsed.version !== 2 || !Array.isArray(parsed.nodes) || !Array.isArray(parsed.edges)) return null;
    // 后端旧数据会把未保存的坐标序列化为 0，全部重合时按节点顺序重新布局。
    const hasStoredLayout = parsed.nodes.some(
      node => typeof node.x === "number" && typeof node.y === "number" && (node.x !== 0 || node.y !== 0)
    );
    parsed.nodes = parsed.nodes.map((node, index) => ({
      ...node,
      ...(node.type === "TASK" ? { strategy: node.strategy ?? "ALL" } : {}),
      x: hasStoredLayout && typeof node.x === "number" ? node.x : 40 + (index % 4) * 200,
      y: hasStoredLayout && typeof node.y === "number" ? node.y : 40 + Math.floor(index / 4) * 120
    }));
    return parsed;
  } catch {
    return null;
  }
}

/** 为画布与配置表单统一新增连线，拒绝缺失端点、自连线和重复连线。 */
export function addGraphEdge(draft: GraphDraft, from: string, to: string, condition = ""): string {
  if (!from || !to) return "system.workflow.definition.error.edge_required";
  if (from === to) return "system.workflow.definition.error.edge_self";
  if (!draft.nodes.some(node => node.id === from) || !draft.nodes.some(node => node.id === to)) {
    return "system.workflow.definition.error.unknown_edge_node";
  }
  if (draft.edges.some(edge => edge.from === from && edge.to === to)) return "system.workflow.definition.error.edge_duplicate";
  draft.edges.push({ from, to, condition });
  return "";
}

/**
 * 校验节点图草稿，返回首个错误文案 key；通过返回空串。
 * 规则与后端 engine 包一致：出入度约束、审批人配置与双向可达性。
 */
export function validateGraphDraft(draft: GraphDraft): string {
  const nodes = new Map(draft.nodes.map(node => [node.id, node]));
  let startCount = 0;
  let endCount = 0;
  const inDegree = new Map<string, number>();
  const outEdges = new Map<string, GraphEdgeDraft[]>();
  for (const node of draft.nodes) {
    inDegree.set(node.id, 0);
    outEdges.set(node.id, []);
  }
  for (const edge of draft.edges) {
    if (!nodes.has(edge.from) || !nodes.has(edge.to)) return "system.workflow.definition.error.unknown_edge_node";
    inDegree.set(edge.to, (inDegree.get(edge.to) ?? 0) + 1);
    outEdges.get(edge.from)?.push(edge);
  }

  for (const node of draft.nodes) {
    const out = outEdges.get(node.id) ?? [];
    const ind = inDegree.get(node.id) ?? 0;
    switch (node.type) {
      case "START":
        startCount += 1;
        if (out.length !== 1) return "system.workflow.definition.error.start_out_degree";
        break;
      case "END":
        endCount += 1;
        if (ind < 1) return "system.workflow.definition.error.end_in_degree";
        if (out.length !== 0) return "system.workflow.definition.error.end_out_degree";
        break;
      case "TASK": {
        if (out.length !== 1 || ind !== 1) return "system.workflow.definition.error.task_degree";
        const approvers = node.approvers ?? [];
        if ((approvers.length === 0) === !node.approver_script_id) return "system.workflow.definition.error.task_approvers";
        const seen = new Set<string>();
        for (const approver of approvers) {
          if (!approver.id || approver.id <= 0) return "system.workflow.definition.error.approver_id";
          const key = `${approver.type}:${approver.id}`;
          if (seen.has(key)) return "system.workflow.definition.error.approver_duplicate";
          seen.add(key);
        }
        break;
      }
      case "SCRIPT":
        if (out.length !== 1 || ind !== 1) return "system.workflow.definition.error.task_degree";
        if (!node.script_id || node.script_id <= 0) return "system.workflow.script.required";
        break;
      case "EXCLUSIVE_GATEWAY": {
        if (out.length < 2) return "system.workflow.definition.error.gateway_out_degree";
        const defaults = out.filter(edge => edge.condition === CONDITION_DEFAULT);
        if (defaults.length !== 1) return "system.workflow.definition.error.gateway_default";
        if (out.some(edge => !edge.condition)) return "system.workflow.definition.error.gateway_condition_missing";
        break;
      }
      case "PARALLEL_GATEWAY_FORK":
        if (out.length < 2) return "system.workflow.definition.error.fork_out_degree";
        if (out.some(edge => edge.condition)) return "system.workflow.definition.error.fork_condition";
        break;
      case "PARALLEL_GATEWAY_JOIN":
        if (ind < 2) return "system.workflow.definition.error.join_in_degree";
        if (out.length !== 1) return "system.workflow.definition.error.join_out_degree";
        break;
      default:
        return "system.workflow.definition.error.unknown_node_type";
    }
  }
  if (startCount !== 1) return "system.workflow.definition.error.start_count";
  if (endCount !== 1) return "system.workflow.definition.error.end_count";

  const startId = draft.nodes.find(node => node.type === "START")?.id ?? "";
  const endId = draft.nodes.find(node => node.type === "END")?.id ?? "";
  const forward = bfsReach(draft, startId, id => (outEdges.get(id) ?? []).map(edge => edge.to));
  const backward = bfsReach(draft, endId, id => {
    const preds: string[] = [];
    for (const edge of draft.edges) if (edge.to === id) preds.push(edge.from);
    return preds;
  });
  for (const node of draft.nodes) {
    if (!forward.get(node.id)) return "system.workflow.definition.error.not_reachable";
    if (!backward.get(node.id)) return "system.workflow.definition.error.cannot_reach_end";
  }
  return "";
}

/** 广度优先标记可达节点。 */
function bfsReach(draft: GraphDraft, from: string, next: (id: string) => string[]): Map<string, boolean> {
  const visited = new Map<string, boolean>();
  if (!draft.nodes.some(node => node.id === from)) return visited;
  visited.set(from, true);
  const queue = [from];
  while (queue.length) {
    const current = queue.shift() as string;
    for (const id of next(current)) {
      if (!visited.get(id)) {
        visited.set(id, true);
        queue.push(id);
      }
    }
  }
  return visited;
}
