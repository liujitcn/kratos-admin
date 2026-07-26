package biz

import (
	"context"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/event"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"
	"github.com/liujitcn/kratos-admin/backend/service/base/utils"
)

// BaseUserCase 用户业务实例
type BaseUserCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseUserRepository
	baseDeptRepo *data.BaseDeptRepository
	basePostRepo *data.BasePostRepository
	baseRoleCase *BaseRoleCase
	baseDeptCase *BaseDeptCase
	baseMenuCase *BaseMenuCase
	userEvents   *event.UserEvents
	formMapper   *mapper.CopierMapper[systemadminv1.BaseUserForm, models.BaseUser]
	mapper       *mapper.CopierMapper[systemadminv1.BaseUser, models.BaseUser]
}

// NewBaseUserCase 创建用户业务实例
func NewBaseUserCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseUserRepo *data.BaseUserRepository,
	baseDeptRepo *data.BaseDeptRepository,
	basePostRepo *data.BasePostRepository,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	baseMenuCase *BaseMenuCase,
	userEvents *event.UserEvents,
) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseUserRepository: baseUserRepo,
		baseDeptRepo:       baseDeptRepo,
		basePostRepo:       basePostRepo,
		baseRoleCase:       baseRoleCase,
		baseDeptCase:       baseDeptCase,
		baseMenuCase:       baseMenuCase,
		userEvents:         userEvents,
		formMapper:         mapper.NewCopierMapper[systemadminv1.BaseUserForm, models.BaseUser](),
		mapper:             mapper.NewCopierMapper[systemadminv1.BaseUser, models.BaseUser](),
	}
}

// OptionBaseUser 查询用户选项
func (c *BaseUserCase) OptionBaseUser(ctx context.Context, req *systemadminv1.OptionBaseUserRequest) (*commonv1.SelectOptionResponse, error) {
	keyword := req.GetKeyword()
	// 未传关键字时，直接返回空选项集。
	if keyword == "" {
		return &commonv1.SelectOptionResponse{List: []*commonv1.SelectOptionResponse_Option{}}, nil
	}

	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.NickName.Like("%"+keyword+"%")))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	opts = append(opts, repository.Limit(100))

	var list []*models.BaseUser
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label: item.NickName,
			Value: item.ID,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseUser 分页查询用户
