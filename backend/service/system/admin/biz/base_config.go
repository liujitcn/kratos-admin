package biz

import (
	"context"

	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
)

// BASE_CONFIG_CACHE_PREFIX 表示基础配置缓存键前缀。
const BASE_CONFIG_CACHE_PREFIX = "config:"

// BaseConfigCase 配置业务实例
type BaseConfigCase struct {
	*biz.BaseCase
	*data.BaseConfigRepository
	formMapper *mapper.CopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig]
	mapper     *mapper.CopierMapper[systemadminv1.BaseConfig, models.BaseConfig]
}

// NewBaseConfigCase 创建配置业务实例
func NewBaseConfigCase(baseCase *biz.BaseCase, baseConfigRepo *data.BaseConfigRepository) *BaseConfigCase {
	return &BaseConfigCase{
		BaseCase:             baseCase,
		BaseConfigRepository: baseConfigRepo,
		formMapper:           mapper.NewCopierMapper[systemadminv1.BaseConfigForm, models.BaseConfig](),
		mapper:               mapper.NewCopierMapper[systemadminv1.BaseConfig, models.BaseConfig](),
	}
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
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
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
	for _, item := range list {
		baseConfig := c.mapper.ToDTO(item)
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
	err = c.syncBaseConfigCache(entity)
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

	// 配置键发生变化时，需要先清理旧缓存键。
	if oldConfig.Key != entity.Key {
		err = c.clearBaseConfigCache(oldConfig.Key)
		if err != nil {
			return err
		}
	}

	err = c.syncBaseConfigCache(entity)
	if err != nil {
		return err
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

	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, item := range list {
		err = c.clearBaseConfigCache(item.Key)
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

	// 启用状态需要同步配置缓存，禁用状态则清理缓存。
	if baseConfig.Status == _const.STATUS_ENABLE {
		err = c.syncBaseConfigCache(baseConfig)
		if err != nil {
			return err
		}
	} else {
		err = c.clearBaseConfigCache(baseConfig.Key)
		if err != nil {
			return err
		}
	}

	return nil
}

// RefreshBaseConfig 刷新配置缓存
func (c *BaseConfigCase) RefreshBaseConfig(ctx context.Context) error {
	query := c.Query(ctx).BaseConfig
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Site.Eq(_const.BASE_CONFIG_SITE_SYSTEM)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}

	for _, item := range list {
		err = c.syncBaseConfigCache(item)
		if err != nil {
			return err
		}
	}
	return nil
}

// syncBaseConfigCache 同步单个配置缓存
func (c *BaseConfigCase) syncBaseConfigCache(item *models.BaseConfig) error {
	return sdk.Runtime.GetCache().Set(c.makeBaseConfigCacheKey(item.Key), item.Value, -1)
}

// clearBaseConfigCache 删除单个配置缓存
func (c *BaseConfigCase) clearBaseConfigCache(key string) error {
	return sdk.Runtime.GetCache().Del(c.makeBaseConfigCacheKey(key))
}

// makeBaseConfigCacheKey 生成配置缓存键
func (c *BaseConfigCase) makeBaseConfigCacheKey(key string) string {
	return BASE_CONFIG_CACHE_PREFIX + key
}
