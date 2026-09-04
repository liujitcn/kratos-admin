package biz

import (
	"context"
	"time"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseTableBackupCase 管理数据库备份配置。
type BaseTableBackupCase struct {
	*biz.BaseCase
	*data.BaseTableBackupRepository
}

// NewBaseTableBackupCase 创建数据库备份配置业务实例。
func NewBaseTableBackupCase(baseCase *biz.BaseCase, repo *data.BaseTableBackupRepository) *BaseTableBackupCase {
	return &BaseTableBackupCase{BaseCase: baseCase, BaseTableBackupRepository: repo}
}

// PageBaseTableBackup 分页查询数据库备份配置。
func (c *BaseTableBackupCase) PageBaseTableBackup(ctx context.Context, req *adminv1.PageBaseTableBackupRequest) (*adminv1.PageBaseTableBackupResponse, error) {
	query := c.Query(ctx).BaseTableBackup
	opts := []repository.QueryOption{repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc())}
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
	items := make([]*adminv1.BaseTableBackupForm, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableBackup(item))
	}
	return &adminv1.PageBaseTableBackupResponse{BaseTableBackups: items, Total: int32(total)}, nil
}

// GetBaseTableBackup 查询数据库备份配置。
func (c *BaseTableBackupCase) GetBaseTableBackup(ctx context.Context, id int64) (*adminv1.BaseTableBackupForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableBackup(item), nil
}

// CreateBaseTableBackup 创建数据库备份配置。
func (c *BaseTableBackupCase) CreateBaseTableBackup(ctx context.Context, req *adminv1.BaseTableBackupForm) error {
	if err := validateTableBackupForm(c.BaseCase, req); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	status := int32(req.GetStatus())
	if status == 0 {
		status = _const.STATUS_STATUS_DISABLE
	}
	now := time.Now()
	entity := &models.BaseTableBackup{SourceName: req.GetSourceName(), BackupType: int32(req.GetBackupType()), OSSPrefix: req.GetOssPrefix(), RetentionCount: req.GetRetentionCount(), Status: status, CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: now, UpdatedAt: now}
	return c.Create(ctx, entity)
}

// UpdateBaseTableBackup 更新数据库备份配置。
func (c *BaseTableBackupCase) UpdateBaseTableBackup(ctx context.Context, req *adminv1.BaseTableBackupForm) error {
	if err := validateTableBackupForm(c.BaseCase, req); err != nil {
		return err
	}
	old, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	status := int32(req.GetStatus())
	if status == 0 {
		status = old.Status
	}
	entity := &models.BaseTableBackup{ID: req.GetId(), SourceName: req.GetSourceName(), BackupType: int32(req.GetBackupType()), OSSPrefix: req.GetOssPrefix(), RetentionCount: req.GetRetentionCount(), Status: status, UpdatedBy: authInfo.UserId, UpdatedAt: time.Now()}
	return c.UpdateByID(ctx, entity)
}

// DeleteBaseTableBackup 删除数据库备份配置。
func (c *BaseTableBackupCase) DeleteBaseTableBackup(ctx context.Context, ids string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(ids))
}

// SetBaseTableBackupStatus 设置数据库备份配置状态。
func (c *BaseTableBackupCase) SetBaseTableBackupStatus(ctx context.Context, req *adminv1.SetBaseTableBackupStatusRequest) error {
	if req.GetStatus() != _const.STATUS_STATUS_ENABLE && req.GetStatus() != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("备份配置状态无效")
	}
	return c.UpdateByID(ctx, &models.BaseTableBackup{ID: req.GetId(), Status: req.GetStatus()})
}

// validateTableBackupForm 校验备份配置的数据源和备份参数。
func validateTableBackupForm(baseCase *biz.BaseCase, req *adminv1.BaseTableBackupForm) error {
	if req.GetBackupType() != adminv1.BaseTableBackupType_BASE_TABLE_BACKUP_TYPE_FULL {
		return errorsx.InvalidArgument("当前仅支持全量备份")
	}
	if _, err := GormClientBySourceName(baseCase, req.GetSourceName()); err != nil {
		return errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	if req.GetRetentionCount() <= 0 {
		return errorsx.InvalidArgument("保留数量必须大于零")
	}
	return nil
}

func toBaseTableBackup(item *models.BaseTableBackup) *adminv1.BaseTableBackupForm {
	return &adminv1.BaseTableBackupForm{Id: item.ID, SourceName: item.SourceName, BackupType: adminv1.BaseTableBackupType(item.BackupType), OssPrefix: item.OSSPrefix, RetentionCount: item.RetentionCount, Status: commonv1.Status(item.Status)}
}
