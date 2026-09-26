package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseRateLimitRuleService 提供限流规则模板管理接口。
type BaseRateLimitRuleService struct {
	adminv1.UnimplementedBaseRateLimitRuleServiceServer
	baseRateLimitRuleCase *biz.BaseRateLimitRuleCase
}

// NewBaseRateLimitRuleService 创建限流规则模板服务。
func NewBaseRateLimitRuleService(baseRateLimitRuleCase *biz.BaseRateLimitRuleCase) *BaseRateLimitRuleService {
	return &BaseRateLimitRuleService{baseRateLimitRuleCase: baseRateLimitRuleCase}
}

// OptionBaseRateLimitRule 查询限流规则选项。
func (s *BaseRateLimitRuleService) OptionBaseRateLimitRule(ctx context.Context, req *adminv1.OptionBaseRateLimitRuleRequest) (*adminv1.OptionBaseRateLimitRuleResponse, error) {
	result, err := s.baseRateLimitRuleCase.OptionBaseRateLimitRule(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "查询限流规则选项失败")
	}
	return result, nil
}

// PageBaseRateLimitRule 分页查询限流规则模板。
func (s *BaseRateLimitRuleService) PageBaseRateLimitRule(ctx context.Context, req *adminv1.PageBaseRateLimitRuleRequest) (*adminv1.PageBaseRateLimitRuleResponse, error) {
	result, err := s.baseRateLimitRuleCase.PageBaseRateLimitRule(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询限流规则失败")
	}
	return result, nil
}

// GetBaseRateLimitRule 查询限流规则模板详情。
func (s *BaseRateLimitRuleService) GetBaseRateLimitRule(ctx context.Context, req *adminv1.GetBaseRateLimitRuleRequest) (*adminv1.BaseRateLimitRuleForm, error) {
	result, err := s.baseRateLimitRuleCase.GetBaseRateLimitRule(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "查询限流规则详情失败")
	}
	return result, nil
}

// CreateBaseRateLimitRule 创建限流规则模板。
func (s *BaseRateLimitRuleService) CreateBaseRateLimitRule(ctx context.Context, req *adminv1.CreateBaseRateLimitRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRateLimitRuleCase.CreateBaseRateLimitRule(ctx, req.GetBaseRateLimitRule())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "创建限流规则失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseRateLimitRule 更新限流规则模板。
func (s *BaseRateLimitRuleService) UpdateBaseRateLimitRule(ctx context.Context, req *adminv1.UpdateBaseRateLimitRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRateLimitRuleCase.UpdateBaseRateLimitRule(ctx, req.GetBaseRateLimitRule())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "更新限流规则失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseRateLimitRule 删除限流规则模板。
func (s *BaseRateLimitRuleService) DeleteBaseRateLimitRule(ctx context.Context, req *adminv1.DeleteBaseRateLimitRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRateLimitRuleCase.DeleteBaseRateLimitRule(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseRateLimitRule %v", err))
		return nil, errorsx.WrapInternal(err, "删除限流规则失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRateLimitRuleStatus 设置限流规则模板状态。
func (s *BaseRateLimitRuleService) SetBaseRateLimitRuleStatus(ctx context.Context, req *adminv1.SetBaseRateLimitRuleStatusRequest) (*emptypb.Empty, error) {
	err := s.baseRateLimitRuleCase.SetBaseRateLimitRuleStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRateLimitRuleStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置限流规则状态失败")
	}
	return new(emptypb.Empty), nil
}
