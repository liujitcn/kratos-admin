package biz

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	"github.com/liujitcn/gorm-kit/repository"
)

// BaseUserCase 基础用户业务处理对象
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepository
}

// NewBaseUserCase 创建基础用户业务处理对象
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepository) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		BaseUserRepository: baseUserRepo,
	}
}

// 按手机号查询用户
func (c *BaseUserCase) findByPhone(ctx context.Context, phone string) (*models.BaseUser, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Select(query.ID, query.TenantID))
	opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)), repository.Where(query.Phone.Eq(phone)))
	return c.Find(ctx, opts...)
}
