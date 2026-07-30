package tool

import (
	"context"
	"encoding/json"
	"fmt"

	componentsTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	einoMiddleware "github.com/liujitcn/kratos-admin/backend/internal/agent/middleware"
)

// Option 表示 Eino 工具调用选项。
type Option = componentsTool.Option

// Info 表示 Eino 工具定义。
type Info = schema.ToolInfo

// Invokable 表示可直接执行的 Eino 工具。
type Invokable = componentsTool.InvokableTool

// Base 表示只提供定义信息的 Eino 工具。
type Base = componentsTool.BaseTool

// Call 表示一次函数工具调用请求。
type Call struct {
	// ID 工具调用 ID。
	ID string
	// Name 工具名称。
	Name string
	// Arguments 工具原始入参 JSON。
	Arguments string
}

// CallResult 表示一次函数工具调用结果。
type CallResult struct {
	// Content 给模型继续推理使用的工具结果文本。
	Content string
	// Type 工具类型。
	Type string
	// Name 工具名称。
	Name string
	// Title 工具展示名称。
	Title string
	// Status 工具执行状态。
	Status string
	// Input 工具原始入参 JSON。
	Input string
	// Output 工具原始出参 JSON。
	Output string
}

// CatalogOptions 表示工具目录工具配置。
type CatalogOptions struct {
	// Name 工具目录工具名。
	Name string
	// Description 工具目录工具描述。
	Description string
	// Terminal 当前终端。
	Terminal string
	// Infos 当前终端完整工具定义。
	Infos []*Info
	// EnabledInfos 当前终端已启用工具定义。
	EnabledInfos []*Info
	// ModelToolsPerTurn 每轮请求暴露给模型的工具数量。
	ModelToolsPerTurn int
}

// catalogTool 是暴露给模型的内部工具目录查询工具。
type catalogTool struct {
	name              string
	description       string
	terminal          string
	infos             []*Info
	enabledNames      map[string]bool
	modelToolsPerTurn int
}

// NameSet 将工具定义列表转换为按名称索引的集合。
func NameSet(infos []*Info) map[string]bool {
	result := make(map[string]bool, len(infos))
	for _, info := range infos {
		// 空名称工具无法被模型稳定调用，不进入名称集合。
		if info == nil || info.Name == "" {
			continue
		}
		result[info.Name] = true
	}
	return result
}

// HasInfo 判断工具是否存在于定义列表中。
func HasInfo(infos []*Info, name string) bool {
	// 空名称不应匹配任何工具，避免误把无效调用视为合法。
	if name == "" {
		return false
	}
	for _, info := range infos {
		if info != nil && info.Name == name {
			return true
		}
	}
	return false
}

// Title 返回函数工具展示名称。
func Title(info *Info) string {
	if info == nil {
		return ""
	}
	if info.Desc != "" {
		return info.Desc
	}
	return info.Name
}

// InfoByTool 读取工具定义，过滤无效工具。
func InfoByTool(ctx context.Context, value Invokable) (*Info, error) {
	if value == nil {
		return nil, nil
	}
	return value.Info(ctx)
}

// NewCatalogTool 创建工具目录查询工具。
func NewCatalogTool(options CatalogOptions) Invokable {
	return &catalogTool{
		name:              options.Name,
		description:       options.Description,
		terminal:          options.Terminal,
		infos:             append([]*Info(nil), options.Infos...),
		enabledNames:      NameSet(options.EnabledInfos),
		modelToolsPerTurn: options.ModelToolsPerTurn,
	}
}

