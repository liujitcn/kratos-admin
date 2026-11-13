package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/id"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	agentmodel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/automation"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const (
	aiHistorySize            = 12
	aiMessagePersistTimeout  = 30 * time.Second
	aiGeneratedFileDirectory = "ai/generated"
	aiGeneratedFileMaxSize   = 20 << 20
)

// AiMessageCase 管理 AI 助手消息数据。
type AiMessageCase struct {
	*biz.BaseCase
	tx                data.Transaction
	aiMessageRepo     *data.AiMessageRepository
	aiApprovalRepo    *data.AiToolApprovalRepository
	aiSessionCase     *AiSessionCase
	baseUserCase      *BaseUserCase
	aiRuntime         *ai.Runtime
	aiPluginCase      *AiPluginCase
	aiFileCase        *AiFileCase
	automationCase    *AiAutomationCase
	aiKnowledgeEngine *ai.KnowledgeEngine
}

// NewAiMessageCase 创建 AI 助手消息业务实例。
func NewAiMessageCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiMessageRepo *data.AiMessageRepository,
	aiApprovalRepo *data.AiToolApprovalRepository,
	aiSessionCase *AiSessionCase,
	baseUserCase *BaseUserCase,
	aiRuntime *ai.Runtime,
	aiKnowledgeEngine *ai.KnowledgeEngine,
	aiPluginCase *AiPluginCase,
	aiFileCase *AiFileCase,
) *AiMessageCase {
	c := &AiMessageCase{
		BaseCase:          baseCase,
		tx:                tx,
		aiMessageRepo:     aiMessageRepo,
		aiApprovalRepo:    aiApprovalRepo,
		aiSessionCase:     aiSessionCase,
		baseUserCase:      baseUserCase,
		aiRuntime:         aiRuntime,
		aiKnowledgeEngine: aiKnowledgeEngine,
		aiPluginCase:      aiPluginCase,
		aiFileCase:        aiFileCase,
	}
	return c
}

// ListAiMessage 终止过期生成后查询指定会话的消息列表。
func (c *AiMessageCase) ListAiMessage(ctx context.Context, req *basev1.ListAiMessageRequest) (*basev1.ListAiMessageResponse, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if err = c.expireAiGeneratingMessages(ctx, session); err != nil {
		return nil, err
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TenantID.Eq(session.TenantID)))
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
	if session.AutomationRunID > 0 {
		return nil, errorsx.PermissionDenied("自动化结果只读，请使用继续讨论")
	}
	err = c.ensureLastAiMessage(ctx, session.TenantID, session.ID, message.ID)
	if err != nil {
		return nil, err
	}
	if message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING) || message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)))
	}
	return c.regenerateAiMessageWithContent(ctx, session, message, content)
}

// DeleteAiMessage 删除当前用户当前会话下的单轮消息。
func (c *AiMessageCase) DeleteAiMessage(ctx context.Context, req *basev1.DeleteAiMessageRequest) error {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return err
	}
	if session.AutomationRunID > 0 {
		return errorsx.PermissionDenied("自动化结果只读，请删除整个结果会话")
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var txErr error
		if txErr = c.aiSessionCase.lockAiSession(txCtx, session); txErr != nil {
			return txErr
		}
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var result gen.ResultInfo
		result, txErr = query.WithContext(txCtx).Where(query.TenantID.Eq(message.TenantID), query.ID.Eq(message.ID), query.Status.Neq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING))).Delete()
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("正在执行的消息不能删除")
		}
		approval := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		_, txErr = approval.WithContext(txCtx).Where(approval.MessageID.Eq(message.ID), approval.Status.Eq(aiRunPending)).UpdateSimple(approval.Status.Value(aiRunRejected), approval.CheckpointData.Value([]byte{}), approval.UpdatedAt.Value(time.Now()))
		return txErr
	})
}

