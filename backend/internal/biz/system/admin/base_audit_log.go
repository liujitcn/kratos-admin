package biz

import (
	"context"
	stdtime "time"

	utiltime "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
)

// BaseAuditLogCase 提供六类日志审计的只读查询能力。
// 所有查询均按当前登录主体应用租户边界，并支持分页、时间范围和关键字筛选。
type BaseAuditLogCase struct {
	*biz.BaseCase
	loginLogRepo            *data.BaseLoginLogRepository
	apiLogRepo              *data.BaseAPILogRepository
	operationLogRepo        *data.BaseOperationLogRepository
	dataAccessLogRepo       *data.BaseDataAccessLogRepository
	permissionLogRepo       *data.BasePermissionLogRepository
	policyEvaluationLogRepo *data.BasePolicyEvaluationLogRepository
}

// NewBaseAuditLogCase 创建日志审计业务实例。
func NewBaseAuditLogCase(
	baseCase *biz.BaseCase,
	loginLogRepo *data.BaseLoginLogRepository,
	apiLogRepo *data.BaseAPILogRepository,
	operationLogRepo *data.BaseOperationLogRepository,
	dataAccessLogRepo *data.BaseDataAccessLogRepository,
	permissionLogRepo *data.BasePermissionLogRepository,
	policyEvaluationLogRepo *data.BasePolicyEvaluationLogRepository,
) *BaseAuditLogCase {
	return &BaseAuditLogCase{
		BaseCase:                baseCase,
		loginLogRepo:            loginLogRepo,
		apiLogRepo:              apiLogRepo,
		operationLogRepo:        operationLogRepo,
		dataAccessLogRepo:       dataAccessLogRepo,
		permissionLogRepo:       permissionLogRepo,
		policyEvaluationLogRepo: policyEvaluationLogRepo,
	}
}

