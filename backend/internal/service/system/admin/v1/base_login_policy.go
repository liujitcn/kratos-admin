package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseLoginPolicyService 提供 Admin 登录策略接口。
type BaseLoginPolicyService struct {
	adminv1.UnimplementedBaseLoginPolicyServiceServer
	baseLoginPolicyCase *biz.BaseLoginPolicyCase
}

// NewBaseLoginPolicyService 创建登录策略服务。
func NewBaseLoginPolicyService(baseLoginPolicyCase *biz.BaseLoginPolicyCase) *BaseLoginPolicyService {
	return &BaseLoginPolicyService{baseLoginPolicyCase: baseLoginPolicyCase}
}

// PageBaseLoginPolicy 分页查询登录策略。
func (s *BaseLoginPolicyService) PageBaseLoginPolicy(ctx context.Context, req *adminv1.PageBaseLoginPolicyRequest) (*adminv1.PageBaseLoginPolicyResponse, error) {
	res, err := s.baseLoginPolicyCase.PageBaseLoginPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询登录策略失败")
	}
	return res, nil
}

// GetBaseLoginPolicy 查询登录策略详情。
func (s *BaseLoginPolicyService) GetBaseLoginPolicy(ctx context.Context, req *adminv1.GetBaseLoginPolicyRequest) (*adminv1.BaseLoginPolicyForm, error) {
	res, err := s.baseLoginPolicyCase.GetBaseLoginPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录策略详情失败")
	}
	return res, nil
}

// CreateBaseLoginPolicy 创建登录策略。
func (s *BaseLoginPolicyService) CreateBaseLoginPolicy(ctx context.Context, req *adminv1.CreateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	res, err := s.baseLoginPolicyCase.CreateBaseLoginPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "创建登录策略失败")
	}
	return res, nil
}

// UpdateBaseLoginPolicy 更新登录策略。
func (s *BaseLoginPolicyService) UpdateBaseLoginPolicy(ctx context.Context, req *adminv1.UpdateBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	res, err := s.baseLoginPolicyCase.UpdateBaseLoginPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "更新登录策略失败")
	}
	return res, nil
}

// DeleteBaseLoginPolicy 删除登录策略。
func (s *BaseLoginPolicyService) DeleteBaseLoginPolicy(ctx context.Context, req *adminv1.DeleteBaseLoginPolicyRequest) (*emptypb.Empty, error) {
	res, err := s.baseLoginPolicyCase.DeleteBaseLoginPolicy(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "删除登录策略失败")
	}
	return res, nil
}

// SetBaseLoginPolicyStatus 设置登录策略状态。
func (s *BaseLoginPolicyService) SetBaseLoginPolicyStatus(ctx context.Context, req *adminv1.SetBaseLoginPolicyStatusRequest) (*emptypb.Empty, error) {
	res, err := s.baseLoginPolicyCase.SetBaseLoginPolicyStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseLoginPolicyStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置登录策略状态失败")
	}
	return res, nil
}
