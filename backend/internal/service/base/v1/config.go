package base

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

	"github.com/go-kratos/kratos/v3/log"
)

// ConfigService 系统配置公共服务
type ConfigService struct {
	basev1.UnimplementedConfigServiceServer
	configCase *biz.ConfigCase
	aiClient   *model.ResponsesClient
}

// NewConfigService 创建系统配置公共服务
func NewConfigService(
	configCase *biz.ConfigCase,
	aiClient *model.ResponsesClient,
) *ConfigService {
	var ss = ConfigService{
		configCase: configCase,
		aiClient:   aiClient,
	}
	return &ss
}

// GetConfig 获取系统配置
func (s *ConfigService) GetConfig(ctx context.Context, req *basev1.GetConfigRequest) (*basev1.GetConfigResponse, error) {
	resp, err := s.configCase.GetConfig(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetConfig %v", err))
		return nil, errorsx.WrapInternal(err, "获取系统配置失败")
	}
	resp.AiEnabled = s.aiClient != nil && s.aiClient.Enabled()

	return resp, nil
}
