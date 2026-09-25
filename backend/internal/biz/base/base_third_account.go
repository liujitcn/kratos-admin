package biz

import (
	"context"
	"errors"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"

	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseThirdAccountCase 处理用户三方登录账号绑定业务。
type BaseThirdAccountCase struct {
	*biz.BaseCase
	*data.BaseThirdAccountRepository
}

// NewBaseThirdAccountCase 创建用户三方登录账号绑定业务实例。
func NewBaseThirdAccountCase(baseCase *biz.BaseCase, baseThirdAccountRepo *data.BaseThirdAccountRepository) *BaseThirdAccountCase {
	return &BaseThirdAccountCase{
		BaseCase:                   baseCase,
		BaseThirdAccountRepository: baseThirdAccountRepo,
	}
}

// ListByUserID 查询指定用户已绑定的三方账号。
func (c *BaseThirdAccountCase) ListByUserID(ctx context.Context, tenantID int64, userID int64) ([]*models.BaseThirdAccount, error) {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.List(ctx, opts...)
}

// CreateBinding 创建三方账号绑定关系。
func (c *BaseThirdAccountCase) CreateBinding(ctx context.Context, tenantID int64, userID int64, provider string, identifier string) error {
	err := c.Create(ctx, &models.BaseThirdAccount{
		TenantID:   tenantID,
		UserID:     userID,
		Provider:   provider,
		Identifier: identifier,
	})
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return thirdAccountUniqueConflict(err)
		}
		return err
	}
	return nil
}

// DeleteByUserProvider 删除指定用户的三方账号绑定关系。
func (c *BaseThirdAccountCase) DeleteByUserProvider(ctx context.Context, tenantID int64, userID int64, provider string) error {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Provider.Eq(provider)))
	return c.Delete(ctx, opts...)
}

// FindByProviderIdentifier 按三方登录方式与唯一标识查询绑定关系。
func (c *BaseThirdAccountCase) FindByProviderIdentifier(ctx context.Context, provider string, identifier string) (*models.BaseThirdAccount, error) {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Provider.Eq(provider)))
	opts = append(opts, repository.Where(query.Identifier.Eq(identifier)))
	return c.Find(ctx, opts...)
}

// FindByUserProvider 按用户与三方登录方式查询绑定关系。
func (c *BaseThirdAccountCase) FindByUserProvider(ctx context.Context, tenantID int64, userID int64, provider string) (*models.BaseThirdAccount, error) {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Provider.Eq(provider)))
	return c.Find(ctx, opts...)
}

// FindAuthorizedUserProvider 按当前身份可访问的租户范围查询用户三方账号。
func (c *BaseThirdAccountCase) FindAuthorizedUserProvider(ctx context.Context, userID int64, provider string) (*models.BaseThirdAccount, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode {
		return c.FindByUserProvider(ctx, authInfo.TenantId, userID, provider)
	}
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Provider.Eq(provider)))
	return c.Find(ctx, opts...)
}

// thirdAccountUniqueConflict 根据命中的唯一索引构造带稳定翻译键的绑定冲突错误。
func thirdAccountUniqueConflict(err error) error {
	message := "三方账号绑定关系已存在"
	messageKey := "base.third_account.error.binding_exists"
	constraint := ""
	field := "provider,identifier"
	if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok {
		// 根据数据库实际命中的唯一索引返回对应的绑定关系描述。
		switch {
		case strings.Contains(mysqlErr.Message, "unique_base_third_account_user"):
			message = "当前用户已绑定该登录方式"
			messageKey = "base.third_account.error.current_user_provider_bound"
			constraint = "unique_base_third_account_user"
			field = "tenant_id,user_id,provider"
		case strings.Contains(mysqlErr.Message, "unique_base_third_account"):
			message = "三方账号已被其他用户绑定"
			messageKey = "base.third_account.error.account_bound_to_other_user"
			constraint = "unique_base_third_account"
		}
	}
	return errorsx.WithMessageKey(errorsx.UniqueConflict(message, "base_third_account", field, constraint), messageKey, nil).WithCause(err)
}
