package biz

import (
	"context"
	"errors"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"gorm.io/gorm"
)

// BaseUserCase 处理基础用户业务。
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepository
}

// NewBaseUserCase 创建基础用户业务实例。
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepository) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		BaseUserRepository: baseUserRepo,
	}
}

// FindUserNameByID 按用户编号查询展示名称。
func (c *BaseUserCase) FindUserNameByID(ctx context.Context, userID int64) (string, error) {
	query := c.Query(ctx).BaseUser
	user, err := c.Find(ctx,
		repository.Select(query.ID, query.UserName, query.NickName),
		repository.Where(query.ID.Eq(userID)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errorsx.ResourceNotFound("用户不存在")
		}
		return "", err
	}
	if user.NickName != "" {
		return user.NickName, nil
	}
	return user.UserName, nil
}
