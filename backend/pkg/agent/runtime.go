package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/adk"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	einoMessage "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
)

const (
	maxHistoryToolText = 2000
)

const fallbackAIInstruction = "You are a general-purpose AI assistant. Answer naturally, helpfully, and accurately. Use internal tools for private system data, do not fabricate results, and reply in the user's current language."

// Runtime 封装流式 AI 助手运行时。
//
// Runtime 只负责把业务层准备好的输入组装为 Eino 消息并交给模型执行，不直接处理数据库、
// OSS、鉴权或前端协议。这样 AI 助手链路可以把“业务准备”和“模型运行”分开维护。
type Runtime struct {
	checkPointStore    CheckPointStore
	toolsMu            sync.RWMutex
	client             *model.AssistantClient
	clientResolver     model.AssistantClientResolver
	adminTools         []tool.Base
	appTools           []tool.Base
	toolGate           ToolAccessChecker
	localizeMessage    MessageLocalizer
	responseBlocksHook func([]ToolUsage) []map[string]any
}

// newRuntime 创建 AI 助手运行时。
func newRuntime(
	client *model.AssistantClient,
	clientResolver model.AssistantClientResolver,
	checker ToolAccessChecker,
	adminTools []Tool,
	appTools []Tool,
	localizeMessage MessageLocalizer,
) *Runtime {
	return &Runtime{
		client:          client,
		clientResolver:  clientResolver,
		adminTools:      append([]tool.Base(nil), adminTools...),
		appTools:        append([]tool.Base(nil), appTools...),
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

// RegisteredTools 返回指定终端静态登记工具切片的副本。
func (r *Runtime) RegisteredTools(terminal string) []tool.Base {
	if r == nil {
		return nil
	}
	r.toolsMu.RLock()
	defer r.toolsMu.RUnlock()
	if terminal == "app" {
		return append([]tool.Base(nil), r.appTools...)
	}
	return append([]tool.Base(nil), r.adminTools...)
}

// InvokeTool 按工具名直接调用当前终端已启用的 Agent 工具。
func (r *Runtime) InvokeTool(ctx context.Context, terminal string, name string, arguments string) (*ToolInvokeResult, error) {
	if r == nil {
		return nil, errors.New("AI assistant runtime is not initialized")
	}
	ctx = ContextWithTerminal(ctx, terminal)
	input := RuntimeInput{Terminal: terminal, Content: name}
	infos := r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input))
	result := tool.ExecuteCall(ctx, r.toolMap(ctx, input), infos, tool.Call{
		ID:        "direct_" + name,
		Name:      name,
		Arguments: arguments,
	}, tool.WithMessageLocalizer(r.localizeMessage))
	usage := toolUsageFromCallResult(result)
	if usage.Status != "success" {
		return &ToolInvokeResult{Output: result.Content, Usage: usage}, errors.New(result.Content)
	}
	return &ToolInvokeResult{Output: result.Content, Usage: usage}, nil
}

// Enabled 判断 AI 助手运行时是否可用。
func (r *Runtime) Enabled() bool {
	if r == nil {
		return false
	}
	if r.clientResolver != nil {
		return r.clientResolver.HasEnabledAssistantClient()
	}
	return r.client != nil && r.client.Enabled()
}

// Model 返回 AI 助手当前使用的模型名称。
func (r *Runtime) Model() string {
	if r == nil {
		return ""
	}
	client := r.client
	if r.clientResolver != nil {
		client = r.clientResolver.DefaultAssistantClient()
	}
	if client == nil {
		return ""
	}
	return client.Name()
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
	client, err := r.resolveAssistantClient(input)
	if err != nil {
		return nil, err
	}
	return r.runGenerate(ctx, input, client)
}

