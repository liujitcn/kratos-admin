package biz

import (
	"context"
	"encoding/json"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gen/field"
)

// BaseConfigCase 配置业务实例
type BaseConfigCase struct {
	*biz.BaseCase
	*data.BaseConfigRepository
	formMapper      *mapper.CopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig]
	mapper          *mapper.CopierMapper[systemadminv1.BaseConfig, models.BaseConfig]
	translationCase *BaseTranslationCase
}

// NewBaseConfigCase 创建配置业务实例
func NewBaseConfigCase(baseCase *biz.BaseCase, baseConfigRepo *data.BaseConfigRepository, translationCase *BaseTranslationCase) *BaseConfigCase {
	return &BaseConfigCase{
		BaseCase:             baseCase,
		BaseConfigRepository: baseConfigRepo,
		formMapper:           mapper.NewCopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig](),
		mapper:               mapper.NewCopierMapper[systemadminv1.BaseConfig, models.BaseConfig](),
		translationCase:      translationCase,
	}
}

// RefreshBaseConfig 刷新配置缓存
func (c *BaseConfigCase) RefreshBaseConfig(ctx context.Context) error {
	sites := []int32{
		_const.BASE_CONFIG_SITE_SYSTEM,
		_const.BASE_CONFIG_SITE_ADMIN,
		_const.BASE_CONFIG_SITE_APP,
	}
	var err error
	for _, site := range sites {
		err = c.refreshBaseConfigSite(ctx, site)
		if err != nil {
			return err
		}
	}
	return nil
}