// SendAiMessage 发送用户消息并生成 AI 助手回复。
func (c *AiMessageCase) SendAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	session, message, content, attachments, aiAttachments, history, userName, err := c.prepareNewAiMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	startAt := time.Now()
	input := ai.ParseInputContent(message.InputContent)
	var reply *ai.Response
	reply, err = c.generateAiReply(ctx, session, message.ID, userName, content, input.ProviderID, input.ProviderName, input.ModelName, req.GetAction(), attachments, aiAttachments, history, req.GetKnowledgeBaseIds(), nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	if err != nil {
		log.Error(fmt.Sprintf("AI message generation failed: %v", err))
		failedReply := c.buildAiFailedReply(ctx, reply, err, input.ProviderName, input.ModelName)
		err = c.finishAiMessage(ctx, session, message, failedReply, finishAt, firstTokenMs, durationMs, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED))
		if err != nil {
			return nil, err
		}
		return &basev1.SendAiMessageResponse{
			Messages: []*basev1.AiMessage{c.ToDTO(message)},
			Session:  c.aiSessionCase.ToDTO(session),
		}, nil
	}

	if err = c.finishAiMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
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
	ctx = callback.WithActivityEmitter(ctx, func(activity callback.Activity) {
		activityErr := emitter.EmitAiStream(dto.AiStreamEventActivity, dto.AiStreamPayload{
			SessionID: req.GetSessionId(),
			MessageID: messageID,
			Activity: &dto.AiStreamActivity{
				ID:        activity.ID,
				Phase:     activity.Phase,
				Status:    activity.Status,
				Summary:   activity.Summary,
				ToolName:  activity.ToolName,
				ToolTitle: activity.ToolTitle,
			},
		})
		if activityErr != nil {
			log.Error(fmt.Sprintf("StreamAiMessage EmitActivity %v", activityErr))
		}
	})
	callback.EmitActivity(ctx, callback.Activity{ID: "reasoning:0", Phase: "reasoning", Status: "running"})
	startAt := time.Now()
	input := ai.ParseInputContent(message.InputContent)
	answerStarted := false
	var firstTokenMs int32
	reply, runErr := c.generateAiReply(ctx, session, message.ID, userName, content, input.ProviderID, input.ProviderName, input.ModelName, req.GetAction(), attachments, aiAttachments, history, req.GetKnowledgeBaseIds(), func(delta string) {
		if delta == "" {
			return
		}
		if firstTokenMs == 0 {
			firstTokenMs = durationMilliseconds(startAt, time.Now())
		}
		if !answerStarted {
			answerStarted = true
			callback.EmitActivity(ctx, callback.Activity{ID: "answer", Phase: "answering", Status: "running"})
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
	status := int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
	if runErr != nil {
		log.Error(fmt.Sprintf("StreamAiMessage RunStream %v", runErr))
		reply = c.buildAiFailedReply(ctx, reply, runErr, input.ProviderName, input.ModelName)
		status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)
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

	event := dto.AiStreamEventFinish
	if message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) {
		event = dto.AiStreamEventApprovalRequired
	}
	emitErr := emitter.EmitAiStream(event, dto.AiStreamPayload{
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
	if message.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED) {
		return nil, errorsx.StateConflict("只能重试失败的消息", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)))
	}
	return c.regenerateAiMessage(ctx, session, message)
}

// RegenerateAiMessage 重新生成指定 AI 助手消息。
func (c *AiMessageCase) RegenerateAiMessage(ctx context.Context, req *basev1.RegenerateAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING) || message.Status == int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)))
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

// prepareNewAiMessage 清理过期生成，校验请求并创建新一轮消息记录。
func (c *AiMessageCase) prepareNewAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest) (*models.AiSession, *models.AiMessage, string, []*basev1.AiAttachment, []ai.Attachment, []ai.Message, string, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	if session.AutomationRunID > 0 && automation.ExecutionFromContext(ctx).RunID != session.AutomationRunID {
		return nil, nil, "", nil, nil, nil, "", errorsx.PermissionDenied("自动化结果只读，请使用继续讨论")
	}
	if err = c.expireAiGeneratingMessages(ctx, session); err != nil {
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
	aiAttachments, err = c.buildAiAttachments(attachments)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var userName string
	userName, err = c.baseUserCase.FindUserNameByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var history []ai.Message
	history, err = c.buildHistory(ctx, session.TenantID, session.ID, aiHistorySize)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	selection := agentmodel.ModelOption{}
	if c.aiRuntime != nil {
		selection, err = c.aiRuntime.SelectModel(req.GetProviderId(), req.GetModelName())
		if err != nil {
			return nil, nil, "", nil, nil, nil, "", errorsx.InvalidArgument("AI供应商或模型选择无效").WithCause(err)
		}
	} else if req.GetProviderId() != 0 || req.GetModelName() != "" {
		return nil, nil, "", nil, nil, nil, "", errorsx.InvalidArgument("AI供应商或模型选择无效")
	}

	now := time.Now()
	inputContent := ai.InputContentPayload{
		Kind:             ai.KindText,
		Content:          ai.BuildUserContent(content, attachments, c.localizedAttachmentPrompt(ctx)),
		ProviderID:       selection.ProviderID,
		ProviderName:     selection.ProviderName,
		ModelName:        selection.ModelName,
		KnowledgeBaseIDs: req.GetKnowledgeBaseIds(),
	}
	message := &models.AiMessage{
		TenantID:      session.TenantID,
		SessionID:     session.ID,
		UserID:        session.UserID,
		InputContent:  ai.MarshalInputContentPayload(inputContent),
		OutputContent: ai.MarshalEmptyOutputContent(),
		Attachments:   ai.MarshalAttachments(attachments),
		Tools:         "[]",
		Token:         ai.MarshalTokenUsage(ai.TokenUsage{}),
		FirstTokenMs:  0,
		DurationMs:    0,
		Status:        int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var txErr error
		if txErr = c.aiSessionCase.lockAiSession(txCtx, session); txErr != nil {
			return txErr
		}
		if txErr = c.ensureAiSessionIdle(txCtx, session.TenantID, session.ID); txErr != nil {
			return txErr
		}
		// 与数据库 datetime(3) 精度一致，更新时间同时用于阻止旧请求回填。
		now = time.Now().Truncate(time.Millisecond)
		message.CreatedAt = now
		message.UpdatedAt = now
		if txErr = c.aiMessageRepo.Create(txCtx, message); txErr != nil {
			return txErr
		}
		defaultSummary := c.aiRuntime.LocalizeMessage(ctx, "base.ai.session.default_summary", nil, "New conversation")
		attachmentSummary := c.aiRuntime.LocalizeMessage(ctx, "base.ai.session.attachment_summary", map[string]any{"Count": len(attachments)}, fmt.Sprintf("%d attachment(s)", len(attachments)))
		summary := ai.BuildDynamicSummary(content, attachments, defaultSummary, attachmentSummary)
		return c.aiSessionCase.UpdateSessionSummary(txCtx, session, summary, now)
	})
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	return session, message, content, attachments, aiAttachments, history, userName, nil
}

