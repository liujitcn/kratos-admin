package biz

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	baseBiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/pkg/biz"
	coreconst "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

// UPDATE_PHONE_CODE_CACHE_PREFIX 表示修改手机号验证码缓存前缀。
const UPDATE_PHONE_CODE_CACHE_PREFIX = "admin:update:phone:code:"

// AuthCase 认证业务实例
type AuthCase struct {
	*biz.BaseCase
	baseUserCase   *BaseUserCase
	baseRoleCase   *BaseRoleCase
	baseDeptCase   *BaseDeptCase
	baseTenantCase *BaseTenantCase
	baseMenuCase   *BaseMenuCase
	fileCase       *baseBiz.FileCase
	userInfoMapper *mapper.CopierMapper[
		systemadminv1.UserInfoForm,
		models.BaseUser,
	]
	profileMapper *mapper.CopierMapper[
		systemadminv1.UserProfileForm,
		models.BaseUser,
	]
}

// NewAuthCase 创建认证业务实例
func NewAuthCase(
	baseCase *biz.BaseCase,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	baseTenantCase *BaseTenantCase,
	baseMenuCase *BaseMenuCase,
	fileCase *baseBiz.FileCase,
) *AuthCase {
	return &AuthCase{
		BaseCase:       baseCase,
		baseUserCase:   baseUserCase,
		baseRoleCase:   baseRoleCase,
		baseDeptCase:   baseDeptCase,
		baseTenantCase: baseTenantCase,
		baseMenuCase:   baseMenuCase,
		fileCase:       fileCase,
		userInfoMapper: mapper.NewCopierMapper[
			systemadminv1.UserInfoForm,
			models.BaseUser,
		](),
		profileMapper: mapper.NewCopierMapper[
			systemadminv1.UserProfileForm,
			models.BaseUser,
		](),
	}
}

// TreeUserMenu 获取用户菜单
func (c *AuthCase) TreeUserMenu(ctx context.Context) (*systemadminv1.TreeRouteResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindByID(ctx, authInfo.RoleId)
	if err != nil {
		return nil, errorsx.Internal("获取用户菜单失败").WithCause(err)
	}
	// 角色被停用时，不允许继续获取菜单。
	if baseRole.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("角色已被禁用")
	}

	query := c.baseMenuCase.Query(ctx).BaseMenu
	var appMenuIDs []int64
	appMenuIDs, err = c.baseMenuCase.listSubtreeIDs(ctx, _const.BASE_MENU_APP_ROOT_ID)
	if err != nil {
		return nil, errorsx.Internal("获取用户菜单失败").WithCause(err)
	}

	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	opts = append(opts, repository.Where(query.Status.Eq(coreconst.Status_STATUS_ENABLE)))
	opts = append(opts, repository.Where(query.Type.In(
		_const.BASE_MENU_TYPE_FOLDER,
		_const.BASE_MENU_TYPE_MENU,
		_const.BASE_MENU_TYPE_EXT_LINK,
	)))
	// 移动端菜单树只承载应用端页面和接口权限，不参与管理后台动态路由。
	opts = append(opts, repository.Where(query.ID.NotIn(appMenuIDs...)))
	// 非超级管理员仅允许查看角色菜单里配置过的菜单。
	if baseRole.Code != coreconst.BASE_ROLE_CODE_SUPER {
		ids := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
		// 角色未配置任何菜单时，直接返回空菜单树。
		if len(ids) == 0 {
			return &systemadminv1.TreeRouteResponse{Routes: []*systemadminv1.RouteItem{}}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(ids...)))
	}

	var menuList []*models.BaseMenu
	menuList, err = c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("获取用户菜单失败").WithCause(err)
	}

	var titles map[int64]string
	titles, err = c.baseMenuCase.translatedMenuTitles(ctx, menuList)
	if err != nil {
		return nil, errorsx.Internal("获取用户菜单失败").WithCause(err)
	}
	list := c.baseMenuCase.buildRouteTree(menuList, 0, titles)
	return &systemadminv1.TreeRouteResponse{Routes: list}, nil
}

