package biz

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gorm"
)

const loginPolicyConfigKey = "securityLoginPolicy"

// BaseLoginPolicyCase 提供平台管理员登录来源策略配置。
// 策略持久化在系统配置表，同时刷新运行时缓存供密码登录和 OAuth 登录入口读取。
type BaseLoginPolicyCase struct {
	*biz.BaseCase
	baseConfigRepo *data.BaseConfigRepository
}

// NewBaseLoginPolicyCase 创建登录来源策略业务实例。
func NewBaseLoginPolicyCase(baseCase *biz.BaseCase, baseConfigRepo *data.BaseConfigRepository) *BaseLoginPolicyCase {
	return &BaseLoginPolicyCase{BaseCase: baseCase, baseConfigRepo: baseConfigRepo}
}

// GetBaseLoginPolicy 查询登录来源策略。
func (c *BaseLoginPolicyCase) GetBaseLoginPolicy(ctx context.Context) (*adminv1.BaseLoginPolicy, error) {
	var err error
	err = c.ensurePlatformOperator(ctx)
	if err != nil {
		return nil, err
	}
	var policy loginpolicy.Policy
	policy, err = c.loadPolicy(ctx)
	if err != nil {
		return nil, err
	}
	return toBaseLoginPolicy(policy), nil
}

// RefreshBaseLoginPolicy 从数据库加载登录来源策略到运行时缓存。
func (c *BaseLoginPolicyCase) RefreshBaseLoginPolicy(ctx context.Context) error {
	_, err := c.loadPolicy(ctx)
	return err
}

// UpdateBaseLoginPolicy 更新登录来源策略并刷新运行时缓存。
func (c *BaseLoginPolicyCase) UpdateBaseLoginPolicy(ctx context.Context, input *adminv1.BaseLoginPolicy) (*adminv1.BaseLoginPolicy, error) {
	var err error
	err = c.ensurePlatformOperator(ctx)
	if err != nil {
		return nil, err
	}
	policy := loginpolicy.Policy{
		Enabled:         input.GetEnabled(),
		IPBlacklist:     input.GetIpBlacklist(),
		IPWhitelist:     input.GetIpWhitelist(),
		TimeWindows:     input.GetTimeWindows(),
		DeviceBlacklist: input.GetDeviceBlacklist(),
		DeviceWhitelist: input.GetDeviceWhitelist(),
	}
	for _, item := range input.GetRules() {
		policy.Rules = append(policy.Rules, loginpolicy.Rule{
			TargetType:      item.GetTargetType(),
			TargetValue:     item.GetTargetValue(),
			Enabled:         item.GetEnabled(),
			IPBlacklist:     item.GetIpBlacklist(),
			IPWhitelist:     item.GetIpWhitelist(),
			TimeWindows:     item.GetTimeWindows(),
			DeviceBlacklist: item.GetDeviceBlacklist(),
			DeviceWhitelist: item.GetDeviceWhitelist(),
		})
	}
	err = policy.Validate()
	if err != nil {
		return nil, errorsx.InvalidArgument("登录来源策略格式无效").WithCause(err)
	}
	var payload []byte
	payload, err = json.Marshal(policy)
	if err != nil {
		return nil, errorsx.Internal("保存登录来源策略失败").WithCause(err)
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.baseConfigRepo.Query(ctx).BaseConfig
	var entity *models.BaseConfig
	entity, err = c.baseConfigRepo.Find(ctx,
		repository.Where(query.Site.Eq(_const.BASE_CONFIG_SITE_ADMIN)),
		repository.Where(query.Key.Eq(loginPolicyConfigKey)),
	)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		entity = &models.BaseConfig{
			Site:      _const.BASE_CONFIG_SITE_ADMIN,
			Name:      "登录来源策略",
			Type:      int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT),
			Key:       loginPolicyConfigKey,
			Value:     string(payload),
			Status:    coreconst.STATUS_STATUS_ENABLE,
			CreatedBy: authInfo.UserId,
			UpdatedBy: authInfo.UserId,
		}
		if err = c.baseConfigRepo.Create(ctx, entity); err != nil {
			return nil, err
		}
	} else {
		if err = c.baseConfigRepo.UpdateByID(ctx, &models.BaseConfig{ID: entity.ID, Value: string(payload), UpdatedBy: authInfo.UserId}); err != nil {
			return nil, err
		}
	}
	if err = loginpolicy.SaveToCache(c.Cache, policy); err != nil {
		return nil, errorsx.Internal("刷新登录来源策略失败").WithCause(err)
	}
	return toBaseLoginPolicy(policy), nil
}

// loadPolicy 从系统配置加载登录来源策略，并同步运行时缓存。
func (c *BaseLoginPolicyCase) loadPolicy(ctx context.Context) (loginpolicy.Policy, error) {
	query := c.baseConfigRepo.Query(ctx).BaseConfig
	entity, err := c.baseConfigRepo.Find(ctx,
		repository.Where(query.Site.Eq(_const.BASE_CONFIG_SITE_ADMIN)),
		repository.Where(query.Key.Eq(loginPolicyConfigKey)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return loginpolicy.LoadFromCache(c.Cache), nil
		}
		return loginpolicy.Policy{}, err
	}
	policy := loginpolicy.Policy{}
	if err = json.Unmarshal([]byte(entity.Value), &policy); err != nil {
		return loginpolicy.Policy{}, errorsx.Internal("解析登录来源策略失败").WithCause(err)
	}
	if err = loginpolicy.SaveToCache(c.Cache, policy); err != nil {
		return loginpolicy.Policy{}, errorsx.Internal("刷新登录来源策略失败").WithCause(err)
	}
	return policy, nil
}

// ensurePlatformOperator 校验当前操作者具备平台级登录策略管理权限。
func (c *BaseLoginPolicyCase) ensurePlatformOperator(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.RoleCode != coreconst.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("只有平台管理员可以管理登录来源策略")
	}
	return nil
}

// toBaseLoginPolicy 将运行时登录策略转换为接口响应。
func toBaseLoginPolicy(policy loginpolicy.Policy) *adminv1.BaseLoginPolicy {
	result := &adminv1.BaseLoginPolicy{
		Enabled:         policy.Enabled,
		IpBlacklist:     append([]string(nil), policy.IPBlacklist...),
		IpWhitelist:     append([]string(nil), policy.IPWhitelist...),
		TimeWindows:     append([]string(nil), policy.TimeWindows...),
		DeviceBlacklist: append([]string(nil), policy.DeviceBlacklist...),
		DeviceWhitelist: append([]string(nil), policy.DeviceWhitelist...),
	}
	for _, rule := range policy.Rules {
		result.Rules = append(result.Rules, &adminv1.BaseLoginPolicyRule{
			TargetType:      rule.TargetType,
			TargetValue:     rule.TargetValue,
			Enabled:         rule.Enabled,
			IpBlacklist:     append([]string(nil), rule.IPBlacklist...),
			IpWhitelist:     append([]string(nil), rule.IPWhitelist...),
			TimeWindows:     append([]string(nil), rule.TimeWindows...),
			DeviceBlacklist: append([]string(nil), rule.DeviceBlacklist...),
			DeviceWhitelist: append([]string(nil), rule.DeviceWhitelist...),
		})
	}
	return result
}
