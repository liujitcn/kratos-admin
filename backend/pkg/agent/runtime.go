package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/adk"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	einoMessage "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

const (
	maxModelToolsPerRequest    = 6
	minToolMatchScore          = 4
	maxShortFollowUpLength     = 8
	maxToolQueryAttachmentText = 800
	maxHistoryToolText         = 2000
	agentToolCatalogName       = "internal_agent_tool_catalog"
	aiWebSearchToolName        = "base_v1_ai_search_service_search_ai_web"
)

const fallbackAIInstruction = "You are a general-purpose AI assistant. Answer naturally, helpfully, and accurately. Use internal tools for private system data, do not fabricate results, and reply in the user's current language."

// Runtime 封装流式 AI 助手运行时。
//
// Runtime 只负责把业务层准备好的输入组装为 Eino 消息并交给模型执行，不直接处理数据库、
// OSS、鉴权或前端协议。这样 AI 助手链路可以把“业务准备”和“模型运行”分开维护。
type Runtime struct {
	toolsMu         sync.RWMutex
	client          *model.AssistantClient
	adminTools      []tool.Invokable
	appTools        []tool.Invokable
	toolGate        ToolAccessChecker
	localizeMessage MessageLocalizer
}

// newRuntime 创建 AI 助手运行时。
func newRuntime(
	client *model.AssistantClient,
	checker ToolAccessChecker,
	adminTools []Tool,
	appTools []Tool,
	localizeMessage MessageLocalizer,
) *Runtime {
	return &Runtime{
		client:          client,
		adminTools:      append([]tool.Invokable(nil), adminTools...),
		appTools:        append([]tool.Invokable(nil), appTools...),
		toolGate:        checker,
		localizeMessage: localizeMessage,
	}
}

// RegisterTool 将一个工具追加到指定终端；重复名称按首次注册的工具执行。
func (r *Runtime) RegisterTool(terminal string, value Tool) error {
	return r.RegisterTools(terminal, value)
}

// RegisterTools 将工具追加到指定终端，支持运行时扩展工具集合。
func (r *Runtime) RegisterTools(terminal string, values ...Tool) error {
	if r == nil {
		return errors.New("AI assistant runtime is not initialized")
	}
	if terminal != "" && terminal != "admin" && terminal != "app" {
		return fmt.Errorf("unsupported AI terminal: %s", terminal)
	}
	if len(values) == 0 {
		return nil
	}
	r.toolsMu.Lock()
	defer r.toolsMu.Unlock()
	if terminal == "app" {
		r.appTools = append(r.appTools, values...)
		return nil
	}
	r.adminTools = append(r.adminTools, values...)
	return nil
}

// InvokeTool 按工具名直接调用当前终端已启用的 Agent 工具。
func (r *Runtime) InvokeTool(ctx context.Context, terminal string, name string, arguments string) (*ToolInvokeResult, error) {
	if r == nil {
		return nil, errors.New("AI assistant runtime is not initialized")
	}
	input := RuntimeInput{Terminal: terminal, Content: name}
	infos := r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input))
	result := tool.ExecuteCall(ctx, r.toolMap(ctx, input), infos, tool.Call{
		ID:        "direct_" + name,
		Name:      name,
		Arguments: arguments,
	}, tool.WithCatalogName(agentToolCatalogName), tool.WithMessageLocalizer(r.localizeMessage))
	usage := toolUsageFromCallResult(result)
	if usage.Status != "success" {
		return &ToolInvokeResult{Output: result.Content, Usage: usage}, errors.New(result.Content)
	}
	return &ToolInvokeResult{Output: result.Content, Usage: usage}, nil
}

// Enabled 判断 AI 助手运行时是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.AgenticModel != nil
}

// Model 返回 AI 助手当前使用的模型名称。
func (r *Runtime) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.client.Name()
}

// EnabledToolNames 返回当前终端实际启用的 Agent 工具名集合。
func (r *Runtime) EnabledToolNames(ctx context.Context, terminal string) map[string]bool {
	if r == nil {
		return map[string]bool{}
	}
	input := RuntimeInput{Terminal: terminal}
	return tool.NameSet(r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input)))
}

// Run 使用生成式模式运行助手。
//
// 该方法用于普通 RPC 或非流式调用：先构建带历史上下文的 Eino 消息列表，
// 再等待模型完整回复。
func (r *Runtime) Run(ctx context.Context, input RuntimeInput) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai ai client is not configured")
	}
	output, token, tools, err := r.runGenerate(ctx, input, r.buildMessages(ctx, input))
	if err != nil {
		return nil, err
	}
	return r.buildResponse(output, token, tools), nil
}

// RunStream 使用流式模式运行助手。
//
// 该方法用于管理端 direct SSE：模型返回文本片段时会透传给 onDelta，
// 最终仍返回完整回复供业务层落库。
func (r *Runtime) RunStream(ctx context.Context, input RuntimeInput, onDelta func(string)) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai ai client is not configured")
	}
	messages := r.buildMessages(ctx, input)
	toolInfos, internalToolMatched := r.toolInfos(ctx, input)
	if disabledCall := r.disabledToolCall(ctx, input, toolInfos); disabledCall != nil {
		if onDelta != nil {
			onDelta(disabledCall.Content)
		}
		return r.buildResponse(einoMessage.AIText(disabledCall.Content), TokenUsage{}, []ToolUsage{disabledCall.Usage}), nil
	}
	result, recorder, err := r.runADK(ctx, input, messages, toolInfos, internalToolMatched, true, onDelta)
	if err != nil {
		return nil, err
	}
	return r.buildResponse(result.Message, tokenFromCallback(result.Token), toolsFromRecorder(recorder)), nil
}

