package biz

import (
	"context"
	"strings"
	"time"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/ratelimit"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen/field"
)

const rateLimitRuleType = "TOKEN_BUCKET"

// BaseRateLimitRuleCase 提供限流规则模板管理能力。
type BaseRateLimitRuleCase struct {
	*biz.BaseCase
	*data.BaseRateLimitRuleRepository
	policyRepo *data.BaseAPIRateLimitPolicyRepository
}

// NewBaseRateLimitRuleCase 创建限流规则模板业务实例。
func NewBaseRateLimitRuleCase(baseCase *biz.BaseCase, repo *data.BaseRateLimitRuleRepository, policyRepo *data.BaseAPIRateLimitPolicyRepository) *BaseRateLimitRuleCase {
	return &BaseRateLimitRuleCase{BaseCase: baseCase, BaseRateLimitRuleRepository: repo, policyRepo: policyRepo}
}

// OptionBaseRateLimitRule 查询限流规则选项。
func (c *BaseRateLimitRuleCase) OptionBaseRateLimitRule(ctx context.Context, req *adminv1.OptionBaseRateLimitRuleRequest) (*adminv1.OptionBaseRateLimitRuleResponse, error) {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRateLimitRule
	opts := make([]repository.QueryOption, 0, 2)
	if req.GetKeyword() != "" {
		keyword := "%" + req.GetKeyword() + "%"
		opts = append(opts, repository.Where(field.Or(query.Code.Like(keyword), query.Name.Like(keyword))))
	}
	opts = append(opts, repository.Order(query.ID.Asc()))
	var list []*models.BaseRateLimitRule
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*adminv1.BaseRateLimitRuleOption, 0, len(list))
	for _, item := range list {
		options = append(options, &adminv1.BaseRateLimitRuleOption{
			Id:            item.ID,
			Label:         item.Name,
			Code:          item.Code,
			RuleType:      item.RuleType,
			DefaultParams: item.DefaultParams,
			Disabled:      item.Status != _const.STATUS_STATUS_ENABLE,
		})
	}
	return &adminv1.OptionBaseRateLimitRuleResponse{List: options}, nil
}

// PageBaseRateLimitRule 分页查询限流规则模板。
func (c *BaseRateLimitRuleCase) PageBaseRateLimitRule(ctx context.Context, req *adminv1.PageBaseRateLimitRuleRequest) (*adminv1.PageBaseRateLimitRuleResponse, error) {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRateLimitRule
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.ID.Asc()))
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseRateLimitRule
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseRateLimitRule, 0, len(list))
	for _, item := range list {
		result = append(result, toBaseRateLimitRule(item))
	}
	return &adminv1.PageBaseRateLimitRuleResponse{BaseRateLimitRules: result, Total: int32(total)}, nil
}

// GetBaseRateLimitRule 查询限流规则模板详情。
func (c *BaseRateLimitRuleCase) GetBaseRateLimitRule(ctx context.Context, id int64) (*adminv1.BaseRateLimitRuleForm, error) {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	var item *models.BaseRateLimitRule
	item, err = c.FindByID(ctx, id)
	if err != nil {
		return nil, errorsx.ResourceNotFound("限流规则不存在").WithCause(err)
	}
	return toBaseRateLimitRuleForm(item), nil
}

