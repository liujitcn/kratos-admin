package biz

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"hash/fnv"
	"net"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/event"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/go-utils/id"
	kitOauth "github.com/liujitcn/kratos-kit/oauth"
	"github.com/liujitcn/kratos-kit/oauth/provider"
	"gorm.io/gorm"
)

const oauthSceneAdminLogin = "admin_login"
const oauthSceneAdminBind = "admin_bind"
const oauthLoginTicketKeyPrefix = "oauth_login_ticket"
const oauthLoginTicketExpire = 2 * time.Minute
const oauthLoginTicketLockShardCount = 64

var oauthLoginTicketLocks [oauthLoginTicketLockShardCount]sync.Mutex

// OauthCase 处理三方登录授权业务。
type OauthCase struct {
	*biz.BaseCase
	tx                   data.Transaction
	oauthManager         *kitOauth.Manager
	baseThirdAccountCase *BaseThirdAccountCase
	baseUserCase         *BaseUserCase
	baseRoleCase         *BaseRoleCase
	baseDeptCase         *BaseDeptCase
	loginCase            *LoginCase
	configCase           *ConfigCase
	userEvents           *event.UserEvents
}

// oauthLoginTicketPayload 表示三方登录一次性票据缓存的令牌信息。
type oauthLoginTicketPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// NewOauthCase 创建三方登录授权业务实例。
func NewOauthCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	oauthManager *kitOauth.Manager,
	baseThirdAccountCase *BaseThirdAccountCase,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	loginCase *LoginCase,
	configCase *ConfigCase,
	userEvents *event.UserEvents,
) *OauthCase {
	return &OauthCase{
		BaseCase:             baseCase,
		tx:                   tx,
		oauthManager:         oauthManager,
		baseThirdAccountCase: baseThirdAccountCase,
		baseUserCase:         baseUserCase,
		baseRoleCase:         baseRoleCase,
		baseDeptCase:         baseDeptCase,
		loginCase:            loginCase,
		configCase:           configCase,
		userEvents:           userEvents,
	}
}

// ListOauthBinding 查询当前用户的三方账号绑定状态。
func (c *OauthCase) ListOauthBinding(ctx context.Context, req *basev1.ListOauthBindingRequest) (*basev1.ListOauthBindingResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var providerRes *basev1.ListOauthProviderResponse
	providerRes, err = c.ListOauthProvider(ctx, nil)
	if err != nil {
		return nil, err
	}

	var thirdAccounts []*models.BaseThirdAccount
	thirdAccounts, err = c.baseThirdAccountCase.ListByUserID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.Internal("查询三方账号绑定失败").WithCause(err)
	}
	boundProviderSet := make(map[string]struct{}, len(thirdAccounts))
	for _, item := range thirdAccounts {
		boundProviderSet[item.Provider] = struct{}{}
	}

	bindings := make([]*basev1.OauthBinding, 0, len(providerRes.GetProviders()))
	for _, item := range providerRes.GetProviders() {
		_, bound := boundProviderSet[item.GetProvider()]
		bindings = append(bindings, &basev1.OauthBinding{
			Provider: item.GetProvider(),
			Bound:    bound,
		})
	}
	return &basev1.ListOauthBindingResponse{Bindings: bindings}, nil
}

// ListOauthProvider 查询可用于管理端展示的三方登录方式。
func (c *OauthCase) ListOauthProvider(ctx context.Context, req *basev1.ListOauthProviderRequest) (*basev1.ListOauthProviderResponse, error) {
	providerNames := c.oauthManager.Providers()
	providers := make([]*basev1.OauthProvider, 0, len(providerNames))
	for _, providerName := range providerNames {
		providers = append(providers, &basev1.OauthProvider{
			Provider: string(providerName),
		})
	}
	return &basev1.ListOauthProviderResponse{Providers: providers}, nil
}

