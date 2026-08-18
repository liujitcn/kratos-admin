package biz

import (
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
)

// BaseRoleCase 基础角色业务处理对象
type BaseRoleCase struct {
	*biz.BaseCase
	*data.BaseRoleRepository
}

// NewBaseRoleCase 创建基础角色业务处理对象
func NewBaseRoleCase(baseCase *biz.BaseCase, baseRoleRepo *data.BaseRoleRepository) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:           baseCase,
		BaseRoleRepository: baseRoleRepo,
	}
}
