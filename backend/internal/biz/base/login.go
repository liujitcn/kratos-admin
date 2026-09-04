package biz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	passwordPolicy "github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	adminconst "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	_const "github.com/liujitcn/kratos-core/const"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/captcha"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const loginCaptchaKeyPrefix = "login_captcha"
const loginCaptchaTokenKeyPrefix = "login_captcha_token"
const loginCaptchaTypeKeyPrefix = "login_captcha_type"
const refreshTokenAuthKeyPrefix = "refresh_token_auth"
const loginCaptchaTokenExpire = 2 * time.Minute
const loginCaptchaRandomType = "random"
const loginCaptchaTypeDictCode = "captcha_type"
const loginFailureKeyPrefix = "login_failure_v2"

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
	baseDeptCase     *BaseDeptCase
	baseRoleCase     *BaseRoleCase
	baseUserCase     *BaseUserCase
	baseTenantRepo   *data.BaseTenantRepository
	baseDictRepo     *data.BaseDictRepository
	baseDictItemRepo *data.BaseDictItemRepository
	mfaCase          *MfaCase
	userToken        *authData.UserToken
	loginPolicyMu    sync.Mutex
}

// NewLoginCase 创建登录业务实例。
func NewLoginCase(
	baseCase *biz.BaseCase,
	baseDeptRepo *BaseDeptCase,
	baseRoleRepo *BaseRoleCase,
	baseUserRepo *BaseUserCase,
	baseTenantRepo *data.BaseTenantRepository,
	baseDictRepo *data.BaseDictRepository,
	baseDictItemRepo *data.BaseDictItemRepository,
	mfaCase *MfaCase,
	userToken *authData.UserToken,
) *LoginCase {
	return &LoginCase{
		BaseCase:         baseCase,
		baseDeptCase:     baseDeptRepo,
		baseRoleCase:     baseRoleRepo,
		baseUserCase:     baseUserRepo,
		baseTenantRepo:   baseTenantRepo,
		baseDictRepo:     baseDictRepo,
		baseDictItemRepo: baseDictItemRepo,
		mfaCase:          mfaCase,
		userToken:        userToken,
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
	challenge, err = captcha.NewCaptcha(c.Cache,
		captcha.WithDriverType(driverType),
		captcha.WithKeyPrefix(loginCaptchaKeyPrefix),
	).Generate(ctx)
	if err != nil {
		return nil, errorsx.Internal("生成验证码失败").WithCause(err)
	}
	err = c.Cache.Set(loginCaptchaTypeKey(challenge.ID), string(driverType), captcha.DefaultConfig().Expire)
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
	matched, err := captcha.NewCaptcha(c.Cache,
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
	err = c.Cache.Set(loginCaptchaTokenKey(req.GetCaptchaId(), token), req.GetCaptchaId(), loginCaptchaTokenExpire)
	if err != nil {
		return nil, errorsx.Internal("验证码令牌保存失败").WithCause(err)
	}
	return &basev1.VerifyCaptchaResponse{
		CaptchaToken: token,
		ExpiresIn:    int64(loginCaptchaTokenExpire / time.Second),
	}, nil
}

// PasswordPublicKey 生成密码加密临时公钥。
func (c *LoginCase) PasswordPublicKey(_ context.Context, req *basev1.PasswordPublicKeyRequest) (*basev1.PasswordPublicKeyResponse, error) {
	return utils.GeneratePasswordPublicKey(c.Cache, req.GetScene())
}

// Logout 退出登录。
func (c *LoginCase) Logout(ctx context.Context, _ *basev1.LogoutRequest) error {
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
		err = c.Cache.Del(refreshTokenAuthKey(refreshToken))
		if err != nil {
			return errorsx.Internal("退出登录失败").WithCause(err)
		}
	}
	if err = sessionstate.Clear(c.Cache, authInfo.UserId); err != nil {
		return errorsx.Internal("清理会话状态失败").WithCause(err)
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
	if requiresServerSession(authInfo.RoleCode) {
		_, err = sessionstate.Validate(c.Cache, authInfo.UserId, time.Now())
		if errors.Is(err, sessionstate.ErrStateNotFound) {
			_, err = sessionstate.Start(c.Cache, authInfo.UserId, loginClientIP(ctx), loginDevice(ctx), time.Now())
		}
		if err != nil {
			if errors.Is(err, sessionstate.ErrIdleExpired) || errors.Is(err, sessionstate.ErrMaxLifetimeExpired) {
				_ = c.userToken.RemoveToken(authInfo.UserId)
				_ = sessionstate.Clear(c.Cache, authInfo.UserId)
				return nil, errorsx.Unauthenticated("会话已超时，请重新登录")
			}
			return nil, errorsx.Internal("校验会话状态失败").WithCause(err)
		}
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.Unauthenticated("刷新认证令牌失败")
		}
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	if c.mfaCase != nil {
		var mfaPolicy string
		mfaPolicy, err = c.mfaCase.policy(ctx)
		if err != nil {
			return nil, err
		}
		if mfaPolicy == mfaPolicyAllRequired {
			return nil, errorsx.Unauthenticated("多因素认证策略要求重新登录")
		}
	}
	authInfo, err = c.buildAuthInfo(ctx, user)
	if err != nil {
		return nil, err
	}

	// 每次刷新同时轮换 Refresh Token，旧令牌立即失效，避免被重放。
	var accessToken string
	accessToken, err = c.userToken.GenerateAccessToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	var nextRefreshToken string
	nextRefreshToken, err = c.userToken.GenerateRefreshToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	if err = c.setRefreshTokenAuth(nextRefreshToken, authInfo, c.userToken.GetRefreshTokenExpires()); err != nil {
		return nil, errorsx.Internal("刷新认证令牌失败").WithCause(err)
	}
	if requiresServerSession(authInfo.RoleCode) {
		if err = sessionstate.MarkTokenIssued(c.Cache, authInfo.UserId, time.Now()); err != nil {
			_ = c.userToken.RemoveToken(authInfo.UserId)
			return nil, errorsx.Internal("更新会话状态失败").WithCause(err)
		}
	}
	return &basev1.RefreshTokenResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: nextRefreshToken,
		ExpiresIn:    c.userToken.GetAccessTokenExpires(),
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
	var loginSourcePolicy loginpolicy.PolicySet
	loginSourcePolicy, err = loginpolicy.LoadFromCacheStrict(c.Cache)
	if err != nil {
		return nil, errorsx.Internal("读取登录来源策略失败").WithCause(err)
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, 0, 0); err != nil {
		return nil, err
	}
	var baseTenant *models.BaseTenant
	baseTenant, err = c.findTenantByCode(ctx, tenantCode)
	if err != nil {
		if err = c.recordLoginFailure(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, 0, 0); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if baseTenant.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("租户已被禁用")
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, baseTenant.ID, 0); err != nil {
		return nil, err
	}

	var user *models.BaseUser
	userQuery := c.baseUserCase.Query(ctx).BaseUser
	userOpts := make([]repository.QueryOption, 0, 1)
	userOpts = append(userOpts, repository.Where(userQuery.UserName.Eq(req.GetUserName())))
	user, err = c.baseUserCase.Find(ctx, userOpts...)
	if err != nil {
		if err = c.recordLoginFailure(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, baseTenant.ID, 0); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
		return nil, err
	}
	if blocked, reason := loginSourcePolicy.EvaluateFor(baseTenant.ID, user.ID, loginClientIP(ctx), loginMAC(ctx), loginRegion(ctx), loginDevice(ctx), time.Now()); blocked {
		return nil, errorsx.PermissionDenied(reason)
	}
	var password string
	password, err = utils.DecryptPassword(c.Cache, req.GetPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_LOGIN)
	if err != nil {
		decryptErr := err
		if err = c.recordLoginFailure(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误").WithCause(decryptErr)
	}
	err = crypto.Verify(password, user.Password)
	if err != nil {
		if err = c.recordLoginFailure(ctx, tenantCode, req.GetUserName(), loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if err = c.clearLoginFailures(ctx, tenantCode, req.GetUserName()); err != nil {
		return nil, errorsx.Internal("清理登录失败状态失败").WithCause(err)
	}
	var response *basev1.LoginResponse
	response, err = c.IssueUserLogin(ctx, user)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// ValidateExternalLogin 将来源策略和密码有效期应用到 OAuth 等非密码登录入口。
func (c *LoginCase) ValidateExternalLogin(ctx context.Context, user *models.BaseUser) error {
	var err error
	var loginSourcePolicy loginpolicy.PolicySet
	loginSourcePolicy, err = loginpolicy.LoadFromCacheStrict(c.Cache)
	if err != nil {
		return errorsx.Internal("读取登录来源策略失败").WithCause(err)
	}
	if blocked, reason := loginSourcePolicy.EvaluateFor(user.TenantID, user.ID, loginClientIP(ctx), loginMAC(ctx), loginRegion(ctx), loginDevice(ctx), time.Now()); blocked {
		return errorsx.PermissionDenied(reason)
	}
	return nil
}

// VerifyMfa 校验登录阶段的 TOTP 或恢复码并签发正式令牌。
func (c *LoginCase) VerifyMfa(ctx context.Context, req *basev1.VerifyMfaRequest) (*basev1.LoginResponse, error) {
	user, err := c.mfaCase.VerifyLoginChallenge(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.IssueUserToken(ctx, user)
}

// BeginMfaEnrollment 开始登录阶段的强制 MFA 绑定。
func (c *LoginCase) BeginMfaEnrollment(ctx context.Context, req *basev1.BeginMfaEnrollmentRequest) (*basev1.BeginMfaEnrollmentResponse, error) {
	res, err := c.mfaCase.BeginMfaSetup(ctx, &basev1.BeginMfaSetupRequest{SetupTicket: req.GetSetupTicket()})
	if err != nil {
		return nil, err
	}
	return &basev1.BeginMfaEnrollmentResponse{
		SetupTicket:         res.GetSetupTicket(),
		OtpauthUri:          res.GetOtpauthUri(),
		ExpiresIn:           res.GetExpiresIn(),
		Method:              res.GetMethod(),
		WebauthnOptionsJson: res.GetWebauthnOptionsJson(),
	}, nil
}

// ConfirmMfaEnrollment 确认登录阶段的强制 MFA 绑定。
func (c *LoginCase) ConfirmMfaEnrollment(ctx context.Context, req *basev1.ConfirmMfaEnrollmentRequest) (*basev1.ConfirmMfaEnrollmentResponse, error) {
	res, err := c.mfaCase.ConfirmMfaSetup(ctx, &basev1.ConfirmMfaSetupRequest{SetupTicket: req.GetSetupTicket(), Code: req.GetCode(), WebauthnResponseJson: req.GetWebauthnResponseJson()})
	if err != nil {
		return nil, err
	}
	return &basev1.ConfirmMfaEnrollmentResponse{Enabled: res.GetEnabled(), RecoveryCodes: res.GetRecoveryCodes()}, nil
}

// FindUserByPassword 按租户、用户名和加密密码查找已有用户，不执行验证码校验。
func (c *LoginCase) FindUserByPassword(ctx context.Context, tenantCode string, userName string, encryptedPassword *commonv1.PasswordCrypto) (*models.BaseUser, error) {
	var err error
	if tenantCode == "" {
		tenantCode = databaseGorm.DefaultTenantCode
	}
	var loginSourcePolicy loginpolicy.PolicySet
	loginSourcePolicy, err = loginpolicy.LoadFromCacheStrict(c.Cache)
	if err != nil {
		return nil, errorsx.Internal("读取登录来源策略失败").WithCause(err)
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, userName, loginSourcePolicy, 0, 0); err != nil {
		return nil, err
	}
	var baseTenant *models.BaseTenant
	baseTenant, err = c.findTenantByCode(ctx, tenantCode)
	if err != nil || baseTenant.Status != _const.STATUS_STATUS_ENABLE {
		if err = c.recordLoginFailure(ctx, tenantCode, userName, loginSourcePolicy, 0, 0); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, userName, loginSourcePolicy, baseTenant.ID, 0); err != nil {
		return nil, err
	}

	userQuery := c.baseUserCase.Query(ctx).BaseUser
	userOpts := make([]repository.QueryOption, 0, 1)
	userOpts = append(userOpts, repository.Where(userQuery.UserName.Eq(userName)))
	var user *models.BaseUser
	user, err = c.baseUserCase.Find(ctx, userOpts...)
	if err != nil {
		if err = c.recordLoginFailure(ctx, tenantCode, userName, loginSourcePolicy, baseTenant.ID, 0); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if err = c.checkLoginPolicy(ctx, tenantCode, userName, loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
		return nil, err
	}
	if blocked, reason := loginSourcePolicy.EvaluateFor(baseTenant.ID, user.ID, loginClientIP(ctx), loginMAC(ctx), loginRegion(ctx), loginDevice(ctx), time.Now()); blocked {
		return nil, errorsx.PermissionDenied(reason)
	}
	var password string
	password, err = utils.DecryptPassword(c.Cache, encryptedPassword, basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_LOGIN)
	if err != nil {
		decryptErr := err
		if err = c.recordLoginFailure(ctx, tenantCode, userName, loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误").WithCause(decryptErr)
	}
	if err = crypto.Verify(password, user.Password); err != nil {
		if err = c.recordLoginFailure(ctx, tenantCode, userName, loginSourcePolicy, baseTenant.ID, user.ID); err != nil {
			return nil, errorsx.Internal("记录登录失败状态失败").WithCause(err)
		}
		return nil, errorsx.Unauthenticated("用户名或密码错误")
	}
	if err = c.clearLoginFailures(ctx, tenantCode, userName); err != nil {
		return nil, errorsx.Internal("清理登录失败状态失败").WithCause(err)
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
	var accessToken string
	var refreshToken string
	accessToken, refreshToken, err = c.userToken.GenerateToken(authInfo)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	err = c.setRefreshTokenAuth(refreshToken, authInfo, c.userToken.GetRefreshTokenExpires())
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	if requiresServerSession(authInfo.RoleCode) {
		_, err = sessionstate.Start(c.Cache, authInfo.UserId, loginClientIP(ctx), loginDevice(ctx), time.Now())
		if err != nil {
			_ = c.userToken.RemoveToken(authInfo.UserId)
			return nil, errorsx.Internal("创建会话状态失败").WithCause(err)
		}
	}
	status := basev1.LoginStatus_LOGIN_STATUS_AUTHENTICATED
	passwordExpired := false
	if authInfo.RoleCode != _const.BASE_ROLE_CODE_USER && authInfo.RoleCode != _const.BASE_ROLE_CODE_AUTHUSER {
		var policySet loginpolicy.PolicySet
		policySet, err = loginpolicy.LoadFromCacheStrict(c.Cache)
		if err != nil {
			return nil, errorsx.Internal("读取密码策略失败").WithCause(err)
		}
		passwordExpired = passwordPolicy.IsExpiredAtWithMaxAge(user.PasswordChangedAt, time.Now(), policySet.PasswordMaxAgeDaysFor(user.TenantID, user.ID))
	}
	if authInfo.RoleCode != _const.BASE_ROLE_CODE_USER && authInfo.RoleCode != _const.BASE_ROLE_CODE_AUTHUSER &&
		(user.MustChangePassword == adminconst.BASE_USER_PASSWORD_CHANGE_STATUS_REQUIRED || passwordExpired) {
		status = basev1.LoginStatus_LOGIN_STATUS_PASSWORD_CHANGE_REQUIRED
	}
	return &basev1.LoginResponse{
		TokenType:    engine.BearerWord,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    c.userToken.GetAccessTokenExpires(),
		Status:       status,
	}, nil
}

// IssueUserLogin 根据 MFA 状态决定返回登录挑战或签发正式令牌。
func (c *LoginCase) IssueUserLogin(ctx context.Context, user *models.BaseUser) (*basev1.LoginResponse, error) {
	decision, err := c.mfaCase.PrepareLogin(ctx, user)
	if err != nil {
		return nil, err
	}
	if decision != nil {
		return decision, nil
	}
	return c.IssueUserToken(ctx, user)
}

// RefreshTokenExpiresIn 返回刷新令牌有效期，供 HTTP Cookie 生命周期使用。
func (c *LoginCase) RefreshTokenExpiresIn() int64 {
	if c == nil || c.userToken == nil {
		return 0
	}
	return c.userToken.GetRefreshTokenExpires()
}

// buildAuthInfo 查询用户关联状态并构造认证载荷。
func (c *LoginCase) buildAuthInfo(ctx context.Context, user *models.BaseUser) (*authData.UserTokenPayload, error) {
	// 用户被停用时，不允许签发新的登录令牌。
	if user.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	// 查询角色信息
	role, err := c.baseRoleCase.FindByID(ctx, user.RoleID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 角色被停用时，不允许继续登录后台。
	if role.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("角色已被禁用")
	}

	// 查询部门信息
	var dept *models.BaseDept
	dept, err = c.baseDeptCase.FindByID(ctx, user.DeptID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 部门被停用时，不允许继续登录后台。
	if dept.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("部门已被禁用")
	}

	var baseTenant *models.BaseTenant
	baseTenant, err = c.baseTenantRepo.FindByID(ctx, user.TenantID)
	if err != nil {
		return nil, errorsx.Internal("登录失败").WithCause(err)
	}
	// 租户被停用时，不允许继续登录后台。
	if baseTenant.Status != _const.STATUS_STATUS_ENABLE {
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
func (c *LoginCase) setRefreshTokenAuth(refreshToken string, authInfo *authData.UserTokenPayload, expiresIn int64) error {
	if refreshToken == "" || authInfo == nil {
		return errorsx.Unauthenticated("刷新认证令牌失败")
	}

	payload, err := json.Marshal(authInfo)
	if err != nil {
		return errorsx.Internal("保存刷新认证信息失败").WithCause(err)
	}
	return c.Cache.Set(refreshTokenAuthKey(refreshToken), string(payload), time.Duration(expiresIn)*time.Second)
}

// getAuthInfoByRefreshToken 根据刷新令牌读取认证信息。
func (c *LoginCase) getAuthInfoByRefreshToken(refreshToken string) (*authData.UserTokenPayload, error) {
	payload, err := c.Cache.GetDel(refreshTokenAuthKey(refreshToken))
	if err != nil {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败").WithCause(err)
	}

	authInfo := &authData.UserTokenPayload{}
	err = json.Unmarshal([]byte(payload), authInfo)
	if err != nil {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败").WithCause(err)
	}

	if c.userToken.GetRefreshToken(authInfo.UserId) != refreshToken {
		return nil, errorsx.Unauthenticated("刷新认证令牌失败")
	}
	return authInfo, nil
}

// checkLoginPolicy 检查账号是否处于已启用策略定义的登录失败锁定窗口。
func (c *LoginCase) checkLoginPolicy(ctx context.Context, tenantCode, userName string, policySet loginpolicy.PolicySet, tenantID, userID int64) error {
	maxAttempts, lockWindow := policySet.FailureConfig(tenantID, userID)
	if maxAttempts <= 0 || lockWindow <= 0 {
		return nil
	}
	c.loginPolicyMu.Lock()
	defer c.loginPolicyMu.Unlock()
	var err error
	for _, key := range loginFailureKeys(tenantCode, userName, loginClientIP(ctx)) {
		_, err = c.Cache.Get(loginFailureLockKeyFromKey(key))
		if err == nil {
			return errorsx.PermissionDenied("登录失败次数过多，请稍后再试")
		}
		if !isLoginCacheMiss(err) {
			return errorsx.Internal("读取登录策略状态失败").WithCause(err)
		}
	}
	return nil
}

// recordLoginFailure 记录账号登录失败次数并在达到阈值后短暂锁定。
func (c *LoginCase) recordLoginFailure(ctx context.Context, tenantCode, userName string, policySet loginpolicy.PolicySet, tenantID, userID int64) error {
	maxAttempts, lockWindow := policySet.FailureConfig(tenantID, userID)
	if maxAttempts <= 0 || lockWindow <= 0 {
		return nil
	}
	c.loginPolicyMu.Lock()
	defer c.loginPolicyMu.Unlock()
	var err error
	for _, key := range loginFailureKeys(tenantCode, userName, loginClientIP(ctx)) {
		var attempts int64
		attempts, err = c.Cache.Incr(key)
		if err != nil {
			return err
		}
		if attempts == 1 {
			if err = c.Cache.Expire(key, lockWindow); err != nil {
				return err
			}
		}
		if attempts >= int64(maxAttempts) {
			if err = c.Cache.Set(loginFailureLockKeyFromKey(key), "1", lockWindow); err != nil {
				return err
			}
		}
	}
	return nil
}

// clearLoginFailures 清除账号成功登录后的失败计数。
func (c *LoginCase) clearLoginFailures(ctx context.Context, tenantCode, userName string) error {
	c.loginPolicyMu.Lock()
	defer c.loginPolicyMu.Unlock()
	var err error
	for _, key := range loginFailureKeys(tenantCode, userName, loginClientIP(ctx)) {
		err = c.Cache.Del(key)
		if err != nil {
			return err
		}
		err = c.Cache.Del(loginFailureLockKeyFromKey(key))
		if err != nil {
			return err
		}
	}
	return nil
}

// verifyLoginCaptcha 校验登录请求携带的验证码或行为验证码令牌。
func (c *LoginCase) verifyLoginCaptcha(ctx context.Context, captchaID, captchaCode string) error {
	driverType, ok := c.captchaDriverTypeByID(captchaID)
	// 登录阶段先通过 captcha_id 取回类型，避免依赖验证码 ID 命名格式。
	if !ok {
		return errorsx.InvalidArgument("验证码错误")
	}
	matched, err := captcha.NewCaptcha(c.Cache,
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
	value, err := c.Cache.Get(key)
	if err != nil || value != captchaID {
		return false, nil
	}
	err = c.Cache.Del(key)
	if err != nil {
		return false, errorsx.Internal("验证码令牌消费失败").WithCause(err)
	}
	return true, nil
}

// captchaDriverTypeByID 根据验证码 ID 查询生成时保存的校验驱动类型。
func (c *LoginCase) captchaDriverTypeByID(captchaID string) (captcha.DriverType, bool) {
	captchaType, err := c.Cache.Get(loginCaptchaTypeKey(captchaID))
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

// loginFailureKey 生成带租户隔离的登录失败状态缓存键。
func loginFailureKey(tenantCode, userName, clientIP string) string {
	digest := sha256.Sum256([]byte(tenantCode + "\x00" + userName + "\x00" + clientIP))
	return fmt.Sprintf("%s:%x", loginFailureKeyPrefix, digest[:])
}

// loginFailureAccountKey 生成不受来源地址影响的账号级失败状态键。
func loginFailureAccountKey(tenantCode, userName string) string {
	digest := sha256.Sum256([]byte(tenantCode + "\x00" + userName))
	return fmt.Sprintf("%s:account:%x", loginFailureKeyPrefix, digest[:])
}

// loginFailureKeys 返回账号级和来源级登录失败状态键。
func loginFailureKeys(tenantCode, userName, clientIP string) []string {
	return []string{loginFailureAccountKey(tenantCode, userName), loginFailureKey(tenantCode, userName, clientIP)}
}

// loginFailureLockKeyFromKey 生成指定失败状态的锁定键。
func loginFailureLockKeyFromKey(key string) string {
	return key + ":lock"
}

// requiresServerSession 判断角色是否属于需要后台会话生命周期控制的管理账号。
func requiresServerSession(roleCode string) bool {
	return roleCode != _const.BASE_ROLE_CODE_USER && roleCode != _const.BASE_ROLE_CODE_AUTHUSER
}

// isLoginCacheMiss 判断登录锁定键不存在，而不是缓存服务不可用。
func isLoginCacheMiss(err error) bool {
	if err == nil || errors.Is(err, redis.Nil) {
		return err != nil
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "key not found") || strings.Contains(message, "key expired") || strings.Contains(message, "not found")
}

// loginClientIP 从服务端请求上下文读取对端地址，不信任可伪造的转发请求头。
func loginClientIP(ctx context.Context) string {
	transportValue, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpValue, ok := transportValue.(*http.Transport)
	if !ok || httpValue.Request() == nil {
		return ""
	}
	clientIP, _, err := net.SplitHostPort(httpValue.Request().RemoteAddr)
	if err != nil {
		return httpValue.Request().RemoteAddr
	}
	return clientIP
}

// loginDevice 从请求头提取设备标识，优先使用客户端显式设备标识并回退到 User-Agent。
func loginDevice(ctx context.Context) string {
	transportValue, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpValue, ok := transportValue.(*http.Transport)
	if !ok || httpValue.Request() == nil {
		return ""
	}
	if deviceID := httpValue.RequestHeader().Get("X-Device-ID"); deviceID != "" {
		return deviceID
	}
	return httpValue.RequestHeader().Get("User-Agent")
}

// loginMAC 从请求头读取客户端提供的 MAC/设备硬件标识。
func loginMAC(ctx context.Context) string {
	transportValue, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpValue, ok := transportValue.(*http.Transport)
	if !ok || httpValue.Request() == nil {
		return ""
	}
	return httpValue.RequestHeader().Get("X-MAC-Address")
}

// loginRegion 从受信任网关注入的请求头读取地区编码。
func loginRegion(ctx context.Context) string {
	transportValue, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	httpValue, ok := transportValue.(*http.Transport)
	if !ok || httpValue.Request() == nil {
		return ""
	}
	return httpValue.RequestHeader().Get("X-Region-Code")
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
	dictOpts = append(dictOpts, repository.Where(dictQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	dict, err := c.baseDictRepo.Find(ctx, dictOpts...)
	if err != nil {
		return captcha.DriverDigit, err
	}

	itemQuery := c.baseDictItemRepo.Query(ctx).BaseDictItem
	itemOpts := make([]repository.QueryOption, 0, 2)
	itemOpts = append(itemOpts, repository.Where(itemQuery.DictID.Eq(dict.ID)))
	itemOpts = append(itemOpts, repository.Where(itemQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
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
