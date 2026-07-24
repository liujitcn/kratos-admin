package biz

import (
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
)

// BaseDeptCase 处理基础部门业务。
type BaseDeptCase struct {
	*data.BaseDeptRepository
}

// NewBaseDeptCase 创建基础部门业务实例。
func NewBaseDeptCase(baseDeptRepo *data.BaseDeptRepository) *BaseDeptCase {
	return &BaseDeptCase{
		BaseDeptRepository: baseDeptRepo,
	}
}
