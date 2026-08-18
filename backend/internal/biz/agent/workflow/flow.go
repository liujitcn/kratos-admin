package workflow

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

const unsupportedActionNode = "unsupported_action"

// FlowName 表示模块固定流程名称。
type FlowName string

// Action 表示流程动作定义。
type Action struct {
	// Flow 表示动作所属流程。
	Flow FlowName
	// Step 表示流程步骤。
	Step string
	// Type 表示动作类型。
	Type string
}

// ActionRequest 表示固定流程动作执行输入。
type ActionRequest struct {
	// Flow 表示前端当前所在流程。
	Flow FlowName
	// ActionType 表示前端按钮动作类型。
	ActionType string
	// Payload 表示前端按钮携带的上下文。
	Payload map[string]any
}

// LookupRequest 表示固定流程动作查询输入。
type LookupRequest struct {
	// Flow 表示流程名称。
	Flow FlowName
	// ActionType 表示动作类型。
	ActionType string
}

// LookupResult 表示固定流程动作查询结果。
type LookupResult struct {
	// Definition 表示命中的流程定义。
	Definition Definition
	// Action 表示命中的动作定义。
	Action Action
	// Found 表示是否命中动作。
	Found bool
}

// ActionResult 表示固定流程动作执行结果。
type ActionResult[T any] struct {
	// Definition 表示命中的流程定义。
	Definition Definition
	// Action 表示命中的动作定义。
	Action Action
	// Output 表示业务动作输出。
	Output T
	// Found 表示是否命中动作。
	Found bool
}

// ActionHandler 表示固定流程动作处理函数。
type ActionHandler[T any] func(context.Context, Action, map[string]any) (T, error)

// Definition 表示一个固定流程的动作图定义。
type Definition struct {
	// Name 表示流程名称。
	Name FlowName
	// EntryAction 表示入口动作类型。
	EntryAction string
	// Actions 表示流程可处理动作。
	Actions []Action
}

// Registry 保存模块提供的固定流程定义与已编译图。
type Registry[T any] struct {
	definitions          map[FlowName]Definition
	actionsByType        map[string][]Action
	actionsByFlowAndType map[string]Action
	lookupRunnable       compose.Runnable[LookupRequest, LookupResult]
	actionRunnable       compose.Runnable[ActionRequest, LookupResult]
}

