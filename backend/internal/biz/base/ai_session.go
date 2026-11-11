package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AiSessionCase 管理 AI 助手会话数据。
type AiSessionCase struct {
	*biz.BaseCase
	*data.AiSessionRepository
	tx                data.Transaction
	aiMessageRepo     *data.AiMessageRepository
	aiApprovalRepo    *data.AiToolApprovalRepository
	aiPluginCase      *AiPluginCase
	automationRunRepo *data.AiAutomationRunRepository
	catalog           *i18n.I18n
	mapper            *mapper.CopierMapper[basev1.AiSession, models.AiSession]
}

// NewAiSessionCase 创建 AI 助手会话业务实例。
func NewAiSessionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiSessionRepo *data.AiSessionRepository,
	aiMessageRepo *data.AiMessageRepository,
	aiApprovalRepo *data.AiToolApprovalRepository,
	catalog *i18n.I18n,
	aiPluginCase *AiPluginCase,
	automationRunRepo *data.AiAutomationRunRepository,
) *AiSessionCase {
	return &AiSessionCase{
		BaseCase:            baseCase,
		AiSessionRepository: aiSessionRepo,
		tx:                  tx,
		aiMessageRepo:       aiMessageRepo,
		aiApprovalRepo:      aiApprovalRepo,
		aiPluginCase:        aiPluginCase,
		automationRunRepo:   automationRunRepo,
		catalog:             catalog,
		mapper:              mapper.NewCopierMapper[basev1.AiSession, models.AiSession](),
	}
}

// ListAiSession 查询当前租户与用户在所有客户端共享的 AI 对话。
func (c *AiSessionCase) ListAiSession(ctx context.Context, req *basev1.ListAiSessionRequest) (*basev1.ListAiSessionResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).AiSession
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Order(query.UpdatedAt.Desc(), query.ID.Desc()))
	var list []*models.AiSession
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	sessions := make([]*basev1.AiSession, 0, len(list))
	for _, item := range list {
		sessions = append(sessions, c.ToDTO(item))
	}
	return &basev1.ListAiSessionResponse{Sessions: sessions}, nil
}