// ToolInvokeResult 表示直接调用 Agent 工具后的结果。
type ToolInvokeResult struct {
	// Output 工具原始输出 JSON。
	Output string
	// Usage 本次工具调用的后台展示记录。
	Usage ToolUsage
}

// disabledToolCall 保存禁用工具命中后的回复与展示记录。
type disabledToolCall struct {
	// Content 面向用户展示的禁用原因。
	Content string
	// Usage 禁用工具对应的错误工具卡记录。
	Usage ToolUsage
}

// runGenerate 执行非流式模型调用，并在需要时继续执行工具回填。
func (r *Runtime) runGenerate(ctx context.Context, input RuntimeInput, messages []*einoMessage.AgenticMessage) (*einoMessage.AgenticMessage, TokenUsage, []ToolUsage, error) {
	toolInfos, internalToolMatched := r.toolInfos(ctx, input)
	if disabledCall := r.disabledToolCall(ctx, input, toolInfos); disabledCall != nil {
		return einoMessage.AIText(disabledCall.Content), TokenUsage{}, []ToolUsage{disabledCall.Usage}, nil
	}
	result, recorder, err := r.runADK(ctx, input, messages, toolInfos, internalToolMatched, false, nil)
	if err != nil {
		return nil, TokenUsage{}, nil, err
	}
	return result.Message, tokenFromCallback(result.Token), toolsFromRecorder(recorder), nil
}

// disabledToolCall 在本轮命中已禁用工具时构造明确错误回复。
func (r *Runtime) disabledToolCall(ctx context.Context, input RuntimeInput, enabledCandidates []*tool.Info) *disabledToolCall {
	registeredInfos := r.allToolInfos(ctx, input)
	enabledInfos := r.enabledToolInfos(ctx, input, registeredInfos)
	enabledNames := tool.NameSet(enabledInfos)
	disabledInfos := make([]*tool.Info, 0, len(registeredInfos))
	for _, info := range registeredInfos {
		if info == nil || info.Name == "" || info.Name == agentToolCatalogName {
			continue
		}
		if enabledNames[info.Name] {
			continue
		}
		disabledInfos = append(disabledInfos, info)
	}
	if len(disabledInfos) == 0 {
		return nil
	}
	if info := selectExplicitToolInfo(input, disabledInfos); info != nil {
		return r.newDisabledToolCall(ctx, info)
	}

	terms := toolQueryTerms(input)
	disabledMatches := scoredToolInfos(disabledInfos, terms)
	enabledMatches := scoredToolInfos(enabledCandidates, terms)
	if len(disabledMatches) > 0 && shouldReturnDisabledToolCall(disabledMatches[0], enabledMatches) {
		return r.newDisabledToolCall(ctx, disabledMatches[0].info)
	}

	if len(disabledMatches) > 0 || len(enabledMatches) > 0 || !isHistoryToolFollowUp(input) {
		return nil
	}
	matchedInfos := selectHistoryToolInfos(input, disabledInfos)
	if len(matchedInfos) == 0 {
		return nil
	}
	return r.newDisabledToolCall(ctx, matchedInfos[0])
}

// runADK 通过 Eino ADK ChatModelAgent / Runner 执行模型和工具循环。
func (r *Runtime) runADK(
	ctx context.Context,
	input RuntimeInput,
	messages []*einoMessage.AgenticMessage,
	toolInfos []*tool.Info,
	internalToolMatched bool,
	stream bool,
	onDelta func(string),
) (*adk.Result, *callback.Recorder, error) {
	recorder := &callback.Recorder{}
	// Responses 协议才支持服务端工具（联网搜索等），聊天补全协议使用同名 function 工具替代。
	serverTools := r.client != nil && r.client.SupportsResponsesServerTools() && !internalToolMatched
	description := r.localizePrompt(ctx, "base.ai.prompt.assistant_description", nil, "Admin AI assistant for conversation, internal tools, and web search.")
	runner := adk.NewRunner(adk.Config{
		Model:       r.client.AgenticModel,
		Name:        "admin_ai",
		Description: description,
		ServerTools: serverTools,
	})
	result, err := runner.Run(ctx, adk.Request{
		Messages:        append([]*einoMessage.AgenticMessage(nil), messages...),
		Tools:           r.runnerTools(ctx, input),
		ToolInfos:       toolInfos,
		Recorder:        recorder,
		LocalizeMessage: r.localizeMessage,
		Stream:          stream,
		OnDelta:         onDelta,
	})
	return result, recorder, err
}

// runnerTools 收集当前终端可由 ADK 执行的工具。
func (r *Runtime) runnerTools(ctx context.Context, input RuntimeInput) []tool.Base {
	toolMap := r.toolMap(ctx, input)
	tools := make([]tool.Base, 0, len(toolMap))
	for _, item := range toolMap {
		if item == nil {
			continue
		}
		tools = append(tools, item)
	}
	return tools
}

