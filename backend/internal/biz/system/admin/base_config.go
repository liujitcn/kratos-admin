package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"buf.build/go/protovalidate"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// BaseConfigCase 配置业务实例
type BaseConfigCase struct {
	*biz.BaseCase
	*data.BaseConfigRepository
	tx           data.Transaction
	baseI18nCase *BaseI18nCase
	formMapper   *mapper.CopierMapper[adminv1.BaseConfigForm, models.BaseConfig]
	mapper       *mapper.CopierMapper[adminv1.BaseConfig, models.BaseConfig]
}

// NewBaseConfigCase 创建配置业务实例
func NewBaseConfigCase(baseCase *biz.BaseCase, tx data.Transaction, baseConfigRepo *data.BaseConfigRepository, baseI18nCase *BaseI18nCase) *BaseConfigCase {
	return &BaseConfigCase{
		BaseCase:             baseCase,
		tx:                   tx,
		BaseConfigRepository: baseConfigRepo,
		baseI18nCase:         baseI18nCase,
		formMapper:           mapper.NewCopierMapper[adminv1.BaseConfigForm, models.BaseConfig](),
		mapper:               mapper.NewCopierMapper[adminv1.BaseConfig, models.BaseConfig](),
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
func (c *BaseConfigCase) PageBaseConfig(ctx context.Context, req *adminv1.PageBaseConfigRequest) (*adminv1.PageBaseConfigResponse, error) {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Site != nil {
		opts = append(opts, repository.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	var err error
	// 传入名称关键字时，按配置名称模糊匹配。
	if req.GetName() != "" {
		var targetIds []int64
		targetIds, err = c.baseI18nCase.GetTargetIdsByName(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME, req.GetName())
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

	var list []*models.BaseConfig
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseConfig, 0, len(list))
	targetIds := make([]int64, 0, len(list))
	for _, item := range list {
		targetIds = append(targetIds, item.ID)
	}
	var i18ns map[int64][]*adminv1.BaseI18n
	i18ns, err = c.baseI18nCase.GetBaseI18nMapByTargetType(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME, targetIds)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		baseConfig := c.mapper.ToDTO(item)
		if item.Type == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
			baseConfig.Value = ""
		}
		baseConfig.I18ns = i18ns[item.ID]
		resList = append(resList, baseConfig)
	}

	return &adminv1.PageBaseConfigResponse{
		BaseConfigs: resList,
		Total:       int32(total),
	}, nil
}

// GetBaseConfig 根据主键查询配置
func (c *BaseConfigCase) GetBaseConfig(ctx context.Context, id int64) (*adminv1.BaseConfigForm, error) {
	baseConfig, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseConfig)
	if baseConfig.Type == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
		res.Value, err = runtimeconfig.RedactJSON(baseConfig.Key, baseConfig.Value)
		if err != nil {
			return nil, errorsx.Internal("脱敏系统配置失败").WithCause(err)
		}
	}
	var nameI18ns, valueI18ns map[int64][]*adminv1.BaseI18n
	nameI18ns, err = c.baseI18nCase.GetBaseI18nMapByTargetType(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME, []int64{id})
	if err != nil {
		return nil, err
	}
	if isTranslatableConfigType(baseConfig.Type) {
		valueI18ns, err = c.baseI18nCase.GetBaseI18nMapByTargetType(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_VALUE, []int64{id})
		if err != nil {
			return nil, err
		}
	}
	res.NameI18ns = nameI18ns[id]
	res.ValueI18ns = valueI18ns[id]
	return res, nil
}