// CreateAiSession 创建当前用户的新会话。
func (c *AiSessionCase) CreateAiSession(ctx context.Context, req *basev1.CreateAiSessionRequest) (*basev1.AiSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := ai.NormalizeTerminal(req.GetTerminal())
	selections := req.Plugins
	if selections == nil {
		selections = []*basev1.AiPluginSelection{{PluginId: "1"}}
	}
	_, err = c.aiPluginCase.ResolveToolPool(ctx, ai.NormalizeTerminalString(terminal), selections)
	if err != nil {
		return nil, err
	}
	var raw []byte
	raw, err = json.Marshal(selections)
	if err != nil {
		return nil, err
	}

	title := req.GetTitle()
	if title == "" {
		title = localizedAiSessionText(c.catalog, biz.LocaleFromContext(ctx), "base.ai.session.default_title", "New conversation")
	}
	summary := localizedAiSessionText(c.catalog, biz.LocaleFromContext(ctx), "base.ai.session.default_summary", "New conversation")
	now := time.Now()
	model := &models.AiSession{
		TenantID:  authInfo.TenantId,
		UserID:    authInfo.UserId,
		Terminal:  terminal,
		Plugins:   string(raw),
		Title:     title,
		Summary:   summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = c.Create(ctx, model); err != nil {
		return nil, err
	}
	return c.ToDTO(model), nil
}

// CreateAiSessionBranch 从指定消息创建当前用户的新分支会话。
func (c *AiSessionCase) CreateAiSessionBranch(ctx context.Context, req *basev1.CreateAiSessionBranchRequest) (*basev1.CreateAiSessionBranchResponse, error) {
	sourceSession, err := c.FindCurrentUserSessionByRawID(ctx, req.GetSourceSessionId())
	if err != nil {
		return nil, err
	}
	if sourceSession.AutomationRunID > 0 {
		return nil, errorsx.PermissionDenied("自动化结果已冻结，请使用继续讨论创建普通会话")
	}
	terminal := ai.NormalizeTerminal(req.GetTerminal())
	var selections []*basev1.AiPluginSelection
	selections, err = c.Plugins(sourceSession)
	if err != nil {
		return nil, err
	}
	_, err = c.aiPluginCase.ResolveToolPool(ctx, ai.NormalizeTerminalString(terminal), selections)
	if err != nil {
		return nil, err
	}

	var anchorMessageID int64
	anchorMessageID, err = strconv.ParseInt(req.GetAnchorMessageId(), 10, 64)
	if err != nil || anchorMessageID <= 0 {
		return nil, errorsx.InvalidArgument("分支锚点消息编号不合法")
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.ID.Eq(anchorMessageID)))
	opts = append(opts, repository.Where(query.TenantID.Eq(sourceSession.TenantID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(sourceSession.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(sourceSession.UserID)))
	var anchorMessage *models.AiMessage
	anchorMessage, err = c.aiMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("分支锚点消息不存在")
		}
		return nil, err
	}

	messageOpts := make([]repository.QueryOption, 0, 7)
	messageOpts = append(messageOpts, repository.Where(query.TenantID.Eq(sourceSession.TenantID)))
	messageOpts = append(messageOpts, repository.Where(query.SessionID.Eq(sourceSession.ID)))
	messageOpts = append(messageOpts, repository.Where(query.UserID.Eq(sourceSession.UserID)))
	messageOpts = append(messageOpts, repository.Where(query.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS))))
	messageOpts = append(messageOpts, repository.Where(query.CreatedAt.Lte(anchorMessage.CreatedAt)))
	messageOpts = append(messageOpts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	var sourceMessages []*models.AiMessage
	sourceMessages, err = c.aiMessageRepo.List(ctx, messageOpts...)
	if err != nil {
		return nil, err
	}
	if len(sourceMessages) == 0 {
		return nil, errorsx.ResourceNotFound("分支消息不存在")
	}

	now := time.Now()
	title := req.GetTitle()
	if title == "" {
		title = localizedAiSessionText(c.catalog, biz.LocaleFromContext(ctx), "base.ai.session.branch_title", "Branch conversation")
	}
	plugins := sourceSession.Plugins
	if plugins == "" {
		plugins = "null"
	}
	branchSession := &models.AiSession{
		TenantID:  sourceSession.TenantID,
		UserID:    sourceSession.UserID,
		Terminal:  terminal,
		Plugins:   plugins,
		Title:     title,
		Summary:   sourceSession.Summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	branchMessages := make([]*models.AiMessage, 0, len(sourceMessages))
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.Create(txCtx, branchSession); err != nil {
			return err
		}
		for _, item := range sourceMessages {
			branchMessage := &models.AiMessage{
				TenantID:      branchSession.TenantID,
				SessionID:     branchSession.ID,
				UserID:        branchSession.UserID,
				InputContent:  item.InputContent,
				OutputContent: item.OutputContent,
				Attachments:   item.Attachments,
				Tools:         item.Tools,
				Token:         item.Token,
				FirstTokenMs:  item.FirstTokenMs,
				DurationMs:    item.DurationMs,
				Status:        item.Status,
				CreatedAt:     item.CreatedAt,
				UpdatedAt:     now,
			}
			normalizeAiMessageJSON(branchMessage)
			if err = c.aiMessageRepo.Create(txCtx, branchMessage); err != nil {
				return err
			}
			branchMessages = append(branchMessages, branchMessage)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiMessage, 0, len(branchMessages))
	for _, item := range branchMessages {
		messages = append(messages, toAiMessageDTO(item))
	}
	return &basev1.CreateAiSessionBranchResponse{
		Session:  c.ToDTO(branchSession),
		Messages: messages,
	}, nil
}

// UpdateAiSession 更新当前用户的会话标题。
func (c *AiSessionCase) UpdateAiSession(ctx context.Context, req *basev1.UpdateAiSessionRequest) (*basev1.AiSession, error) {
	title := req.GetTitle()
	session, err := c.FindCurrentUserSessionByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if session.AutomationRunID > 0 {
		return nil, errorsx.PermissionDenied("自动化结果会话不允许重命名")
	}

	now := time.Now()
	query := c.Query(ctx).AiSession
	_, err = query.WithContext(ctx).
		Where(query.TenantID.Eq(session.TenantID), query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Title.Value(title),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return nil, err
	}
	session.Title = title
	session.UpdatedAt = now
	return c.ToDTO(session), nil
}

// UpdateSessionSummary 更新会话摘要与更新时间。
func (c *AiSessionCase) UpdateSessionSummary(ctx context.Context, session *models.AiSession, summary string, now time.Time) error {
	query := c.Query(ctx).AiSession
	_, err := query.WithContext(ctx).
		Where(query.TenantID.Eq(session.TenantID), query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Summary.Value(summary),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.Summary = summary
	session.UpdatedAt = now
	return nil
}

// DeleteAiSession 在会话锁内作废未完成运行、清空全部审批快照并删除当前用户的会话。
func (c *AiSessionCase) DeleteAiSession(ctx context.Context, req *basev1.DeleteAiSessionRequest) (*emptypb.Empty, error) {
	session, err := c.FindCurrentUserSessionByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.lockAiSession(txCtx, session); err != nil {
			return err
		}
		if session.AutomationRunID > 0 {
			query := c.automationRunRepo.Query(txCtx).AiAutomationRun
			var count int64
			count, err = c.automationRunRepo.Count(txCtx, repository.Where(query.ID.Eq(session.AutomationRunID)), repository.Where(query.TenantID.Eq(session.TenantID)), repository.Where(query.OwnerUserID.Eq(session.UserID)), repository.Where(query.Status.In("queued", "running", "waiting_approval")))
			if err != nil {
				return err
			}
			if count > 0 {
				return errorsx.Conflict("自动化仍在执行或等待确认，不能删除结果会话")
			}
			return c.DeleteByID(txCtx, session.ID)
		}

		approval := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		runs := approval.WithContext(txCtx).Where(approval.TenantID.Eq(session.TenantID), approval.SessionID.Eq(session.ID), approval.UserID.Eq(session.UserID))
		_, err = runs.WithContext(txCtx).Where(approval.Status.In(aiRunRunning, aiRunPending)).UpdateSimple(approval.Status.Value(aiRunRejected))
		if err != nil {
			return err
		}
		_, err = runs.WithContext(txCtx).UpdateSimple(approval.CheckpointData.Value([]byte{}))
		if err != nil {
			return err
		}
		query := c.Query(txCtx).AiSession
		opts := []repository.QueryOption{
			repository.Where(query.TenantID.Eq(session.TenantID)),
			repository.Where(query.ID.Eq(session.ID)),
			repository.Where(query.UserID.Eq(session.UserID)),
		}
		return c.Delete(txCtx, opts...)
	})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// lockAiSession 串行化会话内的消息、审批和删除操作，调用方须位于事务中。
func (c *AiSessionCase) lockAiSession(ctx context.Context, session *models.AiSession) error {
	query := c.Query(ctx).AiSession
	locked, err := query.WithContext(ctx).Where(query.ID.Eq(session.ID), query.TenantID.Eq(session.TenantID), query.UserID.Eq(session.UserID)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
	if err != nil {
		return err
	}
	*session = *locked
	return nil
}

// FindCurrentUserSessionByRawID 按当前用户与字符串会话编号查询会话。
func (c *AiSessionCase) FindCurrentUserSessionByRawID(ctx context.Context, rawID string) (*models.AiSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var sessionID int64
	sessionID, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || sessionID <= 0 {
		return nil, errorsx.InvalidArgument("会话编号不合法")
	}

	query := c.Query(ctx).AiSession
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)))
	opts = append(opts, repository.Where(query.ID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var session *models.AiSession
	session, err = c.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("会话不存在")
		}
		return nil, err
	}
	return session, nil
}