// CreateOauthAuthorization 创建三方登录授权地址。
func (c *OauthCase) CreateOauthAuthorization(ctx context.Context, req *basev1.CreateOauthAuthorizationRequest) (*basev1.CreateOauthAuthorizationResponse, error) {
	var err error
	var redirectURL string
	redirectURL, err = normalizeOauthLoginURL(ctx, req.GetRedirectUrl())
	if err != nil {
		return nil, err
	}

	oauthType := kitOauth.Type(req.GetProvider())
	var oauthProvider provider.OAuth
	oauthProvider, err = c.oauthManager.Get(oauthType)
	if err != nil {
		return nil, errorsx.InvalidArgument("登录方式不支持").WithCause(err)
	}
	var state string
	var pkce provider.PKCEChallenge
	state, pkce, err = kitOauth.NewStateWithPKCE(c.Cache, kitOauth.StatePayload{
		Provider:    oauthType,
		Scene:       oauthSceneAdminLogin,
		RedirectURL: redirectURL,
	}, 0)
	if err != nil {
		return nil, errorsx.Internal("创建三方登录授权失败").WithCause(err)
	}
	authorizationURL := oauthProvider.AuthURL(state, provider.WithPKCE(pkce))
	if authorizationURL == "" {
		return nil, errorsx.InvalidArgument("登录方式不支持跳转授权")
	}
	return &basev1.CreateOauthAuthorizationResponse{AuthorizationUrl: authorizationURL}, nil
}

// CreateOauthBindingAuthorization 创建个人中心三方账号绑定授权地址。
func (c *OauthCase) CreateOauthBindingAuthorization(ctx context.Context, req *basev1.CreateOauthBindingAuthorizationRequest) (*basev1.CreateOauthBindingAuthorizationResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	if authInfo.UserId <= 0 {
		return nil, errorsx.Unauthenticated("用户认证失败")
	}
	var safeRedirectURL string
	safeRedirectURL, err = normalizeOauthReturnURL(ctx, req.GetRedirectUrl(), true)
	if err != nil {
		return nil, err
	}

	oauthType := kitOauth.Type(req.GetProvider())
	var oauthProvider provider.OAuth
	oauthProvider, err = c.oauthManager.Get(oauthType)
	if err != nil {
		return nil, errorsx.InvalidArgument("登录方式不支持").WithCause(err)
	}
	var state string
	var pkce provider.PKCEChallenge
	state, pkce, err = kitOauth.NewStateWithPKCE(c.Cache, kitOauth.StatePayload{
		Provider:    oauthType,
		Scene:       oauthSceneAdminBind,
		RedirectURL: safeRedirectURL,
		Extra: map[string]string{
			"user_id": strconv.FormatInt(authInfo.UserId, 10),
		},
	}, 0)
	if err != nil {
		return nil, errorsx.Internal("创建三方账号绑定授权失败").WithCause(err)
	}
	authorizationURL := oauthProvider.AuthURL(state, provider.WithPKCE(pkce))
	if authorizationURL == "" {
		return nil, errorsx.InvalidArgument("登录方式不支持跳转授权")
	}
	return &basev1.CreateOauthBindingAuthorizationResponse{AuthorizationUrl: authorizationURL}, nil
}

