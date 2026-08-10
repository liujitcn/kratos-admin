package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	coreconst "github.com/liujitcn/kratos-core/pkg/const"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/captcha"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

const loginCaptchaKeyPrefix = "login_captcha"
const loginCaptchaTokenKeyPrefix = "login_captcha_token"
const loginCaptchaTypeKeyPrefix = "login_captcha_type"
const refreshTokenAuthKeyPrefix = "refresh_token_auth"
const loginCaptchaTokenExpire = 2 * time.Minute
const loginCaptchaRandomType = "random"
const loginCaptchaTypeDictCode = "captcha_type"

var supportedCaptchaDriverTypes = [...]captcha.DriverType{
	captcha.DriverDigit,
	captcha.DriverString,
	captcha.DriverMath,
	captcha.DriverChinese,
	captcha.DriverSlide,
	captcha.DriverClick,
	captcha.DriverRotate,
}

// LoginCase 处理基础登录认证业务。
type LoginCase struct {
	*biz.BaseCase
	userToken        *authData.UserToken
	baseDeptCase     *BaseDeptCase
	baseRoleCase     *BaseRoleCase
	baseUserCase     *BaseUserCase
	baseTenantRepo   *data.BaseTenantRepository
	baseDictRepo     *data.BaseDictRepository
	baseDictItemRepo *data.BaseDictItemRepository
}

// NewLoginCase 创建登录业务实例。
func NewLoginCase(
	baseCase *biz.BaseCase,
	userToken *authData.UserToken,
	baseDeptRepo *BaseDeptCase,
	baseRoleRepo *BaseRoleCase,
	baseUserRepo *BaseUserCase,
	baseTenantRepo *data.BaseTenantRepository,
	baseDictRepo *data.BaseDictRepository,
	baseDictItemRepo *data.BaseDictItemRepository,
) *LoginCase {
	return &LoginCase{
		BaseCase:         baseCase,
		userToken:        userToken,
		baseDeptCase:     baseDeptRepo,
		baseRoleCase:     baseRoleRepo,
		baseUserCase:     baseUserRepo,
		baseTenantRepo:   baseTenantRepo,
		baseDictRepo:     baseDictRepo,
		baseDictItemRepo: baseDictItemRepo,
	}
}

