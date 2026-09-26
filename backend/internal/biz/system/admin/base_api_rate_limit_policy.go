package biz

import (
	"context"
	"encoding/json"
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
)

// BaseApiRateLimitPolicyCase 提供接口限流策略管理能力。
type BaseApiRateLimitPolicyCase struct {
	*biz.BaseCase
	*data.BaseAPIRateLimitPolicyRepository
	apiRepo  *data.BaseAPIRepository
	ruleRepo *data.BaseRateLimitRuleRepository
	resolver *kit.RateLimitPolicyResolver
}

// NewBaseApiRateLimitPolicyCase 创建接口限流策略业务实例。
func NewBaseApiRateLimitPolicyCase(baseCase *biz.BaseCase, repo *data.BaseAPIRateLimitPolicyRepository, apiRepo *data.BaseAPIRepository, ruleRepo *data.BaseRateLimitRuleRepository, resolver *kit.RateLimitPolicyResolver) *BaseApiRateLimitPolicyCase {
	return &BaseApiRateLimitPolicyCase{BaseCase: baseCase, BaseAPIRateLimitPolicyRepository: repo, apiRepo: apiRepo, ruleRepo: ruleRepo, resolver: resolver}
}

// PageBaseApiRateLimitPolicy 分页查询接口限流策略。
func (c *BaseApiRateLimitPolicyCase) PageBaseApiRateLimitPolicy(ctx context.Context, req *adminv1.PageBaseApiRateLimitPolicyRequest) (*adminv1.PageBaseApiRateLimitPolicyResponse, error) {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseAPIRateLimitPolicy
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.ID.Desc()))
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operations.Like("%"+req.GetOperation()+"%")))
	}
	if req.Dimension != nil && req.GetDimension() != adminv1.BaseApiRateLimitDimension_BASE_API_RATE_LIMIT_DIMENSION_UNSPECIFIED {
		var dimension string
		dimension, err = kit.RateLimitDimensionFromProto(req.GetDimension())
		if err != nil {
			return nil, errorsx.InvalidArgument("限流维度无效").WithCause(err)
		}
		opts = append(opts, repository.Where(query.Dimension.Eq(dimension)))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseAPIRateLimitPolicy
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	operations := make([]string, 0, len(list))
	operationSet := make(map[string]struct{}, len(list))
	operationsByPolicy := make(map[int64][]string, len(list))
	ruleIDs := make([]int64, 0, len(list))
	ruleIDSet := make(map[int64]struct{}, len(list))
	for _, item := range list {
		var policyOperations []string
		policyOperations, err = parseRateLimitOperations(item.Operations)
		if err != nil {
			return nil, errorsx.Internal("接口限流策略接口数据无效").WithCause(err)
		}
		operationsByPolicy[item.ID] = policyOperations
		for _, operation := range policyOperations {
			if _, exists := operationSet[operation]; !exists {
				operations = append(operations, operation)
				operationSet[operation] = struct{}{}
			}
		}
		if _, exists := ruleIDSet[item.RuleID]; !exists {
			ruleIDs = append(ruleIDs, item.RuleID)
			ruleIDSet[item.RuleID] = struct{}{}
		}
	}
	var apiList []*models.BaseAPI
	if len(operations) > 0 {
		apiQuery := c.apiRepo.Query(ctx).BaseAPI
		apiList, err = c.apiRepo.List(ctx, repository.Where(apiQuery.Operation.In(operations...)))
		if err != nil {
			return nil, err
		}
	}
	rules := make(map[int64]*models.BaseRateLimitRule, len(ruleIDs))
	if len(ruleIDs) > 0 {
		var ruleList []*models.BaseRateLimitRule
		ruleList, err = c.ruleRepo.ListByIDs(ctx, ruleIDs)
		if err != nil {
			return nil, err
		}
		for _, item := range ruleList {
			rules[item.ID] = item
		}
	}
	apiByOperation := make(map[string]*models.BaseAPI, len(apiList))
	for _, item := range apiList {
		apiByOperation[item.Operation] = item
	}
	result := make([]*adminv1.BaseApiRateLimitPolicy, 0, len(list))
	for _, item := range list {
		result = append(result, toBaseApiRateLimitPolicy(item, operationsByPolicy[item.ID], apiByOperation, rules[item.RuleID]))
	}
	return &adminv1.PageBaseApiRateLimitPolicyResponse{BaseApiRateLimitPolicies: result, Total: int32(total)}, nil
}

