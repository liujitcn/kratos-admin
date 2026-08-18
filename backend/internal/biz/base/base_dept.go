package biz

import (
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
)

// BaseDeptCase 处理基础部门业务。
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepository
}

// NewBaseDeptCase 创建基础部门业务实例。
func NewBaseDeptCase(baseCase *biz.BaseCase, baseDeptRepo *data.BaseDeptRepository) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:           baseCase,
		BaseDeptRepository: baseDeptRepo,
	}
}