// toolInfos 收集可传给模型的工具定义。
func (r *Runtime) toolInfos(ctx context.Context, input RuntimeInput) ([]*tool.Info, bool) {
	registeredInfos := r.allToolInfos(ctx, input)
	infos := r.enabledToolInfos(ctx, input, registeredInfos)
	catalogTool := r.newAgentToolCatalogTool(ctx, input.Terminal, registeredInfos, infos, maxModelToolsPerRequest)
	catalogInfo, err := catalogTool.Info(ctx)
	if err == nil && catalogInfo != nil {
		infos = append(infos, catalogInfo)
	}
	return selectToolInfos(input, infos)
}

// toolMap 按工具名构造本地执行索引。
func (r *Runtime) toolMap(ctx context.Context, input RuntimeInput) map[string]tool.Invokable {
	registeredInfos := r.allToolInfos(ctx, input)
	enabledInfos := r.enabledToolInfos(ctx, input, registeredInfos)
	enabledNames := tool.NameSet(enabledInfos)
	tools := r.terminalTools(input.Terminal)
	result := make(map[string]tool.Invokable, len(enabledNames)+1)
	var err error
	if r == nil {
		return result
	}
	for _, item := range tools {
		if item == nil {
			continue
		}
		var info *tool.Info
		info, err = item.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		if _, ok := result[info.Name]; ok {
			continue
		}
		if !enabledNames[info.Name] {
			continue
		}
		result[info.Name] = item
	}
	catalogTool := r.newAgentToolCatalogTool(ctx, input.Terminal, registeredInfos, enabledInfos, maxModelToolsPerRequest)
	var catalogInfo *tool.Info
	catalogInfo, err = catalogTool.Info(ctx)
	if err == nil && catalogInfo != nil && catalogInfo.Name != "" {
		result[catalogInfo.Name] = catalogTool
	}
	return result
}

// buildMessages 构建当前轮次发送给 Eino 模型的消息列表。
func (r *Runtime) buildMessages(ctx context.Context, input RuntimeInput) []*einoMessage.AgenticMessage {
	messages := []*einoMessage.AgenticMessage{einoMessage.SystemText(r.resolvePrompt(ctx, input))}
	enabledNames := tool.NameSet(r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input)))
	for _, item := range input.History {
		if item.Content == "" {
			continue
		}
		historyMessage := einoMessage.TextByRole(item.Role, item.Content)
		if historyMessage == nil {
			continue
		}
		messages = append(messages, historyMessage)
		toolContext := r.buildHistoryToolContext(ctx, item.Tools, enabledNames)
		if toolContext != "" {
			messages = append(messages, einoMessage.SystemText(toolContext))
		}
	}
	messages = append(messages, r.buildUserMessage(ctx, input))
	return messages
}

// enabledToolInfos 按 base_api.agent_status 过滤当前终端可暴露给 Agent 的工具。
func (r *Runtime) enabledToolInfos(ctx context.Context, input RuntimeInput, infos []*tool.Info) []*tool.Info {
	if len(infos) == 0 || r == nil || r.toolGate == nil {
		return infos
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == "" {
			continue
		}
		names = append(names, info.Name)
	}
	toolConfigs, err := r.toolGate.ToolConfigs(ctx, input.Terminal, names)
	if err != nil {
		return nil
	}
	result := make([]*tool.Info, 0, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == "" {
			continue
		}
		config := toolConfigs[info.Name]
		if !config.Enabled {
			continue
		}
		result = append(result, withToolInfoConfig(info, config))
	}
	return result
}

// allToolInfos 收集当前终端完整工具定义，不做本轮相关性筛选。
func (r *Runtime) allToolInfos(ctx context.Context, input RuntimeInput) []*tool.Info {
	tools := r.terminalTools(input.Terminal)
	if len(tools) == 0 {
		return nil
	}
	infos := make([]*tool.Info, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))
	for _, item := range tools {
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		if _, ok := seen[info.Name]; ok {
			continue
		}
		seen[info.Name] = struct{}{}
		infos = append(infos, info)
	}
	toolConfigs := r.toolInfoConfigs(ctx, input, infos)
	if len(toolConfigs) == 0 {
		return infos
	}
	for index, info := range infos {
		if info == nil {
			continue
		}
		infos[index] = withToolInfoConfig(info, toolConfigs[info.Name])
	}
	return infos
}

// toolInfoConfigs 查询当前终端完整工具配置。
func (r *Runtime) toolInfoConfigs(ctx context.Context, input RuntimeInput, infos []*tool.Info) map[string]ToolConfig {
	result := make(map[string]ToolConfig, len(infos))
	if len(infos) == 0 || r == nil || r.toolGate == nil {
		for _, info := range infos {
			if info == nil || info.Name == "" {
				continue
			}
			result[info.Name] = ToolConfig{Enabled: true}
		}
		return result
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == "" {
			continue
		}
		names = append(names, info.Name)
	}
	toolConfigs, err := r.toolGate.ToolConfigs(ctx, input.Terminal, names)
	if err != nil {
		return result
	}
	return toolConfigs
}

// terminalTools 按终端选择当前智能体可用工具。
func (r *Runtime) terminalTools(terminal string) []tool.Invokable {
	if r == nil {
		return nil
	}
	r.toolsMu.RLock()
	defer r.toolsMu.RUnlock()
	if terminal == "app" {
		return append([]tool.Invokable(nil), r.appTools...)
	}
	return append([]tool.Invokable(nil), r.adminTools...)
}

