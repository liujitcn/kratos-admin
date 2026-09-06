package biz

import (
	"context"
	"net/http"
	"strings"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/adapter/kit"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/redact"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseRedactOutputPolicyCase 提供出库脱敏策略管理能力。
type BaseRedactOutputPolicyCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRedactOutputPolicyRepository
	baseAPIRepo *data.BaseAPIRepository
	ruleRepo    *data.BaseRedactRuleRepository
	resolver    *kit.RedactPolicyResolver
}

// NewBaseRedactOutputPolicyCase 创建出库脱敏策略业务实例。
func NewBaseRedactOutputPolicyCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.BaseRedactOutputPolicyRepository, baseAPIRepo *data.BaseAPIRepository, ruleRepo *data.BaseRedactRuleRepository, resolver *kit.RedactPolicyResolver) *BaseRedactOutputPolicyCase {
	return &BaseRedactOutputPolicyCase{BaseCase: baseCase, tx: tx, BaseRedactOutputPolicyRepository: repo, baseAPIRepo: baseAPIRepo, ruleRepo: ruleRepo, resolver: resolver}
}

// PageBaseRedactOutputPolicy 分页查询出库脱敏策略。
func (c *BaseRedactOutputPolicyCase) PageBaseRedactOutputPolicy(ctx context.Context, req *adminv1.PageBaseRedactOutputPolicyRequest) (*adminv1.PageBaseRedactOutputPolicyResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRedactOutputPolicy
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.ID.Asc()))
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	if req.GetMessageRef() != "" {
		opts = append(opts, repository.Where(query.MessageRef.Like("%"+req.GetMessageRef()+"%")))
	}
	if req.GetFieldPath() != "" {
		opts = append(opts, repository.Where(query.FieldPath.Like("%"+req.GetFieldPath()+"%")))
	}
	if req.GetServiceName() != "" {
		opts = append(opts, repository.Where(query.ServiceName.Eq(req.GetServiceName())))
	}
	if req.Mode != nil {
		opts = append(opts, repository.Where(query.Mode.Eq(int32(req.GetMode()))))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseRedactOutputPolicy
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseRedactOutputPolicy, 0, len(list))
	for _, item := range list {
		var dto *adminv1.BaseRedactOutputPolicy
		dto, err = c.toBaseRedactOutputPolicy(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, dto)
	}
	return &adminv1.PageBaseRedactOutputPolicyResponse{BaseRedactOutputPolicies: result, Total: int32(total)}, nil
}

// GetBaseRedactOutputPolicy 查询出库脱敏策略详情。
func (c *BaseRedactOutputPolicyCase) GetBaseRedactOutputPolicy(ctx context.Context, req *adminv1.GetBaseRedactOutputPolicyRequest) (*adminv1.BaseRedactOutputPolicyForm, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	var item *models.BaseRedactOutputPolicy
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseRedactOutputPolicyForm{Id: item.ID, Operation: item.Operation, ServiceName: item.ServiceName, MessageRef: item.MessageRef, FieldPath: item.FieldPath, Mode: adminv1.BaseRedactOutputPolicyMode(item.Mode), RuleId: item.RuleID, RuleParams: item.RuleParams, Status: commonv1.Status(item.Status), Remark: item.Remark}, nil
}

