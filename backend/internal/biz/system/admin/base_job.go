package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	corejob "github.com/liujitcn/kratos-core/job"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseJobCase 定时任务业务实例
type BaseJobCase struct {
	*biz.BaseCase
	*data.BaseJobRepository
	baseJobLogCase *BaseJobLogCase
	job            *corejob.Job
	formMapper     *_mapper.CopierMapper[systemadminv1.BaseJobForm, models.BaseJob]
	mapper         *_mapper.CopierMapper[systemadminv1.BaseJob, models.BaseJob]
}

// NewBaseJobCase 创建定时任务业务实例
func NewBaseJobCase(baseCase *biz.BaseCase, job *corejob.Job, baseJobRepo *data.BaseJobRepository, baseJobLogCase *BaseJobLogCase) *BaseJobCase {
	formMapper := _mapper.NewCopierMapper[systemadminv1.BaseJobForm, models.BaseJob]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*systemadminv1.BaseJobArgs]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[systemadminv1.BaseJob, models.BaseJob]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*systemadminv1.BaseJobArgs]().NewConverterPair())

	return &BaseJobCase{
		BaseCase:          baseCase,
		BaseJobRepository: baseJobRepo,
		baseJobLogCase:    baseJobLogCase,
		job:               job,
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
	err := validateBaseJobStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	err = c.Create(ctx, baseJob)
	if err != nil {
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseJob 更新定时任务
func (c *BaseJobCase) UpdateBaseJob(ctx context.Context, req *systemadminv1.BaseJobForm) error {
	err := validateBaseJobStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	var previousJob *models.BaseJob
	previousJob, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	wasRunning := previousJob.EntryID > 0
	if wasRunning {
		err = c.job.Stop(ctx, previousJob.ID, previousJob.EntryID)
		if err != nil {
			return err
		}
	}

	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	if wasRunning {
		baseJob.EntryID = 0
	}
	err = c.UpdateByID(ctx, baseJob)
	if err != nil {
		if wasRunning {
			restoreErr := c.restoreBaseJob(ctx, previousJob)
			if restoreErr != nil {
				return errorsx.WrapInternal(restoreErr, "恢复定时任务调度失败")
			}
		}
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	if !wasRunning || baseJob.Status != coreconst.STATUS_STATUS_ENABLE {
		return nil
	}
	var entryID int32
	entryID, err = c.job.Start(ctx, baseJob.ID, baseJob.CronExpression, baseJob.InvokeTarget, baseJob.Args, 0)
	if err != nil {
		restoreErr := c.restoreBaseJob(ctx, previousJob)
		if restoreErr != nil {
			return errorsx.WrapInternal(restoreErr, "恢复定时任务配置失败")
		}
		return err
	}
	err = c.updateBaseJobEntryID(ctx, baseJob.ID, entryID)
	if err != nil {
		_ = c.job.Stop(ctx, baseJob.ID, entryID)
		restoreErr := c.restoreBaseJob(ctx, previousJob)
		if restoreErr != nil {
			return errorsx.WrapInternal(restoreErr, "恢复定时任务配置失败")
		}
		return err
	}
	return nil
}

// DeleteBaseJob 删除定时任务
func (c *BaseJobCase) DeleteBaseJob(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseJobs, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	baseJobMap := make(map[int64]*models.BaseJob, len(baseJobs))
	for _, baseJob := range baseJobs {
		baseJobMap[baseJob.ID] = baseJob
	}
	stoppedJobs := make([]*models.BaseJob, 0, len(baseJobs))
	stoppedJobIDs := make(map[int64]struct{}, len(baseJobs))
	for _, jobID := range ids {
		baseJob, exists := baseJobMap[jobID]
		if !exists {
			restoreErr := c.restoreBaseJobs(ctx, stoppedJobs)
			if restoreErr != nil {
				return errorsx.WrapInternal(restoreErr, "恢复定时任务调度失败")
			}
			return errorsx.ResourceNotFound("定时任务不存在")
		}
		if baseJob.EntryID == 0 {
			continue
		}
		if _, stopped := stoppedJobIDs[baseJob.ID]; stopped {
			continue
		}
		err = c.job.Stop(ctx, baseJob.ID, baseJob.EntryID)
		if err != nil {
			restoreErr := c.restoreBaseJobs(ctx, stoppedJobs)
			if restoreErr != nil {
				return errorsx.WrapInternal(restoreErr, "恢复定时任务调度失败")
			}
			return err
		}
		stoppedJobIDs[baseJob.ID] = struct{}{}
		stoppedJobs = append(stoppedJobs, baseJob)
	}
	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		restoreErr := c.restoreBaseJobs(ctx, stoppedJobs)
		if restoreErr != nil {
			return errorsx.WrapInternal(restoreErr, "恢复定时任务调度失败")
		}
	}
	return err
}

// SetBaseJobStatus 设置定时任务状态
func (c *BaseJobCase) SetBaseJobStatus(ctx context.Context, req *systemadminv1.SetBaseJobStatusRequest) error {
	err := validateBaseJobStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	var baseJob *models.BaseJob
	baseJob, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetStatus() == coreconst.STATUS_STATUS_DISABLE && baseJob.EntryID > 0 {
		err = c.job.Stop(ctx, baseJob.ID, baseJob.EntryID)
		if err != nil {
			return err
		}
	}
	updatedJob := &models.BaseJob{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	}
	if req.GetStatus() == coreconst.STATUS_STATUS_DISABLE {
		updatedJob.EntryID = 0
	}
	err = c.UpdateByID(ctx, updatedJob)
	if err != nil && req.GetStatus() == coreconst.STATUS_STATUS_DISABLE && baseJob.EntryID > 0 {
		restoreErr := c.restoreBaseJob(ctx, baseJob)
		if restoreErr != nil {
			return errorsx.WrapInternal(restoreErr, "恢复定时任务调度失败")
		}
	}
	return err
}

// StartBaseJob 启动定时任务
func (c *BaseJobCase) StartBaseJob(ctx context.Context, req *systemadminv1.StartBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if baseJob.Status != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.Conflict("定时任务未启用")
	}
	var entryID int32
	previousEntryID := baseJob.EntryID
	entryID, err = c.job.Start(ctx, baseJob.ID, baseJob.CronExpression, baseJob.InvokeTarget, baseJob.Args, baseJob.EntryID)
	if err != nil {
		return err
	}
	err = c.updateBaseJobEntryID(ctx, baseJob.ID, entryID)
	if err != nil {
		_ = c.job.Stop(ctx, baseJob.ID, entryID)
		if previousEntryID > 0 {
			_ = c.restoreBaseJob(ctx, baseJob)
		}
		return err
	}
	baseJob.EntryID = entryID
	return nil
}

// StopBaseJob 停止定时任务
func (c *BaseJobCase) StopBaseJob(ctx context.Context, req *systemadminv1.StopBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	err = c.job.Stop(ctx, baseJob.ID, baseJob.EntryID)
	if err != nil {
		return err
	}
	err = c.updateBaseJobEntryID(ctx, baseJob.ID, 0)
	if err != nil {
		_ = c.restoreBaseJob(ctx, baseJob)
		return err
	}
	baseJob.EntryID = 0
	return nil
}

// ExecuteBaseJob 立即执行定时任务
func (c *BaseJobCase) ExecuteBaseJob(ctx context.Context, req *systemadminv1.ExecuteBaseJobRequest) error {
	baseJob, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if baseJob.Status != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.Conflict("定时任务未启用")
	}
	return c.job.Run(ctx, baseJob.ID, baseJob.InvokeTarget, baseJob.Args)
}