// CreateBaseRateLimitRule 创建限流规则模板。
func (c *BaseRateLimitRuleCase) CreateBaseRateLimitRule(ctx context.Context, input *adminv1.BaseRateLimitRuleForm) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	ruleType := strings.ToUpper(strings.TrimSpace(input.GetRuleType()))
	_, err = kit.ParseRateLimitParams(ruleType, input.GetDefaultParams())
	if err != nil {
		return errorsx.InvalidArgument("限流规则参数无效").WithCause(err)
	}
	status := int32(input.GetStatus())
	if status == 0 {
		status = _const.STATUS_STATUS_ENABLE
	}
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("限流规则状态无效")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	item := &models.BaseRateLimitRule{
		Code:          input.GetCode(),
		Name:          input.GetName(),
		RuleType:      ruleType,
		DefaultParams: input.GetDefaultParams(),
		Status:        status,
		Remark:        input.GetRemark(),
		CreatedBy:     authInfo.UserId,
		UpdatedBy:     authInfo.UserId,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = c.Create(ctx, item)
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("限流规则编码重复", "base_rate_limit_rule", "code", "unique_base_rate_limit_rule").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseRateLimitRule 更新限流规则默认参数和状态。
func (c *BaseRateLimitRuleCase) UpdateBaseRateLimitRule(ctx context.Context, input *adminv1.BaseRateLimitRuleForm) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	var oldItem *models.BaseRateLimitRule
	oldItem, err = c.FindByID(ctx, input.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("限流规则不存在").WithCause(err)
	}
	if input.GetCode() != oldItem.Code || strings.ToUpper(strings.TrimSpace(input.GetRuleType())) != oldItem.RuleType {
		return errorsx.InvalidArgument("限流规则编码和算法类型不允许修改")
	}
	_, err = kit.ParseRateLimitParams(oldItem.RuleType, input.GetDefaultParams())
	if err != nil {
		return errorsx.InvalidArgument("限流规则参数无效").WithCause(err)
	}
	status := int32(input.GetStatus())
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("限流规则状态无效")
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
	item := &models.BaseRateLimitRule{ID: oldItem.ID, Name: input.GetName(), DefaultParams: input.GetDefaultParams(), Status: status, Remark: input.GetRemark(), UpdatedBy: authInfo.UserId}
	err = c.UpdateByID(ctx, item)
	if err != nil {
		return err
	}
	return nil
}

// DeleteBaseRateLimitRule 删除未被接口策略引用的限流规则。
func (c *BaseRateLimitRuleCase) DeleteBaseRateLimitRule(ctx context.Context, value string) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(value)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("限流规则ID不能为空")
	}
	var items []*models.BaseRateLimitRule
	items, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(items) != len(ids) {
		return errorsx.ResourceNotFound("限流规则不存在")
	}
	for _, item := range items {
		err = c.ensureRuleCanDelete(ctx, item.ID)
		if err != nil {
			return err
		}
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseRateLimitRuleStatus 设置限流规则模板状态。
func (c *BaseRateLimitRuleCase) SetBaseRateLimitRuleStatus(ctx context.Context, req *adminv1.SetBaseRateLimitRuleStatusRequest) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("限流规则状态无效")
	}
	var item *models.BaseRateLimitRule
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("限流规则不存在").WithCause(err)
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
	return c.UpdateByID(ctx, &models.BaseRateLimitRule{ID: item.ID, Status: int32(req.GetStatus())})
}

// ensureRuleCanDelete 检查规则是否仍被任何接口策略引用。
func (c *BaseRateLimitRuleCase) ensureRuleCanDelete(ctx context.Context, ruleID int64) error {
	query := c.policyRepo.Query(ctx).BaseAPIRateLimitPolicy
	var count int64
	var err error
	count, err = c.policyRepo.Count(ctx, repository.Where(query.RuleID.Eq(ruleID)))
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.ProtectedResourceConflict("已有接口限流策略引用该规则，不能删除", "base_rate_limit_rule")
	}
	return nil
}

// ensureRuleCanDisable 检查规则是否仍被启用的接口策略引用。
func (c *BaseRateLimitRuleCase) ensureRuleCanDisable(ctx context.Context, ruleID int64) error {
	query := c.policyRepo.Query(ctx).BaseAPIRateLimitPolicy
	var count int64
	var err error
	count, err = c.policyRepo.Count(ctx,
		repository.Where(query.RuleID.Eq(ruleID)),
		repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)),
	)
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.ProtectedResourceConflict("已有启用的接口限流策略引用该规则，请先停用引用策略", "base_rate_limit_rule")
	}
	return nil
}

// toBaseRateLimitRule 转换限流规则列表项。
func toBaseRateLimitRule(item *models.BaseRateLimitRule) *adminv1.BaseRateLimitRule {
	return &adminv1.BaseRateLimitRule{Id: item.ID, Code: item.Code, Name: item.Name, RuleType: item.RuleType, DefaultParams: item.DefaultParams, Status: commonv1.Status(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05")}
}

// toBaseRateLimitRuleForm 转换限流规则编辑表单。
func toBaseRateLimitRuleForm(item *models.BaseRateLimitRule) *adminv1.BaseRateLimitRuleForm {
	return &adminv1.BaseRateLimitRuleForm{Id: item.ID, Code: item.Code, Name: item.Name, RuleType: item.RuleType, DefaultParams: item.DefaultParams, Status: commonv1.Status(item.Status), Remark: item.Remark}
}