func (c *BaseUserCase) PageBaseUser(ctx context.Context, req *systemadminv1.PageBaseUserRequest) (*systemadminv1.PageBaseUserResponse, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	var err error
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	// 指定部门时，按部门及其子部门范围筛选用户。
	if req.DeptId != nil && req.GetDeptId() > 0 {
		var dept *models.BaseDept
		dept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
		if err != nil {
			return nil, err
		}
		if req.GetTenantId() > 0 && dept.TenantID != req.GetTenantId() {
			return &systemadminv1.PageBaseUserResponse{BaseUsers: []*systemadminv1.BaseUser{}, Total: 0}, nil
		}

		deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
		deptOpts := make([]repository.QueryOption, 0, 2)
		deptOpts = append(deptOpts, repository.Where(deptQuery.Path.Like(dept.Path+"%")))
		deptOpts = append(deptOpts, repository.Where(deptQuery.TenantID.Eq(dept.TenantID)))
		var deptList []*models.BaseDept
		deptList, err = c.baseDeptRepo.List(ctx, deptOpts...)
		if err != nil {
			return nil, err
		}

		deptIDs := make([]int64, 0, len(deptList))
		for _, item := range deptList {
			deptIDs = append(deptIDs, item.ID)
		}
		// 命中部门集合时，按部门编号范围过滤用户。
		if len(deptIDs) > 0 {
			opts = append(opts, repository.Where(query.DeptID.In(deptIDs...)))
		}
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.Gender != nil {
		opts = append(opts, repository.Where(query.Gender.Eq(int32(req.GetGender()))))
	}
	// 传入用户名关键字时，按用户名模糊匹配。
	if req.GetUserName() != "" {
		opts = append(opts, repository.Where(query.UserName.Like("%"+req.GetUserName()+"%")))
	}
	// 传入用户编号关键字时，按用户编号模糊匹配。
	if req.GetUserCode() != "" {
		opts = append(opts, repository.Where(query.UserCode.Like("%"+req.GetUserCode()+"%")))
	}
	// 传入昵称关键字时，按昵称模糊匹配。
	if req.GetNickName() != "" {
		opts = append(opts, repository.Where(query.NickName.Like("%"+req.GetNickName()+"%")))
	}
	// 传入手机号关键字时，按手机号模糊匹配。
	if req.GetPhone() != "" {
		opts = append(opts, repository.Where(query.Phone.Like("%"+req.GetPhone()+"%")))
	}

	var list []*models.BaseUser
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	roleIDSet := make(map[int64]struct{}, len(list))
	roleIDs := make([]int64, 0, len(list))
	for _, item := range list {
		if _, exists := roleIDSet[item.RoleID]; exists {
			continue
		}
		roleIDSet[item.RoleID] = struct{}{}
		roleIDs = append(roleIDs, item.RoleID)
	}
	protectedRoleIDs := make(map[int64]struct{}, len(roleIDs))
	// 包含软删除角色，确保 tenant 模板删除后其历史账号仍保持用户管理保护。
	if len(roleIDs) > 0 {
		var roleQueryCtx context.Context
		roleQueryCtx, err = c.roleProtectionQueryContext(ctx)
		if err != nil {
			return nil, err
		}
		roleQuery := c.baseRoleCase.Query(roleQueryCtx).BaseRole
		roleOpts := make([]repository.QueryOption, 0, 2)
		roleOpts = append(roleOpts, repository.Unscoped())
		roleOpts = append(roleOpts, repository.Where(roleQuery.ID.In(roleIDs...)))
		var baseRoles []*models.BaseRole
		baseRoles, err = c.baseRoleCase.List(roleQueryCtx, roleOpts...)
		if err != nil {
			return nil, errorsx.Internal("查询用户角色失败").WithCause(err)
		}
		for _, baseRole := range baseRoles {
			if _const.IsDefaultBaseRole(baseRole.Code) {
				protectedRoleIDs[baseRole.ID] = struct{}{}
			}
		}
	}
	resList := make([]*systemadminv1.BaseUser, 0, len(list))
	for _, item := range list {
		baseUser := c.mapper.ToDTO(item)
		_, baseUser.IsProtected = protectedRoleIDs[item.RoleID]
		resList = append(resList, baseUser)
	}
	return &systemadminv1.PageBaseUserResponse{BaseUsers: resList, Total: int32(total)}, nil
}

// GetBaseUser 获取用户
func (c *BaseUserCase) GetBaseUser(ctx context.Context, id int64) (*systemadminv1.BaseUserForm, error) {
	baseUser, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = c.validateUserManagementTarget(ctx, baseUser)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseUser), nil
}

// CreateBaseUser 创建用户
func (c *BaseUserCase) CreateBaseUser(ctx context.Context, req *systemadminv1.BaseUserForm) error {
	baseRole, err := c.baseRoleCase.FindByID(ctx, req.GetRoleId())
	if err != nil {
		return errorsx.ResourceNotFound("用户角色不存在").WithCause(err)
	}
	if _const.IsDefaultBaseRole(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("创建用户失败，不能选择内置角色", "base_user")
	}
	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
	if err != nil {
		return errorsx.ResourceNotFound("用户部门不存在").WithCause(err)
	}
	if baseRole.TenantID != baseDept.TenantID {
		return errorsx.InvalidArgument("用户角色与部门所属租户不一致")
	}
	if req.GetTenantId() > 0 && req.GetTenantId() != baseDept.TenantID {
		return errorsx.InvalidArgument("用户所属租户与部门不一致")
	}
	_, err = c.validateBasePost(ctx, req.GetPostId(), baseDept.TenantID, 0)
	if err != nil {
		return err
	}

	var passwordStr string
	// 未显式传入密码时，回退到系统默认密码规则。
	if req.GetPwd() == nil {
		passwordStr = utils.GetDefaultPassword(req.GetUserName(), req.GetPhone())
	} else {
		passwordStr, err = utils.DecryptPassword(req.GetPwd(), commonv1.PasswordCryptoScene_CREATE_BASE_USER)
		if err != nil {
			return err
		}
	}

	var password string
	password, err = crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = password
	baseUser.TenantID = baseDept.TenantID
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, baseUser)
		if err != nil {
			// 命中用户账号或用户编号唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的用户账号或用户编号重复", "base_user", "", "unique_base_user").WithCause(err)
			}
			return err
		}
		return c.updateBaseUserPostID(ctx, baseUser.ID, req.GetPostId())
	})
	if err != nil {
		return err
	}
	// 用户写库成功后，通知已装配模块处理用户资料变更。
	c.userEvents.PublishUserChanged(baseUser.ID)
	return nil
}

