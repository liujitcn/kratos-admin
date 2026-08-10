package biz

import (
	"context"
	"strconv"
	"sync"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/pkg/biz"

	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

var tree *commonv1.AppTreeOptionResponse
var lock sync.RWMutex

// BaseAreaCase 行政区域业务处理对象
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepository
	mapper *mapper.CopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务处理对象
func NewBaseAreaCase(
	baseCase *biz.BaseCase,
	baseAreaRepo *data.BaseAreaRepository,
) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:           baseCase,
		BaseAreaRepository: baseAreaRepo,
		mapper:             mapper.NewCopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea](),
	}
}

// TreeBaseArea 查询行政区域树形列表
func (c *BaseAreaCase) TreeBaseArea(ctx context.Context) (*commonv1.AppTreeOptionResponse, error) {
	lock.RLock()
	defer lock.RUnlock()
	// 树缓存尚未初始化时，从数据库加载并构建整棵区域树。
	if tree == nil {
		// 首次访问时从数据库加载并缓存，避免重复构树
		query := c.Query(ctx).BaseArea
		list, err := c.List(ctx, repository.Order(query.ID.Asc()))
		if err != nil {
			return nil, err
		}
		tree = &commonv1.AppTreeOptionResponse{
			List: c.buildTree(list, 0),
		}
	}
	return tree, nil
}

// 递归构建行政区域树
func (c *BaseAreaCase) buildTree(list []*models.BaseArea, parentID int64) []*commonv1.AppTreeOptionResponse_Option {
	var res []*commonv1.AppTreeOptionResponse_Option
	for _, item := range list {
		// 仅把当前父节点下的区域挂到本层结果里。
		if item.ParentID == parentID {
			option := c.mapper.ToDTO(item)
			option.Value = strconv.FormatInt(item.ID, 10)
			option.Text = item.Name
			option.Children = c.buildTree(list, item.ID)
			res = append(res, option)
		}
	}
	return res
}