// PageBaseConfig 分页查询配置
func (c *BaseConfigCase) PageBaseConfig(ctx context.Context, req *systemadminv1.PageBaseConfigRequest) (*systemadminv1.PageBaseConfigResponse, error) {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Site != nil {
		opts = append(opts, repository.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	// 传入名称关键字时，按配置名称模糊匹配。
	if req.GetName() != "" {
		translatedIDs, err := c.translationCase.TranslatedConfigIDsByName(ctx, req.GetName())
		if err != nil {
			return nil, err
		}
		nameCondition := query.Name.Like("%" + req.GetName() + "%")
		if len(translatedIDs) > 0 {
			opts = append(opts, repository.Where(field.Or(nameCondition, query.ID.In(translatedIDs...))))
		} else {
			opts = append(opts, repository.Where(nameCondition))
		}
	}
	if req.Type != nil {
		opts = append(opts, repository.Where(query.Type.Eq(int32(req.GetType()))))
	}
	// 传入键关键字时，按配置键模糊匹配。
	if req.GetKey() != "" {
		opts = append(opts, repository.Where(query.Key.Like("%"+req.GetKey()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.BaseConfig, 0, len(list))
	sources := make(map[int64]dto.ConfigTranslationSource, len(list))
	for _, item := range list {
		sources[item.ID] = dto.ConfigTranslationSource{Name: item.Name, Value: item.Value, Type: item.Type}
	}
	var translations map[int64][]*systemadminv1.BaseTranslation
	translations, err = c.translationCase.ConfigTranslations(ctx, sources)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		baseConfig := c.mapper.ToDTO(item)
		baseConfig.Translations = translations[item.ID]
		resList = append(resList, baseConfig)
	}

	return &systemadminv1.PageBaseConfigResponse{
		BaseConfigs: resList,
		Total:       int32(total),
	}, nil
}

// GetBaseConfig 根据主键查询配置
func (c *BaseConfigCase) GetBaseConfig(ctx context.Context, id int64) (*systemadminv1.BaseConfigForm, error) {
	baseConfig, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseConfig)
	var translations map[int64][]*systemadminv1.BaseTranslation
	translations, err = c.translationCase.ConfigTranslations(ctx, map[int64]dto.ConfigTranslationSource{id: {Name: baseConfig.Name, Value: baseConfig.Value, Type: baseConfig.Type}})
	if err != nil {
		return nil, err
	}
	res.Translations = translations[id]
	return res, nil
}

// CreateBaseConfig 创建配置
func (c *BaseConfigCase) CreateBaseConfig(ctx context.Context, req *systemadminv1.BaseConfigForm) error {
	nameSource := req.GetName()
	valueSource := req.GetValue()
	var sourceLocale, primaryLocale string
	var err error
	var primaryName string
	primaryName, sourceLocale, primaryLocale, err = c.translationCase.NormalizePrimaryText(ctx, nameSource)
	if err != nil {
		return err
	}
	primaryValue := req.GetValue()
	if isTranslatableConfigType(int32(req.GetType())) {
		primaryValue, sourceLocale, primaryLocale, err = c.translationCase.NormalizePrimaryText(ctx, valueSource)
		if err != nil {
			return err
		}
	}
	req.Name = primaryName
	req.Value = primaryValue
	translations := appendConfigSourceTranslations(req.GetTranslations(), sourceLocale, primaryLocale, nameSource, valueSource, isTranslatableConfigType(int32(req.GetType())), true)
	entity := c.formMapper.ToEntity(req)
	err = c.Create(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.translationCase.SaveConfigTranslations(ctx, entity.ID, dto.ConfigTranslationSource{Name: entity.Name, Value: entity.Value, Type: entity.Type}, translations)
	if err != nil {
		return err
	}
	err = c.refreshBaseConfigSite(ctx, entity.Site)
	if err != nil {
		return err
	}
	c.translationCase.EnqueueTranslation(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, entity.ID)
	if isTranslatableConfigType(entity.Type) {
		c.translationCase.EnqueueTranslation(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, entity.ID)
	}
	return nil
}

// UpdateBaseConfig 更新配置
func (c *BaseConfigCase) UpdateBaseConfig(ctx context.Context, req *systemadminv1.BaseConfigForm) error {
	oldConfig, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	nameSource := oldConfig.Name
	valueSource := req.GetValue()
	primaryName := oldConfig.Name
	var primaryLocale string
	primaryLocale, err = c.translationCase.PrimaryLocale(ctx)
	if err != nil {
		return err
	}
	sourceLocale := primaryLocale
	primaryValue := req.GetValue()
	if isTranslatableConfigType(int32(req.GetType())) {
		primaryValue, sourceLocale, primaryLocale, err = c.translationCase.NormalizePrimaryText(ctx, valueSource)
		if err != nil {
			return err
		}
	}
	req.Name = primaryName
	req.Value = primaryValue
	translations := appendConfigSourceTranslations(req.GetTranslations(), sourceLocale, primaryLocale, nameSource, valueSource, isTranslatableConfigType(int32(req.GetType())), false)
	entity := c.formMapper.ToEntity(req)
	err = c.UpdateByID(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.translationCase.SaveConfigTranslations(ctx, entity.ID, dto.ConfigTranslationSource{Name: entity.Name, Value: entity.Value, Type: entity.Type}, translations)
	if err != nil {
		return err
	}

	err = c.refreshBaseConfigSite(ctx, oldConfig.Site)
	if err != nil {
		return err
	}
	if oldConfig.Site != entity.Site {
		err = c.refreshBaseConfigSite(ctx, entity.Site)
		if err != nil {
			return err
		}
	}
	c.translationCase.EnqueueTranslation(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, entity.ID)
	if isTranslatableConfigType(entity.Type) {
		c.translationCase.EnqueueTranslation(systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, entity.ID)
	}
	return nil
}

// appendConfigSourceTranslations 保留当前请求语言的配置名称和值。
func appendConfigSourceTranslations(translations []*systemadminv1.BaseTranslation, sourceLocale, primaryLocale, sourceName, sourceValue string, translatableValue, includeNameSource bool) []*systemadminv1.BaseTranslation {
	if sourceLocale == primaryLocale {
		return translations
	}
	result := make([]*systemadminv1.BaseTranslation, 0, len(translations)+1)
	nameAdded := false
	valueAdded := false
	for _, translation := range translations {
		if !coreLocale.IsSupported(translation.GetLocale()) || coreLocale.Normalize(translation.GetLocale()) != sourceLocale {
			result = append(result, translation)
			continue
		}
		switch translation.GetTargetType() {
		case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME:
			if !includeNameSource {
				result = append(result, translation)
				continue
			}
			if !nameAdded {
				result = append(result, &systemadminv1.BaseTranslation{TargetType: translation.GetTargetType(), Locale: sourceLocale, Name: sourceName})
				nameAdded = true
			}
		case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
			if translatableValue && !valueAdded {
				result = append(result, &systemadminv1.BaseTranslation{TargetType: translation.GetTargetType(), TargetId: translation.GetTargetId(), Locale: sourceLocale, Name: sourceValue})
				valueAdded = true
			}
		}
	}
	if includeNameSource && !nameAdded {
		result = append(result, &systemadminv1.BaseTranslation{TargetType: systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, Locale: sourceLocale, Name: sourceName})
	}
	if translatableValue && !valueAdded {
		result = append(result, &systemadminv1.BaseTranslation{TargetType: systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, Locale: sourceLocale, Name: sourceValue})
	}
	return result
}

// DeleteBaseConfig 删除配置
func (c *BaseConfigCase) DeleteBaseConfig(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	list, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}

	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		return err
	}
	err = c.translationCase.DeleteConfigTranslations(ctx, ids)
	if err != nil {
		return err
	}

	sites := make(map[int32]struct{}, len(list))
	for _, item := range list {
		sites[item.Site] = struct{}{}
	}
	for site := range sites {
		err = c.refreshBaseConfigSite(ctx, site)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBaseConfigStatus 设置配置状态
func (c *BaseConfigCase) SetBaseConfigStatus(ctx context.Context, req *systemadminv1.SetBaseConfigStatusRequest) error {
	err := c.UpdateByID(ctx, &models.BaseConfig{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
	if err != nil {
		return err
	}

	var baseConfig *models.BaseConfig
	baseConfig, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	err = c.refreshBaseConfigSite(ctx, baseConfig.Site)
	if err != nil {
		return err
	}
	return nil
}

// refreshBaseConfigSite 查询并缓存指定站点的启用配置。
func (c *BaseConfigCase) refreshBaseConfigSite(ctx context.Context, site int32) error {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Site.Eq(site)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	opts = append(opts, repository.Order(query.ID.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}

	configs := make([]*basev1.ConfigItem, 0, len(list))
	for _, item := range list {
		configs = append(configs, &basev1.ConfigItem{
			Id:    item.ID,
			Key:   item.Key,
			Value: item.Value,
		})
	}
	payload, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	return sdk.Runtime.GetCache().Set(_const.BaseConfigCacheKey(site), string(payload), _const.BASE_CONFIG_CACHE_EXPIRE)
}
