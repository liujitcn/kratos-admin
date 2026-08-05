package biz

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
)

// BaseDeptCase 基础部门业务处理对象
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepository
}

// NewBaseDeptCase 创建基础部门业务处理对象
func NewBaseDeptCase(baseCase *biz.BaseCase, baseDeptRepo *data.BaseDeptRepository) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:           baseCase,
		BaseDeptRepository: baseDeptRepo,
	}
}
