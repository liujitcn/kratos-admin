package biz

import (
	"context"

	utilsTime "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	corebiz "github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
)

// BaseScriptLogCase 脚本执行日志业务实例。
type BaseScriptLogCase struct {
	*corebiz.BaseCase
	*data.BaseScriptLogRepository
}

// NewBaseScriptLogCase 创建脚本执行日志业务实例。
func NewBaseScriptLogCase(baseCase *corebiz.BaseCase, repo *data.BaseScriptLogRepository) *BaseScriptLogCase {
	return &BaseScriptLogCase{BaseCase: baseCase, BaseScriptLogRepository: repo}
}

// PageBaseScriptLog 分页查询脚本执行日志。
func (c *BaseScriptLogCase) PageBaseScriptLog(ctx context.Context, req *adminv1.PageBaseScriptLogRequest) (*adminv1.PageBaseScriptLogResponse, error) {
	query := c.Query(ctx).BaseScriptLog
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetScriptId() > 0 {
		opts = append(opts, repository.Where(query.ScriptID.Eq(req.GetScriptId())))
	}
	if req.GetTriggerType() != "" {
		opts = append(opts, repository.Where(query.TriggerType.Eq(req.GetTriggerType())))
	}
	if req.Success != nil {
		success := int32(0)
		if req.GetSuccess() {
			success = 1
		}
		opts = append(opts, repository.Where(query.Success.Eq(success)))
	}
	if len(req.GetCreatedAt()) == 2 {
		startAt := utilsTime.StringTimeToTime(req.GetCreatedAt()[0])
		endAt := utilsTime.StringTimeToTime(req.GetCreatedAt()[1])
		if startAt != nil {
			opts = append(opts, repository.Where(query.CreatedAt.Gte(*startAt)))
		}
		if endAt != nil {
			opts = append(opts, repository.Where(query.CreatedAt.Lte(*endAt)))
		}
	}

	items, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseScriptLog, 0, len(items))
	for _, item := range items {
		result = append(result, toBaseScriptLog(item))
	}
	return &adminv1.PageBaseScriptLogResponse{BaseScriptLogs: result, Total: int32(total)}, nil
}

// GetBaseScriptLog 获取脚本执行日志详情。
func (c *BaseScriptLogCase) GetBaseScriptLog(ctx context.Context, id int64) (*adminv1.BaseScriptLog, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseScriptLog(item), nil
}

// Record 写入一次脚本执行结果。
func (c *BaseScriptLogCase) Record(ctx context.Context, item *models.BaseScriptLog) error {
	return c.Create(ctx, item)
}

// toBaseScriptLog 将脚本执行日志模型转换为协议对象。
func toBaseScriptLog(item *models.BaseScriptLog) *adminv1.BaseScriptLog {
	if item == nil {
		return nil
	}
	return &adminv1.BaseScriptLog{
		Id:          item.ID,
		ScriptId:    item.ScriptID,
		ScriptName:  item.ScriptName,
		Language:    item.Language,
		TriggerType: item.TriggerType,
		HookPoint:   item.HookPoint,
		Version:     item.Version,
		Success:     item.Success == 1,
		DurationMs:  item.DurationMs,
		Error:       item.Error,
		RequestId:   item.RequestID,
		TraceId:     item.TraceID,
		CreatedAt:   item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// scriptStatusValid 判断脚本状态是否为系统支持的启用或停用状态。
func scriptStatusValid(status int32) bool {
	return status == _const.STATUS_STATUS_ENABLE || status == _const.STATUS_STATUS_DISABLE
}

// scriptStatusValue 将协议状态转成模型状态值。
func scriptStatusValue(status int32) int32 {
	if scriptStatusValid(status) {
		return status
	}
	return _const.STATUS_STATUS_ENABLE
}
