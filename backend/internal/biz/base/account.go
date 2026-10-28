package biz

import (
	"context"
	"time"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

// AccountCase 处理当前用户账户安全业务。
type AccountCase struct {
	*biz.BaseCase
	baseUserRepo *data.BaseUserRepository
	userToken    *authData.UserToken
}

// NewAccountCase 创建当前用户账户安全业务实例。
func NewAccountCase(
	baseCase *biz.BaseCase,
	baseUserRepo *data.BaseUserRepository,
	userToken *authData.UserToken,
) *AccountCase {
	return &AccountCase{
		BaseCase:     baseCase,
		baseUserRepo: baseUserRepo,
		userToken:    userToken,
	}
}

// GetCurrentPasswordPolicy 获取当前用户生效的密码策略。
func (c *AccountCase) GetCurrentPasswordPolicy(ctx context.Context) (*basev1.CurrentPasswordPolicy, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	var baseUser *models.BaseUser
	baseUser, err = c.baseUserRepo.Find(ctx,
		repository.Select(userQuery.ID, userQuery.TenantID),
		repository.Where(userQuery.ID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	config, err := loginpolicy.LoadPasswordConfig(c.Cache, baseUser.TenantID, baseUser.ID)
	if err != nil {
		return nil, errorsx.Internal("读取密码策略失败").WithCause(err)
	}
	return &basev1.CurrentPasswordPolicy{
		MinLength:            config.MinLength,
		MinComplexityClasses: config.MinComplexityClasses,
		HistoryCount:         config.HistoryCount,
		MaxAgeDays:           config.MaxAgeDays,
	}, nil
}

// UpdateUserPassword 更新当前用户密码。
func (c *AccountCase) UpdateUserPassword(ctx context.Context, req *basev1.UserPasswordForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	userOpts := []repository.QueryOption{
		repository.Select(userQuery.ID, userQuery.TenantID, userQuery.Password, userQuery.PasswordHistory),
		repository.Where(userQuery.ID.Eq(authInfo.UserId)),
	}
	var baseUser *models.BaseUser
	baseUser, err = c.baseUserRepo.Find(ctx, userOpts...)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	passwordConfig, err := loginpolicy.LoadPasswordConfig(c.Cache, baseUser.TenantID, baseUser.ID)
	if err != nil {
		return errorsx.Internal("读取密码策略失败").WithCause(err)
	}
	oldPwd := req.GetOldPwd().GetText()
	newPwd := req.GetNewPwd().GetText()
	if err = crypto.Verify(oldPwd, baseUser.Password); err != nil {
		return errorsx.InvalidArgument("原密码错误")
	}
	if err = crypto.Verify(newPwd, baseUser.Password); err == nil {
		return errorsx.InvalidArgument("新密码不能与当前密码相同")
	}
	if err = password.CheckHistoryJSON(baseUser.PasswordHistory, newPwd, passwordConfig.HistoryCount); err != nil {
		return errorsx.InvalidArgument("新密码不能重复使用近期历史密码").WithCause(err)
	}
	if err = password.ValidateComplexity(newPwd, passwordConfig); err != nil {
		return errorsx.InvalidArgument("密码长度或复杂度不符合安全策略").WithCause(err)
	}

	var encrypted string
	encrypted, err = crypto.Encrypt(newPwd)
	if err != nil {
		return errorsx.Internal("修改密码失败").WithCause(err)
	}
	if c.userToken != nil {
		if err = c.userToken.RemoveToken(authInfo.UserId); err != nil {
			return errorsx.Internal("撤销用户登录令牌失败").WithCause(err)
		}
	}
	var history string
	history, err = password.AppendHistoryJSON(baseUser.PasswordHistory, baseUser.Password, passwordConfig.HistoryCount)
	if err != nil {
		return errorsx.Internal("记录历史密码失败").WithCause(err)
	}
	now := time.Now()
	_, err = userQuery.WithContext(ctx).
		Where(userQuery.ID.Eq(authInfo.UserId)).
		UpdateSimple(
			userQuery.Password.Value(encrypted),
			userQuery.PasswordChangedAt.Value(now),
			userQuery.PasswordHistory.Value(history),
			userQuery.MustChangePassword.Value(_const.BASE_USER_PASSWORD_CHANGE_STATUS_NOT_REQUIRED),
			userQuery.UpdatedAt.Value(now),
		)
	return err
}
