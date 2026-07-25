package biz

import (
	"context"

	"github.com/liujitcn/go-utils/time"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMigrationCase 提供数据库升级历史查询业务。
type BaseMigrationCase struct {
	*data.BaseMigrationRepository
}

// NewBaseMigrationCase 创建数据库升级历史查询业务实例。
func NewBaseMigrationCase(baseMigrationRepository *data.BaseMigrationRepository) *BaseMigrationCase {
	return &BaseMigrationCase{BaseMigrationRepository: baseMigrationRepository}
}

// LatestSuccessfulVersion 查询指定迁移业务最近一次成功执行的版本。
func (c *BaseMigrationCase) LatestSuccessfulVersion(ctx context.Context, business string) (string, error) {
	query := c.Query(ctx).BaseMigration
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Business.Eq(business)))
	opts = append(opts, repository.Where(query.IsSuccess.Is(true)))
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

// PageBaseMigration 分页查询数据库升级历史。
func (c *BaseMigrationCase) PageBaseMigration(
	ctx context.Context,
	req *systemadminv1.PageBaseMigrationRequest,
) (*systemadminv1.PageBaseMigrationResponse, error) {
	query := c.Query(ctx).BaseMigration
	opts := make([]repository.QueryOption, 0, 4)
	if req.GetBusiness() != "" {
		opts = append(opts, repository.Where(query.Business.Eq(req.GetBusiness())))
	}
	if req.Version != nil {
		opts = append(opts, repository.Where(query.Version.Like("%"+req.GetVersion()+"%")))
	}
	if req.IsSuccess != nil {
		opts = append(opts, repository.Where(query.IsSuccess.Is(req.GetIsSuccess())))
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
	res := &systemadminv1.PageBaseMigrationResponse{
		BaseMigrations: make([]*systemadminv1.BaseMigrationListItem, 0, len(histories)),
		Total:          int32(total),
	}
	for _, item := range histories {
		res.BaseMigrations = append(res.BaseMigrations, &systemadminv1.BaseMigrationListItem{
			Id:        item.ID,
			Business:  item.Business,
			Version:   item.Version,
			CreatedAt: time.TimeToTimeString(item.CreatedAt),
			IsSuccess: item.IsSuccess,
		})
	}
	return res, nil
}

// GetBaseMigration 查询数据库迁移记录详情。
func (c *BaseMigrationCase) GetBaseMigration(ctx context.Context, id int64) (*systemadminv1.BaseMigration, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.BaseMigration{
		Id:          item.ID,
		Business:    item.Business,
		Version:     item.Version,
		UpSql:       item.UpSql,
		DownSql:     item.DownSql,
		Description: item.Description,
		CreatedAt:   time.TimeToTimeString(item.CreatedAt),
		IsSuccess:   item.IsSuccess,
	}, nil
}
