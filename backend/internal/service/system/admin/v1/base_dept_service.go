package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseDeptService Admin部门服务
type BaseDeptService struct {
	adminv1.UnimplementedBaseDeptServiceServer
	baseDeptCase *biz.BaseDeptCase
}

// NewBaseDeptService 创建Admin部门服务
func NewBaseDeptService(
	deptCase *biz.BaseDeptCase,
) *BaseDeptService {
	return &BaseDeptService{
		baseDeptCase: deptCase,
	}
}

// OptionBaseDept 查询部门树形选择
func (s *BaseDeptService) OptionBaseDept(ctx context.Context, req *adminv1.OptionBaseDeptRequest) (*commonv1.TreeOptionResponse, error) {
	tree, err := s.baseDeptCase.OptionBaseDept(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "查询部门树形选择失败")
	}
	return tree, nil
}

// TreeBaseDept 查询部门树形列表
func (s *BaseDeptService) TreeBaseDept(ctx context.Context, req *adminv1.TreeBaseDeptRequest) (*adminv1.TreeBaseDeptResponse, error) {
	tree, err := s.baseDeptCase.TreeBaseDept(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TreeBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "查询部门树形列表失败")
	}

	return tree, nil
}

// GetBaseDept 查询部门
func (s *BaseDeptService) GetBaseDept(ctx context.Context, req *adminv1.GetBaseDeptRequest) (*adminv1.BaseDeptForm, error) {
	baseDept, err := s.baseDeptCase.GetBaseDept(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "查询部门失败")
	}
	return baseDept, nil
}

// CreateBaseDept 创建部门
func (s *BaseDeptService) CreateBaseDept(ctx context.Context, req *adminv1.CreateBaseDeptRequest) (*emptypb.Empty, error) {
	err := s.baseDeptCase.CreateBaseDept(ctx, req.GetBaseDept())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "创建部门失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseDept 更新部门
func (s *BaseDeptService) UpdateBaseDept(ctx context.Context, req *adminv1.UpdateBaseDeptRequest) (*emptypb.Empty, error) {
	err := s.baseDeptCase.UpdateBaseDept(ctx, req.GetBaseDept())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "更新部门失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseDept 删除部门
func (s *BaseDeptService) DeleteBaseDept(ctx context.Context, req *adminv1.DeleteBaseDeptRequest) (*emptypb.Empty, error) {
	err := s.baseDeptCase.DeleteBaseDept(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseDept %v", err))
		return nil, errorsx.WrapInternal(err, "删除部门失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseDeptStatus 设置状态
func (s *BaseDeptService) SetBaseDeptStatus(ctx context.Context, req *adminv1.SetBaseDeptStatusRequest) (*emptypb.Empty, error) {
	err := s.baseDeptCase.SetBaseDeptStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseDeptStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
