package workflow

import (
	"context"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/workflow"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/gorm-kit/repository"
)

// engineAdapter 将注入的实例提交能力适配为公共工作流引擎。
type engineAdapter struct {
	baseCase     *biz.BaseCase
	instanceRepo *data.WorkflowInstanceRepository
	submit       func(context.Context, workflow.SubmitParams) (int64, error)
}

// NewEngine 创建公共引擎适配器，通过回调接入实例提交能力，避免依赖上层 Case。
func NewEngine(
	baseCase *biz.BaseCase,
	instanceRepo *data.WorkflowInstanceRepository,
	submit func(context.Context, workflow.SubmitParams) (int64, error),
) workflow.Engine {
	return &engineAdapter{baseCase: baseCase, instanceRepo: instanceRepo, submit: submit}
}

// Submit 解析提交身份并传递关联回写，保证首次推进前宿主已保存实例关联。
func (a *engineAdapter) Submit(ctx context.Context, params workflow.SubmitParams) (int64, error) {
	var err error
	params.TenantID, params.ApplicantUserID, err = a.resolveIdentity(ctx, params)
	if err != nil {
		return 0, err
	}
	return a.submit(ctx, params)
}

// resolveIdentity 解析提交身份：显式参数优先，缺省回退请求上下文登录态。
func (a *engineAdapter) resolveIdentity(ctx context.Context, params workflow.SubmitParams) (int64, int64, error) {
	tenantID, applicantUserID := params.TenantID, params.ApplicantUserID
	if tenantID == 0 || applicantUserID == 0 {
		authInfo, err := a.baseCase.GetAuthInfo(ctx)
		if err != nil {
			return 0, 0, err
		}
		if tenantID == 0 {
			tenantID = authInfo.TenantId
		}
		if applicantUserID == 0 {
			applicantUserID = authInfo.UserId
		}
	}
	return tenantID, applicantUserID, nil
}

// StatusByBusiness 查询业务单据最新关联实例的状态摘要。
func (a *engineAdapter) StatusByBusiness(ctx context.Context, tenantID int64, businessType string, businessID int64) (workflow.InstanceStatus, error) {
	query := a.instanceRepo.Query(ctx).WorkflowInstance
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Select(query.ID, query.Status, query.EndAt))
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.BusinessType.Eq(businessType)))
	opts = append(opts, repository.Where(query.BusinessID.Eq(businessID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	instance, err := a.instanceRepo.Find(ctx, opts...)
	if err != nil {
		return workflow.InstanceStatus{}, errorsx.ResourceNotFound("未找到业务单据关联的流程实例")
	}
	return workflow.InstanceStatus{
		InstanceID: instance.ID,
		Status:     adminv1.WorkflowInstanceStatus(instance.Status),
		EndAt:      FormatOptionalTime(instance.EndAt),
	}, nil
}
