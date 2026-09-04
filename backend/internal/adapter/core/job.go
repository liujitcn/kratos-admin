package core

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coredata "github.com/liujitcn/kratos-core/data"
)

// JobStoreAdapter 将 Admin 的任务生成仓储适配为 Core 任务接口。
type JobStoreAdapter struct {
	jobRepository    *data.BaseJobRepository
	jobLogRepository *data.BaseJobLogRepository
}

// NewJobStoreAdapter 创建 Core 任务存储适配器。
func NewJobStoreAdapter(
	jobRepository *data.BaseJobRepository,
	jobLogRepository *data.BaseJobLogRepository,
) *JobStoreAdapter {
	return &JobStoreAdapter{jobRepository: jobRepository, jobLogRepository: jobLogRepository}
}

// List 查询 Core 调度所需的任务字段。
func (s *JobStoreAdapter) List(ctx context.Context) ([]coredata.JobRecord, error) {
	items, err := s.jobRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]coredata.JobRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.JobRecord{
			ID:             item.ID,
			InvokeTarget:   item.InvokeTarget,
			Args:           item.Args,
			CronExpression: item.CronExpression,
			Status:         item.Status,
			EntryID:        item.EntryID,
		})
	}
	return result, nil
}

// FindByID 按编号查询 Core 调度所需的任务字段。
func (s *JobStoreAdapter) FindByID(ctx context.Context, id int64) (coredata.JobRecord, error) {
	item, err := s.jobRepository.FindByID(ctx, id)
	if err != nil {
		return coredata.JobRecord{}, err
	}
	return coredata.JobRecord{
		ID:             item.ID,
		InvokeTarget:   item.InvokeTarget,
		Args:           item.Args,
		CronExpression: item.CronExpression,
		Status:         item.Status,
		EntryID:        item.EntryID,
	}, nil
}

// UpdateEntryID 更新任务的 Cron 入口编号。
func (s *JobStoreAdapter) UpdateEntryID(ctx context.Context, id int64, entryID int32) error {
	query := s.jobRepository.Query(ctx).BaseJob
	_, err := query.WithContext(ctx).Where(query.ID.Eq(id)).UpdateSimple(query.EntryID.Value(entryID))
	return err
}

// CreateLog 写入 Core 产生的任务执行日志。
func (s *JobStoreAdapter) CreateLog(ctx context.Context, item coredata.JobLogRecord) error {
	return s.jobLogRepository.Create(ctx, &models.BaseJobLog{
		ID:          item.ID,
		JobID:       item.JobID,
		Input:       item.Input,
		Output:      item.Output,
		Error:       item.Error,
		Status:      item.Status,
		ProcessTime: item.ProcessTime,
		ExecuteTime: item.ExecuteTime,
	})
}

var _ coredata.JobStore = (*JobStoreAdapter)(nil)
