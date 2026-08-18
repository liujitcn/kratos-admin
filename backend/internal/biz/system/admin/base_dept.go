package biz

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDeptCase 部门业务实例
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepository
	formMapper *mapper.CopierMapper[systemadminv1.BaseDeptForm, models.BaseDept]
	mapper     *mapper.CopierMapper[systemadminv1.BaseDept, models.BaseDept]
}

// NewBaseDeptCase 创建部门业务实例
func NewBaseDeptCase(
	baseCase *biz.BaseCase,
	baseDeptRepo *data.BaseDeptRepository,
) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:           baseCase,
		BaseDeptRepository: baseDeptRepo,
		formMapper:         mapper.NewCopierMapper[systemadminv1.BaseDeptForm, models.BaseDept](),
		mapper:             mapper.NewCopierMapper[systemadminv1.BaseDept, models.BaseDept](),
	}
}

// OptionBaseDept 查询部门选项
func (c *BaseDeptCase) OptionBaseDept(ctx context.Context, req *systemadminv1.OptionBaseDeptRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.TenantId != nil && req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseDeptParentIDsWithChildren(ctx, list, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &commonv1.TreeOptionResponse{List: c.buildBaseDeptOption(list, parentID, req.GetLazy(), hasChildren)}, nil
}

// TreeBaseDept 查询部门树
func (c *BaseDeptCase) TreeBaseDept(ctx context.Context, req *systemadminv1.TreeBaseDeptRequest) (*systemadminv1.TreeBaseDeptResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.TenantId != nil && req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseDeptParentIDsWithChildren(ctx, list, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &systemadminv1.TreeBaseDeptResponse{BaseDepts: c.buildBaseDeptTree(list, parentID, req.GetLazy(), hasChildren)}, nil
}

// GetBaseDept 获取部门
func (c *BaseDeptCase) GetBaseDept(ctx context.Context, id int64) (*systemadminv1.BaseDeptForm, error) {
	baseDept, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseDept), nil
}

// CreateBaseDept 创建部门
func (c *BaseDeptCase) CreateBaseDept(ctx context.Context, req *systemadminv1.BaseDeptForm) error {
	baseDept := c.formMapper.ToEntity(req)

	parentID := req.GetParentId()
	// 存在父部门时，继承父部门路径并追加当前节点。
	var parentPath string
	var err error
	if parentID != 0 {
		var parentDept *models.BaseDept
		parentDept, err = c.FindByID(ctx, parentID)
		if err != nil {
			return errorsx.Internal("创建部门失败，更新路径错误").WithCause(err)
		}
		if baseDept.TenantID > 0 && baseDept.TenantID != parentDept.TenantID {
			return errorsx.InvalidArgument("上级部门与所属租户不一致")
		}
		baseDept.TenantID = parentDept.TenantID
		parentPath = parentDept.Path
	}

	err = c.Create(ctx, baseDept)
	if err != nil {
		return err
	}
	if parentPath != "" {
		baseDept.Path = fmt.Sprintf("%s/%d", parentPath, baseDept.ID)
	} else {
		baseDept.Path = fmt.Sprintf("/0/%d", baseDept.ID)
	}
	return c.UpdateByID(ctx, baseDept)
}

// UpdateBaseDept 更新部门
func (c *BaseDeptCase) UpdateBaseDept(ctx context.Context, req *systemadminv1.BaseDeptForm) error {
	oldBaseDept, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	baseDept := c.formMapper.ToEntity(req)
	baseDept.TenantID = oldBaseDept.TenantID
	baseDept.Path = oldBaseDept.Path
	parentID := req.GetParentId()
	if parentID != 0 {
		var parentDept *models.BaseDept
		parentDept, err = c.FindByID(ctx, parentID)
		if err != nil {
			return err
		}
		if parentDept.TenantID != oldBaseDept.TenantID {
			return errorsx.InvalidArgument("上级部门与所属租户不一致")
		}
	}
	return c.UpdateByID(ctx, baseDept)
}

// DeleteBaseDept 删除部门
func (c *BaseDeptCase) DeleteBaseDept(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseDept

	for _, deptID := range ids {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ParentID.Eq(deptID)))
		count, err := c.Count(ctx, opts...)
		if err != nil {
			return err
		}
		// 仍然存在子部门时，禁止删除当前部门。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除部门失败，下面有部门", "base_dept", "base_dept")
		}
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseDeptStatus 设置部门状态
func (c *BaseDeptCase) SetBaseDeptStatus(ctx context.Context, req *systemadminv1.SetBaseDeptStatusRequest) error {
	query := c.Query(ctx).BaseDept

	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ParentID.Eq(req.GetId())))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return err
	}
	// 存在子部门时，不允许直接调整当前部门状态。
	if count > 0 {
		return errorsx.HasChildrenConflict("设置状态失败，下面有部门", "base_dept", "base_dept")
	}

	return c.UpdateByID(ctx, &models.BaseDept{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildBaseDeptTree 构建部门树
func (c *BaseDeptCase) buildBaseDeptTree(
	list []*models.BaseDept,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*systemadminv1.BaseDept {
	res := make([]*systemadminv1.BaseDept, 0)
	for _, item := range list {
		// 非当前父节点的部门不参与当前层级构建。
		if item.ParentID != parentID {
			continue
		}
		baseDept := c.mapper.ToDTO(item)
		_, baseDept.HasChildren = hasChildren[item.ID]
		if !lazy {
			baseDept.Children = c.buildBaseDeptTree(list, item.ID, false, hasChildren)
		}
		res = append(res, baseDept)
	}
	return res
}

// buildBaseDeptOption 构建部门选项树
func (c *BaseDeptCase) buildBaseDeptOption(
	list []*models.BaseDept,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range list {
		// 非当前父节点的部门不参与当前层级选项构建。
		if item.ParentID != parentID {
			continue
		}
		option := &commonv1.TreeOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		}
		_, option.HasChildren = hasChildren[item.ID]
		if !lazy {
			option.Children = c.buildBaseDeptOption(list, item.ID, false, hasChildren)
		}
		res = append(res, option)
	}
	return res
}

// listBaseDeptParentIDsWithChildren 查询存在子节点的部门父级编号。
func (c *BaseDeptCase) listBaseDeptParentIDsWithChildren(
	ctx context.Context,
	list []*models.BaseDept,
	tenantID int64,
) (map[int64]struct{}, error) {
	parentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		parentIDs = append(parentIDs, item.ID)
	}
	hasChildren := make(map[int64]struct{}, len(parentIDs))
	if len(parentIDs) == 0 {
		return hasChildren, nil
	}

	query := c.Query(ctx).BaseDept
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ParentID.In(parentIDs...)))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	children, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range children {
		hasChildren[item.ParentID] = struct{}{}
	}
	return hasChildren, nil
}
