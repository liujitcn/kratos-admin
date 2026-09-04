package biz

import (
	"context"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
)

// BaseTableBackupRecordCase 查询数据库备份记录。
type BaseTableBackupRecordCase struct {
	*biz.BaseCase
	*data.BaseTableBackupRecordRepository
}

// NewBaseTableBackupRecordCase 创建数据库备份记录业务实例。
func NewBaseTableBackupRecordCase(baseCase *biz.BaseCase, repo *data.BaseTableBackupRecordRepository) *BaseTableBackupRecordCase {
	return &BaseTableBackupRecordCase{BaseCase: baseCase, BaseTableBackupRecordRepository: repo}
}

// PageBaseTableBackupRecord 分页查询数据库备份记录。
func (c *BaseTableBackupRecordCase) PageBaseTableBackupRecord(ctx context.Context, req *adminv1.PageBaseTableBackupRecordRequest) (*adminv1.PageBaseTableBackupRecordResponse, error) {
	query := c.Query(ctx).BaseTableBackupRecord
	opts := []repository.QueryOption{repository.Order(query.ID.Desc())}
	if req.GetBackupId() > 0 {
		opts = append(opts, repository.Where(query.BackupID.Eq(req.GetBackupId())))
	}
	if req.GetSourceName() != "" {
		opts = append(opts, repository.Where(query.SourceName.Eq(req.GetSourceName())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseTableBackupRecord, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableBackupRecord(item))
	}
	return &adminv1.PageBaseTableBackupRecordResponse{BaseTableBackupRecords: items, Total: int32(total)}, nil
}

// GetBaseTableBackupRecord 查询数据库备份记录。
func (c *BaseTableBackupRecordCase) GetBaseTableBackupRecord(ctx context.Context, id int64) (*adminv1.BaseTableBackupRecord, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableBackupRecord(item), nil
}

func toBaseTableBackupRecord(item *models.BaseTableBackupRecord) *adminv1.BaseTableBackupRecord {
	return &adminv1.BaseTableBackupRecord{Id: item.ID, BackupId: item.BackupID, SourceName: item.SourceName, DatabaseName: item.DatabaseName, BackupType: adminv1.BaseTableBackupType(item.BackupType), ObjectKey: item.ObjectKey, SizeBytes: item.SizeBytes, Sha256: item.Sha256, Hmac: item.Hmac, Status: adminv1.BaseTableBackupRecordStatus(item.Status), Error: item.Error, StartedAt: item.StartedAt.Format(time.RFC3339), FinishedAt: item.FinishedAt.Format(time.RFC3339), VerifiedAt: item.VerifiedAt.Format(time.RFC3339)}
}