// RunStream 使用流式模式运行助手。
//
// 该方法用于管理端 direct SSE：模型返回文本片段时会透传给 onDelta，
// 最终仍返回完整回复供业务层落库。
func (r *Runtime) RunStream(ctx context.Context, input RuntimeInput, onDelta func(string)) (*Response, error) {
	client, err := r.resolveAssistantClient(input)
	if err != nil {
		return nil, err
	}
	toolInfos, internalToolMatched := r.toolInfos(ctx, input)
	preparedInput, cleanup := r.prepareAttachments(ctx, input, client)
	if cleanup != nil {
		defer cleanup()
	}
	messages := r.buildMessages(ctx, preparedInput, client)
	var result *adk.Result
	var recorder *callback.Recorder
	result, recorder, err = r.runADK(ctx, preparedInput, client, messages, toolInfos, internalToolMatched, true, onDelta)
	if err != nil {
		return nil, err
	}
	return r.responseFromRun(ctx, input, client, toolInfos, internalToolMatched, result, recorder), nil
}

// prepareAttachments 为支持的 Responses 客户端上传图片和 PDF，并返回请求结束后的清理函数。
func (r *Runtime) prepareAttachments(ctx context.Context, input RuntimeInput, client *model.AssistantClient) (RuntimeInput, func()) {
	input.Attachments = append([]Attachment(nil), input.Attachments...)
	for index := range input.Attachments {
		input.Attachments[index].FileID = ""
	}
	// 审批可能跨请求和进程恢复，直接把附件内容存入 checkpoint，避免引用已删除的供应商临时文件。
	if input.CheckPointID != "" || !client.SupportsFileUpload() {
		return input, nil
	}

	fileIDs := make([]string, 0, len(input.Attachments))
	var err error
	for index := range input.Attachments {
		item := &input.Attachments[index]
		mimeType := normalizeRuntimeMIMEType(*item)
		if (!isRuntimeImageMIME(mimeType) && mimeType != "application/pdf") || len(item.Bytes) == 0 {
			continue
		}
		var fileID string
		fileID, err = client.UploadFile(ctx, item.Name, item.Bytes)
		if err != nil {
			log.Warn("上传 AI 助手附件失败，沿用现有处理方式", "filename", item.Name, "error", err)
			continue
		}
		item.FileID = fileID
		fileIDs = append(fileIDs, fileID)
	}
	if len(fileIDs) == 0 {
		return input, nil
	}
	return input, func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		for _, fileID := range fileIDs {
			var cleanupErr error
			cleanupErr = client.DeleteFile(cleanupCtx, fileID)
			if cleanupErr != nil {
				log.Warn("删除 AI 助手临时附件失败", "file_id", fileID, "error", cleanupErr)
			}
		}
	}
}

// ToolInvokeResult 表示直接调用 Agent 工具后的结果。
type ToolInvokeResult struct {
	// Output 工具原始输出 JSON。
	Output string
	// Usage 本次工具调用的后台展示记录。
	Usage ToolUsage
}

// runGenerate 执行非流式模型调用并统一收集工具审批与调用记录。
func (r *Runtime) runGenerate(ctx context.Context, input RuntimeInput, client *model.AssistantClient) (*Response, error) {
	toolInfos, internalToolMatched := r.toolInfos(ctx, input)
	preparedInput, cleanup := r.prepareAttachments(ctx, input, client)
	if cleanup != nil {
		defer cleanup()
	}
	messages := r.buildMessages(ctx, preparedInput, client)
	result, recorder, err := r.runADK(ctx, preparedInput, client, messages, toolInfos, internalToolMatched, false, nil)
	if err != nil {
		return nil, err
	}
	return r.responseFromRun(ctx, input, client, toolInfos, internalToolMatched, result, recorder), nil
}