// Captcha 生成验证码。
func (c *LoginCase) Captcha(ctx context.Context, req *basev1.CaptchaRequest) (*basev1.CaptchaResponse, error) {
	driverType, ok, err := c.resolveCaptchaDriverType(ctx, req.GetType())
	if err != nil {
		return nil, errorsx.Internal("查询启用验证码类型失败").WithCause(err)
	}
	// 请求的验证码类型不在系统字典支持范围内时，直接拒绝生成。
	if !ok {
		return nil, errorsx.InvalidArgument("验证码类型不支持")
	}

	var challenge *captcha.Challenge
	challenge, err = captcha.NewCaptcha(c.GetCache(),
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Generate(ctx)
	if err != nil {
		return nil, errorsx.Internal("生成验证码失败").WithCause(err)
	}
	err = c.GetCache().Set(loginCaptchaTypeKey(challenge.ID), string(driverType), captcha.DefaultConfig().Expire)
	if err != nil {
		return nil, errorsx.Internal("验证码类型保存失败").WithCause(err)
	}
	return &basev1.CaptchaResponse{
		CaptchaId:     challenge.ID,
		CaptchaBase64: challenge.Payload,
		Type:          string(driverType),
	}, nil
}

// VerifyCaptcha 预校验验证码并签发一次性登录令牌。
func (c *LoginCase) VerifyCaptcha(ctx context.Context, req *basev1.VerifyCaptchaRequest) (*basev1.VerifyCaptchaResponse, error) {
	driverType, ok := c.captchaDriverTypeByID(req.GetCaptchaId())
	// 验证码类型缺失时，说明验证码已过期或不是当前系统签发。
	if !ok {
		return nil, errorsx.InvalidArgument("验证码错误")
	}
	matched, err := captcha.NewCaptcha(c.GetCache(),
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Verify(ctx, req.GetCaptchaId(), req.GetCaptchaCode())
	if err != nil {
		return nil, errorsx.Internal("验证码校验失败").WithCause(err)
	}
	// 验证码校验失败时，不签发可用于登录的一次性令牌。
	if !matched {
		return nil, errorsx.InvalidArgument("验证码错误")
	}

	token := id.NewGUIDv4NoHyphen()
	err = c.GetCache().Set(loginCaptchaTokenKey(req.GetCaptchaId(), token), req.GetCaptchaId(), loginCaptchaTokenExpire)
	if err != nil {
		return nil, errorsx.Internal("验证码令牌保存失败").WithCause(err)
	}
	return &basev1.VerifyCaptchaResponse{
		CaptchaToken: token,
		ExpiresIn:    int64(loginCaptchaTokenExpire / time.Second),
	}, nil
}

// PasswordPublicKey 生成密码加密临时公钥。
func (c *LoginCase) PasswordPublicKey(ctx context.Context, req *basev1.PasswordPublicKeyRequest) (*basev1.PasswordPublicKeyResponse, error) {
	return utils.GeneratePasswordPublicKey(req.GetScene())
}

// Logout 退出登录。
func (c *LoginCase) Logout(ctx context.Context, req *basev1.LogoutRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	refreshToken := c.userToken.GetRefreshToken(authInfo.UserId)
	err = c.userToken.RemoveToken(authInfo.UserId)
	if err != nil {
		return errorsx.Internal("退出登录失败").WithCause(err)
	}
	if refreshToken != "" {
		err = c.GetCache().Del(refreshTokenAuthKey(refreshToken))
		if err != nil {
			return errorsx.Internal("退出登录失败").WithCause(err)
		}
	}
	return nil
}

// RefreshToken 刷新认证令牌。
func (c *LoginCase) RefreshToken(ctx context.Context, req *basev1.RefreshTokenRequest) (*basev1.RefreshTokenResponse, error) {
	refreshToken := req.GetRefreshToken()
	authInfo, err := c.getAuthInfoByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.Unauthenticated("刷新认证令牌失败")
		}
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	authInfo, err = c.buildAuthInfo(ctx, user)
	if err != nil {
		return nil, err
	}

	// 生成新的访问令牌
	var accessToken string
	accessToken, err = c.userToken.GenerateAccessToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	// Token 有效期
	expiresIn := c.userToken.GetAccessTokenExpires()

	return &basev1.RefreshTokenResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// Login 执行登录。
func (c *LoginCase) Login(ctx context.Context, req *basev1.LoginRequest) (*basev1.LoginResponse, error) {
	err := c.verifyLoginCaptcha(ctx, req.GetCaptchaId(), req.GetCaptchaCode())
	if err != nil {
		return nil, err
	}

	tenantCode := req.GetTenantCode()
	if tenantCode == "" {
		tenantCode = databaseGorm.DefaultTenantCode
	}
	var baseTenant *models.BaseTenant
	baseTenant, err = c.findTenantByCode(ctx, tenantCode)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if baseTenant.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("租户已被禁用")
	}

	var user *models.BaseUser
	userQuery := c.baseUserCase.Query(ctx).BaseUser
	userOpts := make([]repository.QueryOption, 0, 2)
	userOpts = append(userOpts, repository.Where(userQuery.TenantID.Eq(baseTenant.ID)))
	userOpts = append(userOpts, repository.Where(userQuery.UserName.Eq(req.GetUserName())))
	user, err = c.baseUserCase.Find(ctx, userOpts...)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	var password string
	password, err = utils.DecryptPassword(req.GetPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_LOGIN)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误").WithCause(err)
	}
	err = crypto.Verify(password, user.Password)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}

	return c.IssueUserToken(ctx, user)
}

// FindUserByPassword 按租户、用户名和加密密码查找已有用户，不执行验证码校验。
func (c *LoginCase) FindUserByPassword(ctx context.Context, tenantCode string, userName string, encryptedPassword *commonv1.PasswordCrypto) (*models.BaseUser, error) {
	var err error
	if tenantCode == "" {
		tenantCode = databaseGorm.DefaultTenantCode
	}
	var baseTenant *models.BaseTenant
	baseTenant, err = c.findTenantByCode(ctx, tenantCode)
	if err != nil || baseTenant.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}

	userQuery := c.baseUserCase.Query(ctx).BaseUser
	userOpts := make([]repository.QueryOption, 0, 2)
	userOpts = append(userOpts, repository.Where(userQuery.TenantID.Eq(baseTenant.ID)))
	userOpts = append(userOpts, repository.Where(userQuery.UserName.Eq(userName)))
	var user *models.BaseUser
	user, err = c.baseUserCase.Find(ctx, userOpts...)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	var password string
	password, err = utils.DecryptPassword(encryptedPassword, basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_LOGIN)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误").WithCause(err)
	}
	if err = crypto.Verify(password, user.Password); err != nil {
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	return user, nil
}

