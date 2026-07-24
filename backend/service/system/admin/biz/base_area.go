package biz

import (
	"context"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseAreaCase 行政区域业务实例。
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepository
	formMapper *mapper.CopierMapper[systemadminv1.BaseAreaForm, models.BaseArea]
	mapper     *mapper.CopierMapper[systemadminv1.BaseArea, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务实例。
func NewBaseAreaCase(baseCase *biz.BaseCase, baseAreaRepo *data.BaseAreaRepository) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:           baseCase,
		BaseAreaRepository: baseAreaRepo,
		formMapper:         mapper.NewCopierMapper[systemadminv1.BaseAreaForm, models.BaseArea](),
		mapper:             mapper.NewCopierMapper[systemadminv1.BaseArea, models.BaseArea](),
	}
}

// OptionBaseArea 查询行政区域树形选择。
func (c *BaseAreaCase) OptionBaseArea(ctx context.Context, req *systemadminv1.OptionBaseAreaRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseArea

	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.ID.Asc()))
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseAreaParentIDsWithChildren(ctx, list)
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &commonv1.TreeOptionResponse{List: c.buildOptionBaseAreaOption(list, parentID, req.GetLazy(), hasChildren)}, nil
}

// TreeBaseArea 查询行政区域树形列表。
func (c *BaseAreaCase) TreeBaseArea(ctx context.Context, req *systemadminv1.TreeBaseAreaRequest) (*systemadminv1.TreeBaseAreaResponse, error) {
	query := c.Query(ctx).BaseArea
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.ID.Asc()))
	// 搜索时跨层级匹配，避免懒加载树无法检索未展开节点。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	} else if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseAreaParentIDsWithChildren(ctx, list)
	if err != nil {
		return nil, err
	}
	baseAreas := make([]*systemadminv1.BaseArea, 0, len(list))
	if req.GetName() != "" {
		for _, item := range list {
			baseArea := c.mapper.ToDTO(item)
			_, baseArea.HasChildren = hasChildren[item.ID]
			baseAreas = append(baseAreas, baseArea)
		}
	} else {
		parentID := int64(0)
		if req.GetLazy() {
			parentID = req.GetParentId()
		}
		baseAreas = c.buildBaseAreaTree(list, parentID, req.GetLazy(), hasChildren)
	}
	return &systemadminv1.TreeBaseAreaResponse{BaseAreas: baseAreas}, nil
}

// GetBaseArea 查询行政区域详情。
func (c *BaseAreaCase) GetBaseArea(ctx context.Context, id int64) (*systemadminv1.BaseAreaForm, error) {
	baseArea, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseArea), nil
}

// CreateBaseArea 创建行政区域。
func (c *BaseAreaCase) CreateBaseArea(ctx context.Context, req *systemadminv1.BaseAreaForm) error {
	baseArea := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseArea)
}

// UpdateBaseArea 更新行政区域。
func (c *BaseAreaCase) UpdateBaseArea(ctx context.Context, id int64, req *systemadminv1.BaseAreaForm) error {
	baseArea := c.formMapper.ToEntity(req)
	baseArea.ID = id
	return c.UpdateByID(ctx, baseArea)
}

// DeleteBaseArea 删除行政区域。
func (c *BaseAreaCase) DeleteBaseArea(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	query := c.Query(ctx).BaseArea
	for _, parentID := range idList {
		count, err := c.Count(ctx, repository.Where(query.ParentID.Eq(parentID)))
		if err != nil {
			return err
		}
		if count > 0 {
			return errorsx.HasChildrenConflict("删除行政区域失败，下面有行政区域", "base_area", "base_area")
		}
	}
	return c.DeleteByIDs(ctx, idList)
}

// buildOptionBaseAreaOption 构建行政区域树形选择。
func (c *BaseAreaCase) buildOptionBaseAreaOption(
	list []*models.BaseArea,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range list {
		if int64(item.ParentID) != parentID {
			continue
		}
		option := &commonv1.TreeOptionResponse_Option{Label: item.Name, Value: int64(item.ID)}
		if lazy {
			_, option.HasChildren = hasChildren[item.ID]
		} else {
			option.Children = c.buildOptionBaseAreaOption(list, int64(item.ID), false, hasChildren)
		}
		res = append(res, option)
	}
	return res
}

// buildBaseAreaTree 构建行政区域树。
func (c *BaseAreaCase) buildBaseAreaTree(
	list []*models.BaseArea,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*systemadminv1.BaseArea {
	res := make([]*systemadminv1.BaseArea, 0)
	for _, item := range list {
		if int64(item.ParentID) != parentID {
			continue
		}
		baseArea := c.mapper.ToDTO(item)
		_, baseArea.HasChildren = hasChildren[item.ID]
		if !lazy {
			baseArea.Children = c.buildBaseAreaTree(list, int64(item.ID), false, hasChildren)
		}
		res = append(res, baseArea)
	}
	return res
}

// listBaseAreaParentIDsWithChildren 查询存在子节点的区域父级编号。
func (c *BaseAreaCase) listBaseAreaParentIDsWithChildren(ctx context.Context, list []*models.BaseArea) (map[int64]struct{}, error) {
	parentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		parentIDs = append(parentIDs, item.ID)
	}
	hasChildren := make(map[int64]struct{}, len(parentIDs))
	if len(parentIDs) == 0 {
		return hasChildren, nil
	}

	query := c.Query(ctx).BaseArea
	childList, err := c.List(ctx, repository.Where(query.ParentID.In(parentIDs...)))
	if err != nil {
		return nil, err
	}
	for _, item := range childList {
		hasChildren[item.ParentID] = struct{}{}
	}
	return hasChildren, nil
}
