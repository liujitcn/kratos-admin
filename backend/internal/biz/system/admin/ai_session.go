package biz

import (
	"context"
	"errors"
	"strings"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/go-utils/set"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	ai "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
)

// AiSessionCase 管理AI会话与消息查询。
type AiSessionCase struct {
	*biz.BaseCase
	aiSessionRepo *data.AiSessionRepository
	aiMessageRepo *data.AiMessageRepository
	baseUserRepo  *data.BaseUserRepository
	recordMapper  *mapper.CopierMapper[adminv1.AiSessionRecord, models.AiSession]
}

// NewAiSessionCase 创建AI会话管理业务实例。
func NewAiSessionCase(baseCase *biz.BaseCase, aiSessionRepo *data.AiSessionRepository, aiMessageRepo *data.AiMessageRepository, baseUserRepo *data.BaseUserRepository) *AiSessionCase {
	return &AiSessionCase{
		BaseCase:      baseCase,
		aiSessionRepo: aiSessionRepo,
		aiMessageRepo: aiMessageRepo,
		baseUserRepo:  baseUserRepo,
		recordMapper:  mapper.NewCopierMapper[adminv1.AiSessionRecord, models.AiSession](),
	}
}

// aiSessionStat 会话消息轮次与Token聚合。
type aiSessionStat struct {
	messageCount int32
	inputTokens  int32
	outputTokens int32
	cacheTokens  int32
	totalTokens  int32
}

// aiSessionUserInfo 会话所属用户展示信息。
type aiSessionUserInfo struct {
	userName string
	nickName string
}

// PageAiSession 分页查询AI会话。
func (c *AiSessionCase) PageAiSession(ctx context.Context, req *adminv1.PageAiSessionRequest) (*adminv1.PageAiSessionResponse, error) {
	query := c.aiSessionRepo.Query(ctx).AiSession
	opts := []repository.QueryOption{repository.Order(query.UpdatedAt.Desc()), repository.Order(query.ID.Desc())}
	if title := strings.TrimSpace(req.GetTitle()); title != "" {
		opts = append(opts, repository.Where(query.Title.Like("%"+title+"%")))
	}
	if req.Terminal != nil {
		opts = append(opts, repository.Where(query.Terminal.Eq(int32(req.GetTerminal()))))
	}
	sessions, total, err := c.aiSessionRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	stats := c.sessionStats(ctx, sessions)
	users := c.sessionUsers(ctx, sessions)
	records := make([]*adminv1.AiSessionRecord, 0, len(sessions))
	for _, item := range sessions {
		records = append(records, c.toAiSessionRecord(item, stats[item.ID], users[item.UserID]))
	}
	return &adminv1.PageAiSessionResponse{Sessions: records, Total: int32(total)}, nil
}

// PageAiSessionMessage 分页查询AI会话消息。
func (c *AiSessionCase) PageAiSessionMessage(ctx context.Context, req *adminv1.PageAiSessionMessageRequest) (*adminv1.PageAiSessionMessageResponse, error) {
	if _, err := c.findSession(ctx, req.GetId()); err != nil {
		return nil, err
	}
	query := c.aiMessageRepo.Query(ctx).AiMessage
	opts := []repository.QueryOption{
		repository.Where(query.SessionID.Eq(req.GetId())),
		repository.Order(query.CreatedAt.Asc()),
		repository.Order(query.ID.Asc()),
	}
	messages, total, err := c.aiMessageRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	records := make([]*adminv1.AiSessionMessageRecord, 0, len(messages))
	for _, item := range messages {
		records = append(records, toAiSessionMessageRecord(item))
	}
	return &adminv1.PageAiSessionMessageResponse{Messages: records, Total: int32(total)}, nil
}

// findSession 查询会话，不存在时返回业务错误。
func (c *AiSessionCase) findSession(ctx context.Context, id int64) (*models.AiSession, error) {
	query := c.aiSessionRepo.Query(ctx).AiSession
	session, err := c.aiSessionRepo.Find(ctx, repository.Where(query.ID.Eq(id)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("AI会话不存在")
		}
		return nil, err
	}
	return session, nil
}

