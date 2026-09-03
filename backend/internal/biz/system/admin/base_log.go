package biz

import (
	"context"
	"sort"
	"strconv"
	stdtime "time"

	utiltime "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	corebiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gen/field"
)

// BaseLogCase 提供跨审计表的关联时间线查询能力。
type BaseLogCase struct {
	*corebiz.BaseCase
	baseLoginLogCase            *BaseLoginLogCase
	baseAPILogCase              *BaseAPILogCase
	baseOperationLogCase        *BaseOperationLogCase
	baseDataAccessLogCase       *BaseDataAccessLogCase
	basePermissionLogCase       *BasePermissionLogCase
	basePolicyEvaluationLogCase *BasePolicyEvaluationLogCase
}

// NewBaseLogCase 创建公共审计时间线查询业务实例。
func NewBaseLogCase(
	baseCase *corebiz.BaseCase,
	baseLoginLogCase *BaseLoginLogCase,
	baseAPILogCase *BaseAPILogCase,
	baseOperationLogCase *BaseOperationLogCase,
	baseDataAccessLogCase *BaseDataAccessLogCase,
	basePermissionLogCase *BasePermissionLogCase,
	basePolicyEvaluationLogCase *BasePolicyEvaluationLogCase,
) *BaseLogCase {
	return &BaseLogCase{
		BaseCase:                    baseCase,
		baseLoginLogCase:            baseLoginLogCase,
		baseAPILogCase:              baseAPILogCase,
		baseOperationLogCase:        baseOperationLogCase,
		baseDataAccessLogCase:       baseDataAccessLogCase,
		basePermissionLogCase:       basePermissionLogCase,
		basePolicyEvaluationLogCase: basePolicyEvaluationLogCase,
	}
}

// GetBaseLogTrace 查询同一请求或链路关联的六类审计记录。
func (c *BaseLogCase) GetBaseLogTrace(ctx context.Context, req *adminv1.GetBaseLogTraceRequest) (*adminv1.GetBaseLogTraceResponse, error) {
	items := make([]*adminv1.BaseLogTraceItem, 0)
	var err error

	var loginLogs []*models.BaseLoginLog
	loginLogs, err = c.baseLoginLogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range loginLogs {
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_LOGIN, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: item.UserName,
			Result: adminv1.BaseLogResult(item.Result), Reason: item.Reason,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	var apiLogs []*models.BaseAPILog
	apiLogs, err = c.baseAPILogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range apiLogs {
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_API, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: item.Method + " " + item.Operation,
			Result: adminv1.BaseLogResult(item.Result), Reason: item.Reason, DurationMs: item.LatencyMs,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	var operationLogs []*models.BaseOperationLog
	operationLogs, err = c.baseOperationLogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range operationLogs {
		resource := item.ResourceType
		if item.ResourceID != "" {
			resource += ":" + item.ResourceID
		}
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_OPERATION, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: resource,
			Result: adminv1.BaseLogResult(item.Result), Reason: item.Reason,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	var dataAccessLogs []*models.BaseDataAccessLog
	dataAccessLogs, err = c.baseDataAccessLogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range dataAccessLogs {
		resource := item.ResourceType
		if item.ResourceID != "" {
			resource += ":" + item.ResourceID
		}
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_DATA_ACCESS, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: resource,
			Result: adminv1.BaseLogResult(item.Result), Reason: item.ReasonCode, DurationMs: item.LatencyMs,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	var permissionLogs []*models.BasePermissionLog
	permissionLogs, err = c.basePermissionLogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range permissionLogs {
		resource := item.TargetName
		if resource == "" {
			resource = item.TargetID
		}
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_PERMISSION, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: resource,
			Result: adminv1.BaseLogResult(item.Result), Reason: item.Reason,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	var policyLogs []*models.BasePolicyEvaluationLog
	policyLogs, err = c.basePolicyEvaluationLogCase.listLogTrace(ctx, req.GetRequestId(), req.GetTraceId())
	if err != nil {
		return nil, err
	}
	for _, item := range policyLogs {
		result := adminv1.BaseLogResult_BASE_LOG_RESULT_SUCCESS
		if item.Decision == int32(adminv1.BasePolicyDecision_BASE_POLICY_DECISION_DENY) {
			result = adminv1.BaseLogResult_BASE_LOG_RESULT_FAILURE
		} else if item.Decision == int32(adminv1.BasePolicyDecision_BASE_POLICY_DECISION_ERROR) {
			result = adminv1.BaseLogResult_BASE_LOG_RESULT_ERROR
		}
		items = append(items, &adminv1.BaseLogTraceItem{
			LogType: adminv1.BaseLogType_BASE_LOG_TYPE_POLICY_EVALUATION, Id: item.ID,
			TenantId: item.TenantID, UserId: item.UserID, UserName: item.UserName,
			RequestId: item.RequestID, TraceId: item.TraceID, Resource: item.Action + " " + item.Resource,
			Result: result, Reason: item.Reason, DurationMs: item.DurationMs,
			OccurredAt: formatLogTime(item.OccurredAt), CreatedAt: formatLogTime(item.CreatedAt),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].OccurredAt == items[j].OccurredAt {
			return items[i].Id < items[j].Id
		}
		return items[i].OccurredAt < items[j].OccurredAt
	})
	return &adminv1.GetBaseLogTraceResponse{Items: items}, nil
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

// appendTraceIdentityOptions 按请求编号或链路编号关联同一次审计链路。
func appendTraceIdentityOptions(opts []repository.QueryOption, requestID, traceID string, requestColumn, traceColumn field.String) []repository.QueryOption {
	if requestID != "" && traceID != "" {
		return append(opts, repository.Where(field.Or(requestColumn.Eq(requestID), traceColumn.Eq(traceID))))
	}
	if requestID != "" {
		return append(opts, repository.Where(requestColumn.Eq(requestID)))
	}
	return append(opts, repository.Where(traceColumn.Eq(traceID)))
}

// formatLogTime 格式化审计时间，保持毫秒精度。
func formatLogTime(value stdtime.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(stdtime.RFC3339Nano)
}

// parseLogRecordID 将管理端传入的日志主键解析为数据库整数。
func parseLogRecordID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errorsx.InvalidArgument("日志ID格式无效").WithCause(err)
	}
	return id, nil
}

// formatLogRecordID 将数据库日志主键格式化为不会丢失精度的字符串。
func formatLogRecordID(value int64) string {
	return strconv.FormatInt(value, 10)
}
