package biz

import (
	"context"
	"encoding/json"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
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
	tx                  data.Transaction
	baseTranslationCase *BaseTranslationCase
	formMapper          *mapper.CopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig]
	mapper              *mapper.CopierMapper[systemadminv1.BaseConfig, models.BaseConfig]
}

// NewBaseConfigCase 创建配置业务实例
func NewBaseConfigCase(baseCase *biz.BaseCase, tx data.Transaction, baseConfigRepo *data.BaseConfigRepository, baseTranslationCase *BaseTranslationCase) *BaseConfigCase {
	return &BaseConfigCase{
		BaseCase:             baseCase,
		tx:                   tx,
		BaseConfigRepository: baseConfigRepo,
		baseTranslationCase:  baseTranslationCase,
		formMapper:           mapper.NewCopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig](),
		mapper:               mapper.NewCopierMapper[systemadminv1.BaseConfig, models.BaseConfig](),
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
		targetIds, err := c.baseTranslationCase.GetTargetIdsByName(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, req.GetName())
		if err != nil {
			return nil, err
		}
		nameCondition := query.Name.Like("%" + req.GetName() + "%")
		if len(targetIds) > 0 {
			opts = append(opts, repository.Where(field.Or(nameCondition, query.ID.In(targetIds...))))
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
	targetIds := make([]int64, 0, len(list))
	for _, item := range list {
		targetIds = append(targetIds, item.ID)
	}
	var translations map[int64][]*systemadminv1.BaseTranslation
	translations, err = c.baseTranslationCase.GetBaseTranslationMapByTargetType(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, targetIds)
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
	var nameTranslations, valueTranslations map[int64][]*systemadminv1.BaseTranslation
	nameTranslations, err = c.baseTranslationCase.GetBaseTranslationMapByTargetType(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, []int64{id})
	if err != nil {
		return nil, err
	}
	if isTranslatableConfigType(baseConfig.Type) {
		valueTranslations, err = c.baseTranslationCase.GetBaseTranslationMapByTargetType(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, []int64{id})
		if err != nil {
			return nil, err
		}
	}
	res.NameTranslations = nameTranslations[id]
	res.ValueTranslations = valueTranslations[id]
	return res, nil
}

// CreateBaseConfig 创建配置
func (c *BaseConfigCase) CreateBaseConfig(ctx context.Context, req *systemadminv1.BaseConfigForm) error {
	entity := c.formMapper.ToEntity(req)
	err := c.Create(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.saveBaseTranslation(ctx, req, entity)
	if err != nil {
		return err
	}
	err = c.refreshBaseConfigSite(ctx, entity.Site)
	if err != nil {
		return err
	}
	return nil
}

// UpdateBaseConfig 更新配置
func (c *BaseConfigCase) UpdateBaseConfig(ctx context.Context, req *systemadminv1.BaseConfigForm) error {
	oldConfig, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	entity := c.formMapper.ToEntity(req)
	err = c.UpdateByID(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.saveBaseTranslation(ctx, req, entity)
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
	return nil
}

// DeleteBaseConfig 删除配置
func (c *BaseConfigCase) DeleteBaseConfig(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	list, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.deleteBaseTranslation(ctx, ids)
	})
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

// saveBaseTranslation 保存翻译信息
func (c *BaseConfigCase) saveBaseTranslation(ctx context.Context, req *systemadminv1.BaseConfigForm, entity *models.BaseConfig) error {
	err := c.baseTranslationCase.SaveBaseTranslation(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, entity.ID, entity.Name, req.GetNameTranslations(), func(ctx context.Context, name string) error {
		return c.UpdateByID(ctx, &models.BaseConfig{
			ID:   entity.ID,
			Name: name,
		})
	})
	if err != nil {
		return err
	}
	if !isTranslatableConfigType(entity.Type) {
		return nil
	}
	return c.baseTranslationCase.SaveBaseTranslation(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, entity.ID, entity.Value, req.GetValueTranslations(), func(ctx context.Context, value string) error {
		return c.UpdateByID(ctx, &models.BaseConfig{
			ID:    entity.ID,
			Value: value,
		})
	})
}

// deleteBaseTranslation 删除翻译信息
func (c *BaseConfigCase) deleteBaseTranslation(ctx context.Context, ids []int64) error {
	err := c.baseTranslationCase.DeleteBaseTranslation(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, ids)
	if err != nil {
		return err
	}
	return c.baseTranslationCase.DeleteBaseTranslation(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, ids)
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
	var payload []byte
	payload, err = json.Marshal(configs)
	if err != nil {
		return err
	}
	return sdk.Runtime.GetCache().Set(_const.BaseConfigCacheKey(site), string(payload), _const.BASE_CONFIG_CACHE_EXPIRE)
}

// isTranslatableConfigType 判断配置值是否支持机器翻译和动态译文。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT) || configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
}
