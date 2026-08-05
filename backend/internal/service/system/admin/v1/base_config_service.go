package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseConfigService Admin系统配置服务
type BaseConfigService struct {
	systemadminv1.UnimplementedBaseConfigServiceServer
	baseConfigCase *biz.BaseConfigCase
}

// NewBaseConfigService 创建Admin系统配置服务
func NewBaseConfigService(
	configCase *biz.BaseConfigCase,
) *BaseConfigService {
	return &BaseConfigService{
		baseConfigCase: configCase,
	}
}

// RefreshBaseConfigCache 刷新缓存
func (s *BaseConfigService) RefreshBaseConfigCache(ctx context.Context, req *systemadminv1.RefreshBaseConfigCacheRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.RefreshBaseConfig(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("RefreshBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "刷新缓存失败")
	}
	return new(emptypb.Empty), nil
}

// PageBaseConfig 查询系统配置分页列表
func (s *BaseConfigService) PageBaseConfig(ctx context.Context, req *systemadminv1.PageBaseConfigRequest) (*systemadminv1.PageBaseConfigResponse, error) {
	page, err := s.baseConfigCase.PageBaseConfig(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "查询系统配置分页列表失败")
	}

	return page, nil
}

// GetBaseConfig 查询系统配置
func (s *BaseConfigService) GetBaseConfig(ctx context.Context, req *systemadminv1.GetBaseConfigRequest) (*systemadminv1.BaseConfigForm, error) {
	config, err := s.baseConfigCase.GetBaseConfig(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "查询系统配置失败")
	}
	return config, nil
}

// CreateBaseConfig 创建系统配置
func (s *BaseConfigService) CreateBaseConfig(ctx context.Context, req *systemadminv1.CreateBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.CreateBaseConfig(ctx, req.GetBaseConfig())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "创建系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseConfig 更新系统配置
func (s *BaseConfigService) UpdateBaseConfig(ctx context.Context, req *systemadminv1.UpdateBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.UpdateBaseConfig(ctx, req.GetBaseConfig())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "更新系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseConfig 删除系统配置
func (s *BaseConfigService) DeleteBaseConfig(ctx context.Context, req *systemadminv1.DeleteBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.DeleteBaseConfig(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "删除系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseConfigStatus 设置状态
func (s *BaseConfigService) SetBaseConfigStatus(ctx context.Context, req *systemadminv1.SetBaseConfigStatusRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.SetBaseConfigStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseConfigStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
