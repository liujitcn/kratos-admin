package base

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

	"github.com/go-kratos/kratos/v3/log"
)

// ConfigService 系统配置公共服务
type ConfigService struct {
	basev1.UnimplementedConfigServiceServer
	configCase *biz.ConfigCase
}

// NewConfigService 创建系统配置公共服务
func NewConfigService(
	configCase *biz.ConfigCase,
) *ConfigService {
	var ss = ConfigService{
		configCase: configCase}
	return &ss
}

// GetConfig 获取系统配置
func (s *ConfigService) GetConfig(ctx context.Context, req *basev1.GetConfigRequest) (*basev1.GetConfigResponse, error) {
	resp, err := s.configCase.GetConfig(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetConfig %v", err))
		return nil, errorsx.WrapInternal(err, "获取系统配置失败")
	}

	return resp, nil
}
