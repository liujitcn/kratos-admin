package biz

import (
	"context"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDictItemCase 字典项业务处理对象
type BaseDictItemCase struct {
	*biz.BaseCase
	*data.BaseDictItemRepository
	baseDictRepo *data.BaseDictRepository
}

// NewBaseDictItemCase 创建字典项业务处理对象
func NewBaseDictItemCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepository, baseDictItemRepo *data.BaseDictItemRepository) *BaseDictItemCase {
	return &BaseDictItemCase{
		BaseCase:               baseCase,
		baseDictRepo:           baseDictRepo,
		BaseDictItemRepository: baseDictItemRepo,
	}
}

// FindLabelByCodeAndValue 按字典编码和值查询标签。
func (c *BaseDictItemCase) FindLabelByCodeAndValue(ctx context.Context, code, value string) (string, error) {
	baseDictItemQuery := c.Query(ctx).BaseDictItem
	baseDictQuery := c.baseDictRepo.Query(ctx).BaseDict
	// 通过字典表和字典项表联查，直接返回展示标签
	query := baseDictItemQuery.WithContext(ctx).Select(baseDictItemQuery.Label).Join(baseDictQuery, baseDictItemQuery.DictID.EqCol(baseDictQuery.ID))
	query = query.Where(baseDictQuery.Code.Eq(code))
	query = query.Where(baseDictItemQuery.Value.Eq(value))

	var label string
	err := query.Scan(&label)
	return label, err
}

// 按字典编号列表查询启用中的字典项
func (c *BaseDictItemCase) findByDictIDs(ctx context.Context, dictIDs []int64) ([]*models.BaseDictItem, error) {
	query := c.Query(ctx).BaseDictItem
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.DictID.In(dictIDs...)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	return c.List(ctx, opts...)
}
