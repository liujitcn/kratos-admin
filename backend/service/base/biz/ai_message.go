package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"
	"github.com/liujitcn/kratos-admin/backend/service/base/agent/ai"
	"github.com/liujitcn/kratos-admin/backend/service/base/dto"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const aiHistorySize = 12

// AiMessageCase 管理 AI 助手消息数据。
type AiMessageCase struct {
	*biz.BaseCase
	tx            data.Transaction
	aiMessageRepo *data.AiMessageRepository
	aiSessionCase *AiSessionCase
	baseAPIRepo   *data.BaseAPIRepository
	baseUserCase  *BaseUserCase
	aiRuntime     *ai.Runtime
}

// NewAiMessageCase 创建 AI 助手消息业务实例。
func NewAiMessageCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiMessageRepo *data.AiMessageRepository,
	aiSessionCase *AiSessionCase,
	baseAPIRepo *data.BaseAPIRepository,
	baseUserCase *BaseUserCase,
	aiRuntime *ai.Runtime,
) *AiMessageCase {
	c := &AiMessageCase{
		BaseCase:      baseCase,
		tx:            tx,
		aiMessageRepo: aiMessageRepo,
		aiSessionCase: aiSessionCase,
		baseAPIRepo:   baseAPIRepo,
		baseUserCase:  baseUserCase,
		aiRuntime:     aiRuntime,
	}
	if aiRuntime != nil {
		aiRuntime.SetToolAccessChecker(c)
	}
	return c
}

// ListAiMessage 查询指定会话的消息列表。
func (c *AiMessageCase) ListAiMessage(ctx context.Context, req *basev1.ListAiMessageRequest) (*basev1.ListAiMessageResponse, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.SessionID.Eq(session.ID)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	var list []*models.AiMessage
	list, err = c.aiMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiMessage, 0, len(list))
	for _, item := range list {
		messages = append(messages, c.ToDTO(item))
	}
	return &basev1.ListAiMessageResponse{Messages: messages}, nil
}

// UpdateAiMessage 更新当前用户消息文本并重新生成同一轮助手输出。
func (c *AiMessageCase) UpdateAiMessage(ctx context.Context, req *basev1.UpdateAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	content := req.GetContent()
	if content == "" {
		return nil, errorsx.InvalidArgument("消息内容不能为空")
	}
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	err = c.ensureLastAiMessage(ctx, session.ID, message.ID)
	if err != nil {
		return nil, err
	}
	if message.Status == int32(commonv1.AiMessageStatus_GENERATING_AAMS) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiMessageStatus_SUCCESS_AAMS)))
	}
	return c.regenerateAiMessageWithContent(ctx, session, message, content)
}

// DeleteAiMessage 删除当前用户当前会话下的单轮消息。
func (c *AiMessageCase) DeleteAiMessage(ctx context.Context, req *basev1.DeleteAiMessageRequest) error {
	message, _, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return err
	}
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(message.ID)))
	return c.aiMessageRepo.Delete(ctx, opts...)
}

// SendAiMessage 发送用户消息并生成 AI 助手回复。
func (c *AiMessageCase) SendAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	session, message, content, attachments, aiAttachments, history, userName, err := c.prepareNewAiMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	startAt := time.Now()
	var reply *ai.Response
	reply, err = c.generateAiReply(ctx, session, userName, content, req.GetAction(), attachments, aiAttachments, history, nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	if err != nil {
		failedReply := c.buildAiFailedReply(reply, err)
		err = c.finishAiMessage(ctx, session, message, failedReply, finishAt, firstTokenMs, durationMs, int32(commonv1.AiMessageStatus_FAILED_AAMS))
		if err != nil {
			return nil, err
		}
		return &basev1.SendAiMessageResponse{
			Messages: []*basev1.AiMessage{c.ToDTO(message)},
			Session:  c.aiSessionCase.ToDTO(session),
		}, nil
	}

	if err = c.finishAiMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, int32(commonv1.AiMessageStatus_SUCCESS_AAMS)); err != nil {
		return nil, err
	}
	return &basev1.SendAiMessageResponse{
		Messages: []*basev1.AiMessage{c.ToDTO(message)},
		Session:  c.aiSessionCase.ToDTO(session),
	}, nil
}