// CreateBaseConfig 创建配置并校验表单类型的结构和值域。
func (c *BaseConfigCase) CreateBaseConfig(ctx context.Context, req *adminv1.BaseConfigForm) error {
	entity := c.formMapper.ToEntity(req)
	var err error
	err = validateFormConfig(entity, nil)
	if err != nil {
		return err
	}
	err = c.Create(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.saveBaseI18n(ctx, req, entity)
	if err != nil {
		return err
	}
	err = c.refreshBaseConfigSite(ctx, entity.Site)
	if err != nil {
		return err
	}
	return nil
}

// UpdateBaseConfig 更新配置，合并表单敏感值并刷新运行缓存。
func (c *BaseConfigCase) UpdateBaseConfig(ctx context.Context, req *adminv1.BaseConfigForm) error {
	oldConfig, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	entity := c.formMapper.ToEntity(req)
	err = validateFormConfig(entity, oldConfig)
	if err != nil {
		return err
	}
	err = c.UpdateByID(ctx, entity)
	if err != nil {
		// 命中配置键唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置的配置键重复", "base_config", "", "unique_base_config").WithCause(err)
		}
		return err
	}
	err = c.saveBaseI18n(ctx, req, entity)
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

	for _, item := range list {
		if item.Type == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
			return errorsx.WithMessageKey(errorsx.InvalidArgument("表单配置不允许删除或停用"), "system.admin.base.config.form.protected", nil)
		}
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.deleteBaseI18n(ctx, ids)
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
func (c *BaseConfigCase) SetBaseConfigStatus(ctx context.Context, req *adminv1.SetBaseConfigStatusRequest) error {
	baseConfig, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if baseConfig.Type == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) && req.GetStatus() != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.WithMessageKey(errorsx.InvalidArgument("表单配置不允许删除或停用"), "system.admin.base.config.form.protected", nil)
	}
	err = c.UpdateByID(ctx, &models.BaseConfig{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
	if err != nil {
		return err
	}

	baseConfig = nil
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

// saveBaseI18n 保存翻译信息
func (c *BaseConfigCase) saveBaseI18n(ctx context.Context, req *adminv1.BaseConfigForm, entity *models.BaseConfig) error {
	err := c.baseI18nCase.SaveBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME, entity.ID, entity.Name, req.GetNameI18ns(), func(ctx context.Context, name string) error {
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
	return c.baseI18nCase.SaveBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_VALUE, entity.ID, entity.Value, req.GetValueI18ns(), func(ctx context.Context, value string) error {
		return c.UpdateByID(ctx, &models.BaseConfig{
			ID:    entity.ID,
			Value: value,
		})
	})
}

// deleteBaseI18n 删除翻译信息
func (c *BaseConfigCase) deleteBaseI18n(ctx context.Context, ids []int64) error {
	err := c.baseI18nCase.DeleteBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME, ids)
	if err != nil {
		return err
	}
	return c.baseI18nCase.DeleteBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_VALUE, ids)
}

// refreshBaseConfigSite 查询并缓存指定站点的启用配置。
func (c *BaseConfigCase) refreshBaseConfigSite(ctx context.Context, site int32) error {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.Site.Eq(site)))
	opts = append(opts, repository.Where(query.Type.Neq(int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM))))
	opts = append(opts, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)))
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
	err = c.Cache.Set(_const.BaseConfigCacheKey(site), string(payload), _const.BASE_CONFIG_CACHE_EXPIRE)
	if err != nil {
		return err
	}
	if site == _const.BASE_CONFIG_SITE_SYSTEM {
		return c.refreshFormBaseConfig(ctx)
	}
	return nil
}

