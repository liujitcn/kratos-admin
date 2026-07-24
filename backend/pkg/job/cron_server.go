package job

import (
	"context"
	"encoding/json"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
	"github.com/robfig/cron/v3"
)

// CronServer 定时任务服务启动器，同时负责运行时任务调度。
type CronServer struct {
	*cronTransport.Server
	baseJobRepo *data.BaseJobRepository
	registry    *Registry
}

// NewCronServer 创建定时任务服务实例。
func NewCronServer(baseJobRepo *data.BaseJobRepository, registry *Registry) *CronServer {
	return &CronServer{
		Server:      cronTransport.NewServer(cronTransport.WithEnableKeepAlive(false)),
		baseJobRepo: baseJobRepo,
		registry:    registry,
	}
}

// Start 启动定时任务服务并重载启用中的任务。
func (c *CronServer) Start(ctx context.Context) error {
	err := c.Server.Start(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("cron server start failed, err=%v", err))
		return err
	}

	err = c.reloadJobs(ctx)
	// 任务重载失败时，回滚当前 cron 服务启动状态。
	if err != nil {
		// 停服失败只记录日志，原始重载错误仍然优先返回。
		stopErr := c.Stop(ctx)
		if stopErr != nil {
			log.Error(fmt.Sprintf("cron server stop failed, err=%v", stopErr))
		}
		return err
	}
	return nil
}

// Stop 停止定时任务服务。
func (c *CronServer) Stop(ctx context.Context) error {
	return c.Server.Stop(ctx)
}

// StartJob 启动单个定时任务。
func (c *CronServer) StartJob(ctx context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续启动调度。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}

	invokeTarget, err := c.lookupTaskExec(baseJob.InvokeTarget)
	if err != nil {
		return err
	}

	var argsMap map[string]string
	argsMap, err = parseJobArgs(baseJob.Args)
	if err != nil {
		return err
	}

	// 任务已经存在调度记录时，先移除旧调度再重新注册。
	if baseJob.EntryID > 0 {
		c.Server.StopTimerJob(cron.EntryID(baseJob.EntryID))
	}

	jobID := baseJob.ID
	var entryID cron.EntryID
	entryID, err = c.Server.StartTimerJob(baseJob.CronExpression, func() {
		clonedArgs := make(map[string]string, len(argsMap))
		for key, value := range argsMap {
			clonedArgs[key] = value
		}
		execJob := &ExecJob{
			JobID:        jobID,
			Args:         clonedArgs,
			InvokeTarget: invokeTarget,
		}
		// 单次调度执行失败时，仅记录错误日志，不影响后续调度。
		execErr := execJob.Execute()
		if execErr != nil {
			log.Error(fmt.Sprintf("cron job execute failed, jobID=%d err=%v", jobID, execErr))
		}
	})
	if err != nil {
		return err
	}

	err = c.updateBaseJobEntryID(ctx, baseJob.ID, int32(entryID))
	// 调度记录落库失败时，立即回滚刚注册的内存任务，避免运行态与数据库状态分叉。
	if err != nil {
		c.Server.StopTimerJob(entryID)
		return err
	}

	baseJob.EntryID = int32(entryID)
	return nil
}

// StopJob 停止单个定时任务。
func (c *CronServer) StopJob(ctx context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续停止调度。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}

	err := c.updateBaseJobEntryID(ctx, baseJob.ID, 0)
	if err != nil {
		return err
	}

	// 任务存在调度记录时，先从运行中的 cron 服务里移除。
	if baseJob.EntryID > 0 {
		c.Server.StopTimerJob(cron.EntryID(baseJob.EntryID))
	}

	baseJob.EntryID = 0
	return nil
}

