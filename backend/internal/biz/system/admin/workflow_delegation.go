package biz

import (
	"context"
	"errors"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
	gormDB "gorm.io/gorm"
)

// WorkflowDelegationCase 工作流审批委托业务实例。
type WorkflowDelegationCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.WorkflowDelegationRepository
	baseUserRepo *data.BaseUserRepository
	mapper       *mapper.CopierMapper[adminv1.WorkflowDelegation, models.WorkflowDelegation]
}

// NewWorkflowDelegationCase 创建工作流审批委托业务实例。
func NewWorkflowDelegationCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	delegationRepo *data.WorkflowDelegationRepository,
	baseUserRepo *data.BaseUserRepository,
) *WorkflowDelegationCase {
	return &WorkflowDelegationCase{
		BaseCase:                     baseCase,
		tx:                           tx,
		WorkflowDelegationRepository: delegationRepo,
		baseUserRepo:                 baseUserRepo,
		mapper:                       mapper.NewCopierMapper[adminv1.WorkflowDelegation, models.WorkflowDelegation](),
	}
}

// PageWorkflowDelegation 分页查询审批委托，合并双方用户昵称。
func (c *WorkflowDelegationCase) PageWorkflowDelegation(ctx context.Context, req *adminv1.PageWorkflowDelegationRequest) (*adminv1.PageWorkflowDelegationResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).WorkflowDelegation
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		if authInfo.TenantCode != gorm.DefaultTenantCode && req.GetTenantId() != authInfo.TenantId {
			return nil, errorsx.PermissionDenied("不能查询其他租户的审批委托")
		}
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	} else if authInfo.TenantCode != gorm.DefaultTenantCode {
		opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)))
	}
	if req.GetDelegatorUserId() > 0 {
		opts = append(opts, repository.Where(query.DelegatorUserID.Eq(req.GetDelegatorUserId())))
	}
	if req.GetDelegateUserId() > 0 {
		opts = append(opts, repository.Where(query.DelegateUserID.Eq(req.GetDelegateUserId())))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(list)*2)
	for _, item := range list {
		userIDs = append(userIDs, item.DelegatorUserID, item.DelegateUserID)
	}
	userMap, err := c.mapNickNames(ctx, dedupeInt64(userIDs))
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.WorkflowDelegation, 0, len(list))
	for _, item := range list {
		dto := c.mapper.ToDTO(item)
		dto.DelegatorName = userMap[item.DelegatorUserID]
		dto.DelegateName = userMap[item.DelegateUserID]
		resList = append(resList, dto)
	}
	return &adminv1.PageWorkflowDelegationResponse{Delegations: resList, Total: int32(total)}, nil
}

// SetWorkflowDelegation 设置当前用户的审批委托，重复设置覆盖被委托人。
func (c *WorkflowDelegationCase) SetWorkflowDelegation(ctx context.Context, req *adminv1.SetWorkflowDelegationRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	delegateID := req.GetDelegation().GetDelegateUserId()
	if delegateID == 0 || delegateID == authInfo.UserId {
		return errorsx.InvalidArgument("被委托人无效")
	}
	if err := c.ensureUserActive(ctx, authInfo.TenantId, delegateID); err != nil {
		return err
	}

	delegationQuery := c.Query(ctx).WorkflowDelegation
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(delegationQuery.TenantID.Eq(authInfo.TenantId)))
	opts = append(opts, repository.Where(delegationQuery.DelegatorUserID.Eq(authInfo.UserId)))
	existing, err := c.Find(ctx, opts...)
	if err != nil && !errors.Is(err, gormDB.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		existing.DelegateUserID = delegateID
		existing.Remark = req.GetDelegation().GetRemark()
		existing.Status = _const.STATUS_STATUS_ENABLE
		return c.tx.Transaction(ctx, func(ctx context.Context) error {
			return c.UpdateByID(ctx, existing)
		})
	}

	delegation := &models.WorkflowDelegation{
		TenantID:        authInfo.TenantId,
		DelegatorUserID: authInfo.UserId,
		DelegateUserID:  delegateID,
		Remark:          req.GetDelegation().GetRemark(),
		Status:          _const.STATUS_STATUS_ENABLE,
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, delegation)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一用户只能设置一条审批委托", "workflow_delegation", "delegator_user_id", "unique_workflow_delegation").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// DeleteWorkflowDelegation 删除审批委托，仅委托人本人可删除。
func (c *WorkflowDelegationCase) DeleteWorkflowDelegation(ctx context.Context, id string) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(id)
	list, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	existing := make(map[int64]*models.WorkflowDelegation, len(list))
	for _, item := range list {
		existing[item.ID] = item
	}
	for _, delegationID := range ids {
		item, found := existing[delegationID]
		if !found {
			return errorsx.ResourceNotFound("删除审批委托失败，委托不存在")
		}
		if item.DelegatorUserID != authInfo.UserId && authInfo.TenantCode != gorm.DefaultTenantCode {
			return errorsx.PermissionDenied("只能删除自己的审批委托")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.DeleteByIDs(ctx, ids)
	})
}

// ensureUserActive 校验用户存在且启用。
func (c *WorkflowDelegationCase) ensureUserActive(ctx context.Context, tenantID, userID int64) error {
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Select(userQuery.ID, userQuery.Status))
	opts = append(opts, repository.Where(userQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(userQuery.ID.Eq(userID)))
	user, err := c.baseUserRepo.Find(ctx, opts...)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在")
	}
	if user.Status != _const.STATUS_STATUS_ENABLE {
		return errorsx.InvalidArgument("用户已禁用")
	}
	return nil
}

// mapNickNames 按用户 ID 批量查询昵称映射，昵称缺失回退用户名。
func (c *WorkflowDelegationCase) mapNickNames(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	users, err := c.baseUserRepo.List(ctx,
		repository.Select(userQuery.ID, userQuery.NickName, userQuery.UserName),
		repository.Where(userQuery.ID.In(userIDs...)),
	)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.NickName != "" {
			result[user.ID] = user.NickName
		} else {
			result[user.ID] = user.UserName
		}
	}
	return result, nil
}
