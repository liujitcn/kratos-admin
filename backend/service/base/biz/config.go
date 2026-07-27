package biz

import (
	"context"
	"encoding/json"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-kit/sdk"

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
	site := int32(req.GetSite())
	var cached string
	var err error
	cached, err = sdk.Runtime.GetCache().Get(_const.BaseConfigCacheKey(site))
	if err == nil {
		configs := make([]*basev1.ConfigItem, 0)
		err = json.Unmarshal([]byte(cached), &configs)
		if err == nil {
			return &basev1.GetConfigResponse{Configs: configs}, nil
		}
	}

	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Site.Eq(site)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	opts = append(opts, repository.Order(query.ID.Asc()))
	var list []*models.BaseConfig
	list, err = c.List(ctx, opts...)
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
	response := &basev1.GetConfigResponse{
		Configs: configs,
	}
	var payload []byte
	payload, err = json.Marshal(configs)
	if err != nil {
		log.Error(fmt.Sprintf("MarshalBaseConfigCache %v", err))
		return response, nil
	}
	err = sdk.Runtime.GetCache().Set(_const.BaseConfigCacheKey(site), string(payload), _const.BASE_CONFIG_CACHE_EXPIRE)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseConfigCache %v", err))
	}
	return response, nil
}