// RunJob 立即执行单个定时任务。
func (c *CronServer) RunJob(_ context.Context, baseJob *models.BaseJob) error {
	// 任务实体为空时，无法继续执行。
	if baseJob == nil {
		return errorsx.ResourceNotFound("定时任务不存在")
	}

	invokeTarget, err := c.lookupTaskExec(baseJob.InvokeTarget)
	if err != nil {
		// 立即执行在进入任务体前失败时，也要补充失败日志，方便排查配置问题。
		LogJobFailureWithInput(baseJob.ID, baseJob.Args, err)
		return err
	}

	var argsMap map[string]string
	argsMap, err = parseJobArgs(baseJob.Args)
	if err != nil {
		// 参数解析失败时，保留原始入参到任务日志，便于定位非法配置。
		LogJobFailureWithInput(baseJob.ID, baseJob.Args, err)
		return err
	}

	execJob := &ExecJob{
		JobID:        baseJob.ID,
		Args:         argsMap,
		InvokeTarget: invokeTarget,
	}
	return execJob.Execute()
}

// reloadJobs 重载数据库中的全部定时任务状态。
func (c *CronServer) reloadJobs(ctx context.Context) error {
	list, err := c.baseJobRepo.List(ctx)
	if err != nil {
		return err
	}

	startedJobs := make([]*models.BaseJob, 0)
	for _, item := range list {
		// 空任务记录不参与后续重载。
		if item == nil {
			continue
		}

		err = c.StopJob(ctx, item)
		if err != nil {
			return err
		}

		// 未启用任务只重置状态，不重新注册。
		if item.Status != _const.STATUS_ENABLE {
			continue
		}

		err = c.StartJob(ctx, item)
		// 重载启用任务失败时，直接中断启动流程。
		if err != nil {
			// 回滚失败时仅记录日志，优先保留原始启动错误。
			for i := len(startedJobs) - 1; i >= 0; i-- {
				startedJob := startedJobs[i]
				// 空任务记录不参与回滚。
				if startedJob == nil {
					continue
				}
				rollbackErr := c.StopJob(ctx, startedJob)
				if rollbackErr != nil {
					log.Error(fmt.Sprintf("cron rollback started jobs failed, err=%v", rollbackErr))
					break
				}
			}
			return err
		}
		startedJobs = append(startedJobs, item)
	}
	return nil
}

// lookupTaskExec 按调用目标名称查找任务执行器。
func (c *CronServer) lookupTaskExec(invokeTarget string) (TaskExec, error) {
	// 调用目标为空时，无法定位实际任务实现。
	if invokeTarget == "" {
		return nil, errorsx.ResourceNotFound("调用目标不存在")
	}

	invokeTargetExec, ok := c.registry.Lookup(invokeTarget)
	// 调用目标未注册时，直接返回明确错误。
	if !ok || invokeTargetExec == nil {
		return nil, errorsx.ResourceNotFound("调用目标不存在")
	}
	return invokeTargetExec, nil
}

// updateBaseJobEntryID 更新任务的调度 entryID。
func (c *CronServer) updateBaseJobEntryID(ctx context.Context, jobID int64, entryID int32) error {
	query := c.baseJobRepo.Query(ctx).BaseJob
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(jobID)).
		Updates(map[string]interface{}{
			"entry_id": entryID,
		})
	return err
}

// parseJobArgs 解析任务参数 JSON 为执行参数 map。
func parseJobArgs(rawArgs string) (map[string]string, error) {
	// 空参数直接返回空 map，避免上层判空分支过多。
	if rawArgs == "" {
		return map[string]string{}, nil
	}

	args := make([]*systemadminv1.BaseJobArgs, 0)
	err := json.Unmarshal([]byte(rawArgs), &args)
	if err != nil {
		return nil, err
	}

	argsMap := make(map[string]string, len(args))
	for _, item := range args {
		// 空参数项或空 key 不参与最终执行参数组装。
		if item == nil || item.GetKey() == "" {
			continue
		}
		argsMap[item.GetKey()] = item.GetValue()
	}
	return argsMap, nil
}
