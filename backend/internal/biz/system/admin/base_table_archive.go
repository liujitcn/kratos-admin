package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	stringutil "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	corebiz "github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
)

var tableNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// BaseTableArchiveCase 管理表归档配置。
type BaseTableArchiveCase struct {
	*corebiz.BaseCase
	*data.BaseTableArchiveRepository
}

// NewBaseTableArchiveCase 创建表归档配置业务实例。
func NewBaseTableArchiveCase(baseCase *corebiz.BaseCase, repo *data.BaseTableArchiveRepository) *BaseTableArchiveCase {
	return &BaseTableArchiveCase{BaseCase: baseCase, BaseTableArchiveRepository: repo}
}

// PageBaseTableArchive 分页查询表归档配置。
func (c *BaseTableArchiveCase) PageBaseTableArchive(ctx context.Context, req *adminv1.PageBaseTableArchiveRequest) (*adminv1.PageBaseTableArchiveResponse, error) {
	query := c.Query(ctx).BaseTableArchive
	opts := []repository.QueryOption{repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc())}
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
	items := make([]*adminv1.BaseTableArchiveForm, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableArchive(item))
	}
	return &adminv1.PageBaseTableArchiveResponse{BaseTableArchives: items, Total: int32(total)}, nil
}

// GetBaseTableArchive 查询表归档配置。
func (c *BaseTableArchiveCase) GetBaseTableArchive(ctx context.Context, id int64) (*adminv1.BaseTableArchiveForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableArchive(item), nil
}

// CreateBaseTableArchive 创建表归档配置。
func (c *BaseTableArchiveCase) CreateBaseTableArchive(ctx context.Context, req *adminv1.BaseTableArchiveForm) error {
	if err := validateTableArchiveForm(c.BaseCase, req); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	status := int32(req.GetStatus())
	if status == 0 {
		status = coreconst.STATUS_STATUS_DISABLE
	}
	entity := &models.BaseTableArchive{
		SourceName: req.GetSourceName(), TableName_: req.GetTableName(), ArchiveMode: int32(req.GetArchiveMode()),
		OnlineRetentionDays: req.GetOnlineRetentionDays(), ArchiveRetentionDays: req.GetArchiveRetentionDays(), BatchSize: req.GetBatchSize(),
		DeleteAfterVerify: boolToTinyint(req.GetDeleteAfterVerify()), OSSPrefix: req.GetOssPrefix(), Status: status,
		CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	return c.Create(ctx, entity)
}

// UpdateBaseTableArchive 更新表归档配置。
func (c *BaseTableArchiveCase) UpdateBaseTableArchive(ctx context.Context, req *adminv1.BaseTableArchiveForm) error {
	if err := validateTableArchiveForm(c.BaseCase, req); err != nil {
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
	entity := &models.BaseTableArchive{
		ID: req.GetId(), SourceName: req.GetSourceName(), TableName_: req.GetTableName(), ArchiveMode: int32(req.GetArchiveMode()),
		OnlineRetentionDays: req.GetOnlineRetentionDays(), ArchiveRetentionDays: req.GetArchiveRetentionDays(), BatchSize: req.GetBatchSize(),
		DeleteAfterVerify: boolToTinyint(req.GetDeleteAfterVerify()), OSSPrefix: req.GetOssPrefix(), Status: status,
		UpdatedBy: authInfo.UserId, UpdatedAt: time.Now(),
	}
	return c.UpdateByID(ctx, entity)
}

// DeleteBaseTableArchive 删除表归档配置。
func (c *BaseTableArchiveCase) DeleteBaseTableArchive(ctx context.Context, ids string) error {
	return c.DeleteByIDs(ctx, stringutil.ConvertStringToInt64Array(ids))
}

// SetBaseTableArchiveStatus 设置表归档配置状态。
func (c *BaseTableArchiveCase) SetBaseTableArchiveStatus(ctx context.Context, req *adminv1.SetBaseTableArchiveStatusRequest) error {
	if req.GetStatus() != coreconst.STATUS_STATUS_ENABLE && req.GetStatus() != coreconst.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("归档配置状态无效")
	}
	return c.UpdateByID(ctx, &models.BaseTableArchive{ID: req.GetId(), Status: req.GetStatus()})
}

// validateTableArchiveForm 校验归档配置的数据源、数据表和归档参数。
func validateTableArchiveForm(baseCase *corebiz.BaseCase, req *adminv1.BaseTableArchiveForm) error {
	if req.GetArchiveMode() != adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE && req.GetArchiveMode() != adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_OSS {
		return errorsx.InvalidArgument("归档模式无效")
	}
	if !tableNamePattern.MatchString(req.GetTableName()) || strings.HasPrefix(req.GetTableName(), "base_table_") {
		return errorsx.InvalidArgument("数据表名称不合法")
	}
	client, err := GormClientBySourceName(baseCase, req.GetSourceName())
	if err != nil {
		return errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	if !client.Migrator().HasTable(req.GetTableName()) {
		return errorsx.InvalidArgument("所选数据源中不存在该数据表")
	}
	if req.GetBatchSize() <= 0 {
		return errorsx.InvalidArgument("批处理数量必须大于零")
	}
	return nil
}

func boolToTinyint(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func toBaseTableArchive(item *models.BaseTableArchive) *adminv1.BaseTableArchiveForm {
	return &adminv1.BaseTableArchiveForm{Id: item.ID, SourceName: item.SourceName, TableName: item.TableName_, ArchiveMode: adminv1.BaseTableArchiveMode(item.ArchiveMode), OnlineRetentionDays: item.OnlineRetentionDays, ArchiveRetentionDays: item.ArchiveRetentionDays, BatchSize: item.BatchSize, DeleteAfterVerify: item.DeleteAfterVerify != 0, OssPrefix: item.OSSPrefix, Status: commonv1.Status(item.Status)}
}
