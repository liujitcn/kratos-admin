package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
)

// BiQueryRecordCase 管理当前租户和项目范围内的 BI 查询记录。
type BiQueryRecordCase struct {
	*biz.BaseCase
	biQueryRepo  *data.BiQueryRepository
	baseUserRepo *data.BaseUserRepository
	recordMapper *mapper.CopierMapper[adminv1.BiQueryRecord, models.BiQuery]
}

// NewBiQueryRecordCase 创建智能问数记录管理业务实例。
func NewBiQueryRecordCase(baseCase *biz.BaseCase, biQueryRepo *data.BiQueryRepository, baseUserRepo *data.BaseUserRepository) *BiQueryRecordCase {
	return &BiQueryRecordCase{
		BaseCase:     baseCase,
		biQueryRepo:  biQueryRepo,
		baseUserRepo: baseUserRepo,
		recordMapper: mapper.NewCopierMapper[adminv1.BiQueryRecord, models.BiQuery](),
	}
}

// PageBiQuery 分页查询当前访问范围内的 BI 查询记录。
func (c *BiQueryRecordCase) PageBiQuery(ctx context.Context, req *adminv1.PageBiQueryRequest) (*adminv1.PageBiQueryResponse, error) {
	opts, err := c.scopeOptions(ctx)
	if err != nil {
		return nil, err
	}
	query := c.biQueryRepo.Query(ctx).BiQuery
	opts = append(opts, repository.Where(query.Question.Neq("")), repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc()))
	if question := strings.TrimSpace(req.GetQuestion()); question != "" {
		opts = append(opts, repository.Where(query.Question.Like("%"+question+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.GetSourceCode() != "" {
		opts = append(opts, repository.Where(query.SourceCode.Eq(req.GetSourceCode())))
	}
	var queries []*models.BiQuery
	var total int64
	queries, total, err = c.biQueryRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	users := c.queryUsers(ctx, queries)
	records := make([]*adminv1.BiQueryRecord, 0, len(queries))
	for _, item := range queries {
		records = append(records, c.toBiQueryRecord(item, users[item.UserID]))
	}
	return &adminv1.PageBiQueryResponse{Queries: records, Total: int32(total)}, nil
}

// GetBiQuery 查询当前访问范围内的 BI 查询记录详情。
func (c *BiQueryRecordCase) GetBiQuery(ctx context.Context, req *adminv1.GetBiQueryRequest) (*adminv1.BiQueryRecord, error) {
	opts, err := c.scopeOptions(ctx)
	if err != nil {
		return nil, err
	}
	query := c.biQueryRepo.Query(ctx).BiQuery
	opts = append(opts, repository.Where(query.ID.Eq(req.GetId())))
	var record *models.BiQuery
	record, err = c.biQueryRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("智能问数记录不存在")
		}
		return nil, err
	}
	result := c.toBiQueryRecord(record, c.queryUserNames(ctx, []int64{record.UserID})[record.UserID])
	var snapshot bi.QueryResult
	if record.Status == int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS) {
		if err = json.Unmarshal([]byte(record.ResultSnapshot), &snapshot); err != nil {
			return nil, errorsx.Internal("读取查询结果快照失败").WithCause(err)
		}
		snapshot.QueryRecordID = record.ID
		result.Result = toProtoQueryResult(snapshot)
	}
	return result, nil
}

// scopeOptions 限制查询记录的租户和项目范围，避免结果快照跨范围泄露。
func (c *BiQueryRecordCase) scopeOptions(ctx context.Context) ([]repository.QueryOption, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var projectID int64
	projectID, err = projectIDForAuth(authInfo)
	if err != nil {
		return nil, err
	}
	query := c.biQueryRepo.Query(ctx).BiQuery
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(authInfo.TenantId), query.UserID.Eq(authInfo.UserId))}
	if projectID != 0 {
		opts = append(opts, repository.Where(query.ProjectID.Eq(projectID)))
	}
	return opts, nil
}

// queryUsers 查询当前页记录所属用户的账号展示名。
func (c *BiQueryRecordCase) queryUsers(ctx context.Context, queries []*models.BiQuery) map[int64]string {
	userIDs := make([]int64, 0, len(queries))
	for _, item := range queries {
		userIDs = append(userIDs, item.UserID)
	}
	return c.queryUserNames(ctx, userIDs)
}

// queryUserNames 批量查询用户账号，去重后按用户 ID 返回。
func (c *BiQueryRecordCase) queryUserNames(ctx context.Context, userIDs []int64) map[int64]string {
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

// toBiQueryRecord 将问数记录模型转换为管理端记录，账号展示名由调用方补充。
func (c *BiQueryRecordCase) toBiQueryRecord(item *models.BiQuery, userName string) *adminv1.BiQueryRecord {
	record := c.recordMapper.ToDTO(item)
	record.UserName = userName
	return record
}