// StreamAiMessage 发送用户消息并流式返回单助手回复。
func (c *AiMessageCase) StreamAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest, emitter dto.AiStreamEmitter) error {
	if emitter == nil {
		return errorsx.Internal("AI助手流式响应未初始化")
	}
	session, message, content, attachments, aiAttachments, history, userName, err := c.prepareNewAiMessage(ctx, req)
	if err != nil {
		return err
	}

	messageID := strconv.FormatInt(message.ID, 10)
	startAt := time.Now()
	var firstTokenMs int32
	reply, runErr := c.generateAiReply(ctx, session, userName, content, req.GetAction(), attachments, aiAttachments, history, func(delta string) {
		if delta == "" {
			return
		}
		if firstTokenMs == 0 {
			firstTokenMs = durationMilliseconds(startAt, time.Now())
		}
		emitErr := emitter.EmitAiStream(dto.AiStreamEventDelta, dto.AiStreamPayload{
			SessionID: req.GetSessionId(),
			MessageID: messageID,
			Delta:     delta,
		})
		if emitErr != nil {
			log.Error(fmt.Sprintf("StreamAiMessage EmitDelta %v", emitErr))
		}
	})

	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	if firstTokenMs == 0 && durationMs > 0 {
		firstTokenMs = durationMs
	}
	status := int32(commonv1.AiMessageStatus_SUCCESS_AAMS)
	if runErr != nil {
		log.Error(fmt.Sprintf("StreamAiMessage RunStream %v", runErr))
		reply = c.buildAiFailedReply(reply, runErr)
		status = int32(commonv1.AiMessageStatus_FAILED_AAMS)
	}

	saveErr := c.finishAiMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, status)
	if saveErr != nil {
		log.Error(fmt.Sprintf("StreamAiMessage SaveReply %v", saveErr))
		_ = emitter.EmitAiStream(dto.AiStreamEventError, dto.AiStreamPayload{
			SessionID: req.GetSessionId(),
			MessageID: messageID,
		})
		return nil
	}

	emitErr := emitter.EmitAiStream(dto.AiStreamEventFinish, dto.AiStreamPayload{
		SessionID: req.GetSessionId(),
		MessageID: messageID,
		Messages:  []*basev1.AiMessage{c.ToDTO(message)},
		Session:   c.aiSessionCase.ToDTO(session),
	})
	if emitErr != nil {
		log.Error(fmt.Sprintf("StreamAiMessage EmitFinish %v", emitErr))
	}
	return nil
}

// RetryAiUserMessage 重试失败的 AI 助手消息。
func (c *AiMessageCase) RetryAiUserMessage(ctx context.Context, req *basev1.RetryAiUserMessageRequest) (*basev1.SendAiMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if message.Status != int32(commonv1.AiMessageStatus_FAILED_AAMS) {
		return nil, errorsx.StateConflict("只能重试失败的消息", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiMessageStatus_FAILED_AAMS)))
	}
	return c.regenerateAiMessage(ctx, session, message)
}

// RegenerateAiMessage 重新生成指定 AI 助手消息。
func (c *AiMessageCase) RegenerateAiMessage(ctx context.Context, req *basev1.RegenerateAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if message.Status == int32(commonv1.AiMessageStatus_GENERATING_AAMS) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiMessageStatus_SUCCESS_AAMS)))
	}
	return c.regenerateAiMessage(ctx, session, message)
}

// ToDTO 转换消息模型到接口对象。
func (c *AiMessageCase) ToDTO(model *models.AiMessage) *basev1.AiMessage {
	if model == nil {
		return nil
	}

	return toAiMessageDTO(model)
}