// RefreshSessionUpdatedAt 更新会话更新时间。
func (c *AiSessionCase) RefreshSessionUpdatedAt(ctx context.Context, session *models.AiSession, now time.Time) error {
	query := c.Query(ctx).AiSession
	_, err := query.WithContext(ctx).
		Where(query.TenantID.Eq(session.TenantID), query.ID.Eq(session.ID)).
		UpdateSimple(
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.UpdatedAt = now
	return nil
}

// ToDTO 转换会话模型到接口对象。
func (c *AiSessionCase) ToDTO(model *models.AiSession) *basev1.AiSession {
	if model == nil {
		return nil
	}
	session := c.mapper.ToDTO(model)
	session.Id = strconv.FormatInt(model.ID, 10)
	session.UpdatedAt = timestamppb.New(model.UpdatedAt)
	session.Terminal = ai.NormalizeTerminalEnum(model.Terminal)
	selections, err := c.Plugins(model)
	if err != nil {
		log.Error("读取会话插件配置失败", "session_id", model.ID, "error", err)
		selections = []*basev1.AiPluginSelection{}
	}
	session.Plugins = selections
	session.AutomationId = ""
	if model.AutomationID > 0 {
		session.AutomationId = strconv.FormatInt(model.AutomationID, 10)
	}
	session.AutomationRunId = ""
	if model.AutomationRunID > 0 {
		session.AutomationRunId = strconv.FormatInt(model.AutomationRunID, 10)
	}
	session.ReadOnly = model.AutomationRunID > 0

	return session
}

// SetAiSessionPlugins 仅在本人普通会话空闲时保存明确选择；空列表明确禁用全部插件。
func (c *AiSessionCase) SetAiSessionPlugins(ctx context.Context, req *basev1.SetAiSessionPluginsRequest) (*basev1.AiSession, error) {
	session, err := c.FindCurrentUserSessionByRawID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	selections := req.Plugins
	if selections == nil {
		selections = []*basev1.AiPluginSelection{}
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.lockAiSession(txCtx, session)
		if err != nil {
			return err
		}
		if session.AutomationRunID > 0 {
			return errorsx.PermissionDenied("自动化结果会话不允许修改插件")
		}
		query := c.aiMessageRepo.Query(txCtx).AiMessage
		var count int64
		count, err = c.aiMessageRepo.Count(txCtx, repository.Where(query.TenantID.Eq(session.TenantID)), repository.Where(query.SessionID.Eq(session.ID)), repository.Where(query.UserID.Eq(session.UserID)), repository.Where(query.Status.In(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING), int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL))))
		if err != nil {
			return err
		}
		if count > 0 {
			return errorsx.Conflict("会话仍在执行或等待确认，暂不能修改插件")
		}
		approvalQuery := c.aiApprovalRepo.Query(txCtx).AiToolApproval
		count, err = c.aiApprovalRepo.Count(txCtx, repository.Where(approvalQuery.SessionID.Eq(session.ID)), repository.Where(approvalQuery.TenantID.Eq(session.TenantID)), repository.Where(approvalQuery.UserID.Eq(session.UserID)), repository.Where(approvalQuery.Status.In(aiRunRunning, aiRunPending)), repository.Where(approvalQuery.ExpiresAt.Gt(time.Now())))
		if err != nil {
			return err
		}
		if count > 0 {
			return errorsx.Conflict("会话仍有未完成运行，暂不能修改插件")
		}
		_, err = c.aiPluginCase.ResolveToolPool(txCtx, ai.NormalizeTerminalString(session.Terminal), selections)
		if err != nil {
			return err
		}
		var raw []byte
		raw, err = json.Marshal(selections)
		if err != nil {
			return err
		}
		session.Plugins = string(raw)
		session.UpdatedAt = time.Now()
		sessionQuery := c.Query(txCtx).AiSession
		return c.Update(txCtx, session, repository.Where(sessionQuery.ID.Eq(session.ID)), repository.Select(sessionQuery.Plugins, sessionQuery.UpdatedAt))
	})
	if err != nil {
		return nil, err
	}
	return c.ToDTO(session), nil
}