// refreshFormBaseConfig 初始化并刷新表单系统配置缓存。
func (c *BaseConfigCase) refreshFormBaseConfig(ctx context.Context) error {
	query := c.Query(ctx).BaseConfig
	var err error
	for _, key := range runtimeconfig.Keys() {
		var list []*models.BaseConfig
		list, err = c.List(ctx,
			repository.Where(query.Site.Eq(_const.BASE_CONFIG_SITE_SYSTEM)),
			repository.Where(query.Key.Eq(key)),
		)
		if err != nil {
			return fmt.Errorf("查询表单系统配置失败: %w", err)
		}
		var entity *models.BaseConfig
		if len(list) == 0 {
			var value string
			value, err = runtimeconfig.DefaultJSON(key)
			if err != nil {
				return err
			}
			entity = &models.BaseConfig{
				Site:   _const.BASE_CONFIG_SITE_SYSTEM,
				Name:   key,
				Type:   int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM),
				Key:    key,
				Value:  value,
				Status: coreconst.STATUS_STATUS_ENABLE,
			}
			err = c.Create(ctx, entity)
			if err != nil {
				return fmt.Errorf("初始化表单系统配置失败: %w", err)
			}
		} else {
			entity = list[0]
		}
		if entity.Type != int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
			return fmt.Errorf("系统配置 %s 未标记为表单配置", key)
		}
		if entity.Status != coreconst.STATUS_STATUS_ENABLE {
			return fmt.Errorf("系统配置 %s 未启用", key)
		}
		if err = runtimeconfig.SaveJSON(c.Cache, key, entity.Value); err != nil {
			return fmt.Errorf("刷新表单系统配置缓存失败: %w", err)
		}
	}
	return nil
}

// wrapRuntimeConfigValidationError 将运行配置 Proto 校验错误转换为可国际化的业务错误。
func wrapRuntimeConfigValidationError(err error) error {
	if validationErr, ok := errors.AsType[*protovalidate.ValidationError](err); ok && len(validationErr.Violations) > 0 {
		violation := validationErr.Violations[0].Proto
		message := violation.GetMessage()
		if message == "" {
			message = "系统配置内容无效"
		}
		messageKey := violation.GetRuleId()
		if messageKey == "" {
			messageKey = "system.admin.runtime_config.invalid_json"
		}
		field := "value"
		fields := make([]string, 0, len(violation.GetField().GetElements()))
		for _, element := range violation.GetField().GetElements() {
			if element.GetFieldName() != "" {
				fields = append(fields, element.GetFieldName())
			}
		}
		if len(fields) > 0 {
			field = strings.Join(fields, ".")
		}
		return errorsx.WithMessageKey(errorsx.InvalidArgument(message), messageKey, map[string]string{"Field": field}).WithCause(err)
	}
	return errorsx.WithMessageKey(errorsx.InvalidArgument("系统配置内容无效"), "system.admin.runtime_config.invalid_json", nil).WithCause(err)
}

// isTranslatableConfigType 判断配置值是否支持机器翻译和动态译文。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT) || configType == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
}

// validateFormConfig 校验表单配置归属、不可变标识和 JSON，并保留未修改的敏感值。
func validateFormConfig(entity, previous *models.BaseConfig) error {
	if previous != nil && previous.Type == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
		if entity.Type != previous.Type || entity.Site != previous.Site || entity.Key != previous.Key {
			return errorsx.WithMessageKey(errorsx.InvalidArgument("表单配置的位置、类型和编码不可修改"), "system.admin.base.config.form.identity", nil)
		}
	}
	if entity.Type != int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM) {
		if entity.Site == _const.BASE_CONFIG_SITE_SYSTEM && runtimeconfig.IsSupportedKey(entity.Key) {
			return errorsx.WithMessageKey(errorsx.InvalidArgument("该配置编码必须使用表单类型"), "system.admin.base.config.form.required", nil)
		}
		return nil
	}
	if entity.Site != _const.BASE_CONFIG_SITE_SYSTEM || !runtimeconfig.IsSupportedKey(entity.Key) {
		return errorsx.WithMessageKey(errorsx.InvalidArgument("请选择已注册的系统配置表单"), "system.admin.base.config.form.unsupported", nil)
	}
	if entity.Status != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.WithMessageKey(errorsx.InvalidArgument("表单配置不允许删除或停用"), "system.admin.base.config.form.protected", nil)
	}
	var err error
	if previous != nil {
		entity.Value, err = runtimeconfig.MergeSensitiveJSON(entity.Key, previous.Value, entity.Value)
		if err != nil {
			return wrapRuntimeConfigValidationError(err)
		}
	}
	err = runtimeconfig.ValidateJSON(entity.Key, entity.Value)
	if err != nil {
		return wrapRuntimeConfigValidationError(err)
	}
	return nil
}
