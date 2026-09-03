package biz

import (
	"context"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	corebiz "github.com/liujitcn/kratos-core/biz"
	"gorm.io/gen/field"
)

// BasePolicyEvaluationLogCase 提供策略评估日志表的查询业务。
type BasePolicyEvaluationLogCase struct {
	*corebiz.BaseCase
	*data.BasePolicyEvaluationLogRepository
}

// NewBasePolicyEvaluationLogCase 创建策略评估日志查询业务实例。
func NewBasePolicyEvaluationLogCase(baseCase *corebiz.BaseCase, basePolicyEvaluationLogRepo *data.BasePolicyEvaluationLogRepository) *BasePolicyEvaluationLogCase {
	return &BasePolicyEvaluationLogCase{BaseCase: baseCase, BasePolicyEvaluationLogRepository: basePolicyEvaluationLogRepo}
}

// PageBasePolicyEvaluationLog 分页查询策略评估日志。
func (c *BasePolicyEvaluationLogCase) PageBasePolicyEvaluationLog(ctx context.Context, req *adminv1.PageBasePolicyEvaluationLogRequest) (*adminv1.PageBasePolicyEvaluationLogResponse, error) {
	query := c.Query(ctx).BasePolicyEvaluationLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.EvaluationType != nil {
		opts = append(opts, repository.Where(query.EvaluationType.Eq(int32(req.GetEvaluationType()))))
	}
	if req.Decision != nil {
		opts = append(opts, repository.Where(query.Decision.Eq(int32(req.GetDecision()))))
	}
	if req.GetResource() != "" {
		opts = append(opts, repository.Where(query.Resource.Like("%"+req.GetResource()+"%")))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.Resource.Like(keyword), query.Action.Like(keyword), query.RequestID.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BasePolicyEvaluationLog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BasePolicyEvaluationLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBasePolicyEvaluationLog(item))
	}
	return &adminv1.PageBasePolicyEvaluationLogResponse{BasePolicyEvaluationLogs: items, Total: int32(total)}, nil
}

// GetBasePolicyEvaluationLog 查询策略评估日志详情。
func (c *BasePolicyEvaluationLogCase) GetBasePolicyEvaluationLog(ctx context.Context, idText string) (*adminv1.BasePolicyEvaluationLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BasePolicyEvaluationLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BasePolicyEvaluationLog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBasePolicyEvaluationLog(item), nil
}

// listLogTrace 查询策略评估日志关联的审计记录。
func (c *BasePolicyEvaluationLogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BasePolicyEvaluationLog, error) {
	query := c.Query(ctx).BasePolicyEvaluationLog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBasePolicyEvaluationLog 转换策略评估日志响应。
func toBasePolicyEvaluationLog(item *models.BasePolicyEvaluationLog) *adminv1.BasePolicyEvaluationLog {
	return &adminv1.BasePolicyEvaluationLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RoleId: item.RoleID, RoleCode: item.RoleCode, RequestId: item.RequestID, TraceId: item.TraceID, ClientIp: item.ClientIP, Engine: item.Engine, EvaluationType: adminv1.BasePolicyEvaluationType(item.EvaluationType), Resource: item.Resource, Action: item.Action, Project: item.Project, Decision: adminv1.BasePolicyDecision(item.Decision), ReasonCode: item.ReasonCode, Reason: item.Reason, DurationMs: item.DurationMs, CandidateCount: item.CandidateCount, MatchedCount: item.MatchedCount, InputHash: item.InputHash, OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
