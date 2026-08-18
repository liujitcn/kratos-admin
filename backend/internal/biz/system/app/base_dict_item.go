package biz

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"

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

// 按字典编号列表查询启用中的字典项
func (c *BaseDictItemCase) findByDictIDs(ctx context.Context, dictIDs []int64) ([]*models.BaseDictItem, error) {
	query := c.Query(ctx).BaseDictItem
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.DictID.In(dictIDs...)))
	opts = append(opts, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)))
	return c.List(ctx, opts...)
}