// NewRegistry 校验模块流程定义并编译动作路由图。
func NewRegistry[T any](definitions []Definition, scope string, graphName string) (*Registry[T], error) {
	err := validateDefinitions(definitions, scope)
	if err != nil {
		return nil, err
	}
	registry := &Registry[T]{
		definitions:          make(map[FlowName]Definition, len(definitions)),
		actionsByType:        make(map[string][]Action, len(definitions)*2),
		actionsByFlowAndType: make(map[string]Action, len(definitions)*4),
	}
	for _, definition := range definitions {
		registry.definitions[definition.Name] = definition
		for _, action := range definition.Actions {
			registry.actionsByType[action.Type] = append(registry.actionsByType[action.Type], action)
			registry.actionsByFlowAndType[actionKey(action.Flow, action.Type)] = action
		}
	}
	registry.lookupRunnable, err = compileLookupWorkflow(registry, graphName)
	if err != nil {
		return nil, err
	}
	registry.actionRunnable, err = compileActionGraph(registry, graphName+"_action_graph")
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// MustNewRegistry 创建固定流程注册表，定义非法时直接中止启动。
func MustNewRegistry[T any](definitions []Definition, scope string, graphName string) *Registry[T] {
	registry, err := NewRegistry[T](definitions, scope, graphName)
	if err != nil {
		panic(err)
	}
	return registry
}

// EntryAction 返回流程入口动作。
func (r *Registry[T]) EntryAction(flow FlowName) string {
	if r == nil {
		return ""
	}
	return r.definitions[flow].EntryAction
}

// Action 返回指定流程内的动作定义。
func (r *Registry[T]) Action(flow FlowName, actionType string) (Action, bool) {
	if r == nil || flow == "" || actionType == "" {
		return Action{}, false
	}
	result := r.lookup(LookupRequest{Flow: flow, ActionType: actionType})
	if !result.Found {
		return Action{}, false
	}
	return result.Action, true
}

// UniqueAction 返回全局唯一动作定义；有多个同名动作时必须提供流程标识。
func (r *Registry[T]) UniqueAction(actionType string) (Action, bool) {
	if r == nil || actionType == "" {
		return Action{}, false
	}
	result := r.lookup(LookupRequest{ActionType: actionType})
	if !result.Found {
		return Action{}, false
	}
	return result.Action, true
}

// Run 通过 Eino 图路由并执行固定流程动作。
func (r *Registry[T]) Run(ctx context.Context, request ActionRequest, handler ActionHandler[T]) (ActionResult[T], error) {
	if r == nil {
		return ActionResult[T]{}, nil
	}
	var lookup LookupResult
	if r.actionRunnable == nil {
		lookup = r.lookup(LookupRequest{Flow: request.Flow, ActionType: request.ActionType})
	} else {
		var err error
		lookup, err = r.actionRunnable.Invoke(ctx, request)
		if err != nil {
			return ActionResult[T]{}, err
		}
	}
	return r.executeAction(ctx, request, lookup, handler)
}

// HasFlow 判断流程是否存在。
func (r *Registry[T]) HasFlow(flow FlowName) bool {
	if r == nil {
		return false
	}
	_, exists := r.definitions[flow]
	return exists
}

// Lookup 通过 Eino 图查询固定流程动作。
func (r *Registry[T]) Lookup(ctx context.Context, request LookupRequest) (LookupResult, error) {
	if r == nil {
		return LookupResult{}, nil
	}
	if r.lookupRunnable == nil {
		return r.lookup(request), nil
	}
	return r.lookupRunnable.Invoke(ctx, request)
}

// lookup 执行固定流程动作查询的纯业务逻辑。
func (r *Registry[T]) lookup(request LookupRequest) LookupResult {
	if request.ActionType == "" {
		return LookupResult{}
	}
	if request.Flow != "" {
		action, exists := r.actionsByFlowAndType[actionKey(request.Flow, request.ActionType)]
		if !exists {
			return LookupResult{Definition: r.definitions[request.Flow]}
		}
		return LookupResult{Definition: r.definitions[action.Flow], Action: action, Found: true}
	}
	actions := r.actionsByType[request.ActionType]
	if len(actions) != 1 {
		return LookupResult{}
	}
	action := actions[0]
	return LookupResult{Definition: r.definitions[action.Flow], Action: action, Found: true}
}

// executeAction 执行已路由到具体节点的固定流程动作。
func (r *Registry[T]) executeAction(ctx context.Context, request ActionRequest, lookup LookupResult, handler ActionHandler[T]) (ActionResult[T], error) {
	if !lookup.Found {
		return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action}, nil
	}
	if handler == nil {
		return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action, Found: true}, fmt.Errorf("固定流程动作处理器未配置")
	}
	output, err := handler(ctx, lookup.Action, request.Payload)
	return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action, Output: output, Found: true}, err
}

// validateDefinitions 校验模块流程定义，避免运行期发现路由配置错误。
func validateDefinitions(definitions []Definition, scope string) error {
	flows := make(map[FlowName]bool, len(definitions))
	actions := make(map[string]bool, len(definitions)*4)
	for _, definition := range definitions {
		if definition.Name == "" {
			return fmt.Errorf("%s固定流程名称不能为空", scope)
		}
		if flows[definition.Name] {
			return fmt.Errorf("%s固定流程重复定义: %s", scope, definition.Name)
		}
		flows[definition.Name] = true
		if definition.EntryAction == "" {
			return fmt.Errorf("%s固定流程 %s 缺少入口动作", scope, definition.Name)
		}
		for _, action := range definition.Actions {
			if action.Flow != definition.Name {
				return fmt.Errorf("%s固定流程 %s 包含跨流程动作 %s", scope, definition.Name, action.Type)
			}
			if action.Step == "" || action.Type == "" {
				return fmt.Errorf("%s固定流程 %s 包含无效动作", scope, definition.Name)
			}
			key := actionKey(action.Flow, action.Type)
			if actions[key] {
				return fmt.Errorf("%s固定流程动作重复定义: %s/%s", scope, action.Flow, action.Type)
			}
			actions[key] = true
		}
		if !actions[actionKey(definition.Name, definition.EntryAction)] {
			return fmt.Errorf("%s固定流程 %s 入口动作未注册: %s", scope, definition.Name, definition.EntryAction)
		}
	}
	return nil
}