// GetBaseApiRateLimitPolicy 查询接口限流策略详情。
func (c *BaseApiRateLimitPolicyCase) GetBaseApiRateLimitPolicy(ctx context.Context, id int64) (*adminv1.BaseApiRateLimitPolicyForm, error) {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	var item *models.BaseAPIRateLimitPolicy
	item, err = c.FindByID(ctx, id)
	if err != nil {
		return nil, errorsx.ResourceNotFound("接口限流策略不存在").WithCause(err)
	}
	var form *adminv1.BaseApiRateLimitPolicyForm
	form, err = toBaseApiRateLimitPolicyForm(item)
	if err != nil {
		return nil, errorsx.Internal("接口限流策略接口数据无效").WithCause(err)
	}
	return form, nil
}

// CreateBaseApiRateLimitPolicy 创建接口限流策略。
func (c *BaseApiRateLimitPolicyCase) CreateBaseApiRateLimitPolicy(ctx context.Context, input *adminv1.BaseApiRateLimitPolicyForm) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	var dimension string
	dimension, err = kit.RateLimitDimensionFromProto(input.GetDimension())
	if err != nil {
		return errorsx.InvalidArgument("限流维度无效").WithCause(err)
	}
	var operations []string
	operations, err = c.validateOperations(ctx, input.GetOperations())
	if err != nil {
		return err
	}
	err = c.ensureOperationsAvailable(ctx, operations, dimension, 0)
	if err != nil {
		return err
	}
	rule, err := c.findEnabledRule(ctx, input.GetRuleId())
	if err != nil {
		return err
	}
	_, err = kit.ParseRateLimitParams(rule.RuleType, input.GetRuleParams())
	if err != nil {
		return errorsx.InvalidArgument("接口限流参数无效").WithCause(err)
	}
	status := int32(input.GetStatus())
	if status == 0 {
		status = _const.STATUS_STATUS_ENABLE
	}
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("接口限流策略状态无效")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	var rawOperations []byte
	rawOperations, err = json.Marshal(operations)
	if err != nil {
		return err
	}
	item := &models.BaseAPIRateLimitPolicy{
		Operations: string(rawOperations),
		Dimension:  dimension,
		RuleID:     rule.ID,
		RuleParams: input.GetRuleParams(),
		Status:     status,
		Remark:     input.GetRemark(),
		CreatedBy:  authInfo.UserId,
		UpdatedBy:  authInfo.UserId,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = c.Create(ctx, item)
	if err != nil {
		return err
	}
	return ratelimit.RefreshRateLimitRuntime(ctx, c.resolver)
}