// restoreBaseJob 恢复任务配置并在原任务运行时重新注册调度。
func (c *BaseJobCase) restoreBaseJob(ctx context.Context, previousJob *models.BaseJob) error {
	restoredJob := *previousJob
	restoredJob.EntryID = 0
	err := c.UpdateByID(ctx, &restoredJob)
	if err != nil {
		return err
	}
	if previousJob.EntryID == 0 {
		return nil
	}
	var entryID int32
	entryID, err = c.job.Start(ctx, restoredJob.ID, restoredJob.CronExpression, restoredJob.InvokeTarget, restoredJob.Args, 0)
	if err != nil {
		return err
	}
	err = c.updateBaseJobEntryID(ctx, restoredJob.ID, entryID)
	if err != nil {
		_ = c.job.Stop(ctx, restoredJob.ID, entryID)
	}
	return err
}

// restoreBaseJobs 按逆序恢复一批被停止的定时任务调度。
func (c *BaseJobCase) restoreBaseJobs(ctx context.Context, jobs []*models.BaseJob) error {
	var err error
	for index := len(jobs) - 1; index >= 0; index-- {
		if jobs[index] == nil {
			continue
		}
		err = c.restoreBaseJob(ctx, jobs[index])
		if err != nil {
			return err
		}
	}
	return nil
}

// validateBaseJobStatus 校验定时任务只能使用启用或禁用状态。
func validateBaseJobStatus(status int32) error {
	if status != coreconst.STATUS_STATUS_ENABLE && status != coreconst.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("定时任务状态无效")
	}
	return nil
}

// updateBaseJobEntryID 持久化定时任务当前的调度入口编号。
func (c *BaseJobCase) updateBaseJobEntryID(ctx context.Context, jobID int64, entryID int32) error {
	return c.UpdateByID(ctx, &models.BaseJob{ID: jobID, EntryID: entryID})
}