// ensureAiSessionIdle 确认会话没有正在生成或等待确认的消息，调用方须持有会话锁。
func (c *AiMessageCase) ensureAiSessionIdle(ctx context.Context, tenantID int64, sessionID int64) error {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	active, err := query.WithContext(ctx).Where(query.TenantID.Eq(tenantID), query.SessionID.Eq(sessionID), query.Status.In(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING), int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))).Count()
	if err != nil {
		return err
	}
	if active > 0 {
		return errorsx.Conflict("请先等待当前回复完成或处理待确认操作")
	}
	return nil
}

// buildHistory 构造问答历史上下文。
func (c *AiMessageCase) buildHistory(ctx context.Context, tenantID int64, sessionID int64, historySize int) ([]ai.Message, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS))))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	history := buildHistoryFromMessages(list)
	c.fillLatestHistoryImageBytes(ctx, history)
	return history, nil
}

// regenerateAiMessage 使用已有输入重新生成当前轮次输出。
func (c *AiMessageCase) regenerateAiMessage(ctx context.Context, session *models.AiSession, message *models.AiMessage) (*basev1.SendAiMessageResponse, error) {
	input := ai.ParseInputContent(message.InputContent)
	content := input.Content
	return c.regenerateAiMessageWithContent(ctx, session, message, content)
}

