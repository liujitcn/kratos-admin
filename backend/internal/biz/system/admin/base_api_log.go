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

// BaseAPILogCase 提供 API 访问日志表的查询业务。
type BaseAPILogCase struct {
	*corebiz.BaseCase
	*data.BaseAPILogRepository
}

// NewBaseAPILogCase 创建 API 访问日志查询业务实例。
func NewBaseAPILogCase(baseCase *corebiz.BaseCase, baseAPILogRepo *data.BaseAPILogRepository) *BaseAPILogCase {
	return &BaseAPILogCase{BaseCase: baseCase, BaseAPILogRepository: baseAPILogRepo}
}

// PageBaseApiLog 分页查询 API 访问日志。
func (c *BaseAPILogCase) PageBaseApiLog(ctx context.Context, req *adminv1.PageBaseApiLogRequest) (*adminv1.PageBaseApiLogResponse, error) {
	query := c.Query(ctx).BaseAPILog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.Result != nil {
		opts = append(opts, repository.Where(query.Result.Eq(int32(req.GetResult()))))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.Operation.Like(keyword), query.Path.Like(keyword), query.RequestID.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BaseAPILog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseApiLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseApiLog(item))
	}
	return &adminv1.PageBaseApiLogResponse{BaseApiLogs: items, Total: int32(total)}, nil
}

// GetBaseApiLog 查询 API 访问日志详情。
func (c *BaseAPILogCase) GetBaseApiLog(ctx context.Context, idText string) (*adminv1.BaseApiLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseAPILog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BaseAPILog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseApiLog(item), nil
}

// listLogTrace 查询 API 访问日志关联的审计记录。
func (c *BaseAPILogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BaseAPILog, error) {
	query := c.Query(ctx).BaseAPILog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBaseApiLog 转换 API 访问日志响应。
func toBaseApiLog(item *models.BaseAPILog) *adminv1.BaseApiLog {
	return &adminv1.BaseApiLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ServiceName: item.ServiceName, Operation: item.Operation, Method: item.Method, Path: item.Path, StatusCode: item.StatusCode, Result: adminv1.BaseLogResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, LatencyMs: item.LatencyMs, RequestSize: item.RequestSize, ResponseSize: item.ResponseSize, ClientIp: item.ClientIP, UserAgent: item.UserAgent, OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