// ListUserButton 获取用户按钮
func (c *AuthCase) ListUserButton(ctx context.Context) (*commonv1.StringValues, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续获取按钮权限。
	if baseUser.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	// 查询角色信息
	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindByID(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errorsx.Internal("查询用户按钮权限失败").WithCause(err)
	}

	query := c.baseMenuCase.Query(ctx).BaseMenu

	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(coreconst.Status_STATUS_ENABLE)))
	opts = append(opts, repository.Where(query.Type.In(_const.BASE_MENU_TYPE_BUTTON)))
	// 非超级管理员仅允许查看角色菜单里配置过的按钮。
	if baseRole.Code != coreconst.BASE_ROLE_CODE_SUPER {
		ids := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
		// 角色未配置任何按钮时，直接返回空按钮集。
		if len(ids) == 0 {
			return &commonv1.StringValues{}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(ids...)))
	}

	var baseMenus []*models.BaseMenu
	baseMenus, err = c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("查询用户按钮权限失败").WithCause(err)
	}

	permission := make([]string, 0, len(baseMenus))
	for _, item := range baseMenus {
		permission = append(permission, item.Path)
	}

	return &commonv1.StringValues{
		Value: permission,
	}, nil
}

// GetUserInfo 获取用户信息
func (c *AuthCase) GetUserInfo(ctx context.Context) (*systemadminv1.UserInfoForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续访问后台信息。
	if baseUser.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	// 查询角色信息
	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindByID(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errorsx.Internal("获取用户信息失败").WithCause(err)
	}

	// 查询部门信息
	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptCase.FindByID(ctx, baseUser.DeptID)
	if err != nil {
		return nil, errorsx.Internal("获取用户信息失败").WithCause(err)
	}

	// 查询租户信息，用于前端区分默认租户与普通租户展示范围。
	var baseTenant *models.BaseTenant
	baseTenant, err = c.baseTenantCase.FindByID(ctx, baseUser.TenantID)
	if err != nil {
		return nil, errorsx.Internal("获取用户信息失败").WithCause(err)
	}

	res := c.userInfoMapper.ToDTO(baseUser)
	res.RoleCode = baseRole.Code
	res.RoleName = baseRole.Name
	res.DeptName = baseDept.Name
	res.TenantCode = baseTenant.Code
	res.TenantName = baseTenant.Name
	return res, nil
}

// GetUserProfile 获取用户资料
func (c *AuthCase) GetUserProfile(ctx context.Context) (*systemadminv1.UserProfileForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续获取个人资料。
	if baseUser.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindByID(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errorsx.Internal("获取个人资料失败").WithCause(err)
	}

	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptCase.FindByID(ctx, baseUser.DeptID)
	if err != nil {
		return nil, errorsx.Internal("获取个人资料失败").WithCause(err)
	}

	res := c.profileMapper.ToDTO(baseUser)
	res.RoleName = baseRole.Name
	res.DeptName = baseDept.Name
	return res, nil
}

