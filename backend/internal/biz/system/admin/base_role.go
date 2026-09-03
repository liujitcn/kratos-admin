package biz

import (
	"context"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseRoleCase 角色业务实例
type BaseRoleCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *mapper.CopierMapper[adminv1.BaseRoleForm, models.BaseRole]
	mapper         *mapper.CopierMapper[adminv1.BaseRole, models.BaseRole]
}

// NewBaseRoleCase 创建角色业务实例
func NewBaseRoleCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	casbinRuleCase *CasbinRuleCase,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseRoleRepository: baseRoleRepo,
		baseTenantRepo:     baseTenantRepo,
		casbinRuleCase:     casbinRuleCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseRoleForm, models.BaseRole](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseRole, models.BaseRole](),
	}
}

// OptionBaseRole 查询角色选项
func (c *BaseRoleCase) OptionBaseRole(ctx context.Context, req *adminv1.OptionBaseRoleRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		var disabled bool
		if _const.IsDefaultBaseRole(item.Code) {
			disabled = true
		}

		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: disabled,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseRole 分页查询角色
func (c *BaseRoleCase) PageBaseRole(ctx context.Context, req *adminv1.PageBaseRoleRequest) (*adminv1.PageBaseRoleResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配角色。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入编码关键字时，按编码模糊匹配角色。
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseRole, 0, len(list))
	for _, item := range list {
		baseRole := c.mapper.ToDTO(item)
		baseRole.Menus = _string.ConvertJsonStringToInt64Array(item.Menus)
		baseRole.IsProtected = isBaseRoleProtected(authInfo, item)
		resList = append(resList, baseRole)
	}
	return &adminv1.PageBaseRoleResponse{BaseRoles: resList, Total: int32(total)}, nil
}

// GetBaseRole 获取角色
func (c *BaseRoleCase) GetBaseRole(ctx context.Context, id int64) (*adminv1.BaseRoleForm, error) {
	baseRole, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = c.validateBaseRoleManagementTarget(ctx, baseRole)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseRole)
	res.Menus = _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	return res, nil
}

// CreateBaseRole 创建角色
func (c *BaseRoleCase) CreateBaseRole(ctx context.Context, req *adminv1.BaseRoleForm) error {
	baseRole := c.formMapper.ToEntity(req)
	err := c.validateCreateBaseRole(ctx, baseRole)
	if err != nil {
		return err
	}
	err = c.validateAssignableMenus(ctx, baseRole.TenantID, req.GetMenus())
	if err != nil {
		return err
	}
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, baseRole)
		if err != nil {
			// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的角色编码重复", "base_role", "", "unique_base_role").WithCause(err)
			}
			return err
		}
		// 创建默认 tenant 模板后，将菜单重新同步到普通租户副本。
		if baseRole.Code == _const.BASE_ROLE_CODE_TENANT {
			return c.syncTenantRoleMenus(ctx, baseRole)
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// UpdateBaseRole 更新角色
func (c *BaseRoleCase) UpdateBaseRole(ctx context.Context, req *adminv1.BaseRoleForm) error {
	oldBaseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	err = c.validateBaseRoleManagementTarget(ctx, oldBaseRole)
	if err != nil {
		return err
	}
	// tenant 模板允许修改资料，但角色编码必须保持不变。
	if oldBaseRole.Code == _const.BASE_ROLE_CODE_TENANT && req.GetCode() != oldBaseRole.Code {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能修改内置角色编码", "base_role")
	}
	if oldBaseRole.Code != _const.BASE_ROLE_CODE_TENANT && _const.IsDefaultBaseRole(req.GetCode()) {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能使用内置角色编码", "base_role")
	}
	if _const.IsBaseRoleStatusProtected(oldBaseRole.Code) && req.GetStatus() != 0 && int32(req.GetStatus()) != oldBaseRole.Status {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能修改默认角色状态", "base_role")
	}
	err = c.validateAssignableMenus(ctx, oldBaseRole.TenantID, req.GetMenus())
	if err != nil {
		return err
	}

	baseRole := c.formMapper.ToEntity(req)
	baseRole.TenantID = oldBaseRole.TenantID
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseRole)
		if err != nil {
			// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的角色编码重复", "base_role", "", "unique_base_role").WithCause(err)
			}
			return err
		}
		// 默认 tenant 模板通过编辑表单保存菜单后，同步所有普通租户副本。
		if baseRole.Code == _const.BASE_ROLE_CODE_TENANT {
			return c.syncTenantRoleMenus(ctx, baseRole)
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// DeleteBaseRole 删除角色
func (c *BaseRoleCase) DeleteBaseRole(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseRoles, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return errorsx.Internal("删除角色失败").WithCause(err)
	}
	baseRoleMap := make(map[int64]*models.BaseRole, len(baseRoles))
	for _, baseRole := range baseRoles {
		baseRoleMap[baseRole.ID] = baseRole
	}
	for _, roleID := range ids {
		baseRole, exists := baseRoleMap[roleID]
		if !exists {
			return errorsx.ResourceNotFound("删除角色失败，角色不存在")
		}
		// admin、authuser、user 固定角色与状态保护使用同一保护集合，不允许删除。
		if _const.IsBaseRoleStatusProtected(baseRole.Code) {
			return errorsx.ProtectedResourceConflict("删除角色失败，不能删除默认角色", "base_role")
		}
		err = c.validateBaseRoleManagementTarget(ctx, baseRole)
		if err != nil {
			return err
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.casbinRuleCase.DeleteCasbinRuleByRoleIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(ctx, ids)
	})
}

