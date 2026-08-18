package biz

import (
	"context"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	"github.com/liujitcn/go-utils/mapper"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseLogCase 日志业务实例
type BaseLogCase struct {
	*biz.BaseCase
	*data.BaseLogRepository
	mapper *mapper.CopierMapper[systemadminv1.BaseLog, models.BaseLog]
}

// NewBaseLogCase 创建日志业务实例
func NewBaseLogCase(baseCase *biz.BaseCase, baseLogRepo *data.BaseLogRepository) *BaseLogCase {
	c := &BaseLogCase{
		BaseCase:          baseCase,
		BaseLogRepository: baseLogRepo,
		mapper:            mapper.NewCopierMapper[systemadminv1.BaseLog, models.BaseLog](),
	}

	return c
}

// PageBaseLog 分页查询日志
func (c *BaseLogCase) PageBaseLog(ctx context.Context, req *systemadminv1.PageBaseLogRequest) (*systemadminv1.PageBaseLogResponse, error) {
	query := c.Query(ctx).BaseLog
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.RequestTime.Desc()))
	// 传入操作名时，按操作名模糊匹配日志。
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	if req.StatusCode != nil {
		opts = append(opts, repository.Where(query.StatusCode.Eq(req.GetStatusCode())))
	}

	requestTime := req.GetRequestTime()
	// 仅在传入完整时间区间时，按请求时间范围过滤日志。
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		// 开始时间解析成功时，补充请求时间下界。
		if startTime != nil {
			opts = append(opts, repository.Where(query.RequestTime.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充请求时间上界。
		if endTime != nil {
			endValue := endTime.AddDate(0, 0, 1)
			opts = append(opts, repository.Where(query.RequestTime.Lt(endValue)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.BaseLog, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseLog(item))
	}
	return &systemadminv1.PageBaseLogResponse{BaseLogs: resList, Total: int32(total)}, nil
}

// GetBaseLog 获取日志
func (c *BaseLogCase) GetBaseLog(ctx context.Context, id int64) (*systemadminv1.BaseLog, error) {
	baseLog, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseLog(baseLog), nil
}

// toBaseLog 转换日志响应数据
func (c *BaseLogCase) toBaseLog(item *models.BaseLog) *systemadminv1.BaseLog {
	costTime := time.Duration(item.CostTime) * time.Millisecond
	baseLog := c.mapper.ToDTO(item)
	baseLog.CostTime = costTime.String()
	return baseLog
}
