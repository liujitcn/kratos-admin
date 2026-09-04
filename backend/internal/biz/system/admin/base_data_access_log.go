package biz

import (
	"context"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"gorm.io/gen/field"
)

// BaseDataAccessLogCase 提供数据访问日志表的查询业务。
type BaseDataAccessLogCase struct {
	*biz.BaseCase
	*data.BaseDataAccessLogRepository
}

// NewBaseDataAccessLogCase 创建数据访问日志查询业务实例。
func NewBaseDataAccessLogCase(baseCase *biz.BaseCase, baseDataAccessLogRepo *data.BaseDataAccessLogRepository) *BaseDataAccessLogCase {
	return &BaseDataAccessLogCase{BaseCase: baseCase, BaseDataAccessLogRepository: baseDataAccessLogRepo}
}

// PageBaseDataAccessLog 分页查询数据访问日志。
func (c *BaseDataAccessLogCase) PageBaseDataAccessLog(ctx context.Context, req *adminv1.PageBaseDataAccessLogRequest) (*adminv1.PageBaseDataAccessLogResponse, error) {
	query := c.Query(ctx).BaseDataAccessLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.AccessType != nil {
		opts = append(opts, repository.Where(query.AccessType.Eq(int32(req.GetAccessType()))))
	}
	if req.Sensitive != nil {
		sensitive := int32(0)
		if req.GetSensitive() {
			sensitive = 1
		}
		opts = append(opts, repository.Where(query.Sensitive.Eq(sensitive)))
	}
	if req.Result != nil {
		opts = append(opts, repository.Where(query.Result.Eq(int32(req.GetResult()))))
	}
	if req.GetResourceType() != "" {
		opts = append(opts, repository.Where(query.ResourceType.Eq(req.GetResourceType())))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.TableName_.Like(keyword), query.ResourceID.Like(keyword), query.RequestID.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BaseDataAccessLog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseDataAccessLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseDataAccessLog(item))
	}
	return &adminv1.PageBaseDataAccessLogResponse{BaseDataAccessLogs: items, Total: int32(total)}, nil
}

// GetBaseDataAccessLog 查询数据访问日志详情。
func (c *BaseDataAccessLogCase) GetBaseDataAccessLog(ctx context.Context, idText string) (*adminv1.BaseDataAccessLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseDataAccessLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BaseDataAccessLog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseDataAccessLog(item), nil
}

// listLogTrace 查询数据访问日志关联的审计记录。
func (c *BaseDataAccessLogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BaseDataAccessLog, error) {
	query := c.Query(ctx).BaseDataAccessLog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBaseDataAccessLog 转换数据访问日志响应。
func toBaseDataAccessLog(item *models.BaseDataAccessLog) *adminv1.BaseDataAccessLog {
	return &adminv1.BaseDataAccessLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ResourceType: item.ResourceType, ResourceId: item.ResourceID, AccessType: adminv1.BaseDataAccessType(item.AccessType), DataSource: item.DataSource, TableName: item.TableName_, FieldScope: item.FieldScope, DataScope: item.DataScope, AffectedRows: item.AffectedRows, Sensitive: item.Sensitive != 0, SqlText: item.SqlText, SqlDigest: item.SqlDigest, Result: adminv1.BaseLogResult(item.Result), ReasonCode: item.ReasonCode, LatencyMs: item.LatencyMs, OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
