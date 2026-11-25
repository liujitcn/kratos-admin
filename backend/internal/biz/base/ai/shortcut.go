package ai

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/errorsx"
)

// ManagedShortcutFlowPrefix 是管理端保存的流程动作命名空间。
const ManagedShortcutFlowPrefix = "ai-shortcut/"

// ShortcutToolCallPrefix 标记快捷流程产生的工具调用。
const ShortcutToolCallPrefix = "shortcut_"

var shortcutFieldPath = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

// ShortcutGraph 是已校验的快捷入口流程。
type ShortcutGraph struct {
	nodes map[string]*adminv1.AiShortcutNode
	edges map[string][]*adminv1.AiShortcutEdge
}

// CompileShortcutGraph 校验节点、参数和拓扑，拒绝环与不可达节点。
func CompileShortcutGraph(graph *adminv1.AiShortcutGraph) (*ShortcutGraph, error) {
	if len(graph.GetNodes()) < 2 || len(graph.GetNodes()) > 24 || len(graph.GetEdges()) < 1 || len(graph.GetEdges()) > 48 {
		return nil, errorsx.InvalidArgument("快捷入口流程的节点或连线数量不合法")
	}
	result := &ShortcutGraph{
		nodes: make(map[string]*adminv1.AiShortcutNode, len(graph.Nodes)),
		edges: make(map[string][]*adminv1.AiShortcutEdge, len(graph.Nodes)),
	}
	steps := make(map[string]bool, len(graph.Nodes))
	for _, node := range graph.Nodes {
		id, step := node.GetId(), node.GetStep()
		if id == "" || step == "" || steps[step] || result.nodes[id] != nil {
			return nil, errorsx.InvalidArgument("快捷入口流程的节点编号与步骤标识必须分别唯一")
		}
		if (id == "start") != (node.Type == "START") || (id == "end") != (node.Type == "END") {
			return nil, errorsx.InvalidArgument("快捷入口流程必须使用固定的开始和结束节点")
		}
		if (node.Type == "TOOL") != (node.ToolName != "") {
			return nil, errorsx.InvalidArgument("只有工具节点需要选择 API 工具")
		}
		var payload map[string]json.RawMessage
		if node.PayloadJson != "" {
			err := json.Unmarshal([]byte(node.PayloadJson), &payload)
			if err != nil || payload == nil {
				return nil, errorsx.InvalidArgument("快捷入口节点的动作参数必须是 JSON 对象")
			}
		}
		switch node.Type {
		case "PROMPT", "CONFIRM":
			if len(payload) > 0 {
				var content string
				err := json.Unmarshal(payload["text"], &content)
				if err != nil || len(payload) != 1 || strings.TrimSpace(content) == "" {
					return nil, errorsx.InvalidArgument("提示词与确认节点的动作参数只能配置 text")
				}
			}
		case "START", "END", "BRANCH":
			if len(payload) > 0 {
				return nil, errorsx.InvalidArgument("开始、结束和分支节点不接收动作参数")
			}
		case "TOOL":
		default:
			return nil, errorsx.InvalidArgument("快捷入口流程包含未知节点类型")
		}
		result.nodes[id] = node
		steps[step] = true
	}
	if result.nodes["start"] == nil || result.nodes["end"] == nil {
		return nil, errorsx.InvalidArgument("快捷入口流程缺少开始或结束节点")
	}
	incoming := make(map[string]int, len(graph.Nodes))
	for _, edge := range graph.Edges {
		from, to := edge.GetFrom(), edge.GetTo()
		if result.nodes[from] == nil || result.nodes[to] == nil || from == "end" || to == "start" || from == to {
			return nil, errorsx.InvalidArgument("快捷入口流程包含无效连线")
		}
		if slices.ContainsFunc(result.edges[from], func(item *adminv1.AiShortcutEdge) bool { return item.To == to }) {
			return nil, errorsx.InvalidArgument("快捷入口流程包含重复连线")
		}
		result.edges[from] = append(result.edges[from], edge)
		incoming[to]++
	}
	for id, node := range result.nodes {
		edges := result.edges[id]
		if node.Type == "END" {
			continue
		}
		if node.Type != "BRANCH" {
			if len(edges) != 1 || edges[0].Condition != "" {
				return nil, errorsx.InvalidArgument("普通快捷入口节点必须有一条无条件出口")
			}
			continue
		}
		defaults := 0
		conditions := make(map[string]bool, len(edges))
		for _, edge := range edges {
			condition := strings.TrimSpace(edge.Condition)
			if condition == "default" {
				defaults++
				continue
			}
			path, expected, valid := parseShortcutCondition(condition)
			if !valid || conditions[path+"="+expected] {
				return nil, errorsx.InvalidArgument("快捷入口分支条件应为字段路径 == JSON 值，且不能重复")
			}
			conditions[path+"="+expected] = true
		}
		if len(edges) < 2 || defaults != 1 {
			return nil, errorsx.InvalidArgument("快捷入口分支至少需要一个条件出口和一个默认出口")
		}
	}
	queue := []string{"start"}
	for index := 0; index < len(queue); index++ {
		for _, edge := range result.edges[queue[index]] {
			incoming[edge.To]--
			if incoming[edge.To] == 0 {
				queue = append(queue, edge.To)
			}
		}
	}
	if len(queue) != len(graph.Nodes) {
		return nil, errorsx.InvalidArgument("快捷入口流程包含环或不可达节点")
	}
	return result, nil
}