// UpdateBaseUser 更新用户
func (c *BaseUserCase) UpdateBaseUser(ctx context.Context, req *systemadminv1.BaseUserForm) error {
	oldBaseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("更新用户失败，用户信息不存在").WithCause(err)
	}
	err = c.validateUserManagementTarget(ctx, oldBaseUser)
	if err != nil {
		return err
	}
	// 用户账号作为稳定登录标识，创建后不允许通过编辑接口修改。
	if req.GetUserName() != oldBaseUser.UserName {
		return errorsx.ProtectedResourceConflict("更新用户失败，用户账号不能修改", "base_user")
	}
	// 用户编号作为稳定用户标识，创建后不允许通过编辑接口修改。
	if req.GetUserCode() != oldBaseUser.UserCode {
		return errorsx.ProtectedResourceConflict("更新用户失败，用户编号不能修改", "base_user")
	}
	var newBaseRole *models.BaseRole
	newBaseRole, err = c.baseRoleCase.FindByID(ctx, req.GetRoleId())
	if err != nil {
		return errorsx.ResourceNotFound("用户角色不存在").WithCause(err)
	}
	if _const.IsDefaultBaseRole(newBaseRole.Code) {
		return errorsx.ProtectedResourceConflict("更新用户失败，不能选择内置角色", "base_user")
	}
	if newBaseRole.TenantID != oldBaseUser.TenantID {
		return errorsx.InvalidArgument("用户角色与所属租户不一致")
	}
	var newBaseDept *models.BaseDept
	newBaseDept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
	if err != nil {
		return errorsx.ResourceNotFound("用户部门不存在").WithCause(err)
	}
	if newBaseDept.TenantID != oldBaseUser.TenantID {
		return errorsx.InvalidArgument("用户部门与所属租户不一致")
	}
	_, err = c.validateBasePost(ctx, req.GetPostId(), oldBaseUser.TenantID, oldBaseUser.PostID)
	if err != nil {
		return err
	}

	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = oldBaseUser.Password
	baseUser.TenantID = oldBaseUser.TenantID
	baseUser.UserName = oldBaseUser.UserName
	baseUser.UserCode = oldBaseUser.UserCode
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseUser)
		if err != nil {
			// 命中用户账号或用户编号唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的用户账号或用户编号重复", "base_user", "", "unique_base_user").WithCause(err)
			}
			return err
		}
		return c.updateBaseUserPostID(ctx, baseUser.ID, req.GetPostId())
	})
	if err != nil {
		return err
	}
	// 用户更新成功后，通知已装配模块处理用户资料变更。
	c.userEvents.PublishUserChanged(baseUser.ID)
	return nil
}

// DeleteBaseUser 删除用户
func (c *BaseUserCase) DeleteBaseUser(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseUserList, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	baseUserMap := make(map[int64]*models.BaseUser, len(baseUserList))
	for _, baseUser := range baseUserList {
		baseUserMap[baseUser.ID] = baseUser
	}
	visibleIDs := make([]int64, 0, len(ids))
	for _, userID := range ids {
		baseUser, exists := baseUserMap[userID]
		if !exists {
			return errorsx.ResourceNotFound("删除用户失败，用户不存在")
		}
		err = c.validateUserManagementTarget(ctx, baseUser)
		if err != nil {
			return err
		}
		visibleIDs = append(visibleIDs, baseUser.ID)
	}
	err = c.DeleteByIDs(ctx, visibleIDs)
	if err != nil {
		return err
	}
	// 用户删除成功后，通知已装配模块清理关联用户数据。
	c.userEvents.PublishUsersDeleted(visibleIDs)
	return nil
}

