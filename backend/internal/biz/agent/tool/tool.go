package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/middleware"
)

// Option 表示 Eino 工具调用选项。
type Option = tool.Option

// Info 表示 Eino 工具定义。
type Info = schema.ToolInfo

// Invokable 表示可直接执行的 Eino 工具。
type Invokable = tool.InvokableTool

// Base 表示只提供定义信息的 Eino 工具。
type Base = tool.BaseTool

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
func InfoByTool(ctx context.Context, value Base) (*Info, error) {
	if value == nil {
		return nil, nil
	}
	return value.Info(ctx)
}

// ExecuteCall 执行单个函数工具调用并输出稳定记录字段。
func ExecuteCall(ctx context.Context, toolMap map[string]Base, infos []*Info, call Call, options ...CallOption) CallResult {
	config := callConfig{}
	for _, option := range options {
		option(&config)
	}
	result := newCallResult(infos, call)
	// 直接调用同样受当前已授权工具定义约束。
	if !HasInfo(infos, call.Name) {
		result.Status = "error"
		result.Output = MarshalError(DisabledMessage(ctx, call.Name, config.localize))
		return result.withContent(result.Output)
	}
	item := toolMap[call.Name]
	// 工具定义存在但执行器缺失时返回稳定错误 JSON，便于调用方展示工具卡。
	if item == nil {
		result.Status = "error"
		result.Output = MarshalError(localizeToolMessage(ctx, "system.ai.chat.error.tool_unavailable", call.Name, fmt.Sprintf("Tool %s is not available.", call.Name), config.localize))
		return result.withContent(result.Output)
	}
	var output string
	var summary string
	var err error
	switch actual := item.(type) {
	case tool.EnhancedInvokableTool:
		var resultValue *schema.ToolResult
		resultValue, err = actual.InvokableRun(ctx, &schema.ToolArgument{Text: call.Arguments})
		if err == nil {
			var raw []byte
			raw, err = json.Marshal(resultValue)
			output = string(raw)
			if err == nil {
				summary, err = middleware.ToolResultSummary(resultValue)
			}
		}
	case Invokable:
		output, err = actual.InvokableRun(ctx, call.Arguments)
	default:
		err = fmt.Errorf("tool %s has no invokable implementation", call.Name)
	}
	// 工具内部错误也转成 JSON 文本返回，保持直接调用与 ADK 调用协议一致。
	if err != nil {
		result.Status = "error"
		result.Output = MarshalError(localizeToolMessage(ctx, "system.ai.chat.error.tool_failed", call.Name, fmt.Sprintf("Tool %s failed. Check the service logs.", call.Name), config.localize))
		return result.withContent(result.Output)
	}
	// 空输出统一表示为成功空对象，避免调用方难以区分“无数据”和“未执行”。
	if output == "" {
		output = "{}"
	}
	result.Output = output
	if summary != "" {
		result.Output = summary
	}
	return result.withContent(output)
}

// MarshalError 将工具错误转换成稳定 JSON 文本。
func MarshalError(message string) string {
	return middleware.MarshalToolError(message)
}

// CallOption 表示工具调用执行配置。
type CallOption func(*callConfig)

type callConfig struct {
	localize func(context.Context, string, map[string]any, string) string
}

// WithMessageLocalizer 设置 Agent 工具用户消息的本地化回调。
func WithMessageLocalizer(localize func(context.Context, string, map[string]any, string) string) CallOption {
	return func(config *callConfig) {
		config.localize = localize
	}
}

// DisabledMessage 返回按请求语言本地化的 Agent 工具禁用提示。
func DisabledMessage(ctx context.Context, name string, localize func(context.Context, string, map[string]any, string) string) string {
	fallback := "The requested agent tool is disabled and cannot be called."
	if localize == nil {
		return fallback
	}
	return localize(ctx, "base.ai.tool.disabled", map[string]any{"Name": name}, fallback)
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

// localizeToolMessage 按当前请求语言生成工具调用错误文案。
func localizeToolMessage(ctx context.Context, key, name, fallback string, localize func(context.Context, string, map[string]any, string) string) string {
	if localize == nil {
		return fallback
	}
	return localize(ctx, key, map[string]any{"Name": name}, fallback)
}
