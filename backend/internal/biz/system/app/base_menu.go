package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/pkg/biz"
	coreconst "github.com/liujitcn/kratos-core/pkg/const"
	coreLocale "github.com/liujitcn/kratos-core/pkg/locale"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMenuCase 移动端菜单业务处理对象。
type BaseMenuCase struct {
	*biz.BaseCase
	*data.BaseMenuRepository
	translationCase *adminbiz.BaseTranslationCase
	mapper          *mapper.CopierMapper[systemappv1.BaseMenu, models.BaseMenu]
}

// NewBaseMenuCase 创建移动端菜单业务处理对象。
func NewBaseMenuCase(baseCase *biz.BaseCase, baseMenuRepo *data.BaseMenuRepository, translationCase *adminbiz.BaseTranslationCase) *BaseMenuCase {
	menuMapper := mapper.NewCopierMapper[systemappv1.BaseMenu, models.BaseMenu]()
	menuMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemappv1.BaseMenuMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:           baseCase,
		BaseMenuRepository: baseMenuRepo,
		translationCase:    translationCase,
		mapper:             menuMapper,
	}
}

// ListBaseMenu 查询固定移动端根目录下的完整启用页面层级。
func (c *BaseMenuCase) ListBaseMenu(ctx context.Context) ([]*systemappv1.BaseMenu, error) {
	query := c.Query(ctx).BaseMenu
	items := make([]*systemappv1.BaseMenu, 0)
	menuIDs := make([]int64, 0)
	parentIDs := []int64{_const.BASE_MENU_APP_ROOT_ID}
	visited := map[int64]struct{}{_const.BASE_MENU_APP_ROOT_ID: {}}
	var err error
	for len(parentIDs) > 0 {
		opts := make([]repository.QueryOption, 0, 4)
		opts = append(opts, repository.Where(query.ParentID.In(parentIDs...)))
		opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
		opts = append(opts, repository.Where(query.Status.Eq(coreconst.Status_STATUS_ENABLE)))
		opts = append(opts, repository.Order(query.Sort.Asc(), query.ID.Asc()))
		var children []*models.BaseMenu
		children, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}

		parentIDs = make([]int64, 0, len(children))
		for _, child := range children {
			if _, exists := visited[child.ID]; exists {
				continue
			}
			visited[child.ID] = struct{}{}
			items = append(items, c.mapper.ToDTO(child))
			menuIDs = append(menuIDs, child.ID)
			parentIDs = append(parentIDs, child.ID)
		}
	}
	var titles map[int64]string
	titles, err = c.translationCase.GetBaseTranslationNameMapByLocale(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU, coreLocale.FromContext(ctx), menuIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if title := titles[item.GetId()]; title != "" && item.GetMeta() != nil {
			item.Meta.Title = title
		}
	}
	return items, nil
}