// ToolConfigs 查询当前终端允许暴露给 Agent 的工具配置。
func (c *AiMessageCase) ToolConfigs(ctx context.Context, terminal string, names []string) (map[string]ai.ToolConfig, error) {
	result := make(map[string]ai.ToolConfig)
	if c == nil || c.baseAPIRepo == nil || len(names) == 0 {
		return result, nil
	}
	filteredNames := make([]string, 0, len(names))
	for _, name := range names {
		if !matchToolTerminal(terminal, name) {
			continue
		}
		filteredNames = append(filteredNames, name)
	}
	if len(filteredNames) == 0 {
		return result, nil
	}
	query := c.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ToolName.In(filteredNames...)))
	list, err := c.baseAPIRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	totalByName := make(map[string]int, len(filteredNames))
	enabledByName := make(map[string]int, len(filteredNames))
	promptsByName := make(map[string][]string, len(filteredNames))
	for _, item := range list {
		totalByName[item.ToolName]++
		if item.AgentStatus == int32(commonv1.Status_ENABLE) {
			enabledByName[item.ToolName]++
		}
		if len(promptsByName[item.ToolName]) == 0 {
			promptsByName[item.ToolName] = parseToolPrompts(item.ToolPrompts)
		}
	}
	for _, name := range filteredNames {
		if totalByName[name] == 0 {
			continue
		}
		result[name] = ai.ToolConfig{
			Enabled: totalByName[name] == enabledByName[name],
			Prompts: promptsByName[name],
		}
	}
	return result, nil
}

// prepareNewAiMessage 校验请求并创建生成中的消息记录。
func (c *AiMessageCase) prepareNewAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest) (*models.AiSession, *models.AiMessage, string, []*basev1.AiAttachment, []ai.Attachment, []ai.Message, string, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}

	content := req.GetContent()
	attachments := ai.NormalizeAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 && req.GetAction() == nil {
		return nil, nil, "", nil, nil, nil, "", errorsx.InvalidArgument("消息内容不能为空")
	}
	err = c.ensureAiActionCurrent(ctx, session, req.GetAction())
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var aiAttachments []ai.Attachment
	aiAttachments, err = c.buildAiAttachments(ctx, attachments)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var userName string
	userName, err = c.baseUserCase.FindUserNameByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var history []ai.Message
	history, err = c.buildHistory(ctx, session.ID, aiHistorySize)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}

	now := time.Now()
	message := &models.AiMessage{
		SessionID:     session.ID,
		UserID:        session.UserID,
		InputContent:  ai.MarshalInputContent(content, attachments),
		OutputContent: ai.MarshalEmptyOutputContent(),
		Attachments:   ai.MarshalAttachments(attachments),
		Tools:         "[]",
		Token:         ai.MarshalTokenUsage(ai.TokenUsage{}),
		FirstTokenMs:  0,
		DurationMs:    0,
		Status:        int32(commonv1.AiMessageStatus_GENERATING_AAMS),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if createErr := c.aiMessageRepo.Create(txCtx, message); createErr != nil {
			return createErr
		}
		summary := ai.BuildDynamicSummary(content, attachments)
		return c.aiSessionCase.UpdateSessionSummary(txCtx, session, summary, now)
	})
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	return session, message, content, attachments, aiAttachments, history, userName, nil
}

// buildHistory 构造问答历史上下文。
func (c *AiMessageCase) buildHistory(ctx context.Context, sessionID int64, historySize int) ([]ai.Message, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.AiMessageStatus_SUCCESS_AAMS))))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return buildHistoryFromMessages(list), nil
}

// regenerateAiMessage 使用已有输入重新生成当前轮次输出。
func (c *AiMessageCase) regenerateAiMessage(ctx context.Context, session *models.AiSession, message *models.AiMessage) (*basev1.SendAiMessageResponse, error) {
	input := ai.ParseInputContent(message.InputContent)
	content := input.Content
	return c.regenerateAiMessageWithContent(ctx, session, message, content)
}