// runADK 通过 Eino ADK ChatModelAgent / Runner 执行模型和工具循环。
func (r *Runtime) runADK(
	ctx context.Context,
	input RuntimeInput,
	client *model.AssistantClient,
	messages []*einoMessage.AgenticMessage,
	toolInfos []*tool.Info,
	internalToolMatched bool,
	stream bool,
	onDelta func(string),
) (*adk.Result, *callback.Recorder, error) {
	ctx = ContextWithTerminal(ctx, input.Terminal)
	ctx = ContextWithAssistantClient(ctx, client)
	recorder := &callback.Recorder{}
	// Responses 协议才支持服务端工具（联网搜索等），聊天补全协议使用同名 function 工具替代。
	serverTools := client.SupportsResponsesServerTools() && !internalToolMatched
	description := r.localizePrompt(ctx, "base.ai.prompt.assistant_description", nil, "Admin AI assistant for conversation, internal tools, and web search.")
	runner := adk.NewRunner(adk.Config{
		CheckPointStore: r.checkPointStore,
		Model:           client.AgenticModel,
		Name:            "admin_ai",
		Description:     description,
		ServerTools:     serverTools,
	})
	modelOptions := []model.Option(nil)
	if client.SupportsResponsesServerTools() {
		modelOptions = model.ResponsesReasoningOptions()
	}
	result, err := runner.Run(ctx, adk.Request{
		CheckPointID:     input.CheckPointID,
		Messages:         append([]*einoMessage.AgenticMessage(nil), messages...),
		Tools:            r.runnerTools(ctx, input, toolInfos),
		ToolInfos:        toolInfos,
		DynamicTools:     true,
		ResolveToolInfos: func(callCtx context.Context) []*tool.Info { return r.enabledToolInfos(callCtx, input, toolInfos) },
		Recorder:         recorder,
		LocalizeMessage:  r.localizeMessage,
		ModelOptions:     modelOptions,
		// Stream 控制 SSE 增量透传意图；ModelStreaming 控制模型实际调用方式，
		// 禁用流式的供应商（聚合网关流式协议不兼容）回退非流式调用并整段透传。
		Stream:         stream,
		ModelStreaming: stream && !client.DisableStreaming,
		OnDelta:        onDelta,
	})
	if err == nil && result != nil && len(result.Pending) == 0 && len(einoMessage.ToolCalls(result.Message)) == 0 && hasUnparsedToolCall(result.Message) {
		return nil, recorder, fmt.Errorf("model returned an unparsed tool call")
	}
	return result, recorder, err
}

// hasUnparsedToolCall 判断模型是否把工具调用当作普通文本返回。
func hasUnparsedToolCall(value *einoMessage.AgenticMessage) bool {
	content := strings.ToLower(einoMessage.Text(value))
	return strings.Contains(content, "<tool_calls>") ||
		strings.Contains(content, "<tool_call>") ||
		strings.Contains(content, "<|tool_calls|") ||
		strings.Contains(content, "<｜tool")
}

// runnerTools 收集当前终端可由 ADK 执行的工具。
func (r *Runtime) runnerTools(ctx context.Context, input RuntimeInput, infos []*tool.Info) []tool.Base {
	toolMap := r.toolMap(ctx, input)
	allowed := tool.NameSet(infos)
	tools := make([]tool.Base, 0, len(toolMap))
	for name, item := range toolMap {
		if item == nil || !allowed[name] {
			continue
		}
		tools = append(tools, item)
	}
	return tools
}

// toolInfos 收集当前请求完整授权工具池；相关性选择交给原生动态发现。
func (r *Runtime) toolInfos(ctx context.Context, input RuntimeInput) ([]*tool.Info, bool) {
	return r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input)), false
}

// toolMap 按工具名构造本地执行索引。
func (r *Runtime) toolMap(ctx context.Context, input RuntimeInput) map[string]tool.Base {
	registeredInfos := r.allToolInfos(ctx, input)
	enabledInfos := r.enabledToolInfos(ctx, input, registeredInfos)
	enabledNames := tool.NameSet(enabledInfos)
	pool := r.requestToolPool(ctx, input.Terminal)
	tools := pool.Tools
	result := make(map[string]tool.Base, len(enabledNames)+1)
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
		name := info.Name
		effectiveInfo := info
		for _, enabledInfo := range enabledInfos {
			if enabledInfo.Name == name {
				effectiveInfo = enabledInfo
				break
			}
		}
		item = tool.NewGuardedBaseTool(item, func(callCtx context.Context, arguments string) (string, bool, error) {
			if pool.Checker == nil {
				return arguments, false, nil
			}
			return pool.Checker.PrepareToolCall(callCtx, input.Terminal, name, arguments)
		}, r.checkPointStore != nil && input.CheckPointID != "", effectiveInfo)
		result[info.Name] = item
	}

	return result
}