// ExecuteCall 执行单个函数工具调用并输出稳定记录字段。
func ExecuteCall(ctx context.Context, toolMap map[string]Invokable, infos []*Info, call Call, options ...CallOption) CallResult {
	config := callConfig{}
	for _, option := range options {
		option(&config)
	}
	result := newCallResult(infos, call)
	// 除内置目录工具外，直接调用也必须受本轮已启用工具定义约束。
	if call.Name != config.catalogName && !HasInfo(infos, call.Name) {
		result.Status = "error"
		result.Output = MarshalError(DisabledMessage(call.Name))
		return result.withContent(result.Output)
	}
	item := toolMap[call.Name]
	// 工具定义存在但执行器缺失时返回稳定错误 JSON，便于调用方展示工具卡。
	if item == nil {
		result.Status = "error"
		result.Output = MarshalError(fmt.Sprintf("tool %s is not available", call.Name))
		return result.withContent(result.Output)
	}
	output, err := item.InvokableRun(ctx, call.Arguments)
	// 工具内部错误也转成 JSON 文本返回，保持直接调用与 ADK 调用协议一致。
	if err != nil {
		result.Status = "error"
		result.Output = MarshalError(err.Error())
		return result.withContent(result.Output)
	}
	// 空输出统一表示为成功空对象，避免调用方难以区分“无数据”和“未执行”。
	if output == "" {
		output = "{}"
	}
	result.Output = output
	return result.withContent(output)
}

// WithCatalogName 设置内置工具目录名称。
func WithCatalogName(name string) CallOption {
	return func(config *callConfig) {
		config.catalogName = name
	}
}

// MarshalError 将工具错误转换成稳定 JSON 文本。
func MarshalError(message string) string {
	return einoMiddleware.MarshalToolError(message)
}

// CallOption 表示工具调用执行配置。
type CallOption func(*callConfig)

type callConfig struct {
	catalogName string
}

// DisabledMessage 返回 Agent 工具禁用提示。
func DisabledMessage(name string) string {
	return einoMiddleware.DisabledToolMessage(name)
}

// Info 返回工具目录查询工具定义。
func (t *catalogTool) Info(context.Context) (*Info, error) {
	return &Info{
		Name: t.name,
		Desc: t.description,
	}, nil
}

// InvokableRun 返回当前终端完整工具目录。
func (t *catalogTool) InvokableRun(context.Context, string, ...Option) (string, error) {
	items := make([]map[string]any, 0, len(t.infos))
	enabledCount := 0
	for _, info := range t.infos {
		// 目录工具只展示有名称的工具，避免模型收到不可调用的条目。
		if info == nil || info.Name == "" {
			continue
		}
		enabled := t.enabledNames[info.Name]
		if enabled {
			enabledCount++
		}
		items = append(items, map[string]any{
			"name":        info.Name,
			"description": info.Desc,
			"enabled":     enabled,
		})
	}
	// payload 同时返回完整注册数和启用数，方便模型回答“有哪些工具”和排查禁用原因。
	payload := map[string]any{
		"terminal":                 t.terminal,
		"registered_tool_count":    len(items),
		"enabled_tool_count":       enabledCount,
		"model_tools_per_request":  t.modelToolsPerTurn,
		"catalog_tool_name":        t.name,
		"catalog_tool_description": "当前结果是完整注册工具目录；enabled=false 的工具已禁用，不会作为候选工具，也不能被 Agent 调用。",
		"tools":                    items,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// withContent 写入供模型消费的工具结果文本。
func (r CallResult) withContent(content string) CallResult {
	r.Content = content
	return r
}

// newCallResult 构造函数工具调用的基础记录。
func newCallResult(infos []*Info, call Call) CallResult {
	infoMap := make(map[string]*Info, len(infos))
	for _, info := range infos {
		// 过滤无效工具定义，避免空名称覆盖真实调用名。
		if info == nil || info.Name == "" {
			continue
		}
		infoMap[info.Name] = info
	}
	title := call.Name
	// 工具描述通常来自生成接口说明或后台覆盖提示词，优先作为展示标题。
	if info := infoMap[call.Name]; info != nil && info.Desc != "" {
		title = Title(info)
	}
	return CallResult{
		Type:   "function",
		Name:   call.Name,
		Title:  title,
		Status: "success",
		Input:  call.Arguments,
	}
}
