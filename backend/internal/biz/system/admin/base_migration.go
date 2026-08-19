package biz

import (
	"context"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm/migration"
	"gorm.io/gen/field"
)

// BaseMigrationCase 提供数据库升级历史查询业务。
type BaseMigrationCase struct {
	*biz.BaseCase
	*data.BaseMigrationRepository
	listMapper *mapper.CopierMapper[adminv1.BaseMigrationListItem, models.BaseMigration]
	mapper     *mapper.CopierMapper[adminv1.BaseMigration, models.BaseMigration]
}

// NewBaseMigrationCase 创建数据库升级历史查询业务实例。
func NewBaseMigrationCase(baseCase *biz.BaseCase, baseMigrationRepository *data.BaseMigrationRepository) *BaseMigrationCase {
	return &BaseMigrationCase{
		BaseCase:                baseCase,
		BaseMigrationRepository: baseMigrationRepository,
		listMapper:              mapper.NewCopierMapper[adminv1.BaseMigrationListItem, models.BaseMigration](),
		mapper:                  mapper.NewCopierMapper[adminv1.BaseMigration, models.BaseMigration](),
	}
}

// PageBaseMigration 分页查询数据库升级历史。
func (c *BaseMigrationCase) PageBaseMigration(
	ctx context.Context,
	req *adminv1.PageBaseMigrationRequest,
) (*adminv1.PageBaseMigrationResponse, error) {
	query := c.Query(ctx).BaseMigration
	opts := make([]repository.QueryOption, 0, 4)
	if req.GetDataSource() != "" {
		opts = append(opts, repository.Where(query.DataSource.Eq(req.GetDataSource())))
	}
	if req.Module != nil && req.GetModule() != "" {
		opts = append(opts, repository.Where(query.Module.Eq(req.GetModule())))
	}
	if req.Version != nil {
		opts = append(opts, repository.Where(query.Version.Like("%"+req.GetVersion()+"%")))
	}
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Order(query.ID.Desc()))
	histories, total, err := c.Page(
		ctx,
		req.GetPageNum(),
		req.GetPageSize(),
		opts...,
	)
	if err != nil {
		return nil, err
	}
	res := &adminv1.PageBaseMigrationResponse{
		BaseMigrations: make([]*adminv1.BaseMigrationListItem, 0, len(histories)),
		Total:          int32(total),
	}
	for _, item := range histories {
		res.BaseMigrations = append(res.BaseMigrations, c.listMapper.ToDTO(item))
	}
	return res, nil
}

// GetBaseMigration 查询数据库迁移记录详情。
func (c *BaseMigrationCase) GetBaseMigration(ctx context.Context, id int64) (*adminv1.BaseMigration, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.mapper.ToDTO(item), nil
}

// LatestVersion 查询指定模块和数据源最近一次已记录的版本。
func (c *BaseMigrationCase) LatestVersion(ctx context.Context, module migration.ModuleName, dataSource string) (string, error) {
	query := c.Query(ctx).BaseMigration
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(field.Or(
		query.Module.Eq(module.String()),
		query.Module.Eq(""),
	)))
	opts = append(opts, repository.Where(query.DataSource.Eq(dataSource)))
	opts = append(opts, repository.Order(query.ID.Desc()))
	opts = append(opts, repository.Limit(1))
	histories, err := c.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	if len(histories) == 0 {
		return "", nil
	}
	return histories[0].Version, nil
}
