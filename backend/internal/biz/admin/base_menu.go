package biz

import (
	"context"

	systemcommonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/common/v1"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm/clause"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

const (
	baseMenuChildSequenceMax int64 = 99
	baseMenuMaxLevel               = 4
)

// BaseMenuCase 菜单业务实例
type BaseMenuCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMenuRepository
	baseRoleRepo   *data.BaseRoleRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *_mapper.CopierMapper[systemadminv1.BaseMenuForm, models.BaseMenu]
	mapper         *_mapper.CopierMapper[systemadminv1.BaseMenu, models.BaseMenu]
	routerMapper   *_mapper.CopierMapper[systemadminv1.RouteItem, models.BaseMenu]
}

// NewBaseMenuCase 创建菜单业务实例
func NewBaseMenuCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseMenuRepo *data.BaseMenuRepository,
	baseRoleRepo *data.BaseRoleRepository,
	casbinRuleCase *CasbinRuleCase,
) *BaseMenuCase {
	formMapper := _mapper.NewCopierMapper[systemadminv1.BaseMenuForm, models.BaseMenu]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[*systemadminv1.BaseMenuMeta]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[systemadminv1.BaseMenu, models.BaseMenu]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[*systemadminv1.BaseMenuMeta]().NewConverterPair())
	routerMapper := _mapper.NewCopierMapper[systemadminv1.RouteItem, models.BaseMenu]()
	routerMapper.AppendConverters(_mapper.NewJSONTypeConverter[*systemadminv1.RouteMeta]().NewConverterPair())
	return &BaseMenuCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseMenuRepository: baseMenuRepo,
		baseRoleRepo:       baseRoleRepo,
		casbinRuleCase:     casbinRuleCase,
		formMapper:         formMapper,
		mapper:             mapper,
		routerMapper:       routerMapper,
	}
}

// OptionBaseMenu 查询菜单选项
func (c *BaseMenuCase) OptionBaseMenu(ctx context.Context, req *systemadminv1.OptionBaseMenuRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	allowedMenuIDs, isSuperRole, err := c.listAssignableMenuIDs(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	// 非超级管理员没有菜单权限时，菜单选项接口直接返回空树。
	if !isSuperRole && len(allowedMenuIDs) == 0 {
		return &commonv1.TreeOptionResponse{}, nil
	}
	// 非超级管理员只能看到当前角色已经拥有的菜单上限。
	if !isSuperRole {
		opts = append(opts, repository.Where(query.ID.In(allowedMenuIDs...)))
	}
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	var list []*models.BaseMenu
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseMenuParentIDsWithChildren(ctx, list, allowedMenuIDs, isSuperRole)
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &commonv1.TreeOptionResponse{List: c.buildBaseMenuOption(list, parentID, req.GetLazy(), hasChildren)}, nil
}

// TreeBaseMenu 查询菜单树
func (c *BaseMenuCase) TreeBaseMenu(ctx context.Context, req *systemadminv1.TreeBaseMenuRequest) (*systemadminv1.TreeBaseMenuResponse, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	allowedMenuIDs, isSuperRole, err := c.listAssignableMenuIDs(ctx, 0)
	if err != nil {
		return nil, err
	}
	// 非超级管理员没有菜单权限时，菜单管理页直接返回空树。
	if !isSuperRole && len(allowedMenuIDs) == 0 {
		return &systemadminv1.TreeBaseMenuResponse{}, nil
	}
	// 非超级管理员只能看到当前角色已经拥有的菜单上限。
	if !isSuperRole {
		opts = append(opts, repository.Where(query.ID.In(allowedMenuIDs...)))
	}
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	var list []*models.BaseMenu
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listBaseMenuParentIDsWithChildren(ctx, list, allowedMenuIDs, isSuperRole)
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &systemadminv1.TreeBaseMenuResponse{BaseMenus: c.buildBaseMenuTree(list, parentID, req.GetLazy(), hasChildren)}, nil
}

// GetBaseMenu 获取菜单
func (c *BaseMenuCase) GetBaseMenu(ctx context.Context, id int64) (*systemadminv1.BaseMenuForm, error) {
	baseMenu, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseMenu), nil
}

