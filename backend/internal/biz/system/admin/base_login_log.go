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

// BaseLoginLogCase 提供登录日志表的查询业务。
type BaseLoginLogCase struct {
	*biz.BaseCase
	*data.BaseLoginLogRepository
}

// NewBaseLoginLogCase 创建登录日志查询业务实例。
func NewBaseLoginLogCase(baseCase *biz.BaseCase, baseLoginLogRepo *data.BaseLoginLogRepository) *BaseLoginLogCase {
	return &BaseLoginLogCase{BaseCase: baseCase, BaseLoginLogRepository: baseLoginLogRepo}
}

// PageBaseLoginLog 分页查询登录日志。
func (c *BaseLoginLogCase) PageBaseLoginLog(ctx context.Context, req *adminv1.PageBaseLoginLogRequest) (*adminv1.PageBaseLoginLogResponse, error) {
	query := c.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.LoginType != nil {
		opts = append(opts, repository.Where(query.LoginType.Eq(int32(req.GetLoginType()))))
	}
	if req.Result != nil {
		opts = append(opts, repository.Where(query.Result.Eq(int32(req.GetResult()))))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.TenantCode.Like(keyword), query.UserName.Like(keyword), query.ClientIP.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BaseLoginLog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseLoginLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseLoginLog(item))
	}
	return &adminv1.PageBaseLoginLogResponse{BaseLoginLogs: items, Total: int32(total)}, nil
}

// GetBaseLoginLog 查询登录日志详情。
func (c *BaseLoginLogCase) GetBaseLoginLog(ctx context.Context, idText string) (*adminv1.BaseLoginLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BaseLoginLog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseLoginLog(item), nil
}

// listLogTrace 查询登录日志关联的审计记录。
func (c *BaseLoginLogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BaseLoginLog, error) {
	query := c.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBaseLoginLog 转换登录日志响应。
func toBaseLoginLog(item *models.BaseLoginLog) *adminv1.BaseLoginLog {
	return &adminv1.BaseLoginLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, LoginType: adminv1.BaseLoginLogType(item.LoginType), Result: adminv1.BaseLogResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, ClientIp: item.ClientIP, UserAgent: item.UserAgent, DeviceId: item.DeviceID, RequestId: item.RequestID, TraceId: item.TraceID, OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