// buildMessages 按当前所选模型协议构建发送给 Eino 的消息列表。
func (r *Runtime) buildMessages(ctx context.Context, input RuntimeInput, client *model.AssistantClient) []*einoMessage.AgenticMessage {
	messages := []*einoMessage.AgenticMessage{einoMessage.SystemText(r.resolvePrompt(ctx, input))}
	enabledNames := tool.NameSet(r.enabledToolInfos(ctx, input, r.allToolInfos(ctx, input)))
	for _, item := range input.History {
		if item.Content == "" {
			continue
		}
		historyMessage := historyMessageWithImages(item)
		if historyMessage == nil {
			historyMessage = einoMessage.TextByRole(item.Role, item.Content)
		}
		if historyMessage == nil {
			continue
		}
		messages = append(messages, historyMessage)
		toolContext := r.buildHistoryToolContext(ctx, item.Tools, enabledNames)
		if toolContext != "" {
			messages = append(messages, einoMessage.SystemText(toolContext))
		}
	}
	messages = append(messages, r.buildUserMessage(ctx, input, client))
	return messages
}

// historyMessageWithImages 把携带图片字节的历史用户消息构造为多模态消息，
// 使多轮追问时模型仍能看到最近一张图片；无图片字节时返回 nil 走纯文本构造。
func historyMessageWithImages(item Message) *einoMessage.AgenticMessage {
	if item.Role != RoleUser {
		return nil
	}
	images := make([]einoMessage.ImageData, 0, len(item.Attachments))
	attachmentLines := make([]string, 0, len(item.Attachments))
	for _, attachment := range item.Attachments {
		if len(attachment.Bytes) == 0 {
			continue
		}
		images = append(images, einoMessage.ImageData{Bytes: attachment.Bytes, MIMEType: attachment.MIMEType})
		attachmentLines = append(attachmentLines, attachment.Name)
	}
	if len(images) == 0 {
		return nil
	}
	return buildUserMessageParts(item.Content, attachmentLines, images, nil, "")
}