// compileLookupWorkflow 编译固定流程查询图。
func compileLookupWorkflow[T any](registry *Registry[T], graphName string) (compose.Runnable[LookupRequest, LookupResult], error) {
	workflow := compose.NewWorkflow[LookupRequest, LookupResult]()
	workflow.AddLambdaNode("lookup_action", compose.InvokableLambda(func(_ context.Context, input LookupRequest) (LookupResult, error) {
		if registry == nil {
			return LookupResult{}, nil
		}
		return registry.lookup(input), nil
	}, compose.WithLambdaType("fixed_flow.lookup_action"))).AddDependency(compose.START)
	workflow.End().AddDependency("lookup_action")
	runnable, err := workflow.Compile(context.Background(), compose.WithGraphName(graphName))
	if err != nil {
		return nil, fmt.Errorf("编译固定流程查询图失败: %w", err)
	}
	return runnable, nil
}

// compileActionGraph 编译固定流程动作图。
func compileActionGraph[T any](registry *Registry[T], graphName string) (compose.Runnable[ActionRequest, LookupResult], error) {
	graph := compose.NewGraph[ActionRequest, LookupResult]()
	endNodes := make(map[string]bool, len(registry.actionsByFlowAndType)+1)
	for _, actions := range registry.actionsByType {
		for _, action := range actions {
			nodeKey := actionNodeKey(action)
			endNodes[nodeKey] = true
			currentAction := action
			err := graph.AddLambdaNode(nodeKey, compose.InvokableLambda(func(_ context.Context, input ActionRequest) (LookupResult, error) {
				lookup := registry.lookup(LookupRequest{Flow: input.Flow, ActionType: input.ActionType})
				if !lookup.Found {
					return LookupResult{}, nil
				}
				return LookupResult{Definition: lookup.Definition, Action: currentAction, Found: true}, nil
			}, compose.WithLambdaType("fixed_flow."+nodeKey)))
			if err != nil {
				return nil, fmt.Errorf("注册固定流程动作节点失败: %w", err)
			}
			if err = graph.AddEdge(nodeKey, compose.END); err != nil {
				return nil, fmt.Errorf("连接固定流程动作节点失败: %w", err)
			}
		}
	}
	endNodes[unsupportedActionNode] = true
	err := graph.AddLambdaNode(unsupportedActionNode, compose.InvokableLambda(func(context.Context, ActionRequest) (LookupResult, error) {
		return LookupResult{}, nil
	}, compose.WithLambdaType("fixed_flow.unsupported_action")))
	if err != nil {
		return nil, fmt.Errorf("注册固定流程兜底节点失败: %w", err)
	}
	if err = graph.AddEdge(unsupportedActionNode, compose.END); err != nil {
		return nil, fmt.Errorf("连接固定流程兜底节点失败: %w", err)
	}
	err = graph.AddBranch(compose.START, compose.NewGraphBranch(func(_ context.Context, input ActionRequest) (string, error) {
		lookup := registry.lookup(LookupRequest{Flow: input.Flow, ActionType: input.ActionType})
		if !lookup.Found {
			return unsupportedActionNode, nil
		}
		return actionNodeKey(lookup.Action), nil
	}, endNodes))
	if err != nil {
		return nil, fmt.Errorf("注册固定流程路由分支失败: %w", err)
	}
	var runnable compose.Runnable[ActionRequest, LookupResult]
	runnable, err = graph.Compile(context.Background(), compose.WithGraphName(graphName))
	if err != nil {
		return nil, fmt.Errorf("编译固定流程动作图失败: %w", err)
	}
	return runnable, nil
}

// actionKey 返回流程和动作类型的复合索引键。
func actionKey(flow FlowName, actionType string) string {
	return string(flow) + "\x00" + actionType
}

// actionNodeKey 返回图中对应动作的节点名称。
func actionNodeKey(action Action) string {
	return fmt.Sprintf("%s_%s", action.Flow, action.Type)
}