// regenerateAiMessageWithContent 使用指定输入内容重新生成当前轮次输出。
func (c *AiMessageCase) regenerateAiMessageWithContent(ctx context.Context, session *models.AiSession, message *models.AiMessage, content string) (*basev1.SendAiMessageResponse, error) {
	if session.AutomationRunID > 0 {
		return nil, errorsx.PermissionDenied("自动化结果只读，请使用运行一次")
	}
	attachments := ai.ParseAttachments(message.Attachments)
	input := ai.ParseInputContent(message.InputContent)

	var err error
	var aiAttachments []ai.Attachment
	aiAttachments, err = c.buildAiAttachments(attachments)
	if err != nil {
		return nil, err
	}
	var userName string
	userName, err = c.baseUserCase.FindUserNameByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	var history []ai.Message
	history, err = c.buildHistoryBeforeMessage(ctx, session.TenantID, session.ID, message, aiHistorySize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	input.Content = ai.BuildUserContent(content, attachments, c.localizedAttachmentPrompt(ctx))
	if err = c.markAiMessageGenerating(ctx, session, message, input, now); err != nil {
		return nil, err
	}
	startAt := time.Now()
	var reply *ai.Response
	reply, err = c.generateAiReply(ctx, session, message.ID, userName, content, input.ProviderID, input.ProviderName, input.ModelName, nil, attachments, aiAttachments, history, input.KnowledgeBaseIDs, nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	status := int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
	if err != nil {
		reply = c.buildAiFailedReply(ctx, reply, err, input.ProviderName, input.ModelName)
		status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)
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
func (c *AiMessageCase) buildAiAttachments(attachments []*basev1.AiAttachment) ([]ai.Attachment, error) {
	if len(attachments) == 0 {
		return []ai.Attachment{}, nil
	}
	ossClient := c.OSS
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
			// 前端回传的是带 /data 前缀的浏览器访问路径，先转换为 OSS 对象路径再读取，避免根目录重复拼接。
			objectPath, err := objectFilePath(next.URL)
			if err != nil {
				return nil, err
			}
			var fileBytes []byte
			fileBytes, err = ossClient.GetFileByte(objectPath)
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
func (c *AiMessageCase) buildHistoryBeforeMessage(ctx context.Context, tenantID int64, sessionID int64, message *models.AiMessage, historySize int) ([]ai.Message, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS))))
	opts = append(opts, repository.Where(query.CreatedAt.Lt(message.CreatedAt)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	history := buildHistoryFromMessages(list)
	c.fillLatestHistoryImageBytes(ctx, history)
	return history, nil
}

// generateAiReply 生成当前消息的 AI 助手回复。
func (c *AiMessageCase) generateAiReply(
	ctx context.Context,
	session *models.AiSession,
	messageID int64,
	userName string,
	content string,
	providerID int64,
	providerName string,
	modelName string,
	action *basev1.AiAction,
	attachments []*basev1.AiAttachment,
	aiAttachments []ai.Attachment,
	history []ai.Message,
	knowledgeBaseIDs []int64,
	onDelta func(string),
) (*ai.Response, error) {
	var flowReply *ai.Response
	var handled bool
	var err error
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
			ProviderID:   providerID,
			ModelName:    modelName,
			Summary:      session.Summary,
			Content:      content,
			History:      history,
			Attachments:  aiAttachments,
		}
		if len(knowledgeBaseIDs) > 0 && c.aiKnowledgeEngine != nil {
			retrieveCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			if session.AutomationRunID > 0 {
				input.KnowledgeContext, err = c.aiKnowledgeEngine.RetrieveStrict(retrieveCtx, knowledgeBaseIDs, content)
			} else {
				input.KnowledgeContext = c.aiKnowledgeEngine.Retrieve(retrieveCtx, knowledgeBaseIDs, content, 0)
			}
			cancel()
			if err != nil {
				return nil, err
			}
		}
		var response *ai.Response
		if onDelta != nil {
			response, err = c.runAiRuntime(ctx, session, messageID, input, onDelta)
			if err == nil {
				return response, nil
			}
			return c.buildAiFallbackResponse(ctx, content, attachments, err, providerName, modelName), err
		}
		response, err = c.runAiRuntime(ctx, session, messageID, input, nil)
		if err == nil {
			return response, nil
		}
		log.Error(fmt.Sprintf("AI reply generation failed: %v", err))
		return c.buildAiFallbackResponse(ctx, content, attachments, err, providerName, modelName), err
	}
	err = errorsx.Internal("AI助手运行时未初始化")
	return c.buildAiFallbackResponse(ctx, content, attachments, err, providerName, modelName), err
}

// buildAiFailedReply 构造可展示和可排障的助手异常回复。
func (c *AiMessageCase) buildAiFailedReply(ctx context.Context, reply *ai.Response, cause error, providerName string, modelName string) *ai.Response {
	failedReply := reply
	if failedReply == nil {
		failedReply = c.buildAiFallbackResponse(ctx, "", nil, cause, providerName, modelName)
	}
	// 优先保留上游已记录的真实错误摘要，仅在完全缺失时兜底通用文案。
	reason := failedReply.FallbackReason
	if reason == "" {
		reason = i18n.EncodeMessage("system.ai.chat.error.details_unavailable", nil)
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
	ctx context.Context,
	content string,
	attachments []*basev1.AiAttachment,
	err error,
	providerName string,
	modelName string,
) *ai.Response {
	fallbackReason := ""
	if err != nil {
		// 错误详情落真实摘要供「错误详情」排障，截断防止整段堆栈入库。
		fallbackReason = truncateAiErrorDetail(err.Error())
	}
	modelLabel := ""
	if c != nil && c.aiRuntime != nil {
		modelLabel = c.aiRuntime.SelectedModelName(0, "")
	}
	if providerName != "" && modelName != "" {
		modelLabel = providerName + " / " + modelName
	}
	return &ai.Response{
		Content: ai.BuildFallbackReply(content, attachments, func(key string, args map[string]any, fallback string) string {
			if c == nil || c.aiRuntime == nil {
				return fallback
			}
			return c.aiRuntime.LocalizeMessage(ctx, key, args, fallback)
		}),
		Token:          ai.TokenUsage{},
		Tools:          []ai.ToolUsage{},
		Source:         "fallback",
		Model:          modelLabel,
		Fallback:       true,
		FallbackReason: fallbackReason,
	}
}

// finishAiMessage 在客户端断开后按本轮生成版本回填结果，拒绝过期请求覆盖新状态。
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), aiMessagePersistTimeout)
	defer cancel()
	now = now.Truncate(time.Millisecond)
	if reply != nil && reply.Approval != nil {
		status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL)
	}
	injectAiActionState(reply, message.ID)
	c.persistGeneratedToolFiles(ctx, message.TenantID, reply)
	outputContent := ai.MarshalReplyContent(reply)
	tools := ai.MarshalTools(nil)
	token := ai.MarshalTokenUsage(ai.TokenUsage{})
	attachmentsJSON := ""
	if reply != nil {
		tools = ai.MarshalTools(reply.Tools)
		token = ai.MarshalTokenUsage(reply.Token)
		if len(reply.Attachments) > 0 {
			// 生成附件追加在用户输入附件之后，保持同一条消息内附件完整可见。
			attachmentsJSON = ai.MarshalAttachments(slices.Concat(ai.ParseAttachments(message.Attachments), reply.Attachments))
		}
	}
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var updateErr error
		if execution := automation.ExecutionFromContext(txCtx); execution.RunID > 0 {
			if updateErr = c.automationCase.lockAutomationExecution(txCtx, execution, reply != nil && reply.Approval != nil); updateErr != nil {
				return updateErr
			}
		}
		if updateErr = c.aiSessionCase.lockAiSession(txCtx, session); updateErr != nil {
			return updateErr
		}
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		// 助手回复默认不改写附件列，避免覆盖用户发送消息时记录的输入附件。
		columns := []field.AssignExpr{
			query.OutputContent.Value(outputContent),
			query.Tools.Value(tools),
			query.Token.Value(token),
			query.FirstTokenMs.Value(firstTokenMs),
			query.DurationMs.Value(durationMs),
			query.Status.Value(status),
			query.UpdatedAt.Value(now),
		}
		if attachmentsJSON != "" {
			columns = append(columns, query.Attachments.Value(attachmentsJSON))
		}
		var result gen.ResultInfo
		result, updateErr = query.WithContext(txCtx).
			Where(query.TenantID.Eq(message.TenantID), query.ID.Eq(message.ID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)), query.UpdatedAt.Eq(message.UpdatedAt)).
			UpdateColumnSimple(columns...)
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
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
	if attachmentsJSON != "" {
		message.Attachments = attachmentsJSON
	}
	return nil
}

