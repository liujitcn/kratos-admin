package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-kit/sdk"

	"github.com/liujitcn/gorm-kit/repository"
)

// ConfigCase 处理基础配置查询业务。
type ConfigCase struct {
	*data.BaseConfigRepository
	configTranslationRepo *data.BaseConfigTranslationRepository
	draftConfig           systemConfig.TranslationDraftConfig
}

// NewConfigCase 创建配置业务实例。
func NewConfigCase(baseConfigRepo *data.BaseConfigRepository, configTranslationRepo *data.BaseConfigTranslationRepository, draftConfig systemConfig.TranslationDraftConfig) *ConfigCase {
	return &ConfigCase{
		BaseConfigRepository:  baseConfigRepo,
		configTranslationRepo: configTranslationRepo,
		draftConfig:           draftConfig,
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
		if err == nil && runtimeConfigIDsPresent(configs) {
			var localized []*basev1.ConfigItem
			localized, err = c.localizeRuntimeConfigValues(ctx, configs)
			if err != nil {
				return nil, err
			}
			return &basev1.GetConfigResponse{Configs: appendI18nRuntimeConfig(localized, c.draftConfig.Enabled)}, nil
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
			Id:    item.ID,
			Key:   item.Key,
			Value: item.Value,
		})
	}
	var localized []*basev1.ConfigItem
	localized, err = c.localizeRuntimeConfigValues(ctx, configs)
	if err != nil {
		return nil, err
	}
	response := &basev1.GetConfigResponse{
		Configs: appendI18nRuntimeConfig(localized, c.draftConfig.Enabled),
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

// localizeRuntimeConfigValues 将当前语言已审核的文本配置值覆盖到运行时结果。
func (c *ConfigCase) localizeRuntimeConfigValues(ctx context.Context, configs []*basev1.ConfigItem) ([]*basev1.ConfigItem, error) {
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(configs) == 0 {
		return configs, nil
	}
	configIDs := make([]int64, 0, len(configs))
	for _, item := range configs {
		if item.GetId() > 0 {
			configIDs = append(configIDs, item.GetId())
		}
	}
	if len(configIDs) == 0 {
		return configs, nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.In(configIDs...)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq("value")), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	if err != nil {
		return nil, err
	}
	values := make(map[int64]string, len(rows))
	for _, row := range rows {
		values[row.ConfigID] = row.Text
	}
	localized := make([]*basev1.ConfigItem, 0, len(configs))
	for _, item := range configs {
		copyItem := *item
		if translated := values[item.GetId()]; translated != "" {
			copyItem.Value = translated
		}
		localized = append(localized, &copyItem)
	}
	return localized, nil
}

// runtimeConfigIDsPresent 判断缓存是否包含配置主键，旧缓存不满足时回源刷新。
func runtimeConfigIDsPresent(configs []*basev1.ConfigItem) bool {
	for _, item := range configs {
		if item.GetId() == 0 {
			return false
		}
	}
	return true
}

// appendI18nRuntimeConfig 将部署级翻译草稿开关附加到公共配置结果。
func appendI18nRuntimeConfig(configs []*basev1.ConfigItem, enabled bool) []*basev1.ConfigItem {
	result := make([]*basev1.ConfigItem, 0, len(configs)+1)
	for _, item := range configs {
		if item.GetKey() != _const.I18N_TRANSLATION_DRAFT_CONFIG_KEY {
			result = append(result, item)
		}
	}
	return append(result, &basev1.ConfigItem{Key: _const.I18N_TRANSLATION_DRAFT_CONFIG_KEY, Value: strconv.FormatBool(enabled)})
}