// CreateOauthSession 使用非跳转型 OAuth 授权码创建登录会话。
func (c *OauthCase) CreateOauthSession(ctx context.Context, req *basev1.CreateOauthSessionRequest) (*basev1.CreateOauthSessionResponse, error) {
	// 当前非跳转登录只开放微信小程序，其他 Provider 继续走授权地址回调流程。
	if req.GetProvider() != string(kitOauth.WechatMini) {
		return nil, errorsx.InvalidArgument("登录方式不支持")
	}
	autoRegister, err := c.oauthAutoRegisterEnabled(ctx)
	if err != nil {
		return nil, err
	}
	var openID string
	openID, err = c.getWechatMiniOpenID(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}
	var user *models.BaseUser
	user, err = c.findWechatMiniUserByOpenID(ctx, openID)
	if err != nil {
		if !autoRegister && kratosErrors.Code(err) == 401 {
			return &basev1.CreateOauthSessionResponse{BindingRequired: true}, nil
		}
		if autoRegister && kratosErrors.Code(err) == 401 {
			user, err = c.createWechatMiniUser(ctx, openID)
		}
	}
	if err != nil {
		return nil, err
	}
	var loginRes *basev1.LoginResponse
	loginRes, err = c.loginCase.IssueUserToken(ctx, user)
	if err != nil {
		return nil, err
	}
	c.userEvents.PublishUserChanged(user.ID)
	return &basev1.CreateOauthSessionResponse{
		AccessToken:  loginRes.GetAccessToken(),
		RefreshToken: loginRes.GetRefreshToken(),
		TokenType:    loginRes.GetTokenType(),
		ExpiresIn:    loginRes.GetExpiresIn(),
	}, nil
}

// oauthAutoRegisterEnabled 读取微信未绑定时是否允许自动注册。
func (c *OauthCase) oauthAutoRegisterEnabled(ctx context.Context) (bool, error) {
	config, err := c.configCase.GetConfig(ctx, &basev1.GetConfigRequest{Site: basev1.BaseConfigSite_APP})
	if err != nil {
		return false, errorsx.Internal("读取微信登录配置失败").WithCause(err)
	}
	for _, item := range config.GetConfigs() {
		if item.GetKey() == _const.BASE_CONFIG_KEY_OAUTH_AUTO_REGISTER {
			return strings.EqualFold(item.GetValue(), "true"), nil
		}
	}
	return false, nil
}

// BindOauthSession 校验已有账号并绑定微信小程序账号后创建登录会话。
func (c *OauthCase) BindOauthSession(ctx context.Context, req *basev1.BindOauthSessionRequest) (*basev1.CreateOauthSessionResponse, error) {
	if req.GetProvider() != string(kitOauth.WechatMini) {
		return nil, errorsx.InvalidArgument("登录方式不支持")
	}
	var err error
	err = c.loginCase.verifyLoginCaptcha(ctx, req.GetCaptchaId(), req.GetCaptchaCode())
	if err != nil {
		return nil, err
	}
	var openID string
	openID, err = c.getWechatMiniOpenID(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}
	_, err = c.baseThirdAccountCase.FindByProviderIdentifier(ctx, string(kitOauth.WechatMini), openID)
	if err == nil {
		return nil, errorsx.Conflict("微信账号已绑定")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.Internal("微信登录失败").WithCause(err)
	}

	user, err := c.loginCase.FindUserByPassword(ctx, req.GetTenantCode(), req.GetUserName(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	err = c.baseThirdAccountCase.CreateBinding(ctx, user.ID, string(kitOauth.WechatMini), openID)
	if err != nil {
		return nil, err
	}
	var loginRes *basev1.LoginResponse
	loginRes, err = c.loginCase.IssueUserToken(ctx, user)
	if err != nil {
		return nil, err
	}
	c.userEvents.PublishUserChanged(user.ID)
	return &basev1.CreateOauthSessionResponse{
		AccessToken: loginRes.GetAccessToken(), RefreshToken: loginRes.GetRefreshToken(),
		TokenType: loginRes.GetTokenType(), ExpiresIn: loginRes.GetExpiresIn(),
	}, nil
}

// HandleOauthCallback 处理三方登录回调并跳回管理端登录页。
func (c *OauthCase) HandleOauthCallback(ctx context.Context, req *basev1.HandleOauthCallbackRequest) (*basev1.HandleOauthCallbackResponse, error) {
	payload, err := kitOauth.VerifyState(c.Cache, req.GetState())
	if err != nil {
		return nil, errorsx.InvalidArgument("三方登录状态已失效")
	}
	if payload.Scene == oauthSceneAdminBind {
		return nil, c.handleOauthBindingCallback(ctx, payload, req.GetProvider(), req.GetCode(), req.GetError())
	}
	if payload.Scene != oauthSceneAdminLogin {
		return nil, c.oauthRedirectPayload(payload, "", "三方登录状态无效")
	}

	oauthType := kitOauth.Type(req.GetProvider())
	if payload.Provider != oauthType {
		return nil, c.oauthRedirectPayload(payload, "", "三方登录状态无效")
	}
	if req.GetError() != "" {
		return nil, c.oauthRedirectPayload(payload, "", "三方授权失败")
	}
	var identifier string
	identifier, err = c.fetchOauthIdentifier(ctx, oauthType, req.GetCode(), payload.PKCE)
	if err != nil {
		return nil, c.oauthRedirectPayload(payload, "", kratosErrors.FromError(err).Message)
	}

	var thirdAccount *models.BaseThirdAccount
	thirdAccount, err = c.baseThirdAccountCase.FindByProviderIdentifier(ctx, req.GetProvider(), identifier)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, c.oauthRedirectPayload(payload, "", "三方账号未绑定，请先使用账号密码登录后到个人中心绑定")
		}
		return nil, c.oauthRedirectPayload(payload, "", "三方账号登录失败")
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, thirdAccount.UserID)
	if err != nil {
		return nil, c.oauthRedirectPayload(payload, "", "三方账号登录失败")
	}
	var loginRes *basev1.LoginResponse
	loginRes, err = c.loginCase.IssueUserToken(ctx, user)
	if err != nil {
		return nil, c.oauthRedirectPayload(payload, "", kratosErrors.FromError(err).Message)
	}

	var ticket string
	ticket, err = c.createOauthLoginTicket(loginRes)
	if err != nil {
		return nil, c.oauthRedirectPayload(payload, "", "三方账号登录失败")
	}
	return nil, c.oauthRedirectPayload(payload, ticket, "")
}