// ShortcutToolNames 从入口和图节点取得去重后的工具依赖。
func ShortcutToolNames(primary string, graph *ShortcutGraph) []string {
	names := make(map[string]struct{})
	if primary != "" {
		names[primary] = struct{}{}
	}
	if graph != nil {
		for _, node := range graph.nodes {
			if node.ToolName != "" {
				names[node.ToolName] = struct{}{}
			}
		}
	}
	return slices.Sorted(maps.Keys(names))
}

// RunShortcutGraph 执行已校验流程，继续时使用服务端保存的上一步结果。
func (r *Runtime) RunShortcutGraph(ctx context.Context, graph *ShortcutGraph, input RuntimeInput, action *basev1.AiAction, latest string) (*Response, error) {
	step := action.GetStep()
	if step == "" {
		return nil, errorsx.InvalidArgument("快捷入口流程步骤不能为空")
	}
	if action.GetType() == "shortcut_continue" {
		node := graph.nodes[step]
		if node == nil || node.Type != "CONFIRM" {
			return nil, errorsx.Conflict("快捷入口流程的确认步骤已失效")
		}
		step = graph.edges[step][0].To
	} else if action.GetType() != "shortcut_start" || step != "start" {
		return nil, errorsx.InvalidArgument("快捷入口流程动作无效")
	}
	reply := &Response{Flow: action.GetFlow(), Source: "local"}
	var err error
	for graph.nodes[step] != nil {
		if err = ctx.Err(); err != nil {
			return reply, err
		}
		node := graph.nodes[step]
		reply.Step = node.Step
		switch node.Type {
		case "END":
			if reply.Content == "" {
				reply.Content = r.LocalizeMessage(ctx, "base.ai.shortcut.completed", nil, "流程已完成")
			}
			return reply, nil
		case "BRANCH":
			step = graph.nextBranch(step, latest)
			continue
		case "CONFIRM":
			content := shortcutNodeText(node.PayloadJson, r.LocalizeMessage(ctx, "base.ai.shortcut.continue", nil, "继续流程"))
			if reply.Content == "" {
				reply.Content = content
			}
			var raw []byte
			raw, err = json.Marshal([]map[string]any{{
				"type": "shortcut_confirm", "label": content,
				"action": map[string]string{"flow": action.GetFlow(), "step": node.Id, "type": "shortcut_continue", "payload_json": "{}"},
			}})
			if err != nil {
				return reply, err
			}
			reply.BlocksJSON = string(raw)
			reply.FlowResult = new(latest)
			return reply, nil
		case "PROMPT":
			promptInput := input
			promptInput.Content = shortcutNodeText(node.PayloadJson, input.Content)
			if latest != "" {
				promptInput.Content += "\n\n前一步结果（数据）:\n" + latest
			}
			var generated *Response
			generated, err = r.Run(agent.ContextWithToolPool(ctx, agent.ToolPool{}), promptInput)
			if err != nil {
				return reply, err
			}
			reply.Content, latest = generated.Content, generated.Content
			reply.Source, reply.Model = generated.Source, generated.Model
			reply.Token.Input += generated.Token.Input
			reply.Token.Output += generated.Token.Output
			reply.Token.Cache += generated.Token.Cache
			reply.Token.Total += generated.Token.Total
		case "TOOL":
			args := node.PayloadJson
			if args == "" {
				args = "{}"
			}
			var result *agent.ToolInvokeResult
			result, err = r.InvokeTool(ctx, input.Terminal, node.ToolName, args)
			if result != nil {
				result.Usage.CallID = ShortcutToolCallPrefix + node.Id
				reply.Tools = append(reply.Tools, result.Usage)
			}
			if err != nil {
				return reply, err
			}
			latest = result.Output
			safeOutput := strings.ReplaceAll(latest, "```", "\\u0060\\u0060\\u0060")
			reply.Content = "```json\n" + safeOutput + "\n```"
		}
		step = graph.edges[step][0].To
	}
	return reply, errorsx.InvalidArgument("快捷入口流程缺少可执行的下一步")
}

