package biz

import (
	"context"
	"errors"

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
	user, err := c.FindByID(ctx, userID)
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