// HandleOauthBindingCallback 处理个人中心三方账号绑定回调。
func (c *OauthCase) HandleOauthBindingCallback(ctx context.Context, req *basev1.HandleOauthBindingCallbackRequest) error {
	payload, err := kitOauth.VerifyState(c.Cache, req.GetState())
	if err != nil {
		return errorsx.InvalidArgument("三方账号绑定状态已失效")
	}
	return c.handleOauthBindingCallback(ctx, payload, req.GetProvider(), req.GetCode(), req.GetError())
}

// ExchangeOauthTicket 兑换三方登录一次性票据。
func (c *OauthCase) ExchangeOauthTicket(ctx context.Context, req *basev1.ExchangeOauthTicketRequest) (*basev1.ExchangeOauthTicketResponse, error) {
	value, err := c.consumeOauthLoginTicket(req.GetTicket())
	if err != nil {
		return nil, err
	}

	var payload oauthLoginTicketPayload
	err = json.Unmarshal([]byte(value), &payload)
	if err != nil {
		return nil, errorsx.Unauthenticated("三方登录票据无效").WithCause(err)
	}
	return &basev1.ExchangeOauthTicketResponse{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		ExpiresIn:    payload.ExpiresIn,
	}, nil
}

// UnbindOauthAccount 解绑当前用户三方账号。
func (c *OauthCase) UnbindOauthAccount(ctx context.Context, req *basev1.UnbindOauthAccountRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	_, err = c.baseThirdAccountCase.FindByUserProvider(ctx, authInfo.UserId, req.GetProvider())
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("三方账号未绑定")
		}
		return errorsx.Internal("解绑三方账号失败").WithCause(err)
	}
	err = c.baseThirdAccountCase.DeleteByUserProvider(ctx, authInfo.UserId, req.GetProvider())
	if err != nil {
		return errorsx.Internal("解绑三方账号失败").WithCause(err)
	}
	return nil
}