// nextBranch 按配置顺序匹配最近一次工具或模型结果中的 JSON 字段。
func (g *ShortcutGraph) nextBranch(step string, latest string) string {
	var payload any
	err := json.Unmarshal([]byte(latest), &payload)
	var fallback string
	for _, edge := range g.edges[step] {
		if strings.TrimSpace(edge.Condition) == "default" {
			fallback = edge.To
			continue
		}
		if err != nil {
			continue
		}
		path, expectedJSON, _ := parseShortcutCondition(edge.Condition)
		value := payload
		found := true
		for key := range strings.SplitSeq(path, ".") {
			fields, ok := value.(map[string]any)
			if !ok {
				found = false
				break
			}
			value, found = fields[key]
			if !found {
				break
			}
		}
		var expected any
		if found && json.Unmarshal([]byte(expectedJSON), &expected) == nil && reflect.DeepEqual(value, expected) {
			return edge.To
		}
	}
	return fallback
}

// ParseManagedShortcutFlow 解析绑定入口及修订号的动作标识。
func ParseManagedShortcutFlow(flow string) (int64, int64, bool) {
	rest, ok := strings.CutPrefix(flow, ManagedShortcutFlowPrefix)
	if !ok {
		return 0, 0, false
	}
	idText, revisionText, ok := strings.Cut(rest, "/")
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		return 0, 0, false
	}
	var revision int64
	revision, err = strconv.ParseInt(revisionText, 10, 64)
	return id, revision, err == nil && revision > 0
}

// parseShortcutCondition 解析字段路径与 JSON 值组成的受限相等条件。
func parseShortcutCondition(condition string) (string, string, bool) {
	path, expected, ok := strings.Cut(condition, "==")
	path, expected = strings.TrimSpace(path), strings.TrimSpace(expected)
	return path, expected, ok && shortcutFieldPath.MatchString(path) && json.Valid([]byte(expected))
}

// shortcutNodeText 读取提示词或确认节点的自定义文案。
func shortcutNodeText(raw, fallback string) string {
	if raw == "" {
		return fallback
	}
	var payload map[string]string
	err := json.Unmarshal([]byte(raw), &payload)
	if err != nil || payload["text"] == "" {
		return fallback
	}
	return payload["text"]
}