// CreateBaseRedactOutputPolicy 批量创建出库脱敏策略。
func (c *BaseRedactOutputPolicyCase) CreateBaseRedactOutputPolicy(ctx context.Context, inputs []*adminv1.BaseRedactOutputPolicyForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errorsx.InvalidArgument("出库脱敏策略列表不能为空")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	items := make([]*models.BaseRedactOutputPolicy, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return errorsx.InvalidArgument("出库脱敏策略不能为空")
		}
		var rule *models.BaseRedactRule
		rule, err = c.validateOutputForm(ctx, input)
		if err != nil {
			return err
		}
		item := &models.BaseRedactOutputPolicy{Operation: input.GetOperation(), ServiceName: input.GetServiceName(), MessageRef: input.GetMessageRef(), FieldPath: input.GetFieldPath(), Mode: int32(input.GetMode()), RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: now, UpdatedAt: now}
		if rule == nil {
			item.RuleID = 0
			item.RuleParams = "{}"
		} else {
			item.RuleID = rule.ID
		}
		if item.Status == 0 {
			item.Status = _const.STATUS_STATUS_ENABLE
		}
		items = append(items, item)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.BatchCreate(txCtx, items)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一接口返回字段的出库脱敏策略已存在", "base_redact_output_policy", "operation,message_ref,field_path", "unique_base_redact_output_policy").WithCause(err)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// UpdateBaseRedactOutputPolicy 批量更新出库脱敏策略。
func (c *BaseRedactOutputPolicyCase) UpdateBaseRedactOutputPolicy(ctx context.Context, inputs []*adminv1.BaseRedactOutputPolicyForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errorsx.InvalidArgument("出库脱敏策略列表不能为空")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	items := make([]*models.BaseRedactOutputPolicy, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return errorsx.InvalidArgument("出库脱敏策略不能为空")
		}
		var oldItem *models.BaseRedactOutputPolicy
		oldItem, err = c.FindByID(ctx, input.GetId())
		if err != nil {
			return err
		}
		var rule *models.BaseRedactRule
		rule, err = c.validateOutputForm(ctx, input)
		if err != nil {
			return err
		}
		item := &models.BaseRedactOutputPolicy{ID: oldItem.ID, Operation: input.GetOperation(), ServiceName: input.GetServiceName(), MessageRef: input.GetMessageRef(), FieldPath: input.GetFieldPath(), Mode: int32(input.GetMode()), RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: oldItem.CreatedBy, UpdatedBy: authInfo.UserId, CreatedAt: oldItem.CreatedAt, UpdatedAt: now}
		if rule == nil {
			item.RuleID = 0
			item.RuleParams = "{}"
		} else {
			item.RuleID = rule.ID
		}
		if item.Status == 0 {
			item.Status = oldItem.Status
		}
		items = append(items, item)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		for _, item := range items {
			err = c.UpdateByID(txCtx, item)
			if err != nil {
				if errorsx.IsDuplicateKey(err) {
					return errorsx.UniqueConflict("同一接口返回字段的出库脱敏策略已存在", "base_redact_output_policy", "operation,message_ref,field_path", "unique_base_redact_output_policy").WithCause(err)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// DeleteBaseRedactOutputPolicy 删除出库脱敏策略。
func (c *BaseRedactOutputPolicyCase) DeleteBaseRedactOutputPolicy(ctx context.Context, value string) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(value)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("出库策略ID不能为空")
	}
	var items []*models.BaseRedactOutputPolicy
	items, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(items) != len(ids) {
		return errorsx.ResourceNotFound("出库脱敏策略不存在")
	}
	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// SetBaseRedactOutputPolicyStatus 设置出库脱敏策略状态。
func (c *BaseRedactOutputPolicyCase) SetBaseRedactOutputPolicyStatus(ctx context.Context, req *adminv1.SetBaseRedactOutputPolicyStatusRequest) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("出库脱敏策略状态无效")
	}
	var item *models.BaseRedactOutputPolicy
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	err = c.UpdateByID(ctx, &models.BaseRedactOutputPolicy{ID: item.ID, Status: int32(req.GetStatus())})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// validateOutputForm 校验出库策略接口、模式和规则参数。
func (c *BaseRedactOutputPolicyCase) validateOutputForm(ctx context.Context, input *adminv1.BaseRedactOutputPolicyForm) (*models.BaseRedactRule, error) {
	query := c.baseAPIRepo.Query(ctx).BaseAPI
	var api *models.BaseAPI
	var err error
	api, err = c.baseAPIRepo.Find(ctx, repository.Where(query.Operation.Eq(input.GetOperation())))
	if err != nil {
		return nil, errorsx.ResourceNotFound("接口不存在").WithCause(err)
	}
	if api.ServiceName != input.GetServiceName() {
		return nil, errorsx.InvalidArgument("接口不属于所选服务")
	}
	if !strings.EqualFold(api.Method, http.MethodGet) {
		return nil, errorsx.InvalidArgument("出库脱敏只支持 GET 接口")
	}
	mode := input.GetMode()
	if mode != adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_RULE && mode != adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_HIDE && mode != adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_FULL {
		return nil, errorsx.InvalidArgument("出库脱敏处理模式无效")
	}
	status := input.GetStatus()
	if status != commonv1.Status_STATUS_UNSPECIFIED && status != commonv1.Status_STATUS_ENABLE && status != commonv1.Status_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("出库脱敏策略状态无效")
	}
	if mode != adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_RULE {
		return nil, nil
	}
	var rule *models.BaseRedactRule
	rule, err = c.ruleRepo.FindByID(ctx, input.GetRuleId())
	if err != nil {
		return nil, errorsx.ResourceNotFound("脱敏规则不存在").WithCause(err)
	}
	if rule.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.InvalidArgument("脱敏规则已停用")
	}
	_, err = redact.ValidateRuleTemplate(rule.Code, rule.RuleType, input.GetRuleParams())
	if err != nil {
		return nil, errorsx.InvalidArgument("出库脱敏规则参数无效").WithCause(err)
	}
	return rule, nil
}

// toBaseRedactOutputPolicy 转换出库脱敏策略列表项。
func (c *BaseRedactOutputPolicyCase) toBaseRedactOutputPolicy(ctx context.Context, item *models.BaseRedactOutputPolicy) (*adminv1.BaseRedactOutputPolicy, error) {
	result := &adminv1.BaseRedactOutputPolicy{Id: item.ID, Operation: item.Operation, ServiceName: item.ServiceName, MessageRef: item.MessageRef, FieldPath: item.FieldPath, Mode: adminv1.BaseRedactOutputPolicyMode(item.Mode), RuleId: item.RuleID, RuleParams: item.RuleParams, Status: commonv1.Status(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05")}
	if item.RuleID == 0 {
		return result, nil
	}
	rule, err := c.ruleRepo.FindByID(ctx, item.RuleID)
	if err != nil {
		return nil, err
	}
	result.RuleCode = rule.Code
	result.RuleName = rule.Name
	result.RuleType = rule.RuleType
	return result, nil
}