// generatedFileExtensions 表示允许转存的工具产物文档扩展名。
var generatedFileExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".csv": true, ".txt": true, ".md": true,
	".zip": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
}

// generatedFileURLPattern 匹配工具输出中的文件链接文本。
var generatedFileURLPattern = regexp.MustCompile(`https?://[^\s"'<>\\]+`)

// generatedFileClient 拉取外部生成文件的 HTTP 客户端。
var generatedFileClient = &http.Client{Timeout: 30 * time.Second}

// persistGeneratedToolFiles 识别工具输出中的文档产物并转存为系统持久文件。
// 外部转换服务的签名链接存在有效期，回复落库前把产物拉取到对象存储并登记为生成文件；
// 单个文件失败只记录日志并保留原始链接，不阻断回复落库。
func (c *AiMessageCase) persistGeneratedToolFiles(ctx context.Context, tenantID int64, reply *ai.Response) {
	if reply == nil || len(reply.Tools) == 0 || c.aiFileCase == nil || c.OSS == nil {
		return
	}
	for _, product := range collectGeneratedProducts(reply.Tools) {
		rawURL := product.downloadURL
		content, err := fetchGeneratedFile(ctx, rawURL)
		if err != nil {
			log.Warn("拉取AI生成文件失败，保留原始链接", "url", rawURL, "error", err)
			continue
		}
		fileName := product.filename
		extension := strings.ToLower(path.Ext(fileName))
		mimeType := http.DetectContentType(content)
		// 检测结果可能是通用二进制类型，按扩展名兜底修正为更准确的文档 MIME。
		if inferred := mime.TypeByExtension(extension); inferred != "" {
			mimeType = inferred
		}
		directory := path.Join(aiGeneratedFileDirectory, time.Now().Format("2006/01/02"))
		storedName := fmt.Sprintf("%d%s", id.GenSnowflakeID(), extension)
		if _, err = c.OSS.UploadByByte(storedName, directory, content); err != nil {
			log.Warn("转存AI生成文件失败，保留原始链接", "url", rawURL, "error", err)
			continue
		}
		objectPath := path.Join(directory, storedName)
		hash := sha256.Sum256(content)
		if err = c.aiFileCase.RegisterGeneratedFile(ctx, &models.AiFile{
			TenantID:    tenantID,
			FileName:    fileName,
			Extension:   strings.TrimPrefix(extension, "."),
			MimeType:    mimeType,
			Size:        int64(len(content)),
			LinkURL:     objectPath,
			ContentHash: hex.EncodeToString(hash[:]),
		}); err != nil {
			log.Warn("登记AI生成文件失败", "url", rawURL, "error", err)
			continue
		}
		persistentURL := publicFileURL(objectPath)
		reply.Attachments = append(reply.Attachments, &basev1.AiAttachment{
			Id:       id.NewGUIDv7NoHyphen(),
			Name:     fileName,
			Size:     int64(len(content)),
			Url:      persistentURL,
			MimeType: mimeType,
		})
		// 正文里的临时签名链接一并替换为持久地址，避免过期后无法访问。
		reply.Content = strings.ReplaceAll(reply.Content, rawURL, persistentURL)
	}
}

