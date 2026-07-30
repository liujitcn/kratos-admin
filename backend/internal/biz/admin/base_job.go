package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/job"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseJobCase 定时任务业务实例
type BaseJobCase struct {
	*biz.BaseCase
	*data.BaseJobRepository
	baseJobLogCase *BaseJobLogCase
	cronServer     *job.CronServer
	formMapper     *_mapper.CopierMapper[systemadminv1.BaseJobForm, models.BaseJob]
	mapper         *_mapper.CopierMapper[systemadminv1.BaseJob, models.BaseJob]
}

// NewBaseJobCase 创建定时任务业务实例
func NewBaseJobCase(baseCase *biz.BaseCase, baseJobRepo *data.BaseJobRepository, baseJobLogCase *BaseJobLogCase, cronServer *job.CronServer) *BaseJobCase {
	formMapper := _mapper.NewCopierMapper[systemadminv1.BaseJobForm, models.BaseJob]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*systemadminv1.BaseJobArgs]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[systemadminv1.BaseJob, models.BaseJob]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*systemadminv1.BaseJobArgs]().NewConverterPair())

	return &BaseJobCase{
		BaseCase:          baseCase,
		BaseJobRepository: baseJobRepo,
		baseJobLogCase:    baseJobLogCase,
		cronServer:        cronServer,
		formMapper:        formMapper,
		mapper:            mapper,
	}
}

// PageBaseJob 分页查询定时任务
func (c *BaseJobCase) PageBaseJob(ctx context.Context, req *systemadminv1.PageBaseJobRequest) (*systemadminv1.PageBaseJobResponse, error) {
	query := c.Query(ctx).BaseJob
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入任务名称时，按名称模糊匹配定时任务。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入调用目标时，按调用目标模糊匹配定时任务。
	if req.GetInvokeTarget() != "" {
		opts = append(opts, repository.Where(query.InvokeTarget.Like("%"+req.GetInvokeTarget()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.BaseJob, 0, len(list))
	for _, item := range list {
		baseJob := c.mapper.ToDTO(item)
		resList = append(resList, baseJob)
	}
	return &systemadminv1.PageBaseJobResponse{BaseJobs: resList, Total: int32(total)}, nil
}

// GetBaseJob 获取定时任务
func (c *BaseJobCase) GetBaseJob(ctx context.Context, id int64) (*systemadminv1.BaseJobForm, error) {
	baseJob, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseJob)
	return res, nil
}

// CreateBaseJob 创建定时任务
func (c *BaseJobCase) CreateBaseJob(ctx context.Context, req *systemadminv1.BaseJobForm) error {
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	err := c.Create(ctx, baseJob)
	if err != nil {
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseJob 更新定时任务
func (c *BaseJobCase) UpdateBaseJob(ctx context.Context, req *systemadminv1.BaseJobForm) error {
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	err := c.UpdateByID(ctx, baseJob)
	if err != nil {
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseJob 删除定时任务
func (c *BaseJobCase) DeleteBaseJob(ctx context.Context, id string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(id))
}

// SetBaseJobStatus 设置定时任务状态
func (c *BaseJobCase) SetBaseJobStatus(ctx context.Context, req *systemadminv1.SetBaseJobStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseJob{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// StartBaseJob 启动定时任务
func (c *BaseJobCase) StartBaseJob(ctx context.Context, req *systemadminv1.StartBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.cronServer.StartJob(ctx, baseJob)
}

// StopBaseJob 停止定时任务
func (c *BaseJobCase) StopBaseJob(ctx context.Context, req *systemadminv1.StopBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.cronServer.StopJob(ctx, baseJob)
}

// ExecuteBaseJob 立即执行定时任务
func (c *BaseJobCase) ExecuteBaseJob(ctx context.Context, req *systemadminv1.ExecuteBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.cronServer.RunJob(ctx, baseJob)
}