// IssueUserToken 校验用户关联状态并签发后台访问令牌。
func (c *LoginCase) IssueUserToken(ctx context.Context, user *models.BaseUser) (*basev1.LoginResponse, error) {
	authInfo, err := c.buildAuthInfo(ctx, user)
	if err != nil {
		return nil, err
	}

	// 生成访问令牌
	var accessToken, refreshToken string
	accessToken, refreshToken, err = c.userToken.GenerateToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	err = c.setRefreshTokenAuth(refreshToken, authInfo)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// Token 有效期
	expiresIn := c.userToken.GetAccessTokenExpires()

	return &basev1.LoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// buildAuthInfo 查询用户关联状态并构造认证载荷。
func (c *LoginCase) buildAuthInfo(ctx context.Context, user *models.BaseUser) (*authData.UserTokenPayload, error) {
	// 用户被停用时，不允许签发新的登录令牌。
	if user.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	// 查询角色信息
	role, err := c.baseRoleCase.FindByID(ctx, user.RoleID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 角色被停用时，不允许继续登录后台。
	if role.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("角色已被禁用")
	}

	// 查询部门信息
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindByID(ctx, user.DeptID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 部门被停用时，不允许继续登录后台。
	if dept.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("部门已被禁用")
	}

	var baseTenant *models.BaseTenant
	baseTenant, err = c.baseTenantRepo.FindByID(ctx, user.TenantID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 租户被停用时，不允许继续登录后台。
	if baseTenant.Status != coreconst.Status_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("租户已被禁用")
	}

	return &authData.UserTokenPayload{
		UserId:     user.ID,
		UserCode:   user.UserCode,
		UserName:   user.UserName,
		RoleId:     user.RoleID,
		RoleCode:   role.Code,
		RoleName:   role.Name,
		DataScope:  role.DataScope,
		TenantId:   user.TenantID,
		TenantCode: baseTenant.Code,
		DeptId:     user.DeptID,
		DeptName:   dept.Name,
	}, nil
}

// findTenantByCode 按编码查询租户。
func (c *LoginCase) findTenantByCode(ctx context.Context, code string) (*models.BaseTenant, error) {
	query := c.baseTenantRepo.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(code)))
	return c.baseTenantRepo.Find(ctx, opts...)
}

// setRefreshTokenAuth 保存刷新令牌关联的认证信息。
func (c *LoginCase) setRefreshTokenAuth(refreshToken string, authInfo *authData.UserTokenPayload) error {
	if refreshToken == "" || authInfo == nil {
		return errorsx.Unauthenticated("刷新认证令牌失败")
	}

	payload, err := json.Marshal(authInfo)
	if err != nil {
		return errorsx.Internal("保存刷新认证信息失败").WithCause(err)
	}
	return c.GetCache().Set(refreshTokenAuthKey(refreshToken), string(payload), time.Duration(c.userToken.GetRefreshTokenExpires())*time.Second)
}

// getAuthInfoByRefreshToken 根据刷新令牌读取认证信息。
func (c *LoginCase) getAuthInfoByRefreshToken(refreshToken string) (*authData.UserTokenPayload, error) {
	payload, err := c.GetCache().Get(refreshTokenAuthKey(refreshToken))
	if err != nil {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败").WithCause(err)
	}

	authInfo := &authData.UserTokenPayload{}
	err = json.Unmarshal([]byte(payload), authInfo)
	if err != nil {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败").WithCause(err)
	}

	cachedRefreshToken := c.userToken.GetRefreshToken(authInfo.UserId)
	if cachedRefreshToken != refreshToken {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败")
	}
	return authInfo, nil
}