// regenerateAiMessageWithContent 使用指定输入内容重新生成当前轮次输出。
func (c *AiMessageCase) regenerateAiMessageWithContent(ctx context.Context, session *models.AiSession, message *models.AiMessage, content string) (*basev1.SendAiMessageResponse, error) {
	attachments := ai.ParseAttachments(message.Attachments)

	var err error
	var aiAttachments []ai.Attachment
	aiAttachments, err = c.buildAiAttachments(ctx, attachments)
	if err != nil {
		return nil, err
	}
	var userName string
	userName, err = c.baseUserCase.FindUserNameByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	var history []ai.Message
	history, err = c.buildHistoryBeforeMessage(ctx, session.ID, message, aiHistorySize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err = c.markAiMessageGenerating(ctx, message, content, attachments, now); err != nil {
		return nil, err
	}
	startAt := time.Now()
	var reply *ai.Response
	reply, err = c.generateAiReply(ctx, session, userName, content, nil, attachments, aiAttachments, history, nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	status := int32(commonv1.AiMessageStatus_SUCCESS_AAMS)
	if err != nil {
		reply = c.buildAiFailedReply(reply, err)
		status = int32(commonv1.AiMessageStatus_FAILED_AAMS)
	}
	err = c.finishAiMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, status)
	if err != nil {
		return nil, err
	}
	return &basev1.SendAiMessageResponse{
		Messages: []*basev1.AiMessage{c.ToDTO(message)},
		Session:  c.aiSessionCase.ToDTO(session),
	}, nil
}

// buildAiAttachments 读取附件内容，构造 AI 助手输入附件。
func (c *AiMessageCase) buildAiAttachments(ctx context.Context, attachments []*basev1.AiAttachment) ([]ai.Attachment, error) {
	if len(attachments) == 0 {
		return []ai.Attachment{}, nil
	}
	ossClient := sdk.Runtime.GetOSS()
	result := make([]ai.Attachment, 0, len(attachments))
	if ossClient == nil {
		for _, item := range attachments {
			if item == nil {
				continue
			}
			result = append(result, ai.Attachment{
				Name:     item.GetName(),
				Size:     item.GetSize(),
				URL:      item.GetUrl(),
				MIMEType: ai.DetectAttachmentMIME(item.GetName(), item.GetMimeType()),
			})
		}
		return result, nil
	}
	for _, item := range attachments {
		if item == nil {
			continue
		}
		next := ai.Attachment{
			Name:     item.GetName(),
			Size:     item.GetSize(),
			URL:      item.GetUrl(),
			MIMEType: ai.DetectAttachmentMIME(item.GetName(), item.GetMimeType()),
		}
		if next.URL != "" {
			err := validateFilePath(next.URL)
			if err != nil {
				return nil, err
			}
			var fileBytes []byte
			fileBytes, err = ossClient.GetFileByte(next.URL)
			if err != nil {
				return nil, errorsx.Internal("读取 AI 助手附件失败").WithCause(err)
			}
			next.Content = ai.ExtractAttachmentText(fileBytes, next.MIMEType)
			next.Bytes = fileBytes
		}
		result = append(result, next)
	}
	return result, nil
}

// buildHistoryBeforeMessage 构造指定消息之前的上下文。
func (c *AiMessageCase) buildHistoryBeforeMessage(ctx context.Context, sessionID int64, message *models.AiMessage, historySize int) ([]ai.Message, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.AiMessageStatus_SUCCESS_AAMS))))
	opts = append(opts, repository.Where(query.CreatedAt.Lt(message.CreatedAt)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return buildHistoryFromMessages(list), nil
}

// generateAiReply 生成当前消息的 AI 助手回复。
func (c *AiMessageCase) generateAiReply(
	ctx context.Context,
	session *models.AiSession,
	userName string,
	content string,
	action *basev1.AiAction,
	attachments []*basev1.AiAttachment,
	aiAttachments []ai.Attachment,
	history []ai.Message,
	onDelta func(string),
) (*ai.Response, error) {
	var err error
	var handled bool
	var flowReply *ai.Response
	if c.aiRuntime != nil {
		flowReply, handled, err = c.aiRuntime.GenerateFixedFlowReply(ctx, session.Terminal, content, action)
	}
	if handled {
		// 移动端闭环流程由本地 flow 直接生成结构化回复，先透出正文让前端有流式反馈。
		if err == nil && flowReply != nil && flowReply.Content != "" && onDelta != nil {
			onDelta(flowReply.Content)
		}
		return flowReply, err
	}
	if c.aiRuntime != nil {
		input := ai.RuntimeInput{
			Terminal:     ai.NormalizeTerminalString(session.Terminal),
			UserName:     userName,
			SessionTitle: session.Title,
			SessionID:    strconv.FormatInt(session.ID, 10),
			Summary:      session.Summary,
			Content:      content,
			History:      history,
			Attachments:  aiAttachments,
		}
		var response *ai.Response
		if onDelta != nil {
			response, err = c.aiRuntime.RunStream(ctx, input, onDelta)
			if err == nil {
				return response, nil
			}
			return c.buildAiFallbackResponse(content, attachments, err), err
		}
		response, err = c.aiRuntime.Run(ctx, input)
		if err == nil {
			return response, nil
		}
		return c.buildAiFallbackResponse(content, attachments, err), nil
	}

	err = errorsx.Internal("AI助手运行时未初始化")
	return c.buildAiFallbackResponse(content, attachments, err), err
}

// buildAiFailedReply 构造可展示和可排障的助手异常回复。
func (c *AiMessageCase) buildAiFailedReply(reply *ai.Response, cause error) *ai.Response {
	failedReply := reply
	if failedReply == nil {
		failedReply = c.buildAiFallbackResponse("", nil, cause)
	}
	reason := failedReply.FallbackReason
	if reason == "" && cause != nil {
		reason = cause.Error()
	}
	return &ai.Response{
		Content:        failedReply.Content,
		Token:          failedReply.Token,
		Tools:          failedReply.Tools,
		Source:         "fallback",
		Model:          failedReply.Model,
		Fallback:       true,
		FallbackReason: reason,
	}
}

// buildAiFallbackResponse 构造 AI 助手降级回复。
func (c *AiMessageCase) buildAiFallbackResponse(
	content string,
	attachments []*basev1.AiAttachment,
	err error,
) *ai.Response {
	fallbackReason := ""
	if err != nil {
		fallbackReason = err.Error()
	}
	model := ""
	if c != nil && c.aiRuntime != nil {
		model = c.aiRuntime.Model()
	}
	return &ai.Response{
		Content:        ai.BuildFallbackReply(content, attachments),
		Token:          ai.TokenUsage{},
		Tools:          []ai.ToolUsage{},
		Source:         "fallback",
		Model:          model,
		Fallback:       true,
		FallbackReason: fallbackReason,
	}
}

// finishAiMessage 回填当前轮次输出、工具、token 与耗时。
func (c *AiMessageCase) finishAiMessage(
	ctx context.Context,
	session *models.AiSession,
	message *models.AiMessage,
	reply *ai.Response,
	now time.Time,
	firstTokenMs int32,
	durationMs int32,
	status int32,
) error {
	injectAiActionState(reply, message.ID)
	outputContent := ai.MarshalReplyContent(reply)
	tools := ai.MarshalTools(nil)
	token := ai.MarshalTokenUsage(ai.TokenUsage{})
	if reply != nil {
		tools = ai.MarshalTools(reply.Tools)
		token = ai.MarshalTokenUsage(reply.Token)
	}
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		_, updateErr := query.WithContext(txCtx).
			Where(query.ID.Eq(message.ID)).
			UpdateSimple(
				query.OutputContent.Value(outputContent),
				query.Tools.Value(tools),
				query.Token.Value(token),
				query.FirstTokenMs.Value(firstTokenMs),
				query.DurationMs.Value(durationMs),
				query.Status.Value(status),
				query.UpdatedAt.Value(now),
			)
		if updateErr != nil {
			return updateErr
		}
		return c.aiSessionCase.RefreshSessionUpdatedAt(txCtx, session, now)
	})
	if err != nil {
		return err
	}
	message.OutputContent = outputContent
	message.Tools = tools
	message.Token = token
	message.FirstTokenMs = firstTokenMs
	message.DurationMs = durationMs
	message.Status = status
	message.UpdatedAt = now
	return nil
}