// PageBaseLoginLog 分页查询登录日志。
func (c *BaseAuditLogCase) PageBaseLoginLog(ctx context.Context, req *adminv1.PageBaseLoginLogRequest) (*adminv1.PageBaseLoginLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.loginLogRepo.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.loginLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
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
func (c *BaseAuditLogCase) GetBaseLoginLog(ctx context.Context, id int64) (*adminv1.BaseLoginLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.loginLogRepo.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseLoginLog
	item, err = c.loginLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseLoginLog(item), nil
}

// PageBaseApiLog 分页查询 API 访问日志。
func (c *BaseAuditLogCase) PageBaseApiLog(ctx context.Context, req *adminv1.PageBaseApiLogRequest) (*adminv1.PageBaseApiLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.apiLogRepo.Query(ctx).BaseAPILog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.apiLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
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
func (c *BaseAuditLogCase) GetBaseApiLog(ctx context.Context, id int64) (*adminv1.BaseApiLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.apiLogRepo.Query(ctx).BaseAPILog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseAPILog
	item, err = c.apiLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseApiLog(item), nil
}

// PageBaseOperationLog 分页查询业务操作日志。
func (c *BaseAuditLogCase) PageBaseOperationLog(ctx context.Context, req *adminv1.PageBaseOperationLogRequest) (*adminv1.PageBaseOperationLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.operationLogRepo.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.operationLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
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
func (c *BaseAuditLogCase) GetBaseOperationLog(ctx context.Context, id int64) (*adminv1.BaseOperationLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.operationLogRepo.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseOperationLog
	item, err = c.operationLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseOperationLog(item), nil
}

// PageBaseDataAccessLog 分页查询数据访问日志。
func (c *BaseAuditLogCase) PageBaseDataAccessLog(ctx context.Context, req *adminv1.PageBaseDataAccessLogRequest) (*adminv1.PageBaseDataAccessLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.dataAccessLogRepo.Query(ctx).BaseDataAccessLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.dataAccessLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
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
func (c *BaseAuditLogCase) GetBaseDataAccessLog(ctx context.Context, id int64) (*adminv1.BaseDataAccessLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.dataAccessLogRepo.Query(ctx).BaseDataAccessLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseDataAccessLog
	item, err = c.dataAccessLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBaseDataAccessLog(item), nil
}

// PageBasePermissionLog 分页查询权限日志。
func (c *BaseAuditLogCase) PageBasePermissionLog(ctx context.Context, req *adminv1.PageBasePermissionLogRequest) (*adminv1.PageBasePermissionLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.permissionLogRepo.Query(ctx).BasePermissionLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.permissionLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BasePermissionLog, 0, len(list))
	for _, item := range list {
		items = append(items, toBasePermissionLog(item))
	}
	return &adminv1.PageBasePermissionLogResponse{BasePermissionLogs: items, Total: int32(total)}, nil
}

// GetBasePermissionLog 查询权限日志详情。
func (c *BaseAuditLogCase) GetBasePermissionLog(ctx context.Context, id int64) (*adminv1.BasePermissionLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.permissionLogRepo.Query(ctx).BasePermissionLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BasePermissionLog
	item, err = c.permissionLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBasePermissionLog(item), nil
}

// PageBasePolicyEvaluationLog 分页查询策略评估日志。
func (c *BaseAuditLogCase) PageBasePolicyEvaluationLog(ctx context.Context, req *adminv1.PageBasePolicyEvaluationLogRequest) (*adminv1.PageBasePolicyEvaluationLogResponse, error) {
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.policyEvaluationLogRepo.Query(ctx).BasePolicyEvaluationLog
	opts := []repository.QueryOption{repository.Order(query.OccurredAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
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
	list, total, err = c.policyEvaluationLogRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
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
func (c *BaseAuditLogCase) GetBasePolicyEvaluationLog(ctx context.Context, id int64) (*adminv1.BasePolicyEvaluationLog, error) {
	tenantID, err := c.resolveTenantID(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.policyEvaluationLogRepo.Query(ctx).BasePolicyEvaluationLog
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BasePolicyEvaluationLog
	item, err = c.policyEvaluationLogRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return toBasePolicyEvaluationLog(item), nil
}

// resolveTenantID 将普通租户查询限制在自身租户，默认租户允许按条件查询。
func (c *BaseAuditLogCase) resolveTenantID(ctx context.Context, requested int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode {
		return authInfo.TenantId, nil
	}
	return requested, nil
}

// appendOccurredAtOptions 将完整时间范围转换为类型化查询条件。
func appendOccurredAtOptions(opts []repository.QueryOption, values []string, column field.Time) []repository.QueryOption {
	if len(values) != 2 {
		return opts
	}
	startTime := utiltime.StringTimeToTime(values[0])
	endTime := utiltime.StringTimeToTime(values[1])
	if startTime != nil {
		opts = append(opts, repository.Where(column.Gte(*startTime)))
	}
	if endTime != nil {
		opts = append(opts, repository.Where(column.Lt(endTime.AddDate(0, 0, 1))))
	}
	return opts
}

// formatAuditTime 格式化审计时间，保持毫秒精度。
func formatAuditTime(value stdtime.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(stdtime.RFC3339Nano)
}

// toBaseLoginLog 转换登录日志响应。
func toBaseLoginLog(item *models.BaseLoginLog) *adminv1.BaseLoginLog {
	return &adminv1.BaseLoginLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, LoginType: adminv1.BaseLoginLogType(item.LoginType), Result: adminv1.BaseAuditResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, ClientIp: item.ClientIP, UserAgent: item.UserAgent, DeviceId: item.DeviceID, RequestId: item.RequestID, TraceId: item.TraceID, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}

// toBaseApiLog 转换 API 访问日志响应。
func toBaseApiLog(item *models.BaseAPILog) *adminv1.BaseApiLog {
	return &adminv1.BaseApiLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ServiceName: item.ServiceName, Operation: item.Operation, Method: item.Method, Path: item.Path, StatusCode: item.StatusCode, Result: adminv1.BaseAuditResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, LatencyMs: item.LatencyMs, RequestSize: item.RequestSize, ResponseSize: item.ResponseSize, ClientIp: item.ClientIP, UserAgent: item.UserAgent, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}

// toBaseOperationLog 转换业务操作日志响应。
func toBaseOperationLog(item *models.BaseOperationLog) *adminv1.BaseOperationLog {
	return &adminv1.BaseOperationLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ResourceType: item.ResourceType, ResourceId: item.ResourceID, ResourceName: item.ResourceName, Action: adminv1.BaseOperationAction(item.Action), ChangedFields: item.ChangedFields, BeforeData: item.BeforeData, AfterData: item.AfterData, Result: adminv1.BaseAuditResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}

// toBaseDataAccessLog 转换数据访问日志响应。
func toBaseDataAccessLog(item *models.BaseDataAccessLog) *adminv1.BaseDataAccessLog {
	return &adminv1.BaseDataAccessLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, ResourceType: item.ResourceType, ResourceId: item.ResourceID, AccessType: adminv1.BaseDataAccessType(item.AccessType), DataSource: item.DataSource, TableName: item.TableName_, FieldScope: item.FieldScope, DataScope: item.DataScope, AffectedRows: item.AffectedRows, Sensitive: item.Sensitive != 0, SqlText: item.SqlText, SqlDigest: item.SqlDigest, Result: adminv1.BaseAuditResult(item.Result), ReasonCode: item.ReasonCode, LatencyMs: item.LatencyMs, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}

// toBasePermissionLog 转换权限日志响应。
func toBasePermissionLog(item *models.BasePermissionLog) *adminv1.BasePermissionLog {
	return &adminv1.BasePermissionLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RequestId: item.RequestID, TraceId: item.TraceID, TargetType: adminv1.BasePermissionTargetType(item.TargetType), TargetId: item.TargetID, TargetName: item.TargetName, Action: adminv1.BasePermissionAction(item.Action), OldValue: item.OldValue, NewValue: item.NewValue, Result: adminv1.BaseAuditResult(item.Result), ReasonCode: item.ReasonCode, Reason: item.Reason, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}

// toBasePolicyEvaluationLog 转换策略评估日志响应。
func toBasePolicyEvaluationLog(item *models.BasePolicyEvaluationLog) *adminv1.BasePolicyEvaluationLog {
	return &adminv1.BasePolicyEvaluationLog{Id: item.ID, TenantId: item.TenantID, TenantCode: item.TenantCode, UserId: item.UserID, UserName: item.UserName, RoleId: item.RoleID, RoleCode: item.RoleCode, RequestId: item.RequestID, TraceId: item.TraceID, ClientIp: item.ClientIP, Engine: item.Engine, EvaluationType: adminv1.BasePolicyEvaluationType(item.EvaluationType), Resource: item.Resource, Action: item.Action, Project: item.Project, Decision: adminv1.BasePolicyDecision(item.Decision), ReasonCode: item.ReasonCode, Reason: item.Reason, DurationMs: item.DurationMs, CandidateCount: item.CandidateCount, MatchedCount: item.MatchedCount, InputHash: item.InputHash, OccurredAt: formatAuditTime(item.OccurredAt), CreatedAt: formatAuditTime(item.CreatedAt)}
}