// CreateBaseMenu 创建菜单
func (c *BaseMenuCase) CreateBaseMenu(ctx context.Context, req *systemadminv1.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.createBaseMenu(ctx, baseMenu)
	})
}

// UpdateBaseMenu 更新菜单
func (c *BaseMenuCase) UpdateBaseMenu(ctx context.Context, req *systemadminv1.BaseMenuForm) error {
	baseMenu := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		currentMenu, err := c.FindByID(ctx, req.GetId())
		if err != nil {
			return err
		}
		// 一级菜单属于初始化固定资源，只允许修改展示配置，不能改变编号、父级和类型。
		if currentMenu.ParentID == 0 {
			if req.GetParentId() != 0 || req.GetType() != systemcommonv1.BaseMenuType_FOLDER {
				return errorsx.ProtectedResourceConflict("一级菜单的父级和类型不允许修改", "base_menu")
			}
		} else {
			if req.ParentId != nil && req.GetParentId() != currentMenu.ParentID {
				return errorsx.ProtectedResourceConflict("菜单创建后不允许更换父级", "base_menu")
			}
			var parentMenu *models.BaseMenu
			parentMenu, err = c.FindByID(ctx, currentMenu.ParentID)
			if err != nil {
				return errorsx.Internal("查询父级菜单失败").WithCause(err)
			}
			if err = validateBaseMenuChild(parentMenu, int32(req.GetType())); err != nil {
				return err
			}
		}
		baseMenu.ID = currentMenu.ID
		baseMenu.ParentID = currentMenu.ParentID
		if err = c.validateBaseMenuIdentity(ctx, baseMenu, currentMenu.ID); err != nil {
			return err
		}
		if err = c.UpdateByID(ctx, baseMenu); err != nil {
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, baseMenu.ID)
	})
}

// DeleteBaseMenu 删除菜单
func (c *BaseMenuCase) DeleteBaseMenu(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseMenu
	var err error
	for _, menuID := range ids {
		var menu *models.BaseMenu
		menu, err = c.FindByID(ctx, menuID)
		if err != nil {
			return err
		}
		if menu.ParentID == 0 {
			return errorsx.ProtectedResourceConflict("一级菜单不允许删除", "base_menu")
		}
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ParentID.Eq(menuID)))
		var count int64
		count, err = c.Count(ctx, opts...)
		if err != nil {
			return err
		}
		// 仍然存在子菜单时，禁止删除当前节点。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除菜单失败，下面有菜单", "base_menu", "base_menu")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		if err = c.DeleteByIDs(ctx, ids); err != nil {
			return err
		}
		return c.casbinRuleCase.DeleteCasbinRuleByMenuIDs(ctx, ids)
	})
}

// allocateBaseMenuID 锁定父节点后，从当前层级的01到99中分配首个可用菜单编号。
func (c *BaseMenuCase) allocateBaseMenuID(ctx context.Context, parentID int64, menuType int32) (int64, error) {
	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(parentID)))
	opts = append(opts, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
	parentMenu, err := c.Find(ctx, opts...)
	if err != nil {
		return 0, errorsx.InvalidArgument("父级菜单不存在").WithCause(err)
	}
	if err = validateBaseMenuChild(parentMenu, menuType); err != nil {
		return 0, err
	}

	opts = make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.ParentID.Eq(parentID)))
	children, err := c.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	usedIDs := make(map[int64]struct{}, len(children))
	for _, child := range children {
		usedIDs[child.ID] = struct{}{}
	}
	for sequence := int64(1); sequence <= baseMenuChildSequenceMax; sequence++ {
		menuID := parentID*100 + sequence
		if _, exists := usedIDs[menuID]; !exists {
			return menuID, nil
		}
	}
	return 0, errorsx.StateConflict("当前父级菜单的子编号01-99已用完", "base_menu", "range_exhausted", "available_id")
}