// UpdateBaseApiRateLimitPolicy 更新接口限流策略及其参数快照。
func (c *BaseApiRateLimitPolicyCase) UpdateBaseApiRateLimitPolicy(ctx context.Context, input *adminv1.BaseApiRateLimitPolicyForm) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	var oldItem *models.BaseAPIRateLimitPolicy
	oldItem, err = c.FindByID(ctx, input.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("接口限流策略不存在").WithCause(err)
	}
	var dimension string
	dimension, err = kit.RateLimitDimensionFromProto(input.GetDimension())
	if err != nil {
		return errorsx.InvalidArgument("限流维度无效").WithCause(err)
	}
	var operations []string
	operations, err = c.validateOperations(ctx, input.GetOperations())
	if err != nil {
		return err
	}
	err = c.ensureOperationsAvailable(ctx, operations, dimension, oldItem.ID)
	if err != nil {
		return err
	}
	rule, err := c.findEnabledRule(ctx, input.GetRuleId())
	if err != nil {
		return err
	}
	_, err = kit.ParseRateLimitParams(rule.RuleType, input.GetRuleParams())
	if err != nil {
		return errorsx.InvalidArgument("接口限流参数无效").WithCause(err)
	}
	status := int32(input.GetStatus())
	if status != _const.STATUS_STATUS_ENABLE && status != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("接口限流策略状态无效")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var rawOperations []byte
	rawOperations, err = json.Marshal(operations)
	if err != nil {
		return err
	}
	item := &models.BaseAPIRateLimitPolicy{ID: oldItem.ID, Operations: string(rawOperations), Dimension: dimension, RuleID: rule.ID, RuleParams: input.GetRuleParams(), Status: status, Remark: input.GetRemark(), UpdatedBy: authInfo.UserId}
	err = c.UpdateByID(ctx, item)
	if err != nil {
		return err
	}
	return ratelimit.RefreshRateLimitRuntime(ctx, c.resolver)
}

// DeleteBaseApiRateLimitPolicy 删除接口限流策略。
func (c *BaseApiRateLimitPolicyCase) DeleteBaseApiRateLimitPolicy(ctx context.Context, value string) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(value)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("接口限流策略ID不能为空")
	}
	var items []*models.BaseAPIRateLimitPolicy
	items, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(items) != len(ids) {
		return errorsx.ResourceNotFound("接口限流策略不存在")
	}
	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		return err
	}
	return ratelimit.RefreshRateLimitRuntime(ctx, c.resolver)
}

// SetBaseApiRateLimitPolicyStatus 设置接口限流策略状态。
func (c *BaseApiRateLimitPolicyCase) SetBaseApiRateLimitPolicyStatus(ctx context.Context, req *adminv1.SetBaseApiRateLimitPolicyStatusRequest) error {
	err := ratelimit.EnsureRateLimitPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("接口限流策略状态无效")
	}
	var item *models.BaseAPIRateLimitPolicy
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("接口限流策略不存在").WithCause(err)
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	if req.GetStatus() == commonv1.Status_STATUS_ENABLE {
		var rule *models.BaseRateLimitRule
		rule, err = c.findEnabledRule(ctx, item.RuleID)
		if err != nil {
			return err
		}
		_, err = kit.ParseRateLimitParams(rule.RuleType, item.RuleParams)
		if err != nil {
			return errorsx.InvalidArgument("接口限流参数无效").WithCause(err)
		}
		var operations []string
		operations, err = parseRateLimitOperations(item.Operations)
		if err != nil {
			return errorsx.Internal("接口限流策略接口数据无效").WithCause(err)
		}
		err = c.ensureOperationsAvailable(ctx, operations, item.Dimension, item.ID)
		if err != nil {
			return err
		}
	}
	err = c.UpdateByID(ctx, &models.BaseAPIRateLimitPolicy{ID: item.ID, Status: int32(req.GetStatus())})
	if err != nil {
		return err
	}
	return ratelimit.RefreshRateLimitRuntime(ctx, c.resolver)
}

// validateOperations 校验接口操作列表非空、无重复且全部存在。
func (c *BaseApiRateLimitPolicyCase) validateOperations(ctx context.Context, operations []string) ([]string, error) {
	if len(operations) == 0 || len(operations) > 500 {
		return nil, errorsx.InvalidArgument("请选择1到500个接口")
	}
	operationSet := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if operation == "" || len(operation) > 250 {
			return nil, errorsx.InvalidArgument("接口操作无效")
		}
		if _, exists := operationSet[operation]; exists {
			return nil, errorsx.InvalidArgument("接口操作不能重复")
		}
		operationSet[operation] = struct{}{}
	}
	query := c.apiRepo.Query(ctx).BaseAPI
	items, err := c.apiRepo.List(ctx, repository.Where(query.Operation.In(operations...)))
	if err != nil {
		return nil, err
	}
	if len(items) != len(operations) {
		return nil, errorsx.ResourceNotFound("接口不存在")
	}
	return operations, nil
}