// verifyLoginCaptcha 校验登录请求携带的验证码或行为验证码令牌。
func (c *LoginCase) verifyLoginCaptcha(ctx context.Context, captchaID, captchaCode string) error {
	driverType, ok := c.captchaDriverTypeByID(captchaID)
	// 登录阶段先通过 captcha_id 取回类型，避免依赖验证码 ID 命名格式。
	if !ok {
		return errorsx.InvalidArgument("验证码错误")
	}
	matched, err := captcha.NewCaptcha(c.GetCache(),
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Verify(ctx, captchaID, captchaCode)
	if err != nil {
		return errorsx.Internal("验证码校验失败").WithCause(err)
	}
	if matched {
		return nil
	}
	// 预校验会删除原始答案并签发 token，原始 code 未命中时再兼容 token 登录。
	var consumed bool
	consumed, err = c.consumeLoginCaptchaToken(captchaID, captchaCode)
	if err != nil {
		return err
	}
	if consumed {
		return nil
	}
	return errorsx.InvalidArgument("验证码错误")
}

// consumeLoginCaptchaToken 校验并消费验证码预校验签发的一次性令牌。
func (c *LoginCase) consumeLoginCaptchaToken(captchaID, token string) (bool, error) {
	key := loginCaptchaTokenKey(captchaID, token)
	value, err := c.GetCache().Get(key)
	if err != nil || value != captchaID {
		return false, nil
	}
	err = c.GetCache().Del(key)
	if err != nil {
		return false, errorsx.Internal("验证码令牌消费失败").WithCause(err)
	}
	return true, nil
}

// captchaDriverTypeByID 根据验证码 ID 查询生成时保存的校验驱动类型。
func (c *LoginCase) captchaDriverTypeByID(captchaID string) (captcha.DriverType, bool) {
	captchaType, err := c.GetCache().Get(loginCaptchaTypeKey(captchaID))
	if err != nil {
		return captcha.DriverDigit, false
	}
	driverType, ok := captchaDriverType(captchaType)
	// 验证码 ID 不承载类型语义，需要回读生成时保存的类型。
	return driverType, ok
}

// captchaDriverType 根据配置值解析验证码驱动类型。
func captchaDriverType(captchaType string) (captcha.DriverType, bool) {
	// 兼容未配置验证码类型的历史场景，默认继续使用数字验证码。
	switch captchaType {
	case "", string(captcha.DriverDigit):
		return captcha.DriverDigit, true
	case string(captcha.DriverString):
		return captcha.DriverString, true
	case string(captcha.DriverMath):
		return captcha.DriverMath, true
	case string(captcha.DriverChinese):
		return captcha.DriverChinese, true
	case string(captcha.DriverSlide):
		return captcha.DriverSlide, true
	case string(captcha.DriverClick):
		return captcha.DriverClick, true
	case string(captcha.DriverRotate):
		return captcha.DriverRotate, true
	default:
		return captcha.DriverDigit, false
	}
}

// resolveCaptchaDriverType 解析验证码请求类型，随机类型从启用字典项中选择。
func (c *LoginCase) resolveCaptchaDriverType(ctx context.Context, captchaType string) (captcha.DriverType, bool, error) {
	if captchaType != loginCaptchaRandomType {
		driverType, ok := captchaDriverType(captchaType)
		return driverType, ok, nil
	}

	driverType, err := c.randomCaptchaDriverType(ctx)
	if err != nil {
		return captcha.DriverDigit, false, err
	}
	return driverType, true, nil
}

// randomCaptchaDriverType 从验证码字典的启用项中随机选择驱动类型。
func (c *LoginCase) randomCaptchaDriverType(ctx context.Context) (captcha.DriverType, error) {
	dictQuery := c.baseDictRepo.Query(ctx).BaseDict
	dictOpts := make([]repository.QueryOption, 0, 2)
	dictOpts = append(dictOpts, repository.Where(dictQuery.Code.Eq(loginCaptchaTypeDictCode)))
	dictOpts = append(dictOpts, repository.Where(dictQuery.Status.Eq(coreconst.Status_STATUS_ENABLE)))
	dict, err := c.baseDictRepo.Find(ctx, dictOpts...)
	if err != nil {
		return captcha.DriverDigit, err
	}

	itemQuery := c.baseDictItemRepo.Query(ctx).BaseDictItem
	itemOpts := make([]repository.QueryOption, 0, 2)
	itemOpts = append(itemOpts, repository.Where(itemQuery.DictID.Eq(dict.ID)))
	itemOpts = append(itemOpts, repository.Where(itemQuery.Status.Eq(coreconst.Status_STATUS_ENABLE)))
	var items []*models.BaseDictItem
	items, err = c.baseDictItemRepo.List(ctx, itemOpts...)
	if err != nil {
		return captcha.DriverDigit, err
	}

	enabledTypes := make([]captcha.DriverType, 0, len(supportedCaptchaDriverTypes))
	for _, item := range items {
		for _, driverType := range supportedCaptchaDriverTypes {
			if item.Value != string(driverType) {
				continue
			}
			enabledTypes = append(enabledTypes, driverType)
			break
		}
	}
	if len(enabledTypes) == 0 {
		return captcha.DriverDigit, errorsx.Internal("没有启用的验证码类型")
	}
	return enabledTypes[rand.IntN(len(enabledTypes))], nil
}

// loginCaptchaTokenKey 生成验证码预校验令牌缓存键。
func loginCaptchaTokenKey(captchaID, token string) string {
	return fmt.Sprintf("%s:%s:%s", loginCaptchaTokenKeyPrefix, captchaID, token)
}

// loginCaptchaTypeKey 生成验证码类型缓存键。
func loginCaptchaTypeKey(captchaID string) string {
	return fmt.Sprintf("%s:%s", loginCaptchaTypeKeyPrefix, captchaID)
}

// refreshTokenAuthKey 生成刷新令牌认证信息缓存键。
func refreshTokenAuthKey(refreshToken string) string {
	return fmt.Sprintf("%s:%s", refreshTokenAuthKeyPrefix, refreshToken)
}
