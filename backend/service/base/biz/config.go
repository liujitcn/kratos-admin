package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repository"
)

// ConfigCase 处理基础配置查询业务。
type ConfigCase struct {
	*data.BaseConfigRepository
}

// NewConfigCase 创建配置业务实例。
func NewConfigCase(baseConfigRepo *data.BaseConfigRepository) *ConfigCase {
	return &ConfigCase{
		BaseConfigRepository: baseConfigRepo,
	}
}

// GetConfig 查询系统配置。
func (c *ConfigCase) GetConfig(ctx context.Context, req *basev1.GetConfigRequest) (*basev1.GetConfigResponse, error) {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Site.Eq(int32(req.GetSite()))))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	configs := make([]*basev1.ConfigItem, 0, len(list))
	for _, item := range list {
		configs = append(configs, &basev1.ConfigItem{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return &basev1.GetConfigResponse{
		Configs: configs,
	}, nil
}