// SetBaseMenuStatus 设置菜单状态
func (c *BaseMenuCase) SetBaseMenuStatus(ctx context.Context, req *systemadminv1.SetBaseMenuStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseMenu{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// createBaseMenu 校验父级并按层级编号规则创建菜单。
func (c *BaseMenuCase) createBaseMenu(ctx context.Context, baseMenu *models.BaseMenu) error {
	menuID, err := c.allocateBaseMenuID(ctx, baseMenu.ParentID, baseMenu.Type)
	if err != nil {
		return err
	}
	baseMenu.ID = menuID
	if err = c.validateBaseMenuIdentity(ctx, baseMenu, 0); err != nil {
		return err
	}
	return c.Create(ctx, baseMenu)
}

// validateBaseMenuIdentity 校验菜单路径和路由名称的唯一性。
func (c *BaseMenuCase) validateBaseMenuIdentity(ctx context.Context, menu *models.BaseMenu, currentID int64) error {
	query := c.Query(ctx).BaseMenu
	pathOpts := make([]repository.QueryOption, 0, 3)
	pathOpts = append(pathOpts, repository.Where(query.Type.Neq(_const.BASE_MENU_TYPE_FOLDER)))
	pathOpts = append(pathOpts, repository.Where(query.Path.Eq(menu.Path)))
	if currentID > 0 {
		pathOpts = append(pathOpts, repository.Where(query.ID.Neq(currentID)))
	}
	var count int64
	var err error
	count, err = c.Count(ctx, pathOpts...)
	if err != nil {
		return errorsx.Internal("校验菜单路径失败").WithCause(err)
	}
	if count > 0 {
		return errorsx.UniqueConflict("菜单路径已存在", "base_menu", "path", "")
	}

	if menu.Type != _const.BASE_MENU_TYPE_MENU {
		return nil
	}
	nameOpts := make([]repository.QueryOption, 0, 3)
	nameOpts = append(nameOpts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
	nameOpts = append(nameOpts, repository.Where(query.Name.Eq(menu.Name)))
	if currentID > 0 {
		nameOpts = append(nameOpts, repository.Where(query.ID.Neq(currentID)))
	}
	count, err = c.Count(ctx, nameOpts...)
	if err != nil {
		return errorsx.Internal("校验菜单路由名称失败").WithCause(err)
	}
	if count > 0 {
		return errorsx.UniqueConflict("菜单路由名称已存在", "base_menu", "name", "")
	}
	return nil
}

// listAssignableMenuIDs 根据真实角色归属查询当前操作可分配的菜单 ID 列表。
func (c *BaseMenuCase) listAssignableMenuIDs(ctx context.Context, targetRoleID int64) ([]int64, bool, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, false, err
	}
	var targetRole *models.BaseRole
	if targetRoleID > 0 {
		targetRole, err = c.baseRoleRepo.FindByID(ctx, targetRoleID)
		if err != nil {
			return nil, false, errorsx.Internal("查询目标角色失败").WithCause(err)
		}
	}
	// 默认租户为普通租户维护角色时，以角色真实所属租户的内置管理员角色作为权限上限。
	if targetRole != nil && authInfo.TenantCode == databaseGorm.DefaultTenantCode && targetRole.TenantID != authInfo.TenantId {
		query := c.baseRoleRepo.Query(ctx).BaseRole
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TenantID.Eq(targetRole.TenantID)))
		opts = append(opts, repository.Where(query.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
		var tenantBaseRole *models.BaseRole
		tenantBaseRole, err = c.baseRoleRepo.Find(ctx, opts...)
		if err != nil {
			return nil, false, errorsx.Internal("查询租户最大权限失败").WithCause(err)
		}
		// 租户内置管理员角色停用时，不能再作为权限上限来源。
		if tenantBaseRole.Status != _const.STATUS_ENABLE {
			return nil, false, errorsx.PermissionDenied("租户管理员角色已被禁用")
		}
		return _string.ConvertJsonStringToInt64Array(tenantBaseRole.Menus), false, nil
	}
	// 超级管理员拥有完整菜单管理权限，不需要按角色菜单裁剪。
	if authInfo.RoleCode == _const.BASE_ROLE_CODE_SUPER {
		return nil, true, nil
	}

	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleRepo.FindByID(ctx, authInfo.RoleId)
	if err != nil {
		return nil, false, errorsx.Internal("查询当前角色权限失败").WithCause(err)
	}
	// 当前角色已停用时，不允许继续作为菜单权限上限来源。
	if baseRole.Status != _const.STATUS_ENABLE {
		return nil, false, errorsx.PermissionDenied("角色已被禁用")
	}
	return _string.ConvertJsonStringToInt64Array(baseRole.Menus), false, nil
}

// listSubtreeIDs 从指定根菜单开始按层查询完整子树 ID。
func (c *BaseMenuCase) listSubtreeIDs(ctx context.Context, rootID int64) ([]int64, error) {
	ids := []int64{rootID}
	parentIDs := []int64{rootID}
	visited := map[int64]struct{}{rootID: {}}
	query := c.Query(ctx).BaseMenu
	var err error
	for len(parentIDs) > 0 {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ParentID.In(parentIDs...)))
		var children []*models.BaseMenu
		children, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}

		parentIDs = make([]int64, 0, len(children))
		for _, child := range children {
			// 已访问节点不再重复入队，避免异常菜单环导致查询无法结束。
			if _, exists := visited[child.ID]; exists {
				continue
			}
			visited[child.ID] = struct{}{}
			ids = append(ids, child.ID)
			parentIDs = append(parentIDs, child.ID)
		}
	}
	return ids, nil
}

