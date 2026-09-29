package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
)

// AiQueryCase 管理智能问数记录查询。
type AiQueryCase struct {
	*biz.BaseCase
	aiQueryRepo  *data.AiQueryRepository
	baseUserRepo *data.BaseUserRepository
	recordMapper *mapper.CopierMapper[adminv1.AiQueryRecord, models.AiQuery]
}

// NewAiQueryCase 创建智能问数记录管理业务实例。
func NewAiQueryCase(baseCase *biz.BaseCase, aiQueryRepo *data.AiQueryRepository, baseUserRepo *data.BaseUserRepository) *AiQueryCase {
	return &AiQueryCase{
		BaseCase:     baseCase,
		aiQueryRepo:  aiQueryRepo,
		baseUserRepo: baseUserRepo,
		recordMapper: mapper.NewCopierMapper[adminv1.AiQueryRecord, models.AiQuery](),
	}
}

// PageAiQuery 分页查询智能问数记录。
func (c *AiQueryCase) PageAiQuery(ctx context.Context, req *adminv1.PageAiQueryRequest) (*adminv1.PageAiQueryResponse, error) {
	query := c.aiQueryRepo.Query(ctx).AiQuery
	opts := []repository.QueryOption{repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc())}
	if question := strings.TrimSpace(req.GetQuestion()); question != "" {
		opts = append(opts, repository.Where(query.Question.Like("%"+question+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	queries, total, err := c.aiQueryRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	users := c.queryUsers(ctx, queries)
	records := make([]*adminv1.AiQueryRecord, 0, len(queries))
	for _, item := range queries {
		records = append(records, c.toAiQueryRecord(item, users[item.UserID]))
	}
	return &adminv1.PageAiQueryResponse{Queries: records, Total: int32(total)}, nil
}

// GetAiQuery 查询智能问数记录详情。
func (c *AiQueryCase) GetAiQuery(ctx context.Context, req *adminv1.GetAiQueryRequest) (*adminv1.AiQueryRecord, error) {
	query := c.aiQueryRepo.Query(ctx).AiQuery
	record, err := c.aiQueryRepo.Find(ctx, repository.Where(query.ID.Eq(req.GetId())))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("智能问数记录不存在")
		}
		return nil, err
	}
	result := c.toAiQueryRecord(record, c.queryUserNames(ctx, []int64{record.UserID})[record.UserID])
	applyResultSnapshot(result, record.ResultSnapshot)
	return result, nil
}

// queryUsers 查询当前页记录所属用户的账号展示名。
func (c *AiQueryCase) queryUsers(ctx context.Context, queries []*models.AiQuery) map[int64]string {
	userIDs := make([]int64, 0, len(queries))
	for _, item := range queries {
		userIDs = append(userIDs, item.UserID)
	}
	return c.queryUserNames(ctx, userIDs)
}

// queryUserNames 批量查询用户账号，去重后按用户 ID 返回。
func (c *AiQueryCase) queryUserNames(ctx context.Context, userIDs []int64) map[int64]string {
	result := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	query := c.baseUserRepo.Query(ctx).BaseUser
	users, err := c.baseUserRepo.List(ctx, repository.Where(query.ID.In(userIDs...)))
	if err != nil {
		return result
	}
	for _, item := range users {
		result[item.ID] = item.UserName
	}
	return result
}

// toAiQueryRecord 将问数记录模型转换为管理端记录，账号展示名由调用方补充。
func (c *AiQueryCase) toAiQueryRecord(item *models.AiQuery, userName string) *adminv1.AiQueryRecord {
	record := c.recordMapper.ToDTO(item)
	record.UserName = userName
	return record
}

// applyResultSnapshot 将结果快照 JSON 解码到详情记录，解码失败时静默跳过。
func applyResultSnapshot(record *adminv1.AiQueryRecord, snapshotJSON string) {
	if strings.TrimSpace(snapshotJSON) == "" {
		return
	}
	var snapshot struct {
		Columns []string   `json:"columns"`
		Rows    [][]string `json:"rows"`
	}
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return
	}
	record.ResultColumns = snapshot.Columns
	rows := make([]*adminv1.AiQueryResultRow, 0, len(snapshot.Rows))
	for _, row := range snapshot.Rows {
		rows = append(rows, &adminv1.AiQueryResultRow{Values: row})
	}
	record.ResultRows = rows
}