// sessionStats 聚合当前页会话的消息轮次与Token统计。
func (c *AiSessionCase) sessionStats(ctx context.Context, sessions []*models.AiSession) map[int64]aiSessionStat {
	stats := make(map[int64]aiSessionStat, len(sessions))
	if len(sessions) == 0 {
		return stats
	}
	sessionIDs := make([]int64, 0, len(sessions))
	for _, item := range sessions {
		sessionIDs = append(sessionIDs, item.ID)
		stats[item.ID] = aiSessionStat{}
	}
	query := c.aiMessageRepo.Query(ctx).AiMessage
	messages, err := c.aiMessageRepo.List(ctx, repository.Where(query.SessionID.In(sessionIDs...)))
	if err != nil {
		return stats
	}
	for _, item := range messages {
		stat := stats[item.SessionID]
		stat.messageCount++
		token := ai.ParseTokenUsage(item.Token)
		stat.inputTokens += token.Input
		stat.outputTokens += token.Output
		stat.cacheTokens += token.Cache
		stat.totalTokens += token.Total
		stats[item.SessionID] = stat
	}
	return stats
}

// sessionUsers 批量读取会话所属用户展示信息。
func (c *AiSessionCase) sessionUsers(ctx context.Context, sessions []*models.AiSession) map[int64]aiSessionUserInfo {
	users := make(map[int64]aiSessionUserInfo, len(sessions))
	if len(sessions) == 0 {
		return users
	}
	userIDs := set.New[int64]()
	for _, item := range sessions {
		userIDs.Add(item.UserID)
	}
	query := c.baseUserRepo.Query(ctx).BaseUser
	records, err := c.baseUserRepo.List(ctx, repository.Where(query.ID.In(userIDs.ToSlice()...)))
	if err != nil {
		return users
	}
	for _, item := range records {
		users[item.ID] = aiSessionUserInfo{userName: item.UserName, nickName: item.NickName}
	}
	return users
}

// toAiSessionRecord 转换会话模型与聚合信息到接口对象，用户信息与消息统计由调用方补充。
func (c *AiSessionCase) toAiSessionRecord(item *models.AiSession, stat aiSessionStat, user aiSessionUserInfo) *adminv1.AiSessionRecord {
	record := c.recordMapper.ToDTO(item)
	// 存库终端值可能不在枚举范围内，映射后统一归一化。
	record.Terminal = ai.NormalizeTerminalEnum(item.Terminal)
	record.UserName = user.userName
	record.NickName = user.nickName
	record.MessageCount = stat.messageCount
	record.InputTokens = stat.inputTokens
	record.OutputTokens = stat.outputTokens
	record.CacheTokens = stat.cacheTokens
	record.TotalTokens = stat.totalTokens
	return record
}

// toAiSessionMessageRecord 转换消息模型到接口对象。
func toAiSessionMessageRecord(item *models.AiMessage) *adminv1.AiSessionMessageRecord {
	input := ai.ParseInputContent(item.InputContent)
	output := ai.ParseOutputContent(item.OutputContent)
	token := ai.ParseTokenUsage(item.Token)
	attachments := ai.ParseAttachments(item.Attachments)
	record := &adminv1.AiSessionMessageRecord{
		Id:            item.ID,
		InputContent:  input.Content,
		OutputContent: output.Content,
		Model:         output.Model,
		Status:        basev1.AiMessageStatus(item.Status),
		InputTokens:   token.Input,
		OutputTokens:  token.Output,
		CacheTokens:   token.Cache,
		TotalTokens:   token.Total,
		FirstTokenMs:  item.FirstTokenMs,
		DurationMs:    item.DurationMs,
		CreatedAt:     item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	record.Attachments = make([]*adminv1.AiSessionMessageAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		record.Attachments = append(record.Attachments, &adminv1.AiSessionMessageAttachment{
			Name:     attachment.GetName(),
			Size:     attachment.GetSize(),
			Url:      attachment.GetUrl(),
			MimeType: attachment.GetMimeType(),
		})
	}
	return record
}
