package biz

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos/v3/log"
	appv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/cache"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMenuCase 移动端菜单业务处理对象。
type BaseMenuCase struct {
	*biz.BaseCase
	*data.BaseMenuRepository
	mapper *mapper.CopierMapper[appv1.BaseMenu, models.BaseMenu]
}

// NewBaseMenuCase 创建移动端菜单业务处理对象。
func NewBaseMenuCase(baseCase *biz.BaseCase, baseMenuRepo *data.BaseMenuRepository) *BaseMenuCase {
	menuMapper := mapper.NewCopierMapper[appv1.BaseMenu, models.BaseMenu]()
	menuMapper.AppendConverters(mapper.NewJSONTypeConverter[*appv1.BaseMenuMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:           baseCase,
		BaseMenuRepository: baseMenuRepo,
		mapper:             menuMapper,
	}
}

// ListBaseMenu 查询固定移动端根目录下的完整启用页面层级。
func (c *BaseMenuCase) ListBaseMenu(ctx context.Context) ([]*appv1.BaseMenu, error) {
	revision, cacheEnabled := cache.ReadRevision(c.Cache, _const.APP_MENU_CACHE_REVISION_KEY)
	if cacheEnabled {
		var cached string
		var err error
		cached, err = c.Cache.Get(_const.AppMenuCacheKey(revision))
		if err == nil {
			items := make([]*appv1.BaseMenu, 0)
			err = json.Unmarshal([]byte(cached), &items)
			if err == nil {
				return items, nil
			}
		}
	}
	query := c.Query(ctx).BaseMenu
	items := make([]*appv1.BaseMenu, 0)
	parentIDs := []int64{_const.BASE_MENU_APP_ROOT_ID}
	visited := map[int64]struct{}{_const.BASE_MENU_APP_ROOT_ID: {}}
	var err error
	for len(parentIDs) > 0 {
		opts := make([]repository.QueryOption, 0, 4)
		opts = append(opts, repository.Where(query.ParentID.In(parentIDs...)))
		opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
		opts = append(opts, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)))
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
			parentIDs = append(parentIDs, child.ID)
		}
	}
	if cacheEnabled {
		var payload []byte
		payload, err = json.Marshal(items)
		if err != nil {
			log.Error("MarshalAppBaseMenuCache", "error", err)
		} else {
			err = c.Cache.Set(_const.AppMenuCacheKey(revision), string(payload), _const.APP_DATA_CACHE_EXPIRE)
			if err != nil {
				log.Error("SetAppBaseMenuCache", "error", err)
			}
		}
	}
	return items, nil
}