// resolvePrompt 渲染 AI 助手提示词。
func (r *Runtime) resolvePrompt(ctx context.Context, input RuntimeInput) string {
	now := time.Now().In(time.Local)
	lines := []string{
		r.localizePrompt(ctx, "base.ai.prompt.instruction", nil, fallbackAIInstruction),
		r.localizePrompt(ctx, "base.ai.prompt.tool_routing_rules", nil,
			"Use internal tools for private project data. When an internal tool matches this request, answer only with internal tools and do not use web search. Follow tool descriptions and parameter defaults; do not add or remove query conditions without instruction. For a clarified follow-up, reuse the latest relevant tool and preserve other conditions, changing only the clarified condition. Ask if the target or condition is ambiguous."),
		"",
		r.localizePrompt(ctx, "base.ai.prompt.current_session", nil, "Current session:"),
		r.localizePrompt(ctx, "base.ai.prompt.terminal", map[string]any{"Value": input.Terminal}, "- Terminal: {{.Value}}"),
		r.localizePrompt(ctx, "base.ai.prompt.user", map[string]any{"Value": input.UserName}, "- User: {{.Value}}"),
		r.localizePrompt(ctx, "base.ai.prompt.title", map[string]any{"Value": input.SessionTitle}, "- Title: {{.Value}}"),
		r.localizePrompt(ctx, "base.ai.prompt.summary", map[string]any{"Value": input.Summary}, "- Summary: {{.Value}}"),
		r.localizePrompt(ctx, "base.ai.prompt.business_date", map[string]any{"Value": now.Format("2006-01-02")}, "- Business date: {{.Value}}"),
		r.localizePrompt(ctx, "base.ai.prompt.business_timezone", map[string]any{"Location": now.Location(), "Offset": now.Format("-07:00")}, "- Business time zone: {{.Location}} (UTC{{.Offset}})"),
	}
	if len(input.Attachments) > 0 {
		lines = append(lines, "", r.localizePrompt(ctx, "base.ai.prompt.attachment_notice", nil, "Attachments are included in the user message; refer to them when relevant."))
	}
	return strings.Join(lines, "\n")
}

// localizePrompt 按请求语言渲染 Agent 系统提示词片段。
func (r *Runtime) localizePrompt(ctx context.Context, key string, args map[string]any, fallback string) string {
	if r.localizeMessage != nil {
		return r.localizeMessage(ctx, key, args, fallback)
	}
	for name, value := range args {
		fallback = strings.ReplaceAll(fallback, "{{."+name+"}}", fmt.Sprint(value))
	}
	return fallback
}

// buildUserMessage 构建当前轮次发送给模型的用户消息。
func (r *Runtime) buildUserMessage(ctx context.Context, input RuntimeInput) *einoMessage.AgenticMessage {
	content := input.Content
	attachmentLines := make([]string, 0, len(input.Attachments)*2)
	images := make([]einoMessage.ImageData, 0, len(input.Attachments))

	for _, item := range input.Attachments {
		attachmentContent := item.Content
		name := item.Name
		if name == "" {
			name = r.localizePrompt(ctx, "base.ai.prompt.attachment_unnamed", nil, "Unnamed attachment")
		}
		cleanMIMEType := normalizeRuntimeMIMEType(item)

		// 图片附件走多模态输入，避免把本地 admin 地址误当成公网图片 URL。
		if isRuntimeImageMIME(cleanMIMEType) {
			// 图片必须具备原始字节，才能作为多模态视觉输入传给模型。
			if len(item.Bytes) == 0 {
				continue
			}
			attachmentLines = append(attachmentLines, r.localizePrompt(ctx, "base.ai.prompt.image_attachment", map[string]any{"Name": name}, "Image attachment {{.Name}} is provided as visual input."))
			images = append(images, einoMessage.ImageData{Bytes: item.Bytes, MIMEType: cleanMIMEType})
			continue
		}

		// 文本类附件优先拼入正文，保证模型能直接读取附件内容。
		if attachmentContent != "" {
			attachmentLines = append(attachmentLines, r.localizePrompt(ctx, "base.ai.prompt.text_attachment", map[string]any{"Name": name, "Content": attachmentContent}, "Attachment {{.Name}} content:\n{{.Content}}"))
			continue
		}

		// 没有可读内容的附件仍保留文件元信息，模型至少能知道用户提供了什么文件。
		attachmentLines = append(attachmentLines, r.buildAttachmentDetailLine(ctx, name, item))
	}

	if len(attachmentLines) == 0 && len(images) == 0 {
		return einoMessage.UserText(content)
	}
	return buildUserMessageParts(content, attachmentLines, images, r.localizePrompt(ctx, "base.ai.prompt.attachment_message_intro", nil, "The current message includes the following attachments. Refer to them when relevant:"))
}