// Plugins 读取会话选择，旧NULL仅映射内置工具；显式空数组保持禁用全部。
func (c *AiSessionCase) Plugins(session *models.AiSession) ([]*basev1.AiPluginSelection, error) {
	return parseAiSessionPlugins(session.Plugins)
}

// parseAiSessionPlugins 对持久化JSON失败关闭，不把损坏配置回退为全部工具。
func parseAiSessionPlugins(raw string) ([]*basev1.AiPluginSelection, error) {
	if raw == "" || raw == "null" {
		return []*basev1.AiPluginSelection{{PluginId: "1"}}, nil
	}
	selections := make([]*basev1.AiPluginSelection, 0)
	err := json.Unmarshal([]byte(raw), &selections)
	if err != nil {
		return nil, errorsx.Internal("会话插件配置损坏").WithCause(err)
	}
	for _, selection := range selections {
		if selection == nil {
			return nil, errorsx.Internal("会话插件引用无效")
		}
	}
	return selections, nil
}

// localizedAiSessionText 根据当前请求语言读取会话默认文案。
func localizedAiSessionText(catalog *i18n.I18n, locale, key, fallback string) string {
	return catalog.Localize(locale, "zh-CN", key, nil, fallback)
}

// normalizeAiMessageJSON 为分支复制的历史消息补齐可入库的 JSON 默认值。
func normalizeAiMessageJSON(item *models.AiMessage) {
	if item.InputContent == "" {
		item.InputContent = ai.MarshalInputContentPayload(ai.InputContentPayload{Kind: ai.KindText})
	}
	if item.OutputContent == "" {
		item.OutputContent = ai.MarshalEmptyOutputContent()
	}
	if item.Attachments == "" {
		item.Attachments = "[]"
	}
	if item.Tools == "" {
		item.Tools = "[]"
	}
	if item.Token == "" {
		item.Token = ai.MarshalTokenUsage(ai.TokenUsage{})
	}
}