// SetBaseRoleStatus 设置角色状态
func (c *BaseRoleCase) SetBaseRoleStatus(ctx context.Context, req *adminv1.SetBaseRoleStatusRequest) error {
	baseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if _const.IsBaseRoleStatusProtected(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("操作角色失败，不能修改默认角色状态", "base_role")
	}
	err = c.validateBaseRoleManagementTarget(ctx, baseRole)
	if err != nil {
		return err
	}
	return c.UpdateByID(ctx, &models.BaseRole{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// SetBaseRoleMenu 设置角色菜单
func (c *BaseRoleCase) SetBaseRoleMenu(ctx context.Context, req *adminv1.SetBaseRoleMenuRequest) error {
	oldBaseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	err = c.validateBaseRoleManagementTarget(ctx, oldBaseRole)
	if err != nil {
		return err
	}
	err = c.validateAssignableMenus(ctx, oldBaseRole.TenantID, req.GetMenus())
	if err != nil {
		return err
	}

	baseRole := &models.BaseRole{
		ID:       req.GetId(),
		TenantID: oldBaseRole.TenantID,
		Menus:    _string.ConvertInt64ArrayToString(req.GetMenus()),
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseRole)
		if err != nil {
			return err
		}
		baseRole.Code = oldBaseRole.Code
		// 默认租户的租户管理员角色是权限模板，变更后同步所有租户副本。
		if oldBaseRole.Code == _const.BASE_ROLE_CODE_TENANT {
			return c.syncTenantRoleMenus(ctx, baseRole)
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// validateBaseRoleManagementTarget 校验当前登录租户是否允许操作目标角色。
func (c *BaseRoleCase) validateBaseRoleManagementTarget(ctx context.Context, baseRole *models.BaseRole) error {
	// super 始终由系统维护，任何租户都不能通过角色管理操作。
	if baseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.ProtectedResourceConflict("操作角色失败，不能操作内置角色", "base_role")
	}
	if baseRole.Code != _const.BASE_ROLE_CODE_TENANT {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// tenant 仅允许默认租户操作自己的权限模板，普通租户和其他租户副本均禁止操作。
	if isBaseRoleProtected(authInfo, baseRole) {
		return errorsx.ProtectedResourceConflict("操作角色失败，不能操作租户内置角色", "base_role")
	}
	return nil
}

// validateCreateBaseRole 校验当前登录租户是否允许创建目标角色编码。
func (c *BaseRoleCase) validateCreateBaseRole(ctx context.Context, baseRole *models.BaseRole) error {
	if !_const.IsDefaultBaseRole(baseRole.Code) {
		return nil
	}
	// super 始终由系统初始化维护，不允许通过角色管理创建。
	if baseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.ProtectedResourceConflict("创建角色失败，不能使用内置角色编码", "base_role")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// tenant 模板只允许默认租户在自己的租户范围内创建。
	if authInfo.TenantCode != gorm.DefaultTenantCode || baseRole.TenantID != authInfo.TenantId {
		return errorsx.ProtectedResourceConflict("创建角色失败，不能使用内置角色编码", "base_role")
	}
	return nil
}

// validateAssignableMenus 校验待保存菜单不超过当前操作允许的权限范围。
func (c *BaseRoleCase) validateAssignableMenus(ctx context.Context, targetTenantID int64, menus []int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var allowedBaseRole *models.BaseRole
	// 默认租户为普通租户维护角色时，以目标租户内置管理员角色作为权限上限。
	if authInfo.TenantCode == gorm.DefaultTenantCode && targetTenantID > 0 && targetTenantID != authInfo.TenantId {
		query := c.Query(ctx).BaseRole
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
		allowedBaseRole, err = c.Find(ctx, opts...)
		if err != nil {
			return errorsx.Internal("查询租户最大权限失败").WithCause(err)
		}
	} else {
		// 超级管理员维护默认租户角色时拥有完整菜单分配权限。
		if authInfo.RoleCode == _const.BASE_ROLE_CODE_SUPER {
			return nil
		}
		allowedBaseRole, err = c.FindByID(ctx, authInfo.RoleId)
		if err != nil {
			return errorsx.Internal("查询当前角色权限失败").WithCause(err)
		}
	}
	// 权限上限角色已停用时，不允许继续分配其他角色权限。
	if allowedBaseRole.Status != _const.STATUS_STATUS_ENABLE {
		return errorsx.PermissionDenied("角色已被禁用")
	}

	allowedMenuIDs := _string.ConvertJsonStringToInt64Array(allowedBaseRole.Menus)
	allowedMenuIDSet := make(map[int64]struct{}, len(allowedMenuIDs))
	for _, menuID := range allowedMenuIDs {
		allowedMenuIDSet[menuID] = struct{}{}
	}
	for _, menuID := range menus {
		// 提交菜单不在权限上限范围内时，直接拒绝保存。
		if _, ok := allowedMenuIDSet[menuID]; !ok {
			return errorsx.PermissionDenied("设置角色菜单权限失败，不能分配超出权限上限的菜单权限")
		}
	}
	return nil
}

// syncTenantRoleMenus 同步默认租户管理员角色菜单到所有租户副本并重建权限。
func (c *BaseRoleCase) syncTenantRoleMenus(ctx context.Context, templateRole *models.BaseRole) error {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}

	for _, item := range list {
		if item.ID != templateRole.ID && item.Menus != templateRole.Menus {
			err = c.UpdateByID(ctx, &models.BaseRole{
				ID:       item.ID,
				TenantID: item.TenantID,
				Menus:    templateRole.Menus,
			})
			if err != nil {
				return err
			}
			item.Menus = templateRole.Menus
		}
		err = c.casbinRuleCase.rebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.casbinRuleCase.RebuildPolicyRule(ctx)
}

// isBaseRoleProtected 判断目标角色是否禁止当前账号通过角色管理操作。
func isBaseRoleProtected(authInfo *authData.UserTokenPayload, baseRole *models.BaseRole) bool {
	if baseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return true
	}
	return baseRole.Code == _const.BASE_ROLE_CODE_TENANT &&
		(authInfo == nil || authInfo.TenantCode != gorm.DefaultTenantCode || baseRole.TenantID != authInfo.TenantId)
}