// buildAttachmentDetailLine 构造模型无法直接读取附件内容时的本地化元信息说明。
func (r *Runtime) buildAttachmentDetailLine(ctx context.Context, name string, item Attachment) string {
	details := []string{r.localizePrompt(ctx, "base.ai.prompt.attachment_name", map[string]any{"Name": name}, "Attachment {{.Name}}")}
	if item.MIMEType != "" {
		details = append(details, r.localizePrompt(ctx, "base.ai.prompt.attachment_type", map[string]any{"Value": item.MIMEType}, "Type: {{.Value}}"))
	}
	if item.Size > 0 {
		details = append(details, r.localizePrompt(ctx, "base.ai.prompt.attachment_size", map[string]any{"Value": item.Size}, "Size: {{.Value}} bytes"))
	}
	if item.URL != "" {
		details = append(details, r.localizePrompt(ctx, "base.ai.prompt.attachment_url", map[string]any{"Value": item.URL}, "URL: {{.Value}}"))
	}
	return strings.Join(details, ", ")
}

// buildResponse 将 Eino 消息收敛为业务层统一回复结构。
func (r *Runtime) buildResponse(message *einoMessage.AgenticMessage, token TokenUsage, tools []ToolUsage) *Response {
	return &Response{
		Content:        einoMessage.Text(message),
		Token:          token,
		Tools:          normalizeToolUsages(tools),
		Source:         "llm",
		Model:          r.Model(),
		Fallback:       false,
		FallbackReason: "",
	}
}

// newAgentToolCatalogTool 创建工具目录查询工具。
func (r *Runtime) newAgentToolCatalogTool(ctx context.Context, terminal string, infos []*tool.Info, enabledInfos []*tool.Info, modelToolsPerTurn int) tool.Invokable {
	return tool.NewCatalogTool(tool.CatalogOptions{
		Name: agentToolCatalogName,
		Description: r.localizePrompt(ctx, "base.ai.prompt.catalog_tool_description", nil,
			"Query the complete internal Agent Tool catalog for the current terminal, including tool counts, names, and descriptions. Use when the user asks which tools or APIs are available."),
		Terminal:          terminal,
		Infos:             infos,
		EnabledInfos:      enabledInfos,
		ModelToolsPerTurn: modelToolsPerTurn,
	})
}

// newDisabledToolCall 构造禁用工具对应的错误回复与工具卡。
func (r *Runtime) newDisabledToolCall(ctx context.Context, info *tool.Info) *disabledToolCall {
	content := tool.DisabledMessage(ctx, info.Name, r.localizeMessage)
	return &disabledToolCall{
		Content: content,
		Usage: ToolUsage{
			Type:   "function",
			Name:   info.Name,
			Title:  tool.Title(info),
			Status: "error",
			Output: tool.MarshalError(content),
		},
	}
}

// shouldReturnDisabledToolCall 判断禁用工具是否比当前启用候选更符合本轮问题。
func shouldReturnDisabledToolCall(disabled scoredToolInfo, enabledMatches []scoredToolInfo) bool {
	if disabled.info == nil || disabled.score <= 0 {
		return false
	}
	if len(enabledMatches) == 0 {
		return true
	}
	return disabled.score >= enabledMatches[0].score
}

// selectExplicitToolInfo 按用户本轮直接写出的工具名匹配工具。
func selectExplicitToolInfo(input RuntimeInput, infos []*tool.Info) *tool.Info {
	text := strings.ToLower(toolQueryText(input))
	if text == "" {
		return nil
	}
	for _, info := range infos {
		if info == nil || info.Name == "" {
			continue
		}
		if strings.Contains(text, strings.ToLower(info.Name)) {
			return info
		}
	}
	return nil
}

// isHistoryToolFollowUp 判断本轮是否像是在延续上一轮工具查询。
func isHistoryToolFollowUp(input RuntimeInput) bool {
	if len(input.Attachments) > 0 || !hasHistoryToolUsage(input.History) {
		return false
	}
	text := strings.TrimSpace(input.Content)
	return hasFollowUpReference(text) || hasPaginationReference(text)
}

// hasHistoryToolUsage 判断历史上下文中是否存在可延续的工具调用。
func hasHistoryToolUsage(history []Message) bool {
	for _, item := range history {
		if len(item.Tools) > 0 {
			return true
		}
	}
	return false
}

// hasFollowUpReference 判断文本是否包含泛化的续查引用。
func hasFollowUpReference(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if hasFollowUpCue(text) {
		return true
	}
	return len([]rune(text)) <= maxShortFollowUpLength
}

// hasFollowUpCue 判断文本是否包含明确的续查表达。
func hasFollowUpCue(text string) bool {
	lowerText := strings.ToLower(text)
	for _, cue := range []string{"继续", "更多", "还有", "再", "下一", "上一", "刷新", "换一批", "next", "more"} {
		if strings.Contains(lowerText, cue) {
			return true
		}
	}
	return false
}

