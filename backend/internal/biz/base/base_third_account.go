package biz

import (
	"context"
	"errors"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

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
func (c *BaseThirdAccountCase) ListByUserID(ctx context.Context, userID int64) ([]*models.BaseThirdAccount, error) {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.List(ctx, opts...)
}

// CreateBinding 创建三方账号绑定关系。
func (c *BaseThirdAccountCase) CreateBinding(ctx context.Context, userID int64, provider string, identifier string) error {
	err := c.Create(ctx, &models.BaseThirdAccount{
		UserID:     userID,
		Provider:   provider,
		Identifier: identifier,
	})
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			message := "三方账号绑定关系已存在"
			constraint := ""
			if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok {
				// 根据数据库实际命中的唯一索引返回对应的绑定关系描述。
				switch {
				case strings.Contains(mysqlErr.Message, "unique_base_third_account_user"):
					message = "当前用户已绑定该登录方式"
					constraint = "unique_base_third_account_user"
				case strings.Contains(mysqlErr.Message, "unique_base_third_account"):
					message = "三方账号已被其他用户绑定"
					constraint = "unique_base_third_account"
				}
			}
			return errorsx.UniqueConflict(message, "base_third_account", "", constraint).WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteByUserProvider 删除指定用户的三方账号绑定关系。
func (c *BaseThirdAccountCase) DeleteByUserProvider(ctx context.Context, userID int64, provider string) error {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
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
func (c *BaseThirdAccountCase) FindByUserProvider(ctx context.Context, userID int64, provider string) (*models.BaseThirdAccount, error) {
	query := c.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Provider.Eq(provider)))
	return c.Find(ctx, opts...)
}
