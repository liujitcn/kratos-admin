package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseUserService Admin用户管理服务
type BaseUserService struct {
	adminv1.UnimplementedBaseUserServiceServer
	baseUserCase *biz.BaseUserCase
}

// NewBaseUserService 创建Admin用户管理服务
func NewBaseUserService(
	userCase *biz.BaseUserCase,
) *BaseUserService {
	return &BaseUserService{
		baseUserCase: userCase,
	}
}

// OptionBaseUser 查询用户下拉选择
func (s *BaseUserService) OptionBaseUser(ctx context.Context, req *adminv1.OptionBaseUserRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseUserCase.OptionBaseUser(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询用户下拉选择失败")
	}
	return list, nil
}

// ListBaseUser 查询用户列表。
func (s *BaseUserService) ListBaseUser(ctx context.Context, req *adminv1.ListBaseUserRequest) (*adminv1.ListBaseUserResponse, error) {
	list, err := s.baseUserCase.ListBaseUser(ctx, req.GetIds())
	if err != nil {
		log.Error(fmt.Sprintf("ListBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询用户列表失败")
	}
	return list, nil
}

// SummaryBaseUser 汇总用户注册数据。
func (s *BaseUserService) SummaryBaseUser(ctx context.Context, req *adminv1.SummaryBaseUserRequest) (*adminv1.SummaryBaseUserResponse, error) {
	summary, err := s.baseUserCase.SummaryBaseUser(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "汇总用户注册数据失败")
	}
	return summary, nil
}

// PageBaseUser 查询用户分页列表
func (s *BaseUserService) PageBaseUser(ctx context.Context, req *adminv1.PageBaseUserRequest) (*adminv1.PageBaseUserResponse, error) {
	page, err := s.baseUserCase.PageBaseUser(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询用户分页列表失败")
	}
	return page, nil
}

// GetBaseUser 查询用户
func (s *BaseUserService) GetBaseUser(ctx context.Context, req *adminv1.GetBaseUserRequest) (*adminv1.BaseUserForm, error) {
	baseUser, err := s.baseUserCase.GetBaseUser(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "查询用户失败")
	}
	return baseUser, nil
}

// CreateBaseUser 创建用户
func (s *BaseUserService) CreateBaseUser(ctx context.Context, req *adminv1.CreateBaseUserRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.CreateBaseUser(ctx, req.GetBaseUser())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "创建用户失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseUser 更新用户
func (s *BaseUserService) UpdateBaseUser(ctx context.Context, req *adminv1.UpdateBaseUserRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.UpdateBaseUser(ctx, req.GetBaseUser())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "更新用户失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseUser 删除用户
func (s *BaseUserService) DeleteBaseUser(ctx context.Context, req *adminv1.DeleteBaseUserRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.DeleteBaseUser(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseUser %v", err))
		return nil, errorsx.WrapInternal(err, "删除用户失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseUserStatus 设置状态
func (s *BaseUserService) SetBaseUserStatus(ctx context.Context, req *adminv1.SetBaseUserStatusRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.SetBaseUserStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseUserStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}

// ResetBaseUserPassword 重置密码
func (s *BaseUserService) ResetBaseUserPassword(ctx context.Context, req *adminv1.ResetBaseUserPasswordRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.ResetBaseUserPassword(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ResetBaseUserPassword %v", err))
		return nil, errorsx.WrapInternal(err, "重置密码失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseUserAppRole 设置基础用户应用端角色。
func (s *BaseUserService) SetBaseUserAppRole(ctx context.Context, req *adminv1.SetBaseUserAppRoleRequest) (*emptypb.Empty, error) {
	err := s.baseUserCase.SetBaseUserAppRole(ctx, req.GetUserId(), req.GetRoleCode())
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseUserAppRole %v", err))
		return nil, errorsx.WrapInternal(err, "设置基础用户应用端角色失败")
	}
	return new(emptypb.Empty), nil
}
