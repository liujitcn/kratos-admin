package biz

import (
	"context"

	query "github.com/liujitcn/kratos-admin/backend/internal/data/gen/query"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// findDefaultTenantID 查询默认租户编号，系统工作流动作不依赖请求身份的数据范围。
func findDefaultTenantID(queryProvider func(context.Context) *query.Query) (int64, error) {
	ctx := context.Background()
	baseTenantQuery := queryProvider(ctx).BaseTenant
	baseTenant, err := baseTenantQuery.WithContext(ctx).
		Select(baseTenantQuery.ID).
		Where(baseTenantQuery.Code.Eq(gorm.DefaultTenantCode)).
		First()
	if err != nil {
		return 0, err
	}
	return baseTenant.ID, nil
}

// workflowInstanceBindingExists 校验系统动作携带的流程实例与业务单据绑定关系。
func workflowInstanceBindingExists(queryProvider func(context.Context) *query.Query, tenantID, instanceID, businessID int64, businessType string, status int32) (bool, error) {
	ctx := context.Background()
	instanceQuery := queryProvider(ctx).WorkflowInstance
	count, err := instanceQuery.WithContext(ctx).
		Where(instanceQuery.TenantID.Eq(tenantID), instanceQuery.ID.Eq(instanceID), instanceQuery.BusinessID.Eq(businessID), instanceQuery.BusinessType.Eq(businessType), instanceQuery.Status.Eq(status)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
