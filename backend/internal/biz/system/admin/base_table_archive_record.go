package biz

import (
	"context"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	corebiz "github.com/liujitcn/kratos-core/biz"
)

// BaseTableArchiveRecordCase 查询表归档执行记录。
type BaseTableArchiveRecordCase struct {
	*corebiz.BaseCase
	*data.BaseTableArchiveRecordRepository
}

// NewBaseTableArchiveRecordCase 创建表归档记录业务实例。
func NewBaseTableArchiveRecordCase(baseCase *corebiz.BaseCase, repo *data.BaseTableArchiveRecordRepository) *BaseTableArchiveRecordCase {
	return &BaseTableArchiveRecordCase{BaseCase: baseCase, BaseTableArchiveRecordRepository: repo}
}

// PageBaseTableArchiveRecord 分页查询表归档记录。
func (c *BaseTableArchiveRecordCase) PageBaseTableArchiveRecord(ctx context.Context, req *adminv1.PageBaseTableArchiveRecordRequest) (*adminv1.PageBaseTableArchiveRecordResponse, error) {
	query := c.Query(ctx).BaseTableArchiveRecord
	opts := []repository.QueryOption{repository.Order(query.ID.Desc())}
	if req.GetArchiveId() > 0 {
		opts = append(opts, repository.Where(query.ArchiveID.Eq(req.GetArchiveId())))
	}
	if req.GetSourceName() != "" {
		opts = append(opts, repository.Where(query.SourceName.Eq(req.GetSourceName())))
	}
	if req.GetTableName() != "" {
		opts = append(opts, repository.Where(query.TableName_.Like("%"+req.GetTableName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseTableArchiveRecord, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableArchiveRecord(item))
	}
	return &adminv1.PageBaseTableArchiveRecordResponse{BaseTableArchiveRecords: items, Total: int32(total)}, nil
}

// GetBaseTableArchiveRecord 查询表归档记录。
func (c *BaseTableArchiveRecordCase) GetBaseTableArchiveRecord(ctx context.Context, id int64) (*adminv1.BaseTableArchiveRecord, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableArchiveRecord(item), nil
}

func toBaseTableArchiveRecord(item *models.BaseTableArchiveRecord) *adminv1.BaseTableArchiveRecord {
	return &adminv1.BaseTableArchiveRecord{Id: item.ID, ArchiveId: item.ArchiveID, SourceName: item.SourceName, TableName: item.TableName_, ArchiveMode: adminv1.BaseTableArchiveMode(item.ArchiveMode), CutoffAt: item.CutoffAt.Format(time.RFC3339), Cursor: item.Cursor, ScannedRows: item.ScannedRows, ArchivedRows: item.ArchivedRows, DeletedRows: item.DeletedRows, ArchiveTableName: item.ArchiveTableName, ObjectKey: item.ObjectKey, SizeBytes: item.SizeBytes, Sha256: item.Sha256, Status: adminv1.BaseTableArchiveRecordStatus(item.Status), Error: item.Error, StartedAt: item.StartedAt.Format(time.RFC3339), FinishedAt: item.FinishedAt.Format(time.RFC3339)}
}
