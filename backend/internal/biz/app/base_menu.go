package biz

import (
	"context"

	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMenuCase 移动端菜单业务处理对象。
type BaseMenuCase struct {
	*biz.BaseCase
	*data.BaseMenuRepository
	mapper *mapper.CopierMapper[systemappv1.BaseMenu, models.BaseMenu]
}

// NewBaseMenuCase 创建移动端菜单业务处理对象。
func NewBaseMenuCase(baseCase *biz.BaseCase, baseMenuRepo *data.BaseMenuRepository) *BaseMenuCase {
	menuMapper := mapper.NewCopierMapper[systemappv1.BaseMenu, models.BaseMenu]()
	menuMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemappv1.BaseMenuMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:           baseCase,
		BaseMenuRepository: baseMenuRepo,
		mapper:             menuMapper,
	}
}

// ListBaseMenu 查询固定移动端根菜单下的启用页面配置。
func (c *BaseMenuCase) ListBaseMenu(ctx context.Context) ([]*systemappv1.BaseMenu, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.ParentID.Eq(_const.BASE_MENU_APP_ROOT_ID)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	opts = append(opts, repository.Order(query.Sort.Asc(), query.ID.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	items := make([]*systemappv1.BaseMenu, 0, len(list))
	for _, item := range list {
		items = append(items, c.mapper.ToDTO(item))
	}
	return items, nil
}
