package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/event"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/app/utils"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	kitOauth "github.com/liujitcn/kratos-kit/oauth"
	"github.com/liujitcn/kratos-kit/oauth/provider"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

// CACHE_KEY_WX_ACCESS_TOKEN 表示微信访问令牌缓存键。
const CACHE_KEY_WX_ACCESS_TOKEN = "wx_access_token"

// AuthCase 处理应用端用户认证资料业务。
type AuthCase struct {
	*biz.BaseCase
	baseUserCase  *BaseUserCase
	oauthManager  *kitOauth.Manager
	userEvents    *event.UserEvents
	profileMapper *mapper.CopierMapper[
		systemappv1.UserProfileForm,
		models.BaseUser,
	]
}

// NewAuthCase 创建应用端用户认证资料业务实例。
func NewAuthCase(
	baseCase *biz.BaseCase,
	baseUserCase *BaseUserCase,
	oauthManager *kitOauth.Manager,
	userEvents *event.UserEvents,
) *AuthCase {
	return &AuthCase{
		BaseCase:     baseCase,
		baseUserCase: baseUserCase,
		oauthManager: oauthManager,
		userEvents:   userEvents,
		profileMapper: mapper.NewCopierMapper[
			systemappv1.UserProfileForm,
			models.BaseUser,
		](),
	}
}

// GetUserProfile 获取当前登录用户信息
func (c *AuthCase) GetUserProfile(ctx context.Context) (*systemappv1.UserProfileForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	// 用户被停用时，不允许继续获取个人信息。
	if user.Status != _const.STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("账号已被禁用")
	}

	res := c.profileMapper.ToDTO(user)
	res.Phone = _string.DesensitizePhone(user.Phone)
	return res, nil
}

// UpdateUserProfile 修改个人中心用户信息
func (c *AuthCase) UpdateUserProfile(ctx context.Context, req *systemappv1.UserProfileForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	originalAvatar := oldBaseUser.Avatar

	baseUser := &models.BaseUser{
		ID:       authInfo.UserId,
		NickName: req.GetNickName(),
		Gender:   req.GetGender(),
		Avatar:   req.GetAvatar(),
	}
	// 用户资料更新失败时，直接返回错误交由上层处理。
	if err = c.baseUserCase.UpdateByID(ctx, baseUser); err != nil {
		return errorsx.Internal("修改个人中心用户信息失败").WithCause(err)
	}
	// 用户资料写库成功后，通知已装配模块处理用户资料变更。
	c.userEvents.PublishUserChanged(authInfo.UserId)

	// 删除被替换的旧头像文件
	oss := sdk.Runtime.GetOSS()
	// OSS 可用时，尝试清理被替换掉的历史头像文件。
	if oss != nil {
		// 新头像为空或发生变更时，旧头像文件需要尝试删除。
		if baseUser.Avatar == "" || originalAvatar != baseUser.Avatar {
			// 头像文件删除失败时，只记录日志不影响主流程。
			if err = oss.DeleteFile(originalAvatar); err != nil {
				log.Error(fmt.Sprintf("DeleteFile %v", err))
			}
		}
	}
	return nil
}

// BindUserPhone 手机号授权
func (c *AuthCase) BindUserPhone(ctx context.Context, req *systemappv1.BindUserPhoneRequest) (*systemappv1.BindUserPhoneResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var accessToken string
	accessToken, err = sdk.Runtime.GetCache().Get(CACHE_KEY_WX_ACCESS_TOKEN)
	// 本地缓存未命中 access token 时，回源微信重新获取。
	if err != nil {
		var oauthProvider provider.OAuth
		oauthProvider, err = c.oauthManager.Get(kitOauth.WechatMini)
		if err != nil {
			return nil, errorsx.Internal("微信登录配置信息错误").WithCause(err)
		}
		var token *provider.Token
		token, err = oauthProvider.GetToken(ctx, "", provider.WithGrantType(provider.GrantTypeClientCredentials))
		// 微信 access token 获取失败时，直接返回授权失败。
		if err != nil {
			return nil, errorsx.Internal("手机号授权失败").WithCause(err)
		}
		accessToken = token.AccessToken
		cacheTTL := time.Duration(token.ExpiresIn-300) * time.Second
		// 有效期不足 300 秒时不再预留刷新缓冲，避免生成非正缓存时长。
		if token.ExpiresIn <= 300 {
			cacheTTL = time.Duration(token.ExpiresIn) * time.Second
		}
		// 新 access token 缓存失败时，只记录日志不影响主流程。
		err = sdk.Runtime.GetCache().Set(CACHE_KEY_WX_ACCESS_TOKEN, accessToken, cacheTTL)
		if err != nil {
			log.Error(fmt.Sprintf("SetWxAccessTokenCache %v", err))
		}
	}

	var phone *utils.PhoneNumber
	phone, err = utils.GetPhoneNumber(accessToken, req.GetCode())
	if err != nil {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 微信侧返回手机号授权错误时，直接返回授权失败。
	if phone.ErrCode != 0 {
		return nil, errorsx.InvalidArgument("手机号授权凭据无效").WithMetadata(map[string]string{
			"provider":      "wechat",
			"provider_code": fmt.Sprintf("%d", phone.ErrCode),
		})
	}

	var find *models.BaseUser
	find, err = c.baseUserCase.findByPhone(ctx, phone.PhoneInfo.PhoneNumber)
	// 手机号占用查询异常时，直接返回授权失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 手机号已绑定其他账号时，不允许继续授权。
	if find != nil && find.ID != authInfo.UserId {
		return nil, errorsx.Conflict("手机号已被占用").WithMetadata(map[string]string{
			errorsx.METADATA_KEY_CONFLICT_TYPE: errorsx.CONFLICT_TYPE_UNIQUE_VIOLATION,
			errorsx.METADATA_KEY_RESOURCE:      "base_user",
			errorsx.METADATA_KEY_FIELD:         "phone",
		})
	}

	user := &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: phone.PhoneInfo.PhoneNumber,
	}
	// 绑定手机号写库失败时，直接返回业务错误。
	if err = c.baseUserCase.UpdateByID(ctx, user); err != nil {
		return nil, errorsx.Internal("手机号授权失败").WithCause(err)
	}
	// 手机号绑定成功后，通知已装配模块处理用户资料变更。
	c.userEvents.PublishUserChanged(authInfo.UserId)

	return &systemappv1.BindUserPhoneResponse{
		Phone: _string.DesensitizePhone(user.Phone),
	}, nil
}
