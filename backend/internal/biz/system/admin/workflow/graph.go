// 工作流节点图模型：解析、校验与条件求值。纯逻辑无 IO，便于单测复用。
package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// 节点类型。
const (
	NodeTypeScript           = "SCRIPT"                // 脚本节点
	NodeTypeStart            = "START"                 // 开始节点
	NodeTypeEnd              = "END"                   // 结束节点
	NodeTypeTask             = "TASK"                  // 审批任务节点
	NodeTypeExclusiveGateway = "EXCLUSIVE_GATEWAY"     // 排他网关
	NodeTypeParallelFork     = "PARALLEL_GATEWAY_FORK" // 并行分支节点
	NodeTypeParallelJoin     = "PARALLEL_GATEWAY_JOIN" // 并行汇聚节点
)

// 审批人类型。
const (
	ApproverTypeUser = "USER" // 指定用户
	ApproverTypeRole = "ROLE" // 指定角色下全部启用用户
	ApproverTypePost = "POST" // 指定岗位下全部启用用户
)

// 审批策略。
const (
	StrategyAll = "ALL" // 会签：全员通过才推进，任一驳回即驳回
	StrategyAny = "ANY" // 或签：任一通过即推进
)

// ConditionDefault 排他网关默认出边条件标识。
const ConditionDefault = "default"

// graphVersion 当前节点图 JSON 版本。
const graphVersion = 2

// Approver 单个审批人规格，按类型解析为用户 ID 列表。
type Approver struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

// Edge 图边，排他网关出边的 Condition 为表达式或 ConditionDefault。
type Edge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

// Node 图节点内存表示，仅 TASK 节点有审批人配置。
type Node struct {
	ID               string     `json:"id"`
	ScriptID         int64      `json:"script_id,omitempty"`
	ApproverScriptID int64      `json:"approver_script_id,omitempty"`
	X                float64    `json:"x"`
	Y                float64    `json:"y"`
	Type             string     `json:"type"`
	Approvers        []Approver `json:"approvers,omitempty"`
	Strategy         string     `json:"strategy,omitempty"`
	InDegree         int        `json:"-"`
	OutEdges         []Edge     `json:"-"`
}

// IsAnyStrategy 是否或签，缺省按会签处理。
func (n *Node) IsAnyStrategy() bool {
	return strings.EqualFold(n.Strategy, StrategyAny)
}

// ScriptHooks 流程生命周期脚本引用。
type ScriptHooks struct {
	BeforeSubmit int64 `json:"before_submit,omitempty"`
	Approved     int64 `json:"approved,omitempty"`
	Rejected     int64 `json:"rejected,omitempty"`
	Withdrawn    int64 `json:"withdrawn,omitempty"`
}

// ScriptSnapshot 发布时固定的可信脚本版本。
type ScriptSnapshot struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Version  int32  `json:"version"`
	Source   string `json:"source"`
}

// Graph 图内存表示。
type Graph struct {
	Nodes           map[string]*Node          `json:"nodes"`
	Hooks           ScriptHooks               `json:"hooks"`
	ScriptSnapshots map[string]ScriptSnapshot `json:"script_snapshots,omitempty"`
	nodeOrder       []string
}

// graphJSON 节点图定义侧 JSON 结构。
type graphJSON struct {
	Version         int                       `json:"version"`
	Hooks           ScriptHooks               `json:"hooks"`
	ScriptSnapshots map[string]ScriptSnapshot `json:"script_snapshots,omitempty"`
	Nodes           []*Node                   `json:"nodes"`
	Edges           []*Edge                   `json:"edges"`
}

// Parse 解析节点图配置 JSON 为内存图。
func Parse(nodeConfig string) (*Graph, error) {
	var parsed graphJSON
	if err := json.Unmarshal([]byte(nodeConfig), &parsed); err != nil {
		return nil, fmt.Errorf("节点图配置不是合法 JSON: %w", err)
	}
	if parsed.Version != graphVersion {
		return nil, fmt.Errorf("节点图配置版本不支持: %d", parsed.Version)
	}
	graph := &Graph{Nodes: make(map[string]*Node, len(parsed.Nodes)), Hooks: parsed.Hooks, ScriptSnapshots: parsed.ScriptSnapshots}
	for _, node := range parsed.Nodes {
		if node == nil {
			return nil, fmt.Errorf("节点不能为空")
		}
		if node.ID == "" {
			return nil, fmt.Errorf("节点 ID 不能为空")
		}
		if _, exists := graph.Nodes[node.ID]; exists {
			return nil, fmt.Errorf("节点 ID 重复: %s", node.ID)
		}
		graph.Nodes[node.ID] = node
		graph.nodeOrder = append(graph.nodeOrder, node.ID)
	}
	for _, edge := range parsed.Edges {
		if edge == nil {
			return nil, fmt.Errorf("边不能为空")
		}
		from, okFrom := graph.Nodes[edge.From]
		if !okFrom {
			return nil, fmt.Errorf("边引用不存在的节点: %s", edge.From)
		}
		if _, okTo := graph.Nodes[edge.To]; !okTo {
			return nil, fmt.Errorf("边引用不存在的节点: %s", edge.To)
		}
		from.OutEdges = append(from.OutEdges, *edge)
		graph.Nodes[edge.To].InDegree++
	}
	return graph, nil
}