// ensureAiActionCurrent 确认流程动作来自当前会话最新消息。
func (c *AiMessageCase) ensureAiActionCurrent(ctx context.Context, session *models.AiSession, action *basev1.AiAction) error {
	if action == nil || action.GetType() == "" {
		return nil
	}
	if action.GetSourceMessageId() == "" && action.GetActionId() == "" && action.GetFlowVersion() == 0 {
		if c.aiRuntime != nil && c.aiRuntime.IsFixedFlowEntryAction(session.Terminal, action.GetFlow(), action.GetType()) {
			return nil
		}
		return aiExpiredActionError("", "")
	}
	sourceMessageID, err := strconv.ParseInt(action.GetSourceMessageId(), 10, 64)
	if err != nil || sourceMessageID <= 0 {
		return aiExpiredActionError(action.GetSourceMessageId(), "")
	}
	if action.GetActionId() == "" || action.GetFlowVersion() != sourceMessageID {
		return aiExpiredActionError(action.GetSourceMessageId(), strconv.FormatInt(sourceMessageID, 10))
	}
	var message *models.AiMessage
	message, err = c.findLatestAiMessage(ctx, session.ID, session.UserID)
	if err != nil {
		return err
	}
	if message.ID != sourceMessageID {
		return aiExpiredActionError(action.GetSourceMessageId(), strconv.FormatInt(message.ID, 10))
	}
	if message.Status != int32(commonv1.AiMessageStatus_SUCCESS_AAMS) {
		return aiExpiredActionError(action.GetSourceMessageId(), strconv.Itoa(int(message.Status)))
	}
	outputContent := ai.ParseOutputContent(message.OutputContent)
	if !aiBlocksContainAction(outputContent.BlocksJSON, action) {
		return aiExpiredActionError(action.GetActionId(), "latest")
	}
	return nil
}