// enabledToolInfos 按 base_api.agent_status 过滤当前终端可暴露给 Agent 的工具。
func (r *Runtime) enabledToolInfos(ctx context.Context, input RuntimeInput, infos []*tool.Info) []*tool.Info {
	pool := r.requestToolPool(ctx, input.Terminal)
	if len(infos) == 0 || pool.Checker == nil {
		return infos
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if info == nil || info.Name == "" {
			continue
		}
		names = append(names, info.Name)
	}
	toolConfigs, err := pool.Checker.ToolConfigs(ctx, input.Terminal, names)
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
	pool := r.requestToolPool(ctx, input.Terminal)
	tools := pool.Tools
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
	pool := r.requestToolPool(ctx, input.Terminal)
	if len(infos) == 0 || pool.Checker == nil {
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
	toolConfigs, err := pool.Checker.ToolConfigs(ctx, input.Terminal, names)
	if err != nil {
		return result
	}
	return toolConfigs
}

// resolvePrompt 渲染 AI 助手提示词。
func (r *Runtime) resolvePrompt(ctx context.Context, input RuntimeInput) string {
	now := time.Now().In(time.Local)
	lines := []string{
		r.localizePrompt(ctx, "base.ai.prompt.instruction", nil, fallbackAIInstruction),
		r.localizePrompt(ctx, "base.ai.prompt.tool_routing_rules", nil,
			"Use internal tools for private project data. When an internal tool matches this request, answer only with internal tools and do not use web search. Follow tool descriptions and parameter defaults; do not add or remove query conditions without instruction. For a clarified follow-up, reuse the latest relevant tool and preserve other conditions, changing only the clarified condition. If no exposed tool matches a request about internal project data, call tool_search first to locate the correct internal tool; never answer such requests from memory or web search alone. Ask if the target or condition is ambiguous."),
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
	if input.KnowledgeContext != "" {
		lines = append(lines, "", input.KnowledgeContext)
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

// buildUserMessage 按模型协议选择附件文件输入或已提取文本。
func (r *Runtime) buildUserMessage(ctx context.Context, input RuntimeInput, client *model.AssistantClient) *einoMessage.AgenticMessage {
	content := input.Content
	attachmentLines := make([]string, 0, len(input.Attachments)*2)
	images := make([]einoMessage.ImageData, 0, len(input.Attachments))
	files := make([]einoMessage.FileData, 0, len(input.Attachments))

	for _, item := range input.Attachments {
		attachmentContent := item.Content
		name := item.Name
		if name == "" {
			name = r.localizePrompt(ctx, "base.ai.prompt.attachment_unnamed", nil, "Unnamed attachment")
		}
		cleanMIMEType := normalizeRuntimeMIMEType(item)

		// 图片附件走多模态输入，避免把本地 admin 地址误当成公网图片 URL。
		if isRuntimeImageMIME(cleanMIMEType) {
			if item.FileID != "" {
				attachmentLines = append(attachmentLines, r.localizePrompt(ctx, "base.ai.prompt.image_attachment", map[string]any{"Name": name}, "Image attachment {{.Name}} is provided as visual input."))
				images = append(images, einoMessage.ImageData{FileID: item.FileID, MIMEType: cleanMIMEType})
				continue
			}
			if len(item.Bytes) == 0 {
				continue
			}
			attachmentLines = append(attachmentLines, r.localizePrompt(ctx, "base.ai.prompt.image_attachment", map[string]any{"Name": name}, "Image attachment {{.Name}} is provided as visual input."))
			images = append(images, einoMessage.ImageData{Bytes: item.Bytes, MIMEType: cleanMIMEType})
			continue
		}
		if client.APIType() == modelconfig.APITypeResponses && cleanMIMEType == "application/pdf" && item.FileID != "" {
			attachmentLines = append(attachmentLines, r.localizePrompt(ctx, "base.ai.prompt.file_attachment", map[string]any{"Name": name}, "File attachment {{.Name}} is provided as file input."))
			files = append(files, einoMessage.FileData{FileID: item.FileID, Bytes: item.Bytes, MIMEType: cleanMIMEType, Name: name})
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

	if len(attachmentLines) == 0 && len(images) == 0 && len(files) == 0 {
		return einoMessage.UserText(content)
	}
	return buildUserMessageParts(content, attachmentLines, images, files, r.localizePrompt(ctx, "base.ai.prompt.attachment_message_intro", nil, "The current message includes the following attachments. Refer to them when relevant:"))
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
func (r *Runtime) buildResponse(message *einoMessage.AgenticMessage, token TokenUsage, tools []ToolUsage, modelName string) *Response {
	response := &Response{
		Content:        einoMessage.Text(message),
		Token:          token,
		Tools:          normalizeToolUsages(tools),
		Source:         "llm",
		Model:          modelName,
		Fallback:       false,
		FallbackReason: "",
	}
	// 业务模块可按本轮工具结果注入结构化引导卡片，例如问数成功后的建卡入口。
	if blocks := r.BuildResponseBlocks(response.Tools); len(blocks) > 0 {
		if raw, err := json.Marshal(blocks); err == nil {
			response.BlocksJSON = string(raw)
		}
	}
	return response
}

// BuildResponseBlocks 调用业务注册的响应引导块钩子，供审批恢复等旁路落库路径复用。
func (r *Runtime) BuildResponseBlocks(tools []ToolUsage) []map[string]any {
	if r == nil || r.responseBlocksHook == nil || len(tools) == 0 {
		return nil
	}
	return r.responseBlocksHook(tools)
}

// resolveAssistantClient 按当前请求选择模型客户端，旧调用方使用静态客户端。
func (r *Runtime) resolveAssistantClient(input RuntimeInput) (*model.AssistantClient, error) {
	if r == nil {
		return nil, errors.New("AI助手运行时未初始化")
	}
	if r.clientResolver != nil {
		return r.clientResolver.ResolveAssistantClient(input.ProviderID, input.ModelName)
	}
	if r.client == nil || !r.client.Enabled() {
		return nil, errors.New("AI助手没有已启用的模型")
	}
	return r.client, nil
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

// buildUserMessageParts 合并文本提示和图片输入，形成 Eino 用户消息。
func buildUserMessageParts(content string, attachmentLines []string, images []einoMessage.ImageData, files []einoMessage.FileData, attachmentIntro string) *einoMessage.AgenticMessage {
	textSections := make([]string, 0, 2)
	// 附件说明统一追加到用户正文后，避免模型误以为附件内容来自系统指令。
	if len(attachmentLines) > 0 {
		textSections = append(textSections, attachmentIntro, strings.Join(attachmentLines, "\n\n"))
	}
	return einoMessage.UserTextWithMedia(content, textSections, images, files)
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