// Marshal 按定义节点顺序导出完整图配置与服务端脚本快照。
func (g *Graph) Marshal() (string, error) {
	config := graphJSON{Version: graphVersion, Hooks: g.Hooks, ScriptSnapshots: g.ScriptSnapshots}
	ids := append([]string(nil), g.nodeOrder...)
	if len(ids) == 0 {
		for id := range g.Nodes {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	}
	for _, id := range ids {
		node := g.Nodes[id]
		config.Nodes = append(config.Nodes, node)
		for _, edge := range node.OutEdges {
			edgeCopy := edge
			config.Edges = append(config.Edges, &edgeCopy)
		}
	}
	raw, err := json.Marshal(config)
	return string(raw), err
}

// ScriptIDs 返回去重且升序排列的全部脚本引用编号。
func (g *Graph) ScriptIDs() []int64 {
	seen := make(map[int64]bool)
	for _, id := range []int64{g.Hooks.BeforeSubmit, g.Hooks.Approved, g.Hooks.Rejected, g.Hooks.Withdrawn} {
		if id > 0 {
			seen[id] = true
		}
	}
	for _, node := range g.Nodes {
		if node.ScriptID > 0 {
			seen[node.ScriptID] = true
		}
		if node.ApproverScriptID > 0 {
			seen[node.ApproverScriptID] = true
		}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// StartNode 返回图的开始节点。
func (g *Graph) StartNode() *Node {
	for _, node := range g.Nodes {
		if node.Type == NodeTypeStart {
			return node
		}
	}
	return nil
}

// Validate 校验图结构合法性：节点出入度约束、审批人配置与双向可达性。
func Validate(graph *Graph) error {
	for _, id := range []int64{graph.Hooks.BeforeSubmit, graph.Hooks.Approved, graph.Hooks.Rejected, graph.Hooks.Withdrawn} {
		if id < 0 {
			return fmt.Errorf("生命周期脚本 ID 必须为正数")
		}
	}
	startCount, endCount := 0, 0
	for _, node := range graph.Nodes {
		if utf8.RuneCountInString(node.ID) > 100 {
			return fmt.Errorf("节点 ID 不能超过 100 个字符: %s", node.ID)
		}
		if node.ScriptID < 0 || node.ApproverScriptID < 0 {
			return fmt.Errorf("节点 %s 脚本 ID 必须为正数", node.ID)
		}
		if node.ScriptID != 0 && node.Type != NodeTypeScript {
			return fmt.Errorf("节点 %s 不允许配置执行脚本", node.ID)
		}
		if node.ApproverScriptID != 0 && node.Type != NodeTypeTask {
			return fmt.Errorf("节点 %s 不允许配置审批人脚本", node.ID)
		}
		switch node.Type {
		case NodeTypeScript:
			if node.ScriptID <= 0 || node.InDegree != 1 || len(node.OutEdges) != 1 {
				return fmt.Errorf("脚本节点 %s 必须配置正数脚本 ID 且单入单出", node.ID)
			}
		case NodeTypeStart:
			startCount++
			if len(node.OutEdges) != 1 {
				return fmt.Errorf("开始节点 %s 必须恰好一条出边", node.ID)
			}
			if node.InDegree != 0 {
				return fmt.Errorf("开始节点 %s 不能有入边", node.ID)
			}
		case NodeTypeEnd:
			endCount++
			if node.InDegree < 1 {
				return fmt.Errorf("结束节点 %s 至少一条入边", node.ID)
			}
			if len(node.OutEdges) != 0 {
				return fmt.Errorf("结束节点 %s 不能有出边", node.ID)
			}
		case NodeTypeTask:
			if len(node.OutEdges) != 1 {
				return fmt.Errorf("审批节点 %s 必须恰好一条出边", node.ID)
			}
			if node.InDegree != 1 {
				return fmt.Errorf("审批节点 %s 必须恰好一条入边", node.ID)
			}
			if err := validateApprovers(node); err != nil {
				return err
			}
		case NodeTypeExclusiveGateway:
			if len(node.OutEdges) < 2 {
				return fmt.Errorf("排他网关 %s 至少两条出边", node.ID)
			}
			defaultCount := 0
			for _, edge := range node.OutEdges {
				if edge.Condition == "" {
					return fmt.Errorf("排他网关 %s 出边缺少条件", node.ID)
				}
				if edge.Condition == ConditionDefault {
					defaultCount++
				}
			}
			if defaultCount != 1 {
				return fmt.Errorf("排他网关 %s 必须恰好一条默认出边", node.ID)
			}
		case NodeTypeParallelFork:
			if len(node.OutEdges) < 2 {
				return fmt.Errorf("并行分支 %s 至少两条出边", node.ID)
			}
			for _, edge := range node.OutEdges {
				if edge.Condition != "" {
					return fmt.Errorf("并行分支 %s 出边不能带条件", node.ID)
				}
			}
		case NodeTypeParallelJoin:
			if node.InDegree < 2 {
				return fmt.Errorf("并行汇聚 %s 至少两条入边", node.ID)
			}
			if len(node.OutEdges) != 1 {
				return fmt.Errorf("并行汇聚 %s 必须恰好一条出边", node.ID)
			}
		default:
			return fmt.Errorf("不支持的节点类型: %s(%s)", node.ID, node.Type)
		}
	}
	if startCount != 1 {
		return fmt.Errorf("节点图必须恰好一个开始节点")
	}
	if endCount != 1 {
		return fmt.Errorf("节点图必须恰好一个结束节点")
	}
	if err := checkAcyclic(graph); err != nil {
		return err
	}
	return checkReachability(graph)
}

// validateApprovers 校验审批节点的审批人配置。
func validateApprovers(node *Node) error {
	if (len(node.Approvers) == 0) == (node.ApproverScriptID == 0) {
		return fmt.Errorf("审批节点 %s 必须且只能配置静态审批人或审批人脚本", node.ID)
	}
	seen := make(map[string]struct{}, len(node.Approvers))
	for _, approver := range node.Approvers {
		if approver.ID <= 0 {
			return fmt.Errorf("审批节点 %s 审批人 ID 无效", node.ID)
		}
		switch approver.Type {
		case ApproverTypeUser, ApproverTypeRole, ApproverTypePost:
		default:
			return fmt.Errorf("审批节点 %s 审批人类型不支持: %s", node.ID, approver.Type)
		}
		key := fmt.Sprintf("%s:%d", approver.Type, approver.ID)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("审批节点 %s 审批人重复: %s", node.ID, key)
		}
		seen[key] = struct{}{}
	}
	if node.Strategy != "" && !strings.EqualFold(node.Strategy, StrategyAll) && !strings.EqualFold(node.Strategy, StrategyAny) {
		return fmt.Errorf("审批节点 %s 审批策略无效: %s", node.ID, node.Strategy)
	}
	return nil
}

// checkReachability 校验从开始节点可达全部节点，且全部节点可达结束节点。
func checkReachability(graph *Graph) error {
	start := graph.StartNode()

	forward := bfsCollect(graph, start.ID, func(node *Node) []string {
		next := make([]string, 0, len(node.OutEdges))
		for _, edge := range node.OutEdges {
			next = append(next, edge.To)
		}
		return next
	})
	for id := range graph.Nodes {
		if !forward[id] {
			return fmt.Errorf("节点不可从开始节点到达: %s", id)
		}
	}

	reverseEdges := make(map[string][]string, len(graph.Nodes))
	var endNode *Node
	for _, node := range graph.Nodes {
		if node.Type == NodeTypeEnd {
			endNode = node
		}
		for _, edge := range node.OutEdges {
			reverseEdges[edge.To] = append(reverseEdges[edge.To], node.ID)
		}
	}
	backward := bfsCollect(graph, endNode.ID, func(node *Node) []string {
		return reverseEdges[node.ID]
	})
	for id := range graph.Nodes {
		if !backward[id] {
			return fmt.Errorf("节点不可到达结束节点: %s", id)
		}
	}
	return nil
}

// bfsCollect 从指定节点做广度优先遍历，next 返回后继节点 ID 列表。
func bfsCollect(graph *Graph, from string, next func(*Node) []string) map[string]bool {
	visited := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, id := range next(graph.Nodes[current]) {
			if !visited[id] {
				visited[id] = true
				queue = append(queue, id)
			}
		}
	}
	return visited
}

// checkAcyclic 通过拓扑排序拒绝环，保证节点唯一执行键的语义稳定。
func checkAcyclic(graph *Graph) error {
	degrees := make(map[string]int, len(graph.Nodes))
	queue := make([]string, 0, len(graph.Nodes))
	for id, node := range graph.Nodes {
		degrees[id] = node.InDegree
		if node.InDegree == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, edge := range graph.Nodes[id].OutEdges {
			degrees[edge.To]--
			if degrees[edge.To] == 0 {
				queue = append(queue, edge.To)
			}
		}
	}
	if visited != len(graph.Nodes) {
		return fmt.Errorf("节点图不允许循环")
	}
	return nil
}