// findLatestAiMessage 查询会话中最后一轮消息。
func (c *AiMessageCase) findLatestAiMessage(ctx context.Context, sessionID int64, userID int64) (*models.AiMessage, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(1))
	message, err := c.aiMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, aiExpiredActionError("", "")
		}
		return nil, err
	}
	return message, nil
}

// markAiMessageGenerating 标记消息进入生成中。
func (c *AiMessageCase) markAiMessageGenerating(ctx context.Context, message *models.AiMessage, content string, attachments []*basev1.AiAttachment, now time.Time) error {
	inputContent := ai.MarshalInputContent(content, attachments)
	query := c.aiMessageRepo.Query(ctx).AiMessage
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(message.ID)).
		UpdateSimple(
			query.InputContent.Value(inputContent),
			query.OutputContent.Value(ai.MarshalEmptyOutputContent()),
			query.Tools.Value("[]"),
			query.Token.Value(ai.MarshalTokenUsage(ai.TokenUsage{})),
			query.FirstTokenMs.Value(0),
			query.DurationMs.Value(0),
			query.Status.Value(int32(commonv1.AiMessageStatus_GENERATING_AAMS)),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	message.InputContent = inputContent
	message.OutputContent = ai.MarshalEmptyOutputContent()
	message.Tools = "[]"
	message.Token = ai.MarshalTokenUsage(ai.TokenUsage{})
	message.FirstTokenMs = 0
	message.DurationMs = 0
	message.Status = int32(commonv1.AiMessageStatus_GENERATING_AAMS)
	message.UpdatedAt = now
	return nil
}

// findCurrentUserMessage 查询当前用户当前会话下的消息。
func (c *AiMessageCase) findCurrentUserMessage(ctx context.Context, rawSessionID string, rawMessageID string) (*models.AiMessage, *models.AiSession, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, rawSessionID)
	if err != nil {
		return nil, nil, err
	}
	var messageID int64
	messageID, err = strconv.ParseInt(rawMessageID, 10, 64)
	if err != nil || messageID <= 0 {
		return nil, nil, errorsx.InvalidArgument("消息编号不合法")
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.ID.Eq(messageID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(session.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(session.UserID)))
	var message *models.AiMessage
	message, err = c.aiMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errorsx.ResourceNotFound("消息不存在")
		}
		return nil, nil, err
	}
	return message, session, nil
}

