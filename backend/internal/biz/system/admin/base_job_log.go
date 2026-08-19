package biz

import (
	"context"
	"strconv"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseJobLogCase 任务日志业务实例
type BaseJobLogCase struct {
	*biz.BaseCase
	*data.BaseJobLogRepository
	mapper *mapper.CopierMapper[adminv1.BaseJobLog, models.BaseJobLog]
}

// NewBaseJobLogCase 创建任务日志业务实例
func NewBaseJobLogCase(baseCase *biz.BaseCase, baseJobLogRepo *data.BaseJobLogRepository) *BaseJobLogCase {
	c := &BaseJobLogCase{
		BaseCase:             baseCase,
		BaseJobLogRepository: baseJobLogRepo,
		mapper:               mapper.NewCopierMapper[adminv1.BaseJobLog, models.BaseJobLog](),
	}

	return c
}

// PageBaseJobLog 分页查询任务日志
func (c *BaseJobLogCase) PageBaseJobLog(ctx context.Context, req *adminv1.PageBaseJobLogRequest) (*adminv1.PageBaseJobLogResponse, error) {
	query := c.Query(ctx).BaseJobLog
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.ExecuteTime.Desc()))
	// 传入任务编号时，仅查询对应任务的执行日志。
	if req.GetJobId() > 0 {
		opts = append(opts, repository.Where(query.JobID.Eq(req.GetJobId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 仅在传入完整时间区间时，按执行时间范围过滤任务日志。
	if len(req.GetExecuteTime()) == 2 {
		startTime := time.StringTimeToTime(req.GetExecuteTime()[0])
		endTime := time.StringTimeToTime(req.GetExecuteTime()[1])
		// 开始时间解析成功时，补充执行时间下界。
		if startTime != nil {
			opts = append(opts, repository.Where(query.ExecuteTime.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充执行时间上界。
		if endTime != nil {
			opts = append(opts, repository.Where(query.ExecuteTime.Lte(*endTime)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseJobLog, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseJobLog(item))
	}
	return &adminv1.PageBaseJobLogResponse{BaseJobLogs: resList, Total: int32(total)}, nil
}

// GetBaseJobLog 获取任务日志
func (c *BaseJobLogCase) GetBaseJobLog(ctx context.Context, id int64) (*adminv1.BaseJobLog, error) {
	baseJobLog, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseJobLog(baseJobLog), nil
}

// toBaseJobLog 转换任务日志响应
func (c *BaseJobLogCase) toBaseJobLog(item *models.BaseJobLog) *adminv1.BaseJobLog {
	baseJobLog := c.mapper.ToDTO(item)
	baseJobLog.ProcessTime = strconv.FormatInt(int64(item.ProcessTime), 10)
	return baseJobLog
}
