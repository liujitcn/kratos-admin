package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseRedactRuleService 提供脱敏规则模板管理接口。
type BaseRedactRuleService struct {
	adminv1.UnimplementedBaseRedactRuleServiceServer
	baseRedactRuleCase *biz.BaseRedactRuleCase
}

// NewBaseRedactRuleService 创建脱敏规则模板服务。
func NewBaseRedactRuleService(baseRedactRuleCase *biz.BaseRedactRuleCase) *BaseRedactRuleService {
	return &BaseRedactRuleService{baseRedactRuleCase: baseRedactRuleCase}
}

// OptionBaseRedactRule 查询脱敏规则选项。
func (s *BaseRedactRuleService) OptionBaseRedactRule(ctx context.Context, req *adminv1.OptionBaseRedactRuleRequest) (*commonv1.SelectOptionResponse, error) {
	result, err := s.baseRedactRuleCase.OptionBaseRedactRule(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "查询脱敏规则选项失败")
	}
	return result, nil
}

// PageBaseRedactRule 分页查询脱敏规则模板。
func (s *BaseRedactRuleService) PageBaseRedactRule(ctx context.Context, req *adminv1.PageBaseRedactRuleRequest) (*adminv1.PageBaseRedactRuleResponse, error) {
	result, err := s.baseRedactRuleCase.PageBaseRedactRule(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询脱敏规则失败")
	}
	return result, nil
}

// GetBaseRedactRule 查询脱敏规则模板详情。
func (s *BaseRedactRuleService) GetBaseRedactRule(ctx context.Context, req *adminv1.GetBaseRedactRuleRequest) (*adminv1.BaseRedactRuleForm, error) {
	result, err := s.baseRedactRuleCase.GetBaseRedactRule(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "查询脱敏规则详情失败")
	}
	return result, nil
}

// CreateBaseRedactRule 创建脱敏规则模板。
func (s *BaseRedactRuleService) CreateBaseRedactRule(ctx context.Context, req *adminv1.CreateBaseRedactRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRedactRuleCase.CreateBaseRedactRule(ctx, req.GetBaseRedactRule())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "创建脱敏规则失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseRedactRule 更新脱敏规则模板。
func (s *BaseRedactRuleService) UpdateBaseRedactRule(ctx context.Context, req *adminv1.UpdateBaseRedactRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRedactRuleCase.UpdateBaseRedactRule(ctx, req.GetBaseRedactRule())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "更新脱敏规则失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseRedactRule 删除脱敏规则模板。
func (s *BaseRedactRuleService) DeleteBaseRedactRule(ctx context.Context, req *adminv1.DeleteBaseRedactRuleRequest) (*emptypb.Empty, error) {
	err := s.baseRedactRuleCase.DeleteBaseRedactRule(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseRedactRule %v", err))
		return nil, errorsx.WrapInternal(err, "删除脱敏规则失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseRedactRuleStatus 设置脱敏规则模板状态。
func (s *BaseRedactRuleService) SetBaseRedactRuleStatus(ctx context.Context, req *adminv1.SetBaseRedactRuleStatusRequest) (*emptypb.Empty, error) {
	err := s.baseRedactRuleCase.SetBaseRedactRuleStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseRedactRuleStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置脱敏规则状态失败")
	}
	return new(emptypb.Empty), nil
}