// UpdateUserPassword 更新用户密码
func (c *AuthCase) UpdateUserPassword(ctx context.Context, req *systemadminv1.UserPasswordForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var oldPwd string
	oldPwd, err = utils.DecryptPassword(req.GetOldPwd(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_UPDATE_USER_PASSWORD)
	if err != nil {
		return err
	}
	var newPwd string
	newPwd, err = utils.DecryptPassword(req.GetNewPwd(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_UPDATE_USER_PASSWORD)
	if err != nil {
		return err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	err = crypto.Verify(oldPwd, baseUser.Password)
	if err != nil {
		return errorsx.InvalidArgument("原密码错误")
	}

	var encrypted string
	encrypted, err = crypto.Encrypt(newPwd)
	if err != nil {
		return errorsx.Internal("修改密码失败").WithCause(err)
	}

	return c.baseUserCase.UpdateByID(ctx, &models.BaseUser{
		ID:       authInfo.UserId,
		Password: encrypted,
	})
}

// UpdateUserPhone 更新用户手机号
func (c *AuthCase) UpdateUserPhone(ctx context.Context, req *systemadminv1.UserPhoneForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	cacheKey := c.makeUpdatePhoneCodeCacheKey(authInfo.UserId, req.GetPhone())
	var cacheCode string
	cacheCode, err = sdk.Runtime.GetCache().Get(cacheKey)
	// 验证码不存在或已过期时，直接返回业务错误。
	if err != nil || cacheCode == "" {
		return errorsx.InvalidArgument("验证码已过期")
	}
	// 验证码不匹配时，直接返回业务错误。
	if cacheCode != req.GetCode() {
		return errorsx.InvalidArgument("验证码错误")
	}

	var userID int64
	userID, err = c.findUserIDByPhone(ctx, req.GetPhone())
	// 手机号占用查询异常时，统一返回修改失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errorsx.Internal("修改手机号失败").WithCause(err)
	}
	// 手机号已绑定其他用户时，不允许继续修改。
	if userID > 0 && userID != authInfo.UserId {
		return errorsx.Conflict("手机号已被占用").WithMetadata(map[string]string{
			errorsx.METADATA_KEY_CONFLICT_TYPE: errorsx.CONFLICT_TYPE_UNIQUE_VIOLATION,
			errorsx.METADATA_KEY_RESOURCE:      "base_user",
			errorsx.METADATA_KEY_FIELD:         "phone",
		})
	}

	err = c.baseUserCase.UpdateByID(ctx, &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: req.GetPhone(),
	})
	if err != nil {
		return errorsx.Internal("修改手机号失败").WithCause(err)
	}

	err = sdk.Runtime.GetCache().Del(cacheKey)
	// 验证码缓存删除失败时，只记录日志不影响主流程。
	if err != nil {
		log.Error(fmt.Sprintf("删除修改手机号验证码缓存失败 %v", err))
	}
	return nil
}

// UpdateUserProfile 更新用户资料
func (c *AuthCase) UpdateUserProfile(ctx context.Context, req *systemadminv1.UserProfileForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	baseUser := models.BaseUser{
		ID:       authInfo.UserId,
		UserName: req.GetUserName(),
		NickName: req.GetNickName(),
		Avatar:   req.GetAvatar(),
		Gender:   req.GetGender(),
	}
	err = c.baseUserCase.UpdateByID(ctx, &baseUser)
	if err != nil {
		return errorsx.Internal("修改个人中心用户信息失败").WithCause(err)
	}
	// 删除图片
	c.fileCase.DeleteFile(oldBaseUser.Avatar, baseUser.Avatar)

	return nil
}

// SendPhoneCode 发送更新手机号验证码
func (c *AuthCase) SendPhoneCode(ctx context.Context, req *systemadminv1.SendPhoneCodeRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var userID int64
	userID, err = c.findUserIDByPhone(ctx, req.GetPhone())
	// 手机号占用查询异常时，统一返回发送失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errorsx.Internal("发送验证码失败").WithCause(err)
	}
	// 手机号已绑定其他用户时，不允许继续发送验证码。
	if userID > 0 && userID != authInfo.UserId {
		return errorsx.Conflict("手机号已被占用").WithMetadata(map[string]string{
			errorsx.METADATA_KEY_CONFLICT_TYPE: errorsx.CONFLICT_TYPE_UNIQUE_VIOLATION,
			errorsx.METADATA_KEY_RESOURCE:      "base_user",
			errorsx.METADATA_KEY_FIELD:         "phone",
		})
	}

	code := fmt.Sprintf("%06d", rand.IntN(1000000))
	err = sdk.Runtime.GetCache().Set(c.makeUpdatePhoneCodeCacheKey(authInfo.UserId, req.GetPhone()), code, 5*time.Minute)
	if err != nil {
		return errorsx.Internal("发送验证码失败").WithCause(err)
	}

	// 当前先将验证码写入日志，后续接入短信渠道时替换这里。
	log.Info(fmt.Sprintf("send update phone code userID=%d phone=%s code=%s", authInfo.UserId, req.GetPhone(), code))
	return nil
}

// findUserIDByPhone 根据手机号查询用户ID
func (c *AuthCase) findUserIDByPhone(ctx context.Context, phone string) (int64, error) {
	query := c.baseUserCase.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Phone.Eq(phone)))
	baseUser, err := c.baseUserCase.Find(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return baseUser.ID, nil
}

// makeUpdatePhoneCodeCacheKey 生成更新手机号验证码缓存键
func (c *AuthCase) makeUpdatePhoneCodeCacheKey(userID int64, phone string) string {
	return fmt.Sprintf("%s%d:%s", UPDATE_PHONE_CODE_CACHE_PREFIX, userID, phone)
}