// SetBaseUserStatus 设置用户状态
func (c *BaseUserCase) SetBaseUserStatus(ctx context.Context, req *systemadminv1.SetBaseUserStatusRequest) error {
	baseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("设置状态失败，用户信息不存在").WithCause(err)
	}
	err = c.validateUserManagementTarget(ctx, baseUser)
	if err != nil {
		return err
	}
	baseUser.Status = req.GetStatus()
	err = c.UpdateByID(ctx, baseUser)
	if err != nil {
		return err
	}
	// 用户状态变更成功后，通知已装配模块处理用户资料变更。
	c.userEvents.PublishUserChanged(baseUser.ID)
	return nil
}

// ResetBaseUserPassword 重置用户密码
func (c *BaseUserCase) ResetBaseUserPassword(ctx context.Context, req *systemadminv1.ResetBaseUserPasswordRequest) error {
	baseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("重置密码失败，用户信息不存在").WithCause(err)
	}
	err = c.validateUserManagementTarget(ctx, baseUser)
	if err != nil {
		return err
	}

	var passwordStr string
	// 未显式传入密码时，回退到系统默认密码规则。
	if req.GetPwd() == nil {
		passwordStr = utils.GetDefaultPassword(baseUser.UserName, baseUser.Phone)
	} else {
		passwordStr, err = utils.DecryptPassword(req.GetPwd(), commonv1.PasswordCryptoScene_RESET_BASE_USER_PASSWORD)
		if err != nil {
			return err
		}
	}

	var password string
	password, err = crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	return c.UpdateByID(ctx, &models.BaseUser{
		ID:       req.GetId(),
		Password: password,
	})
}

// validateUserManagementTarget 校验目标用户是否允许通过用户管理接口操作。
func (c *BaseUserCase) validateUserManagementTarget(ctx context.Context, baseUser *models.BaseUser) error {
	queryCtx, err := c.roleProtectionQueryContext(ctx)
	if err != nil {
		return err
	}
	query := c.baseRoleCase.Query(queryCtx).BaseRole
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.ID.Eq(baseUser.RoleID)))
	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.Find(queryCtx, opts...)
	if err != nil {
		return errorsx.Internal("校验用户角色失败").WithCause(err)
	}
	// super 和 tenant 管理员只能通过个人中心维护自身资料与密码。
	if _const.IsDefaultBaseRole(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("操作用户失败，内置管理员账号只能通过个人中心修改", "base_user")
	}
	return nil
}

// roleProtectionQueryContext 构造仅用于内置角色保护判定的全部数据范围查询上下文。
func (c *BaseUserCase) roleProtectionQueryContext(ctx context.Context) (context.Context, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	roleAuthInfo := *authInfo
	roleAuthInfo.DataScope = databaseGorm.DataScopeAll
	return authnEngine.ContextWithAuthClaims(ctx, roleAuthInfo.MakeAuthClaims()), nil
}

// validateBasePost 校验用户岗位属于用户租户，且新选择的岗位处于启用状态。
func (c *BaseUserCase) validateBasePost(ctx context.Context, postID int64, tenantID int64, oldPostID int64) (*models.BasePost, error) {
	if postID == 0 {
		return nil, nil
	}
	basePost, err := c.basePostRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户岗位不存在").WithCause(err)
	}
	if basePost.TenantID != tenantID {
		return nil, errorsx.InvalidArgument("用户岗位与所属租户不一致")
	}
	if basePost.Status != _const.STATUS_ENABLE && basePost.ID != oldPostID {
		return nil, errorsx.PermissionDenied("岗位已被禁用，不能选择")
	}
	return basePost, nil
}

// updateBaseUserPostID 保存用户岗位，未选择岗位时将字段清空为 NULL。
func (c *BaseUserCase) updateBaseUserPostID(ctx context.Context, userID int64, postID int64) error {
	query := c.Query(ctx).BaseUser
	var err error
	if postID > 0 {
		_, err = query.WithContext(ctx).Where(query.ID.Eq(userID)).UpdateSimple(query.PostID.Value(postID))
		return err
	}
	_, err = query.WithContext(ctx).Where(query.ID.Eq(userID)).UpdateSimple(query.PostID.Null())
	return err
}
