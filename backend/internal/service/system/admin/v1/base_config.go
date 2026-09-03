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

// BaseConfigService Admin系统配置服务
type BaseConfigService struct {
	adminv1.UnimplementedBaseConfigServiceServer
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
func (s *BaseConfigService) RefreshBaseConfigCache(ctx context.Context, req *adminv1.RefreshBaseConfigCacheRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.RefreshBaseConfig(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("RefreshBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "刷新缓存失败")
	}
	return new(emptypb.Empty), nil
}

// PageBaseConfig 查询系统配置分页列表
func (s *BaseConfigService) PageBaseConfig(ctx context.Context, req *adminv1.PageBaseConfigRequest) (*adminv1.PageBaseConfigResponse, error) {
	page, err := s.baseConfigCase.PageBaseConfig(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "查询系统配置分页列表失败")
	}

	return page, nil
}

// GetBaseConfig 查询系统配置
func (s *BaseConfigService) GetBaseConfig(ctx context.Context, req *adminv1.GetBaseConfigRequest) (*adminv1.BaseConfigForm, error) {
	config, err := s.baseConfigCase.GetBaseConfig(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "查询系统配置失败")
	}
	return config, nil
}

// GetBaseConfigByKey 按配置键查询隐藏系统配置。
func (s *BaseConfigService) GetBaseConfigByKey(ctx context.Context, req *adminv1.GetBaseConfigByKeyRequest) (*adminv1.BaseConfigValue, error) {
	config, err := s.baseConfigCase.GetBaseConfigByKey(ctx, req.GetKey())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseConfigByKey %v", err))
		return nil, errorsx.WrapInternal(err, "查询隐藏系统配置失败")
	}
	return config, nil
}

// CreateBaseConfig 创建系统配置
func (s *BaseConfigService) CreateBaseConfig(ctx context.Context, req *adminv1.CreateBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.CreateBaseConfig(ctx, req.GetBaseConfig())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "创建系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseConfig 更新系统配置
func (s *BaseConfigService) UpdateBaseConfig(ctx context.Context, req *adminv1.UpdateBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.UpdateBaseConfig(ctx, req.GetBaseConfig())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "更新系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseConfigByKey 按配置键更新隐藏系统配置。
func (s *BaseConfigService) UpdateBaseConfigByKey(ctx context.Context, req *adminv1.UpdateBaseConfigByKeyRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.UpdateBaseConfigByKey(ctx, req.GetKey(), req.GetValueJson())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseConfigByKey %v", err))
		return nil, errorsx.WrapInternal(err, "更新隐藏系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseConfig 删除系统配置
func (s *BaseConfigService) DeleteBaseConfig(ctx context.Context, req *adminv1.DeleteBaseConfigRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.DeleteBaseConfig(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseConfig %v", err))
		return nil, errorsx.WrapInternal(err, "删除系统配置失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseConfigStatus 设置状态
func (s *BaseConfigService) SetBaseConfigStatus(ctx context.Context, req *adminv1.SetBaseConfigStatusRequest) (*emptypb.Empty, error) {
	err := s.baseConfigCase.SetBaseConfigStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseConfigStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