// ensureOperationsAvailable 检查相同维度下的接口是否已被其他策略占用。
func (c *BaseApiRateLimitPolicyCase) ensureOperationsAvailable(ctx context.Context, operations []string, dimension string, excludeID int64) error {
	query := c.Query(ctx).BaseAPIRateLimitPolicy
	opts := []repository.QueryOption{repository.Where(query.Dimension.Eq(dimension))}
	if excludeID > 0 {
		opts = append(opts, repository.Where(query.ID.Neq(excludeID)))
	}
	policies, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}
	operationSet := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		operationSet[operation] = struct{}{}
	}
	for _, policy := range policies {
		var existingOperations []string
		existingOperations, err = parseRateLimitOperations(policy.Operations)
		if err != nil {
			return errorsx.Internal("接口限流策略接口数据无效").WithCause(err)
		}
		for _, operation := range existingOperations {
			if _, exists := operationSet[operation]; exists {
				return errorsx.Conflict("所选接口与已有策略存在相同限流维度")
			}
		}
	}
	return nil
}

// findEnabledRule 查询启用的限流规则模板。
func (c *BaseApiRateLimitPolicyCase) findEnabledRule(ctx context.Context, id int64) (*models.BaseRateLimitRule, error) {
	item, err := c.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errorsx.ResourceNotFound("限流规则不存在").WithCause(err)
	}
	if item.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.Conflict("限流规则已停用，不能用于接口策略")
	}
	return item, nil
}

// toBaseApiRateLimitPolicy 转换接口限流策略列表项。
func toBaseApiRateLimitPolicy(item *models.BaseAPIRateLimitPolicy, operations []string, apiByOperation map[string]*models.BaseAPI, rule *models.BaseRateLimitRule) *adminv1.BaseApiRateLimitPolicy {
	result := &adminv1.BaseApiRateLimitPolicy{
		Id:         item.ID,
		Dimension:  kit.RateLimitDimensionToProto(item.Dimension),
		RuleId:     item.RuleID,
		RuleParams: item.RuleParams,
		Status:     commonv1.Status(item.Status),
		Remark:     item.Remark,
		CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	result.Apis = make([]*adminv1.BaseApiRateLimitPolicyAPI, 0, len(operations))
	for _, operation := range operations {
		api := apiByOperation[operation]
		apiResult := &adminv1.BaseApiRateLimitPolicyAPI{Operation: operation}
		if api != nil {
			apiResult.ServiceName = api.ServiceName
			apiResult.ServiceDesc = api.ServiceDesc
			apiResult.Method = api.Method
			apiResult.Path = api.Path
		}
		result.Apis = append(result.Apis, apiResult)
	}
	if rule != nil {
		result.RuleCode = rule.Code
		result.RuleName = rule.Name
	}
	return result
}

// toBaseApiRateLimitPolicyForm 转换接口限流策略编辑表单。
func toBaseApiRateLimitPolicyForm(item *models.BaseAPIRateLimitPolicy) (*adminv1.BaseApiRateLimitPolicyForm, error) {
	operations, err := parseRateLimitOperations(item.Operations)
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseApiRateLimitPolicyForm{Id: item.ID, Operations: operations, Dimension: kit.RateLimitDimensionToProto(item.Dimension), RuleId: item.RuleID, RuleParams: item.RuleParams, Status: commonv1.Status(item.Status), Remark: item.Remark}, nil
}

// parseRateLimitOperations 解析策略保存的接口操作数组。
func parseRateLimitOperations(raw string) ([]string, error) {
	var operations []string
	err := json.Unmarshal([]byte(raw), &operations)
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, errorsx.InvalidArgument("接口限流策略至少需要配置一个接口")
	}
	return operations, nil
}