// ensureLastAiMessage 确认当前消息是会话最后一轮消息。
func (c *AiMessageCase) ensureLastAiMessage(ctx context.Context, sessionID int64, messageID int64) error {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(1))
	lastMessage, err := c.aiMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("消息不存在")
		}
		return err
	}
	if lastMessage.ID != messageID {
		return errorsx.StateConflict("只能编辑最后一条消息", "ai_message", strconv.FormatInt(messageID, 10), strconv.FormatInt(lastMessage.ID, 10))
	}
	return nil
}

// parseToolPrompts 解析工具提示词 JSON。
func parseToolPrompts(value string) []string {
	if value == "" {
		return nil
	}
	var prompts []string
	err := json.Unmarshal([]byte(value), &prompts)
	if err != nil {
		return nil
	}
	values := make([]string, 0, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		values = append(values, item)
	}
	return values
}

// toAiMessageDTO 转换消息模型到接口对象。
func toAiMessageDTO(model *models.AiMessage) *basev1.AiMessage {
	if model == nil {
		return nil
	}
	inputContent := ai.ParseInputContent(model.InputContent)
	outputContent := ai.ParseOutputContent(model.OutputContent)
	token := ai.ParseTokenUsage(model.Token)
	return &basev1.AiMessage{
		Id:            strconv.FormatInt(model.ID, 10),
		InputContent:  toAiInputContent(inputContent),
		OutputContent: toAiOutputContent(outputContent),
		Attachments:   ai.ParseAttachments(model.Attachments),
		CreatedAt:     timestamppb.New(model.CreatedAt),
		Status:        commonv1.AiMessageStatus(model.Status),
		Token:         toAiToken(token),
		Tools:         toAiToolCalls(ai.ParseTools(model.Tools)),
		FirstTokenMs:  model.FirstTokenMs,
		DurationMs:    model.DurationMs,
	}
}

// injectAiActionState 为流程动作补充来源消息和状态版本。
func injectAiActionState(response *ai.Response, messageID int64) {
	if response == nil || response.BlocksJSON == "" || messageID <= 0 {
		return
	}
	var blocks []any
	err := json.Unmarshal([]byte(response.BlocksJSON), &blocks)
	if err != nil {
		return
	}
	sourceMessageID := strconv.FormatInt(messageID, 10)
	actionIndex := 0
	if !injectAiActionStateValue(blocks, sourceMessageID, messageID, &actionIndex) {
		return
	}
	var raw []byte
	raw, err = json.Marshal(blocks)
	if err != nil {
		return
	}
	response.BlocksJSON = string(raw)
}

// injectAiActionStateValue 递归补齐 blocks 内所有 action 的状态字段。
func injectAiActionStateValue(value any, sourceMessageID string, flowVersion int64, actionIndex *int) bool {
	changed := false
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			if injectAiActionStateValue(item, sourceMessageID, flowVersion, actionIndex) {
				changed = true
			}
		}
	case map[string]any:
		if aiActionStringValue(current["type"]) != "" && aiActionStringValue(current["flow"]) != "" {
			current["source_message_id"] = sourceMessageID
			current["action_id"] = sourceMessageID + ":" + strconv.Itoa(*actionIndex)
			current["flow_version"] = flowVersion
			(*actionIndex)++
			changed = true
		}
		for _, item := range current {
			if injectAiActionStateValue(item, sourceMessageID, flowVersion, actionIndex) {
				changed = true
			}
		}
	}
	return changed
}

// aiBlocksContainAction 判断指定 blocks 中是否存在同一个流程动作。
func aiBlocksContainAction(raw string, action *basev1.AiAction) bool {
	if raw == "" || action == nil {
		return false
	}
	var blocks []any
	err := json.Unmarshal([]byte(raw), &blocks)
	if err != nil {
		return false
	}
	return aiValueContainsAction(blocks, action)
}

// aiValueContainsAction 递归查找 blocks 内的动作定义。
func aiValueContainsAction(value any, action *basev1.AiAction) bool {
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			if aiValueContainsAction(item, action) {
				return true
			}
		}
	case map[string]any:
		if matchAiAction(current, action) {
			return true
		}
		for _, item := range current {
			if aiValueContainsAction(item, action) {
				return true
			}
		}
	}
	return false
}

