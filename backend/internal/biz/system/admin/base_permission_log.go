package biz

import (
	"context"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"gorm.io/gen/field"
)

// BasePermissionLogCase 提供权限日志表的查询业务。
type BasePermissionLogCase struct {
	*biz.BaseCase
	*data.BasePermissionLogRepository
	catalog *i18n.I18n
}

// NewBasePermissionLogCase 创建权限日志查询业务实例。
func NewBasePermissionLogCase(baseCase *biz.BaseCase, basePermissionLogRepo *data.BasePermissionLogRepository, catalog *i18n.I18n) *BasePermissionLogCase {
	return &BasePermissionLogCase{BaseCase: baseCase, BasePermissionLogRepository: basePermissionLogRepo, catalog: catalog}
}

// PageBasePermissionLog 分页查询权限日志。
func (c *BasePermissionLogCase) PageBasePermissionLog(ctx context.Context, req *adminv1.PageBasePermissionLogRequest) (*adminv1.PageBasePermissionLogResponse, error) {
	query := c.Query(ctx).BasePermissionLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.UserId != nil {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	if req.TargetType != nil {
		opts = append(opts, repository.Where(query.TargetType.Eq(int32(req.GetTargetType()))))
	}
	if req.Action != nil {
		opts = append(opts, repository.Where(query.Action.Eq(int32(req.GetAction()))))
	}
	if req.Result != nil {
		opts = append(opts, repository.Where(query.Result.Eq(int32(req.GetResult()))))
	}
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.TargetID.Like(keyword), query.TargetName.Like(keyword), query.RequestID.Like(keyword))))
	}
	opts = appendOccurredAtOptions(opts, req.GetOccurredAt(), query.OccurredAt)

	var list []*models.BasePermissionLog
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BasePermissionLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBasePermissionLog(item, c.catalog, biz.LocaleFromContext(ctx)))
	}
	return &adminv1.PageBasePermissionLogResponse{BasePermissionLogs: items, Total: int32(total)}, nil
}

// GetBasePermissionLog 查询权限日志详情。
func (c *BasePermissionLogCase) GetBasePermissionLog(ctx context.Context, idText string) (*adminv1.BasePermissionLog, error) {
	id, err := parseLogRecordID(idText)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BasePermissionLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	var item *models.BasePermissionLog
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBasePermissionLog(item, c.catalog, biz.LocaleFromContext(ctx)), nil
}

// listLogTrace 查询权限日志关联的审计记录。
func (c *BasePermissionLogCase) listLogTrace(ctx context.Context, requestID, traceID string) ([]*models.BasePermissionLog, error) {
	query := c.Query(ctx).BasePermissionLog
	opts := []repository.QueryOption{repository.Limit(100)}
	opts = appendTraceIdentityOptions(opts, requestID, traceID, query.RequestID, query.TraceID)
	return c.List(ctx, opts...)
}

// toBasePermissionLog 转换权限日志响应。
func toBasePermissionLog(item *models.BasePermissionLog, catalog *i18n.I18n, locale string) *adminv1.BasePermissionLog {
	return &adminv1.BasePermissionLog{Id: formatLogRecordID(item.ID), TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, TargetType: adminv1.BasePermissionTargetType(item.TargetType), TargetId: item.TargetID, TargetName: item.TargetName, Action: adminv1.BasePermissionAction(item.Action), OldValue: item.OldValue, NewValue: item.NewValue, Result: adminv1.BaseLogResult(item.Result), ReasonCode: item.ReasonCode, Reason: localizeLogReason(catalog, locale, item.ReasonCode, item.Reason), OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt)}
}
