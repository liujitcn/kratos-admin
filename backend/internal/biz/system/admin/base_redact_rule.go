package biz

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/redact"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"

	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// BaseRedactRuleCase 提供脱敏规则模板管理能力。
type BaseRedactRuleCase struct {
	*biz.BaseCase
	*data.BaseRedactRuleRepository
	storagePolicyRepo *data.BaseRedactStoragePolicyRepository
	outputPolicyRepo  *data.BaseRedactOutputPolicyRepository
	resolver          *kit.RedactPolicyResolver
}

// NewBaseRedactRuleCase 创建脱敏规则模板业务实例。
func NewBaseRedactRuleCase(baseCase *biz.BaseCase, repo *data.BaseRedactRuleRepository, storagePolicyRepo *data.BaseRedactStoragePolicyRepository, outputPolicyRepo *data.BaseRedactOutputPolicyRepository, resolver *kit.RedactPolicyResolver) *BaseRedactRuleCase {
	return &BaseRedactRuleCase{BaseCase: baseCase, BaseRedactRuleRepository: repo, storagePolicyRepo: storagePolicyRepo, outputPolicyRepo: outputPolicyRepo, resolver: resolver}
}

// OptionBaseRedactRule 查询脱敏规则选项。
func (c *BaseRedactRuleCase) OptionBaseRedactRule(ctx context.Context, req *adminv1.OptionBaseRedactRuleRequest) (*commonv1.SelectOptionResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRedactRule
	opts := make([]repository.QueryOption, 0, 2)
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.Code.Like(keyword), query.Name.Like(keyword))))
	}
	opts = append(opts, repository.Order(query.ID.Asc()))
	var list []*models.BaseRedactRule
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{Label: item.Name, Value: item.ID, Disabled: item.Status != _const.STATUS_STATUS_ENABLE})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseRedactRule 分页查询脱敏规则模板。
func (c *BaseRedactRuleCase) PageBaseRedactRule(ctx context.Context, req *adminv1.PageBaseRedactRuleRequest) (*adminv1.PageBaseRedactRuleResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRedactRule
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.ID.Asc()))
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetRuleType() != "" {
		opts = append(opts, repository.Where(query.RuleType.Eq(req.GetRuleType())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseRedactRule
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseRedactRule, 0, len(list))
	for _, item := range list {
		result = append(result, toBaseRedactRule(item))
	}
	return &adminv1.PageBaseRedactRuleResponse{BaseRedactRules: result, Total: int32(total)}, nil
}

// GetBaseRedactRule 查询脱敏规则模板详情。
func (c *BaseRedactRuleCase) GetBaseRedactRule(ctx context.Context, id int64) (*adminv1.BaseRedactRuleForm, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	var item *models.BaseRedactRule
	item, err = c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseRedactRuleForm(item), nil
}

// CreateBaseRedactRule 创建脱敏规则模板。
func (c *BaseRedactRuleCase) CreateBaseRedactRule(ctx context.Context, _ *adminv1.BaseRedactRuleForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	return errorsx.ProtectedResourceConflict("脱敏规则模板由系统内置，不支持新增", "base_redact_rule")
}

// UpdateBaseRedactRule 更新脱敏规则模板参数和状态。
func (c *BaseRedactRuleCase) UpdateBaseRedactRule(ctx context.Context, input *adminv1.BaseRedactRuleForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if input.GetId() <= 0 {
		return errorsx.InvalidArgument("规则模板ID不能为空")
	}
	var oldItem *models.BaseRedactRule
	oldItem, err = c.FindByID(ctx, input.GetId())
	if err != nil {
		return err
	}
	if input.GetCode() != oldItem.Code {
		return errorsx.InvalidArgument("规则模板编码不允许修改")
	}
	if input.GetRuleType() != oldItem.RuleType {
		return errorsx.InvalidArgument("规则模板类型不允许修改")
	}
	_, err = redact.ValidateRuleTemplate(oldItem.Code, oldItem.RuleType, input.GetRule())
	if err != nil {
		return errorsx.InvalidArgument("规则模板参数无效").WithCause(err)
	}
	status := int32(input.GetStatus())
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("规则模板状态无效")
	}
	if status == _const.STATUS_STATUS_DISABLE {
		err = c.ensureRuleCanDisable(ctx, oldItem.ID)
		if err != nil {
			return err
		}
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item := &models.BaseRedactRule{ID: oldItem.ID, DefaultParams: input.GetRule(), Status: status, Remark: input.GetRemark(), UpdatedBy: authInfo.UserId}
	err = c.UpdateByID(ctx, item)
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// DeleteBaseRedactRule 删除脱敏规则模板。
func (c *BaseRedactRuleCase) DeleteBaseRedactRule(ctx context.Context, _ string) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	return errorsx.ProtectedResourceConflict("脱敏规则模板由系统内置，不支持删除", "base_redact_rule")
}

// SetBaseRedactRuleStatus 设置脱敏规则模板状态。
func (c *BaseRedactRuleCase) SetBaseRedactRuleStatus(ctx context.Context, req *adminv1.SetBaseRedactRuleStatusRequest) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("规则模板状态无效")
	}
	var item *models.BaseRedactRule
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	if req.GetStatus() == commonv1.Status_STATUS_DISABLE {
		err = c.ensureRuleCanDisable(ctx, item.ID)
		if err != nil {
			return err
		}
	}
	err = c.UpdateByID(ctx, &models.BaseRedactRule{ID: item.ID, Status: int32(req.GetStatus())})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// ensureRuleCanDisable 检查规则是否仍被启用策略引用。
func (c *BaseRedactRuleCase) ensureRuleCanDisable(ctx context.Context, ruleID int64) error {
	storageQuery := c.storagePolicyRepo.Query(ctx).BaseRedactStoragePolicy
	var storageCount int64
	var err error
	storageCount, err = c.storagePolicyRepo.Count(ctx,
		repository.Where(storageQuery.RuleID.Eq(ruleID)),
		repository.Where(storageQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)),
	)
	if err != nil {
		return err
	}
	if storageCount > 0 {
		return errorsx.ProtectedResourceConflict("已有启用的入库脱敏策略引用该规则，不能停用", "base_redact_rule")
	}
	outputQuery := c.outputPolicyRepo.Query(ctx).BaseRedactOutputPolicy
	var outputCount int64
	outputCount, err = c.outputPolicyRepo.Count(ctx,
		repository.Where(outputQuery.RuleID.Eq(ruleID)),
		repository.Where(outputQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)),
	)
	if err != nil {
		return err
	}
	if outputCount > 0 {
		return errorsx.ProtectedResourceConflict("已有启用的出库脱敏策略引用该规则，不能停用", "base_redact_rule")
	}
	return nil
}

// toBaseRedactRule 转换脱敏规则模板列表项。
func toBaseRedactRule(item *models.BaseRedactRule) *adminv1.BaseRedactRule {
	return &adminv1.BaseRedactRule{Id: item.ID, Code: item.Code, Name: item.Name, RuleType: item.RuleType, Rule: item.DefaultParams, Status: commonv1.Status(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05")}
}

// toBaseRedactRuleForm 转换脱敏规则模板表单。
func toBaseRedactRuleForm(item *models.BaseRedactRule) *adminv1.BaseRedactRuleForm {
	return &adminv1.BaseRedactRuleForm{Id: item.ID, Code: item.Code, Name: item.Name, RuleType: item.RuleType, Rule: item.DefaultParams, Status: commonv1.Status(item.Status), Remark: item.Remark}
}
