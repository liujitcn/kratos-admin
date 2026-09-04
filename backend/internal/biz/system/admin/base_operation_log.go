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

// BaseOperationLogCase 提供业务操作日志表的查询业务。
type BaseOperationLogCase struct {
	*biz.BaseCase
	*data.BaseOperationLogRepository
}

// NewBaseOperationLogCase 创建业务操作日志查询业务实例。
func NewBaseOperationLogCase(baseCase *biz.BaseCase, baseOperationLogRepo *data.BaseOperationLogRepository) *BaseOperationLogCase {
	return &BaseOperationLogCase{BaseCase: baseCase, BaseOperationLogRepository: baseOperationLogRepo}
}

// PageBaseOperationLog 分页查询业务操作日志。
func (c *BaseOperationLogCase) PageBaseOperationLog(ctx context.Context, req *adminv1.PageBaseOperationLogRequest) (*adminv1.PageBaseOperationLogResponse, error) {
	query := c.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.Action != nil {
		opts = append(opts, repository.Where(query.Action.Eq(int32(req.GetAction()))))
	}
	if req.Result != nil {
		opts = append(opts, repository.Where(query.Result.Eq(int32(req.GetResult()))))
	}
	if req.GetResourceType() != "" {
		opts = append(opts, repository.Where(query.ResourceType.Eq(req.GetResourceType())))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.ResourceID.Like(keyword), query.ResourceName.Like(keyword), query.RequestID.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BaseOperationLog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseOperationLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseOperationLog(item))
	}
	return &adminv1.PageBaseOperationLogResponse{BaseOperationLogs: items, Total: int32(total)}, nil
}

// GetBaseOperationLog 查询业务操作日志详情。
func (c *BaseOperationLogCase) GetBaseOperationLog(ctx context.Context, idText string) (*adminv1.BaseOperationLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BaseOperationLog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseOperationLog(item), nil
}

// listLogTrace 查询业务操作日志关联的审计记录。
func (c *BaseOperationLogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BaseOperationLog, error) {
	query := c.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBaseOperationLog 转换业务操作日志响应。
func toBaseOperationLog(item *models.BaseOperationLog) *adminv1.BaseOperationLog {
	return &adminv1.BaseOperationLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ResourceType: item.ResourceType, ResourceId: item.ResourceID, ResourceName: item.ResourceName, Action: adminv1.BaseOperationAction(item.Action), ChangedFields: item.ChangedFields, BeforeData: item.BeforeData, AfterData: item.AfterData, Result: adminv1.BaseLogResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