// matchAiAction 判断前端回传动作是否命中服务端生成的动作定义。
func matchAiAction(candidate map[string]any, action *basev1.AiAction) bool {
	return aiActionStringValue(candidate["source_message_id"]) == action.GetSourceMessageId() &&
		aiActionStringValue(candidate["action_id"]) == action.GetActionId() &&
		aiActionInt64Value(candidate["flow_version"]) == action.GetFlowVersion() &&
		aiActionStringValue(candidate["flow"]) == action.GetFlow() &&
		aiActionStringValue(candidate["step"]) == action.GetStep() &&
		aiActionStringValue(candidate["type"]) == action.GetType()
}

// aiExpiredActionError 构造流程动作过期错误。
func aiExpiredActionError(currentState string, expectedState string) error {
	return errorsx.StateConflict("该步骤已过期，请从最新消息继续操作", "ai_action", currentState, expectedState)
}

// aiActionStringValue 将 JSON 值转成字符串。
func aiActionStringValue(value any) string {
	if result, ok := value.(string); ok {
		return result
	}
	return ""
}

// aiActionInt64Value 将 JSON 数值转成 int64。
func aiActionInt64Value(value any) int64 {
	switch item := value.(type) {
	case float64:
		return int64(item)
	case int64:
		return item
	case int:
		return int64(item)
	case json.Number:
		result, err := item.Int64()
		if err != nil {
			return 0
		}
		return result
	default:
		return 0
	}
}

// toAiInputContent 转换输入内容 JSON 为接口对象。
func toAiInputContent(value ai.InputContentPayload) *basev1.AiInputContent {
	return &basev1.AiInputContent{
		Kind:    value.Kind,
		Content: value.Content,
	}
}

// toAiOutputContent 转换输出内容 JSON 为接口对象。
func toAiOutputContent(value ai.OutputContentPayload) *basev1.AiOutputContent {
	return &basev1.AiOutputContent{
		Kind:           value.Kind,
		Content:        value.Content,
		ReplySource:    value.ReplySource,
		Model:          value.Model,
		Fallback:       value.Fallback,
		FallbackReason: value.FallbackReason,
		Flow:           value.Flow,
		Step:           value.Step,
		BlocksJson:     value.BlocksJSON,
	}
}

// toAiToken 转换 token 统计为接口对象。
func toAiToken(value ai.TokenUsage) *basev1.AiToken {
	return &basev1.AiToken{
		Input:  value.Input,
		Output: value.Output,
		Cache:  value.Cache,
		Total:  value.Total,
	}
}

// toAiToolCalls 转换工具使用记录为接口对象。
func toAiToolCalls(values []ai.ToolUsage) []*basev1.AiToolCall {
	if len(values) == 0 {
		return []*basev1.AiToolCall{}
	}
	tools := make([]*basev1.AiToolCall, 0, len(values))
	for _, item := range values {
		tools = append(tools, &basev1.AiToolCall{
			Type:   item.Type,
			Name:   item.Name,
			Title:  item.Title,
			Status: item.Status,
			Input:  item.Input,
			Output: item.Output,
		})
	}
	return tools
}

// buildHistoryFromMessages 将一轮一行消息拆成模型需要的 user/ai 上下文。
func buildHistoryFromMessages(list []*models.AiMessage) []ai.Message {
	history := make([]ai.Message, 0, len(list)*2)
	for index := len(list) - 1; index >= 0; index-- {
		item := list[index]
		input := ai.ParseInputContent(item.InputContent)
		if input.Content != "" {
			history = append(history, ai.Message{
				Role:    ai.RoleUser,
				Content: input.Content,
			})
		}
		output := ai.ParseOutputContent(item.OutputContent)
		if output.Content != "" {
			tools := ai.ParseTools(item.Tools)
			history = append(history, ai.Message{
				Role:    ai.RoleAI,
				Content: output.Content,
				Tools:   tools,
			})
		}
	}
	return history
}

// durationMilliseconds 计算两个时间点之间的毫秒数。
func durationMilliseconds(start time.Time, end time.Time) int32 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	ms := end.Sub(start).Milliseconds()
	if ms <= 0 {
		return 0
	}
	if ms > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(ms)
}