// findWechatMiniUserByOpenID 按微信小程序唯一标识查找已绑定的本地用户。
func (c *OauthCase) findWechatMiniUserByOpenID(ctx context.Context, openID string) (*models.BaseUser, error) {
	var err error
	var thirdAccount *models.BaseThirdAccount
	thirdAccount, err = c.baseThirdAccountCase.FindByProviderIdentifier(ctx, string(kitOauth.WechatMini), openID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.Unauthenticated("微信账号未绑定，请先绑定已有账号")
		}
		return nil, errorsx.Internal("微信登录失败").WithCause(err)
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, thirdAccount.UserID)
	if err != nil {
		return nil, errorsx.Internal("微信登录失败").WithCause(err)
	}
	return user, nil
}

// createWechatMiniUser 按默认角色和部门自动创建微信小程序用户。
func (c *OauthCase) createWechatMiniUser(ctx context.Context, openID string) (*models.BaseUser, error) {
	defaultRole, err := c.baseRoleCase.FindDefaultUser(ctx)
	if err != nil {
		return nil, errorsx.Internal("微信登录默认角色配置错误").WithCause(err)
	}
	var defaultDept *models.BaseDept
	defaultDept, err = c.baseDeptCase.FindByID(ctx, _const.BASE_DEPT_ID_APP_USER)
	if err != nil {
		return nil, errorsx.Internal("微信登录默认部门配置错误").WithCause(err)
	}
	if defaultDept.TenantID != defaultRole.TenantID {
		return nil, errorsx.Internal("微信登录默认部门配置错误")
	}
	userCode := id.NewXID()
	user := &models.BaseUser{
		TenantID: defaultRole.TenantID,
		UserName: userCode,
		UserCode: userCode,
		RoleID:   defaultRole.ID,
		DeptID:   defaultDept.ID,
		Gender:   _const.BASE_USER_GENDER_SECRET,
		Status:   _const.STATUS_ENABLE,
		Remark:   "自动注册用户",
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.baseUserCase.Create(txCtx, user)
		if err != nil {
			return errorsx.Internal("微信登录失败").WithCause(err)
		}
		err = c.baseThirdAccountCase.CreateBinding(txCtx, user.ID, string(kitOauth.WechatMini), openID)
		if err != nil {
			return errorsx.Internal("微信登录失败").WithCause(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

// getWechatMiniOpenID 使用授权码获取微信小程序唯一标识。
func (c *OauthCase) getWechatMiniOpenID(ctx context.Context, code string) (string, error) {
	var err error
	var wechatMiniProvider provider.OAuth
	wechatMiniProvider, err = c.oauthManager.Get(kitOauth.WechatMini)
	if err != nil {
		return "", errorsx.Internal("微信登录配置信息错误").WithCause(err)
	}
	var oauthToken *provider.Token
	oauthToken, err = wechatMiniProvider.GetToken(ctx, code, provider.WithGrantType(provider.GrantTypeAuthorizationCode))
	if err != nil {
		return "", errorsx.InvalidArgument("微信登录凭据无效").WithCause(err)
	}
	var oauthUser *provider.User
	oauthUser, err = wechatMiniProvider.GetUser(ctx, oauthToken)
	if err != nil {
		return "", errorsx.InvalidArgument("获取微信用户失败").WithCause(err)
	}
	if oauthUser.OpenID == "" {
		return "", errorsx.Internal("登录失败")
	}
	return oauthUser.OpenID, nil
}

// handleOauthBindingCallback 校验三方账号并写入当前用户绑定关系。
func (c *OauthCase) handleOauthBindingCallback(ctx context.Context, payload *kitOauth.StatePayload, providerName string, code string, providerError string) error {
	if payload.Scene != oauthSceneAdminBind {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号绑定状态无效")
	}

	oauthType := kitOauth.Type(providerName)
	if payload.Provider != oauthType {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号绑定状态无效")
	}
	if providerError != "" {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方授权失败")
	}
	if code == "" {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方授权码不能为空")
	}

	userID, err := strconv.ParseInt(payload.Extra["user_id"], 10, 64)
	if err != nil || userID <= 0 {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号绑定状态无效")
	}

	var identifier string
	identifier, err = c.fetchOauthIdentifier(ctx, oauthType, code, payload.PKCE)
	if err != nil {
		return c.oauthBindingRedirectPayload(payload, providerName, kratosErrors.FromError(err).Message)
	}

	var boundAccount *models.BaseThirdAccount
	boundAccount, err = c.baseThirdAccountCase.FindByProviderIdentifier(ctx, providerName, identifier)
	if err == nil {
		// 已经绑定到当前用户时，直接视为成功，避免重复回调造成误报。
		if boundAccount.UserID == userID {
			return c.oauthBindingRedirectPayload(payload, providerName, "")
		}
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号已被其他用户绑定")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号绑定失败")
	}

	var userProviderAccount *models.BaseThirdAccount
	userProviderAccount, err = c.baseThirdAccountCase.FindByUserProvider(ctx, userID, providerName)
	if err == nil {
		// 同一用户同一 provider 只保留一条绑定，避免登录入口出现歧义。
		if userProviderAccount.Identifier == identifier {
			return c.oauthBindingRedirectPayload(payload, providerName, "")
		}
		return c.oauthBindingRedirectPayload(payload, providerName, "当前用户已绑定该登录方式")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return c.oauthBindingRedirectPayload(payload, providerName, "三方账号绑定失败")
	}

	err = c.baseThirdAccountCase.CreateBinding(ctx, userID, providerName, identifier)
	if err != nil {
		return c.oauthBindingRedirectPayload(payload, providerName, kratosErrors.FromError(err).Message)
	}
	return c.oauthBindingRedirectPayload(payload, providerName, "")
}

// fetchOauthIdentifier 通过授权码读取三方账号唯一标识。
func (c *OauthCase) fetchOauthIdentifier(ctx context.Context, oauthType kitOauth.Type, code string, pkce provider.PKCEChallenge) (string, error) {
	oauthProvider, err := c.oauthManager.Get(oauthType)
	if err != nil {
		return "", errorsx.InvalidArgument("登录方式不支持").WithCause(err)
	}
	var oauthToken *provider.Token
	oauthToken, err = oauthProvider.GetToken(ctx, code, provider.WithGrantType(provider.GrantTypeAuthorizationCode), provider.WithPKCE(pkce))
	if err != nil {
		return "", errorsx.InvalidArgument("三方授权失败").WithCause(err)
	}
	var oauthUser *provider.User
	oauthUser, err = oauthProvider.GetUser(ctx, oauthToken)
	if err != nil {
		return "", errorsx.InvalidArgument("获取三方用户失败").WithCause(err)
	}
	if oauthUser.OpenID == "" {
		return "", errorsx.InvalidArgument("三方账号唯一标识为空")
	}
	return oauthUser.OpenID, nil
}

// createOauthLoginTicket 缓存三方登录结果并返回一次性票据。
func (c *OauthCase) createOauthLoginTicket(loginRes *basev1.LoginResponse) (string, error) {
	payload := oauthLoginTicketPayload{
		AccessToken:  loginRes.GetAccessToken(),
		RefreshToken: loginRes.GetRefreshToken(),
		TokenType:    loginRes.GetTokenType(),
		ExpiresIn:    loginRes.GetExpiresIn(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", errorsx.Internal("三方登录票据创建失败").WithCause(err)
	}
	ticket := id.NewGUIDv7NoHyphen()
	err = c.Cache.Set(oauthLoginTicketKey(ticket), string(body), oauthLoginTicketExpire)
	if err != nil {
		return "", errorsx.Internal("三方登录票据创建失败").WithCause(err)
	}
	return ticket, nil
}

// consumeOauthLoginTicket 串行消费三方登录一次性票据，避免同一票据被并发重复兑换。
func (c *OauthCase) consumeOauthLoginTicket(ticket string) (string, error) {
	lock := oauthLoginTicketLock(ticket)
	lock.Lock()
	defer lock.Unlock()

	cacheKey := oauthLoginTicketKey(ticket)
	value, err := c.Cache.Get(cacheKey)
	if err != nil {
		return "", errorsx.Unauthenticated("三方登录票据已失效").WithCause(err)
	}
	err = c.Cache.Del(cacheKey)
	if err != nil {
		return "", errorsx.Internal("三方登录票据消费失败").WithCause(err)
	}
	return value, nil
}

// oauthRedirectPayload 构造回跳管理端登录页的重定向响应。
func (c *OauthCase) oauthRedirectPayload(payload *kitOauth.StatePayload, ticket string, errorMessage string) error {
	if payload.RedirectURL == "" {
		return errorsx.InvalidArgument(errorMessage)
	}

	redirectURL := appendOauthRedirectQuery(payload.RedirectURL, ticket, errorMessage)
	return kratosHTTP.NewRedirect(redirectURL, stdhttp.StatusFound)
}

// oauthBindingRedirectPayload 构造回跳个人中心的三方账号绑定响应。
func (c *OauthCase) oauthBindingRedirectPayload(payload *kitOauth.StatePayload, providerName string, errorMessage string) error {
	if payload.RedirectURL == "" {
		return errorsx.InvalidArgument(errorMessage)
	}

	redirectURL := appendOauthBindingRedirectQuery(payload.RedirectURL, providerName, errorMessage)
	return kratosHTTP.NewRedirect(redirectURL, stdhttp.StatusFound)
}

// oauthLoginTicketLock 返回三方登录票据消费分片锁。
func oauthLoginTicketLock(ticket string) *sync.Mutex {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(ticket))
	return &oauthLoginTicketLocks[hash.Sum32()%oauthLoginTicketLockShardCount]
}

// appendOauthRedirectQuery 为登录页地址追加 OAuth 登录结果参数。
func appendOauthRedirectQuery(redirectURL string, ticket string, errorMessage string) string {
	return appendOauthQueryToURL(redirectURL, func(query url.Values) {
		if ticket != "" {
			query.Set("oauth_ticket", ticket)
		}
		if errorMessage != "" {
			query.Set("oauth_error", errorMessage)
		}
	})
}

// appendOauthBindingRedirectQuery 为个人中心地址追加 OAuth 绑定结果参数。
func appendOauthBindingRedirectQuery(redirectURL string, providerName string, errorMessage string) string {
	return appendOauthQueryToURL(redirectURL, func(query url.Values) {
		if providerName != "" {
			query.Set("oauth_bind_provider", providerName)
		}
		if errorMessage != "" {
			query.Set("oauth_bind_error", errorMessage)
			return
		}
		query.Set("oauth_bind_success", "1")
	})
}

// appendOauthQueryToURL 兼容普通 URL 与 Hash 路由地址追加 OAuth 结果参数。
func appendOauthQueryToURL(targetURL string, apply func(url.Values)) string {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}
	if parsedURL.Fragment != "" {
		fragmentPath, fragmentQuery, _ := strings.Cut(parsedURL.Fragment, "?")
		query, parseErr := url.ParseQuery(fragmentQuery)
		if parseErr != nil {
			query = url.Values{}
		}
		apply(query)
		parsedURL.Fragment = ""
		baseURL := parsedURL.String()
		if encodedQuery := query.Encode(); encodedQuery != "" {
			return baseURL + "#" + fragmentPath + "?" + encodedQuery
		}
		return baseURL + "#" + fragmentPath
	}
	query := parsedURL.Query()
	apply(query)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

// oauthLoginTicketKey 生成三方登录一次性票据缓存键。
func oauthLoginTicketKey(ticket string) string {
	return fmt.Sprintf("%s:%s", oauthLoginTicketKeyPrefix, ticket)
}

// normalizeOauthLoginURL 校验并规范化 OAuth 登录页回跳地址，避免票据被重定向到外部站点。
func normalizeOauthLoginURL(ctx context.Context, rawURL string) (string, error) {
	loginURL, err := normalizeOauthReturnURL(ctx, rawURL, true)
	if err != nil {
		return "", err
	}
	var parsedURL *url.URL
	parsedURL, err = url.Parse(loginURL)
	if err != nil {
		return "", errorsx.InvalidArgument("登录页地址无效").WithCause(err)
	}
	// 登录票据只能回到管理端登录路由，避免同源下其他页面误消费或泄露票据。
	if parsedURL.Path != "/login" && !strings.HasSuffix(parsedURL.Path, "/login") && !strings.HasPrefix(parsedURL.Fragment, "/login") {
		return "", errorsx.InvalidArgument("登录页地址无效")
	}
	return loginURL, nil
}

// normalizeOauthReturnURL 校验并规范化 OAuth 回跳地址，只允许相对地址、当前服务同 Host 地址或本地开发地址。
func normalizeOauthReturnURL(ctx context.Context, rawURL string, required bool) (string, error) {
	if rawURL == "" {
		if required {
			return "", errorsx.InvalidArgument("回跳地址不能为空")
		}
		return "", nil
	}
	var err error
	var parsedURL *url.URL
	parsedURL, err = url.Parse(rawURL)
	if err != nil {
		return "", errorsx.InvalidArgument("回跳地址无效").WithCause(err)
	}
	if parsedURL.IsAbs() {
		if !isAllowedOauthAbsoluteURL(ctx, parsedURL) {
			return "", errorsx.InvalidArgument("回跳地址无效")
		}
		return parsedURL.String(), nil
	}
	// 禁止 //example.com 这类 scheme-relative URL 伪装成相对地址。
	if parsedURL.Host != "" || !strings.HasPrefix(parsedURL.Path, "/") {
		return "", errorsx.InvalidArgument("回跳地址无效")
	}
	return parsedURL.String(), nil
}

// isAllowedOauthAbsoluteURL 判断绝对回跳地址是否属于当前站点或本地开发地址。
func isAllowedOauthAbsoluteURL(ctx context.Context, targetURL *url.URL) bool {
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return false
	}
	request, ok := kratosHTTP.RequestFromServerContext(ctx)
	if !ok || request == nil {
		return false
	}
	// 生产环境只允许完整 origin 一致，避免 OAuth 票据被回跳到同域名的其他端口或协议。
	if oauthOrigin(targetURL.Scheme, targetURL.Host) == oauthOrigin(oauthRequestScheme(request), request.Host) {
		return true
	}
	targetHost := normalizedHostname(targetURL.Host)
	requestHost := normalizedHostname(request.Host)
	// 本地开发常见前后端不同端口，仅对 loopback 地址放宽端口限制。
	return isLoopbackHost(requestHost) && isLoopbackHost(targetHost)
}

// oauthOrigin 规范化 OAuth 回跳地址比较使用的源。
func oauthOrigin(scheme string, host string) string {
	scheme = strings.ToLower(scheme)
	parsedURL := &url.URL{Scheme: scheme, Host: host}
	hostname := strings.ToLower(parsedURL.Hostname())
	if scheme == "" || hostname == "" {
		return ""
	}
	port := parsedURL.Port()
	if port == "" {
		port = defaultOriginPort(scheme)
	}
	if port == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s:%s", scheme, hostname, port)
}

// oauthRequestScheme 获取当前请求的访问协议。
func oauthRequestScheme(request *stdhttp.Request) string {
	forwardedProto := strings.ToLower(strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0]))
	if forwardedProto == "http" || forwardedProto == "https" {
		return forwardedProto
	}
	if request.TLS != nil {
		return "https"
	}
	return "http"
}

// defaultOriginPort 返回 HTTP Origin 比较使用的默认端口。
func defaultOriginPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

// normalizedHostname 解析并统一 URL Host 中的主机名。
func normalizedHostname(host string) string {
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}
	return strings.ToLower(strings.Trim(hostname, "[]"))
}

// isLoopbackHost 判断主机名是否为本地开发地址。
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