// hasPaginationReference 判断文本是否包含泛化的分页引用。
func hasPaginationReference(text string) bool {
	lowerText := strings.ToLower(text)
	if !strings.Contains(lowerText, "page") && !strings.Contains(text, "页") {
		return false
	}
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// withToolInfoConfig 使用数据库中的工具配置覆盖生成工具描述。
func withToolInfoConfig(info *tool.Info, config ToolConfig) *tool.Info {
	if info == nil || len(config.Prompts) == 0 {
		return info
	}
	copiedInfo := *info
	copiedInfo.Desc = toolPromptsDescription(config.Prompts)
	return &copiedInfo
}

// toolPromptsDescription 将多条工具提示词合并为模型可读的工具描述。
func toolPromptsDescription(prompts []string) string {
	values := make([]string, 0, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		values = append(values, item)
	}
	return strings.Join(values, "\n")
}

// selectToolInfos 从当前终端完整工具池中挑选本轮请求相关工具。
func selectToolInfos(input RuntimeInput, infos []*tool.Info) ([]*tool.Info, bool) {
	terms := toolQueryTerms(input)
	internalInfos := make([]*tool.Info, 0, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == aiWebSearchToolName || info.Name == agentToolCatalogName {
			continue
		}
		internalInfos = append(internalInfos, info)
	}
	if result := selectScoredToolInfos(internalInfos, terms); len(result) > 0 {
		return result, true
	}
	if isHistoryToolFollowUp(input) {
		historyInfos := selectHistoryToolInfos(input, internalInfos)
		if len(historyInfos) > 0 {
			queryMatchedHistoryTool := hasToolQueryTermMatch(historyInfos, terms)
			if hasFollowUpCue(input.Content) || hasPaginationReference(input.Content) || queryMatchedHistoryTool {
				return historyInfos, true
			}
			return selectHistoryToolInfosWithSearch(historyInfos, infos), false
		}
	}
	if len(infos) <= maxModelToolsPerRequest {
		return infos, false
	}
	result := selectScoredToolInfos(infos, terms)
	if len(result) > 0 {
		return result, false
	}
	if !isHistoryToolFollowUp(input) {
		// 内部工具均不匹配时保底暴露联网搜索工具，避免模型无法处理公开实时信息类问题。
		return selectFallbackToolInfos(infos), false
	}
	return selectHistoryToolInfosWithSearch(selectHistoryToolInfos(input, internalInfos), infos), false
}

// hasToolQueryTermMatch 判断短追问是否命中历史工具描述中的任意有效词项。
func hasToolQueryTermMatch(infos []*tool.Info, terms []string) bool {
	for _, info := range infos {
		if info == nil {
			continue
		}
		text := strings.ToLower(info.Name + " " + info.Desc)
		for _, term := range terms {
			if len([]rune(term)) >= 2 && strings.Contains(text, term) {
				return true
			}
		}
	}
	return false
}

// selectFallbackToolInfos 内部工具不匹配时保底返回联网搜索工具。
func selectFallbackToolInfos(infos []*tool.Info) []*tool.Info {
	for _, info := range infos {
		if info != nil && info.Name == aiWebSearchToolName {
			return []*tool.Info{info}
		}
	}
	return nil
}

// selectScoredToolInfos 按关键词从完整工具池中挑选本轮可暴露的工具。
func selectScoredToolInfos(infos []*tool.Info, terms []string) []*tool.Info {
	scoredTools := scoredToolInfos(infos, terms)
	if len(scoredTools) == 0 {
		return nil
	}
	limit := maxModelToolsPerRequest
	if len(scoredTools) < limit {
		limit = len(scoredTools)
	}
	result := make([]*tool.Info, 0, limit)
	for _, item := range scoredTools[:limit] {
		result = append(result, item.info)
	}
	return result
}

// scoredToolInfos 按关键词为工具打分并按相关性排序。
func scoredToolInfos(infos []*tool.Info, terms []string) []scoredToolInfo {
	if len(terms) == 0 {
		return nil
	}
	scoredTools := make([]scoredToolInfo, 0, len(infos))
	for index, info := range infos {
		score := scoreToolInfo(info, terms)
		if score < minToolMatchScore {
			continue
		}
		scoredTools = append(scoredTools, scoredToolInfo{
			info:  info,
			score: score,
			index: index,
		})
	}
	if len(scoredTools) == 0 {
		return nil
	}
	sort.SliceStable(scoredTools, func(i, j int) bool {
		if scoredTools[i].score == scoredTools[j].score {
			return scoredTools[i].index < scoredTools[j].index
		}
		return scoredTools[i].score > scoredTools[j].score
	})
	return scoredTools
}

// selectHistoryToolInfos 在本轮未命中工具时，延续最近历史工具作为短追问候选。
func selectHistoryToolInfos(input RuntimeInput, infos []*tool.Info) []*tool.Info {
	infoMap := make(map[string]*tool.Info, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == "" || info.Name == agentToolCatalogName {
			continue
		}
		infoMap[info.Name] = info
	}
	if len(infoMap) == 0 {
		return nil
	}
	result := make([]*tool.Info, 0, maxModelToolsPerRequest)
	seen := make(map[string]struct{}, maxModelToolsPerRequest)
	for index := len(input.History) - 1; index >= 0; index-- {
		for _, item := range input.History[index].Tools {
			info := infoMap[item.Name]
			if info == nil {
				continue
			}
			if _, ok := seen[info.Name]; ok {
				continue
			}
			seen[info.Name] = struct{}{}
			result = append(result, info)
			if len(result) >= maxModelToolsPerRequest {
				return result
			}
		}
	}
	return result
}

// selectHistoryToolInfosWithSearch 为无法仅凭短追问确定意图的历史工具保留联网搜索候选。
func selectHistoryToolInfosWithSearch(historyInfos, infos []*tool.Info) []*tool.Info {
	searchInfos := selectFallbackToolInfos(infos)
	if len(searchInfos) == 0 {
		return historyInfos
	}
	if len(historyInfos) >= maxModelToolsPerRequest {
		historyInfos = historyInfos[:maxModelToolsPerRequest-1]
	}
	return append(historyInfos, searchInfos...)
}

type scoredToolInfo struct {
	info  *tool.Info
	score int
	index int
}

// toolQueryTerms 只提取本轮问题用于匹配工具名称和描述的关键词。
func toolQueryTerms(input RuntimeInput) []string {
	return splitUniqueToolQueryTerms(strings.ToLower(toolQueryText(input)))
}

// toolQueryText 汇总本轮内容用于工具匹配。
func toolQueryText(input RuntimeInput) string {
	textParts := []string{input.Content}
	for _, item := range input.Attachments {
		textParts = append(textParts, item.Name)
		textParts = append(textParts, limitToolQueryAttachmentText(item.Content))
	}
	return strings.Join(textParts, " ")
}

// limitToolQueryAttachmentText 按字符数限制附件内容，避免截断中文或 emoji 的 UTF-8 字节。
func limitToolQueryAttachmentText(content string) string {
	runes := []rune(content)
	if len(runes) <= maxToolQueryAttachmentText {
		return content
	}
	return string(runes[:maxToolQueryAttachmentText])
}

// splitUniqueToolQueryTerms 将原始文本切词并去重。
func splitUniqueToolQueryTerms(raw string) []string {
	seen := map[string]struct{}{}
	terms := make([]string, 0, 16)
	for _, term := range splitToolQueryTerms(raw) {
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

// splitToolQueryTerms 同时兼容中文短语和英文/数字工具名。
func splitToolQueryTerms(raw string) []string {
	values := make([]string, 0, 16)
	var word strings.Builder
	var hans strings.Builder
	flushWord := func() {
		if word.Len() == 0 {
			return
		}
		if value := word.String(); len([]rune(value)) > 1 {
			values = append(values, value)
		}
		word.Reset()
	}
	flushHans := func() {
		if hans.Len() == 0 {
			return
		}
		values = append(values, chineseNgrams(hans.String())...)
		hans.Reset()
	}
	for _, r := range raw {
		if unicode.Is(unicode.Han, r) {
			flushWord()
			hans.WriteRune(r)
			continue
		}
		flushHans()
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
			continue
		}
		flushWord()
	}
	flushWord()
	flushHans()
	return values
}

// chineseNgrams 把连续中文切成短语，提升“商品列表”对“查询商品信息列表”的命中率。
func chineseNgrams(value string) []string {
	runes := []rune(value)
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= 2 {
		return []string{value}
	}
	values := make([]string, 0, len(runes)*2)
	if len(runes) <= 6 {
		values = append(values, value)
	}
	for size := 2; size <= 4; size++ {
		if len(runes) < size {
			break
		}
		for i := 0; i+size <= len(runes); i++ {
			values = append(values, string(runes[i:i+size]))
		}
	}
	return values
}

// scoreToolInfo 计算工具与用户问题的相关性。
func scoreToolInfo(info *tool.Info, terms []string) int {
	if info == nil {
		return 0
	}
	text := strings.ToLower(info.Name + " " + info.Desc)
	score := 0
	specificScore := 0
	shortPositions := make([]int, 0, 4)
	for _, term := range terms {
		if !strings.Contains(text, term) {
			continue
		}
		termScore := len([]rune(term))
		score += termScore
		if strings.Contains(info.Name, term) {
			score += 3
		}
		if termScore >= 3 {
			specificScore += termScore
			continue
		}
		if termScore == 2 {
			for start := 0; start < len(text); {
				position := strings.Index(text[start:], term)
				if position < 0 {
					break
				}
				position += start
				shortPositions = append(shortPositions, utf8.RuneCountInString(text[:position]))
				start = position + len(term)
			}
		}
	}
	if specificScore == 0 && !hasNearbyShortToolTerms(shortPositions) {
		return 0
	}
	return score
}

// hasNearbyShortToolTerms 判断多个短词是否在工具描述中足够接近。
func hasNearbyShortToolTerms(positions []int) bool {
	for i, left := range positions {
		if left < 0 {
			continue
		}
		for _, right := range positions[i+1:] {
			if right < 0 {
				continue
			}
			distance := left - right
			if distance < 0 {
				distance = -distance
			}
			if distance <= 18 {
				return true
			}
		}
	}
	return false
}

// buildHistoryToolContext 构造当前仍启用工具的历史调用上下文。
func (r *Runtime) buildHistoryToolContext(ctx context.Context, tools []ToolUsage, enabledNames map[string]bool) string {
	if len(tools) == 0 {
		return ""
	}
	lines := make([]string, 0, len(tools)*5+1)
	lines = append(lines, r.localizePrompt(ctx, "base.ai.prompt.history_tool_context", nil,
		"Previous internal tool context is for understanding follow-up questions only. Never show it directly to the user:"))
	for _, item := range tools {
		if item.Name == "" || !enabledNames[item.Name] {
			continue
		}
		lines = append(lines, r.localizePrompt(ctx, "base.ai.prompt.history_tool_name", map[string]any{"Name": item.Name}, "- Tool name: {{.Name}}"))
		if item.Title != "" {
			lines = append(lines, r.localizePrompt(ctx, "base.ai.prompt.history_tool_title", map[string]any{"Title": item.Title}, "  Tool title: {{.Title}}"))
		}
		if item.Input != "" {
			lines = append(lines, r.localizePrompt(ctx, "base.ai.prompt.history_tool_input", map[string]any{"Input": limitHistoryToolText(item.Input)}, "  Input: {{.Input}}"))
		}
		if item.Output != "" {
			lines = append(lines, r.localizePrompt(ctx, "base.ai.prompt.history_tool_output", map[string]any{"Output": limitHistoryToolText(item.Output)}, "  Output: {{.Output}}"))
		}
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// limitHistoryToolText 限制历史工具上下文长度。
func limitHistoryToolText(content string) string {
	runes := []rune(content)
	if len(runes) <= maxHistoryToolText {
		return content
	}
	return string(runes[:maxHistoryToolText]) + "..."
}

// normalizeRuntimeMIMEType 规范化运行时 MIME 类型。
func normalizeRuntimeMIMEType(item Attachment) string {
	cleanMIMEType := strings.ToLower(strings.SplitN(item.MIMEType, ";", 2)[0])
	// image/jpg 不是标准 MIME，统一转成模型更常见支持的 image/jpeg。
	if cleanMIMEType == "image/jpg" {
		return "image/jpeg"
	}
	// 已有 MIME 时不再从文件名推断，避免后缀与真实内容冲突。
	if cleanMIMEType != "" {
		return cleanMIMEType
	}

	// MIME 缺失时仅对图片后缀兜底推断，其他文件保持普通附件元信息。
	switch strings.ToLower(pathExt(item.Name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

// isRuntimeImageMIME 判断 MIME 类型是否可作为图片输入。
func isRuntimeImageMIME(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

// buildAttachmentDetailLine 构造模型无法直接读取附件内容时的元信息说明。
// buildUserMessageParts 合并文本提示和图片输入，形成 Eino 用户消息。
func buildUserMessageParts(content string, attachmentLines []string, images []einoMessage.ImageData, attachmentIntro string) *einoMessage.AgenticMessage {
	textSections := make([]string, 0, 2)
	// 附件说明统一追加到用户正文后，避免模型误以为附件内容来自系统指令。
	if len(attachmentLines) > 0 {
		textSections = append(textSections, attachmentIntro, strings.Join(attachmentLines, "\n\n"))
	}
	return einoMessage.UserTextWithImages(content, textSections, images)
}

// toolUsageFromCallResult 将 Eino 工具调用结果转换为助手协议工具卡。
func toolUsageFromCallResult(result tool.CallResult) ToolUsage {
	return ToolUsage{
		Type:   result.Type,
		Name:   result.Name,
		Title:  result.Title,
		Status: result.Status,
		Input:  result.Input,
		Output: result.Output,
	}
}

// normalizeToolUsages 去重并补齐工具展示字段。
func normalizeToolUsages(values []ToolUsage) []ToolUsage {
	if len(values) == 0 {
		return []ToolUsage{}
	}
	result := make([]ToolUsage, 0, len(values))
	indexMap := make(map[string]int, len(values))
	for _, item := range values {
		if item.Name == "" {
			continue
		}
		if item.Type == "" {
			item.Type = "function"
		}
		if item.Title == "" {
			item.Title = item.Name
		}
		if item.Status == "" {
			item.Status = "success"
		}
		if item.Type == "function" && (item.Input != "" || item.Output != "") {
			result = append(result, item)
			continue
		}
		key := item.Type + ":" + item.Name + ":" + item.Input + ":" + item.Output
		index, ok := indexMap[key]
		if ok {
			if toolStatusRank(item.Status) > toolStatusRank(result[index].Status) {
				result[index] = item
			}
			continue
		}
		indexMap[key] = len(result)
		result = append(result, item)
	}
	return result
}

// toolStatusRank 返回工具状态优先级，成功调用优先覆盖异常记录。
func toolStatusRank(status string) int {
	switch status {
	case "success":
		return 3
	case "error":
		return 2
	default:
		return 0
	}
}

// tokenFromCallback 将 Eino 调用统计转换为助手协议 token。
func tokenFromCallback(token callback.TokenUsage) TokenUsage {
	return TokenUsage{
		Input:  token.Input,
		Output: token.Output,
		Cache:  token.Cache,
		Total:  token.Total,
	}
}

// toolsFromRecorder 将 ADK callback 记录转换为助手协议工具卡。
func toolsFromRecorder(recorder *callback.Recorder) []ToolUsage {
	if recorder == nil {
		return nil
	}
	tools := make([]ToolUsage, 0)
	for _, item := range recorder.ToolCalls() {
		tools = append(tools, ToolUsage{
			Type:   item.Type,
			Name:   item.Name,
			Title:  item.Title,
			Status: item.Status,
			Input:  item.Input,
			Output: item.Output,
		})
	}
	for _, item := range recorder.ServerTools() {
		tools = append(tools, ToolUsage{
			Type:   "server",
			Name:   item.Name,
			Title:  item.Title,
			Status: item.Status,
		})
	}
	return tools
}

// pathExt 返回文件名扩展名。
func pathExt(name string) string {
	_, ext, found := strings.CutLast(name, ".")
	if !found || ext == "" {
		return ""
	}
	return "." + ext
}