// generatedProduct 表示工具输出的单个文件产物。
type generatedProduct struct {
	downloadURL string
	filename    string
}

// collectGeneratedProducts 汇总工具输出里的文件产物并去重。
// 优先读取结构化产物 JSON（含 download_url/filename，如 Gotenberg 的无后缀签名链接），
// 再兜底扫描正文中带文档后缀的裸链接，拦截任意接口地址被误当文件拉取。
func collectGeneratedProducts(tools []agent.ToolUsage) []generatedProduct {
	seen := make(map[string]bool)
	products := make([]generatedProduct, 0, len(tools))
	add := func(rawURL, filename string) {
		cleanURL := trimGeneratedFileURL(rawURL)
		if cleanURL == "" || seen[cleanURL] {
			return
		}
		filename = path.Base(strings.ReplaceAll(filename, `\`, "/"))
		if filename == "" || filename == "." || filename == "/" {
			filename = "生成文件"
		}
		extension := strings.ToLower(path.Ext(filename))
		if !generatedFileExtensions[extension] {
			return
		}
		seen[cleanURL] = true
		products = append(products, generatedProduct{downloadURL: cleanURL, filename: filename})
	}
	for _, item := range tools {
		var parts []struct {
			Text string `json:"text"`
		}
		// 工具输出是 content parts 数组；解析失败时按纯文本整体处理。
		texts := []string{item.Output}
		if err := json.Unmarshal([]byte(item.Output), &parts); err == nil {
			texts = texts[:0]
			for _, part := range parts {
				texts = append(texts, part.Text)
			}
		}
		for _, text := range texts {
			var produced struct {
				DownloadURL string `json:"download_url"`
				Filename    string `json:"filename"`
			}
			if err := json.Unmarshal([]byte(text), &produced); err == nil && produced.DownloadURL != "" {
				add(produced.DownloadURL, produced.Filename)
				continue
			}
			for _, rawURL := range generatedFileURLPattern.FindAllString(text, -1) {
				base := trimGeneratedFileURL(rawURL)
				if query := strings.IndexByte(base, '?'); query >= 0 {
					base = base[:query]
				}
				add(rawURL, path.Base(base))
			}
		}
	}
	return products
}

// trimGeneratedFileURL 去掉链接尾部粘连的标点与引号。
func trimGeneratedFileURL(rawURL string) string {
	return strings.TrimRight(strings.TrimSpace(rawURL), ".,;，。）)】]'\"")
}

// fetchGeneratedFile 拉取外部生成文件字节，限制大小防止异常产物拖垮内存。
func fetchGeneratedFile(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := generatedFileClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("拉取生成文件返回状态 %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, aiGeneratedFileMaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > aiGeneratedFileMaxSize {
		return nil, errors.New("生成文件超过转存大小限制")
	}
	if len(content) == 0 {
		return nil, errors.New("生成文件内容为空")
	}
	return content, nil
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
	message, err = c.findLatestAiMessage(ctx, session.TenantID, session.ID, session.UserID)
	if err != nil {
		return err
	}
	if message.ID != sourceMessageID {
		return aiExpiredActionError(action.GetSourceMessageId(), strconv.FormatInt(message.ID, 10))
	}
	if message.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS) {
		return aiExpiredActionError(action.GetSourceMessageId(), strconv.Itoa(int(message.Status)))
	}
	outputContent := ai.ParseOutputContent(message.OutputContent)
	if !aiBlocksContainAction(outputContent.BlocksJSON, action) {
		return aiExpiredActionError(action.GetActionId(), "latest")
	}
	return nil
}

// findLatestAiMessage 查询会话中最后一轮消息。
func (c *AiMessageCase) findLatestAiMessage(ctx context.Context, tenantID int64, sessionID int64, userID int64) (*models.AiMessage, error) {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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

// markAiMessageGenerating 在会话锁内领取最新消息的新一轮生成版本。
func (c *AiMessageCase) markAiMessageGenerating(ctx context.Context, session *models.AiSession, message *models.AiMessage, input ai.InputContentPayload, now time.Time) error {
	now = now.Truncate(time.Millisecond)
	inputContent := ai.MarshalInputContentPayload(input)
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var txErr error
		if txErr = c.aiSessionCase.lockAiSession(txCtx, session); txErr != nil {
			return txErr
		}
		if txErr = c.ensureLastAiMessage(txCtx, session.TenantID, session.ID, message.ID); txErr != nil {
			return txErr
		}
		if txErr = c.ensureAiSessionIdle(txCtx, session.TenantID, session.ID); txErr != nil {
			return txErr
		}
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var result gen.ResultInfo
		result, txErr = query.WithContext(txCtx).
			Where(query.TenantID.Eq(message.TenantID), query.ID.Eq(message.ID), query.Status.Eq(message.Status)).
			UpdateColumnSimple(
				query.InputContent.Value(inputContent),
				query.OutputContent.Value(ai.MarshalEmptyOutputContent()),
				query.Tools.Value("[]"),
				query.Token.Value(ai.MarshalTokenUsage(ai.TokenUsage{})),
				query.FirstTokenMs.Value(0),
				query.DurationMs.Value(0),
				query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)),
				query.UpdatedAt.Value(now),
			)
		if txErr != nil {
			return txErr
		}
		if result.RowsAffected != 1 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
		}
		return nil
	})
	if err != nil {
		return err
	}
	message.InputContent = inputContent
	message.OutputContent = ai.MarshalEmptyOutputContent()
	message.Tools = "[]"
	message.Token = ai.MarshalTokenUsage(ai.TokenUsage{})
	message.FirstTokenMs = 0
	message.DurationMs = 0
	message.Status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)
	message.UpdatedAt = now
	return nil
}

// localizedAttachmentPrompt 按当前请求语言生成仅附件消息的默认正文。
func (c *AiMessageCase) localizedAttachmentPrompt(ctx context.Context) string {
	return c.aiRuntime.LocalizeMessage(ctx, "base.ai.message.attachment_prompt", nil, "Please analyze the attached content.")
}

// findCurrentUserMessage 终止过期生成后查询当前用户当前会话下的消息。
func (c *AiMessageCase) findCurrentUserMessage(ctx context.Context, rawSessionID string, rawMessageID string) (*models.AiMessage, *models.AiSession, error) {
	session, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, rawSessionID)
	if err != nil {
		return nil, nil, err
	}
	if err = c.expireAiGeneratingMessages(ctx, session); err != nil {
		return nil, nil, err
	}
	var messageID int64
	messageID, err = strconv.ParseInt(rawMessageID, 10, 64)
	if err != nil || messageID <= 0 {
		return nil, nil, errorsx.InvalidArgument("消息编号不合法")
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.ID.Eq(messageID)))
	opts = append(opts, repository.Where(query.TenantID.Eq(session.TenantID)))
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
func (c *AiMessageCase) ensureLastAiMessage(ctx context.Context, tenantID int64, sessionID int64, messageID int64) error {
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
		TenantId:      model.TenantID,
		InputContent:  toAiInputContent(inputContent),
		OutputContent: toAiOutputContent(outputContent),
		Attachments:   ai.ParseAttachments(model.Attachments),
		CreatedAt:     timestamppb.New(model.CreatedAt),
		Status:        basev1.AiMessageStatus(model.Status),
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
	output := &basev1.AiOutputContent{
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
	if value.Approval != nil {
		output.Approval = &basev1.AiToolApproval{Id: value.Approval.ID, ExpiresAt: timestamppb.New(value.Approval.ExpiresAt)}
		for _, call := range value.Approval.Calls {
			output.Approval.Calls = append(output.Approval.Calls, &basev1.AiApprovalTool{Name: call.Name, Title: call.Title, Arguments: call.Arguments})
		}
	}
	return output
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
	for _, item := range slices.Backward(list) {
		input := ai.ParseInputContent(item.InputContent)
		if input.Content != "" {
			history = append(history, ai.Message{
				Role:        ai.RoleUser,
				Content:     input.Content,
				Attachments: historyImageAttachments(item.Attachments),
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

// historyImageAttachments 解析用户历史消息的图片附件元数据，供多模态上下文挑选。
func historyImageAttachments(raw string) []ai.Attachment {
	images := make([]ai.Attachment, 0)
	for _, item := range ai.ParseAttachments(raw) {
		if item == nil || !strings.HasPrefix(item.MimeType, "image/") {
			continue
		}
		images = append(images, ai.Attachment{Name: item.Name, Size: item.Size, URL: item.Url, MIMEType: item.MimeType})
	}
	if len(images) == 0 {
		return nil
	}
	return images
}

// fillLatestHistoryImageBytes 将最近一条带图用户消息的图片字节读入历史上下文，
// 使多轮追问时模型仍能看到该图片；更早的历史图片不再重传，其信息已沉淀在历史回复文本中。
func (c *AiMessageCase) fillLatestHistoryImageBytes(ctx context.Context, history []ai.Message) {
	for index := len(history) - 1; index >= 0; index-- {
		item := history[index]
		if item.Role != ai.RoleUser || len(item.Attachments) == 0 {
			continue
		}
		for i := range item.Attachments {
			attachment := item.Attachments[i]
			objectPath, err := objectFilePath(attachment.URL)
			if err != nil {
				continue
			}
			content, err := c.OSS.GetFileByte(objectPath)
			if err != nil {
				log.Warn("读取历史对话图片失败，本轮多模态上下文将缺少该图片", "url", attachment.URL, "error", err)
				continue
			}
			attachment.Bytes = content
			item.Attachments[i] = attachment
		}
		return
	}
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

// truncateAiErrorDetail 按字符截断降级原因摘要，避免整段堆栈入库。
func truncateAiErrorDetail(detail string) string {
	const maxRunes = 500
	runes := []rune(detail)
	if len(runes) <= maxRunes {
		return detail
	}
	return string(runes[:maxRunes]) + "…"
}

// expireAiGeneratingMessages 在属主会话锁内终止过期生成，释放会话但不重放业务操作。
func (c *AiMessageCase) expireAiGeneratingMessages(ctx context.Context, session *models.AiSession) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		if err = c.aiSessionCase.lockAiSession(txCtx, session); err != nil {
			return err
		}
		now := time.Now().Truncate(time.Millisecond)
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var list []*models.AiMessage
		list, err = query.WithContext(txCtx).Where(query.TenantID.Eq(session.TenantID), query.SessionID.Eq(session.ID), query.UserID.Eq(session.UserID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING)), query.UpdatedAt.Lte(now.Add(-aiApprovalLifetime))).Find()
		if err != nil {
			return err
		}
		approval := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		for _, item := range list {
			runs := approval.WithContext(txCtx).Where(approval.TenantID.Eq(session.TenantID), approval.SessionID.Eq(session.ID), approval.UserID.Eq(session.UserID), approval.MessageID.Eq(item.ID), approval.Status.In(aiRunRunning, aiRunPending))
			var active int64
			active, err = runs.WithContext(txCtx).Where(approval.ExpiresAt.Gt(now)).Count()
			if err != nil {
				return err
			}
			if active > 0 {
				continue
			}
			_, err = runs.WithContext(txCtx).Where(approval.ExpiresAt.Lte(now)).UpdateColumnSimple(approval.Status.Value(aiRunExpired), approval.CheckpointData.Value([]byte{}), approval.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
			output := ai.MarshalReplyContent(&ai.Response{Content: c.aiRuntime.LocalizeMessage(ctx, "system.ai.chat.error.run_expired", nil, "Generation expired. This request will not resume automatically. Check the business result before retrying."), Fallback: true})
			_, err = query.WithContext(txCtx).Where(query.ID.Eq(item.ID), query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING))).UpdateColumnSimple(query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)), query.OutputContent.Value(output), query.UpdatedAt.Value(now))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// bindAiSessionTools 按共享会话的工具配置重建当前用户的请求池。
func (c *AiMessageCase) bindAiSessionTools(ctx context.Context, session *models.AiSession) (context.Context, error) {
	if c.aiPluginCase == nil {
		return ctx, nil
	}
	selections, err := parseAiSessionPlugins(session.Plugins)
	if err != nil {
		return ctx, err
	}
	terminal := ai.NormalizeTerminalString(session.Terminal)
	ctx = agent.ContextWithTerminal(ctx, terminal)
	var pool agent.ToolPool
	pool, err = c.aiPluginCase.ResolveToolPool(ctx, terminal, selections)
	if err != nil {
		return ctx, err
	}
	return agent.ContextWithToolPool(ctx, pool), nil
}

// validatePendingAiTools 在消费审批前重新验证当前用户权限和冻结配置，拒绝不消耗原许可。
func (c *AiMessageCase) validatePendingAiTools(ctx context.Context, session *models.AiSession, message *models.AiMessage) error {
	if c.aiPluginCase == nil {
		return nil
	}
	output := ai.ParseOutputContent(message.OutputContent)
	if output.Approval == nil {
		return errorsx.Conflict("审批已失效或已处理")
	}
	query := c.aiApprovalRepo.Query(ctx).AiToolApproval
	run, err := query.WithContext(ctx).Where(query.ApprovalID.Eq(output.Approval.ID), query.TenantID.Eq(session.TenantID), query.SessionID.Eq(session.ID), query.UserID.Eq(session.UserID), query.Status.Eq(aiRunPending)).First()
	if err != nil {
		return errorsx.Conflict("审批已失效或已处理").WithCause(err)
	}
	var state ai.RunState
	var calls []ai.ApprovalCall
	if err = json.Unmarshal([]byte(run.RuntimeState), &state); err != nil {
		return errorsx.Internal("审批配置损坏").WithCause(err)
	}
	if err = json.Unmarshal([]byte(run.PendingCalls), &calls); err != nil {
		return errorsx.Internal("审批参数损坏").WithCause(err)
	}
	if err = c.aiRuntime.ValidateApproval(ctx, state, calls); err != nil {
		return errorsx.PermissionDenied("工具配置或当前用户权限已变化，请重新发起").WithCause(err)
	}
	return nil
}