// buildRouteTree 构建菜单路由树
func (c *BaseMenuCase) buildRouteTree(menuList []*models.BaseMenu, parentID int64) []*systemadminv1.RouteItem {
	list := make([]*systemadminv1.RouteItem, 0)
	for _, menu := range menuList {
		// 非当前父节点的菜单不参与当前层级路由构建。
		if menu.ParentID != parentID {
			continue
		}

		route := c.routerMapper.ToDTO(menu)

		route.Children = c.buildRouteTree(menuList, menu.ID)
		list = append(list, route)
	}
	return list
}

// buildBaseMenuTree 构建菜单树
func (c *BaseMenuCase) buildBaseMenuTree(
	menuList []*models.BaseMenu,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*systemadminv1.BaseMenu {
	res := make([]*systemadminv1.BaseMenu, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级树构建。
		if item.ParentID != parentID {
			continue
		}
		menu := c.mapper.ToDTO(item)
		_, menu.HasChildren = hasChildren[item.ID]
		if !lazy {
			menu.Children = c.buildBaseMenuTree(menuList, item.ID, false, hasChildren)
		}
		res = append(res, menu)
	}
	return res
}

// buildBaseMenuOption 构建菜单选项树
func (c *BaseMenuCase) buildBaseMenuOption(
	menuList []*models.BaseMenu,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range menuList {
		// 非当前父节点的菜单不参与当前层级选项构建。
		if item.ParentID != parentID {
			continue
		}
		// 按钮和外链不能承载子菜单，不提供为父级菜单选项。
		if item.Type != int32(systemcommonv1.BaseMenuType_FOLDER) && item.Type != int32(systemcommonv1.BaseMenuType_MENU) {
			continue
		}

		label := item.Name
		route := c.routerMapper.ToDTO(item)
		// 路由元信息存在标题时，优先使用前端路由标题作为展示名称。
		if route != nil && route.GetMeta() != nil && route.GetMeta().GetTitle() != "" {
			label = route.GetMeta().GetTitle()
		}

		menu := &commonv1.TreeOptionResponse_Option{
			Label: label,
			Value: item.ID,
		}
		_, menu.HasChildren = hasChildren[item.ID]
		if !lazy {
			menu.Children = c.buildBaseMenuOption(menuList, item.ID, false, hasChildren)
		}
		res = append(res, menu)
	}
	return res
}

// listBaseMenuParentIDsWithChildren 查询存在可选子节点的菜单父级编号。
func (c *BaseMenuCase) listBaseMenuParentIDsWithChildren(
	ctx context.Context,
	list []*models.BaseMenu,
	allowedMenuIDs []int64,
	isSuperRole bool,
) (map[int64]struct{}, error) {
	parentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		parentIDs = append(parentIDs, item.ID)
	}
	hasChildren := make(map[int64]struct{}, len(parentIDs))
	if len(parentIDs) == 0 {
		return hasChildren, nil
	}

	query := c.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ParentID.In(parentIDs...)))
	opts = append(opts, repository.Where(query.Type.In(
		int32(systemcommonv1.BaseMenuType_FOLDER),
		int32(systemcommonv1.BaseMenuType_MENU),
	)))
	if !isSuperRole {
		opts = append(opts, repository.Where(query.ID.In(allowedMenuIDs...)))
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

// validateBaseMenuChild 校验父节点能否承载指定类型的下级菜单。
func validateBaseMenuChild(parentMenu *models.BaseMenu, menuType int32) error {
	if parentMenu.ID == _const.BASE_MENU_APP_ROOT_ID {
		if menuType == _const.BASE_MENU_TYPE_MENU {
			return nil
		}
		return errorsx.InvalidArgument("移动端根目录下只能创建标签页菜单")
	}
	if isAppMenuID(parentMenu.ID) {
		if baseMenuIDLevel(parentMenu.ID) >= baseMenuMaxLevel {
			return errorsx.InvalidArgument("移动端页面菜单已达到最大层级")
		}
		if menuType == _const.BASE_MENU_TYPE_MENU {
			return nil
		}
		return errorsx.InvalidArgument("移动端页面下只能创建页面菜单")
	}
	if parentMenu.Type == _const.BASE_MENU_TYPE_BUTTON || parentMenu.Type == _const.BASE_MENU_TYPE_EXT_LINK {
		return errorsx.InvalidArgument("按钮或外链不能作为父级菜单")
	}
	parentLevel := baseMenuIDLevel(parentMenu.ID)
	if parentLevel == 0 || parentLevel >= baseMenuMaxLevel {
		return errorsx.InvalidArgument("父级菜单ID不符合菜单编号层级规则")
	}
	if menuType != _const.BASE_MENU_TYPE_FOLDER && menuType != _const.BASE_MENU_TYPE_MENU && menuType != _const.BASE_MENU_TYPE_BUTTON && menuType != _const.BASE_MENU_TYPE_EXT_LINK {
		return errorsx.InvalidArgument("菜单类型无效")
	}
	if parentMenu.Type == _const.BASE_MENU_TYPE_FOLDER && parentLevel == 1 {
		if menuType == _const.BASE_MENU_TYPE_FOLDER || menuType == _const.BASE_MENU_TYPE_MENU || menuType == _const.BASE_MENU_TYPE_EXT_LINK {
			return nil
		}
		return errorsx.InvalidArgument("一级目录下只能创建目录、菜单或外链")
	}
	if parentMenu.Type == _const.BASE_MENU_TYPE_FOLDER && parentLevel == 2 {
		if menuType == _const.BASE_MENU_TYPE_MENU || menuType == _const.BASE_MENU_TYPE_EXT_LINK {
			return nil
		}
		return errorsx.InvalidArgument("二级目录下只能创建三级菜单或外链")
	}
	if parentMenu.Type == _const.BASE_MENU_TYPE_MENU && (parentLevel == 2 || parentLevel == 3) {
		if menuType == _const.BASE_MENU_TYPE_BUTTON {
			return nil
		}
		return errorsx.InvalidArgument("页面菜单下只能创建按钮")
	}
	return errorsx.InvalidArgument("父级菜单类型与层级不匹配")
}

// baseMenuIDLevel 根据三、五、七、九位编号识别菜单层级。
func baseMenuIDLevel(menuID int64) int {
	switch {
	case menuID >= 100 && menuID <= 999:
		return 1
	case menuID >= 10000 && menuID <= 99999:
		return 2
	case menuID >= 1000000 && menuID <= 9999999:
		return 3
	case menuID >= 100000000 && menuID <= 999999999:
		return 4
	default:
		return 0
	}
}

// isAppMenuID 判断菜单编号是否属于固定移动端菜单树。
func isAppMenuID(menuID int64) bool {
	for menuID > _const.BASE_MENU_APP_ROOT_ID {
		menuID /= 100
	}
	return menuID == _const.BASE_MENU_APP_ROOT_ID
}
