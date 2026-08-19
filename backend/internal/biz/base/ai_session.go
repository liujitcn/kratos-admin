package biz

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// AiSessionCase 管理 AI 助手会话数据。
type AiSessionCase struct {
	*biz.BaseCase
	*data.AiSessionRepository
	tx            data.Transaction
	aiMessageRepo *data.AiMessageRepository
	mapper        *mapper.CopierMapper[basev1.AiSession, models.AiSession]
}

// NewAiSessionCase 创建 AI 助手会话业务实例。
func NewAiSessionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiSessionRepo *data.AiSessionRepository,
	aiMessageRepo *data.AiMessageRepository,
) *AiSessionCase {
	return &AiSessionCase{
		BaseCase:            baseCase,
		AiSessionRepository: aiSessionRepo,
		tx:                  tx,
		aiMessageRepo:       aiMessageRepo,
		mapper:              mapper.NewCopierMapper[basev1.AiSession, models.AiSession](),
	}
}

// ListAiSession 查询当前用户的 AI 助手会话列表。
func (c *AiSessionCase) ListAiSession(ctx context.Context, req *basev1.ListAiSessionRequest) (*basev1.ListAiSessionResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := ai.NormalizeTerminal(req.GetTerminal())
	query := c.Query(ctx).AiSession
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.Terminal.Eq(terminal)))
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

	title := req.GetTitle()
	if title == "" {
		title = "新对话"
	}
	now := time.Now()
	model := &models.AiSession{
		UserID:    authInfo.UserId,
		Terminal:  ai.NormalizeTerminal(req.GetTerminal()),
		Title:     title,
		Summary:   ai.BuildDefaultSummary(),
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
	var anchorMessageID int64
	anchorMessageID, err = strconv.ParseInt(req.GetAnchorMessageId(), 10, 64)
	if err != nil || anchorMessageID <= 0 {
		return nil, errorsx.InvalidArgument("分支锚点消息编号不合法")
	}

	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.ID.Eq(anchorMessageID)))
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

	messageOpts := make([]repository.QueryOption, 0, 6)
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
		title = "分支会话"
	}
	branchSession := &models.AiSession{
		UserID:    sourceSession.UserID,
		Terminal:  ai.NormalizeTerminal(req.GetTerminal()),
		Title:     title,
		Summary:   sourceSession.Summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	branchMessages := make([]*models.AiMessage, 0, len(sourceMessages))
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if createErr := c.Create(txCtx, branchSession); createErr != nil {
			return createErr
		}
		for _, item := range sourceMessages {
			branchMessage := &models.AiMessage{
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
			if createErr := c.aiMessageRepo.Create(txCtx, branchMessage); createErr != nil {
				return createErr
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
	now := time.Now()
	query := c.Query(ctx).AiSession
	_, err = query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
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
		Where(query.ID.Eq(session.ID)).
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

// DeleteAiSession 删除当前用户的会话。
func (c *AiSessionCase) DeleteAiSession(ctx context.Context, req *basev1.DeleteAiSessionRequest) (*emptypb.Empty, error) {
	session, err := c.FindCurrentUserSessionByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiSession
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(session.ID)))
	if err = c.Delete(ctx, opts...); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
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
	opts := make([]repository.QueryOption, 0, 2)
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
		Where(query.ID.Eq(session.ID)).
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
	return session
}
